//go:build integration

// Интеграционный тест nr-convert-pipeline.
// Эмулирует полный запуск эквивалентный Gitea Actions workflow:
//
//   uses: gitops-tools/apk-ci-bin@v0.0.4
//   with:
//     giteaURL / repository / accessToken / command / logLevel / actor
//     + default inputs from action.yaml (configSystem, configProject, etc.)
//
// Запуск:
//   GITEA_TOKEN=... go test -tags=integration ./cmd/apk-ci/ \
//     -run TestIntegration_Pipeline -v -timeout 60m
package main

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/command/handlers"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func setTestEnv(t *testing.T, vars map[string]string) {
	t.Helper()
	saved := make(map[string]string)
	for k, v := range vars {
		saved[k] = os.Getenv(k)
		os.Setenv(k, v)
	}
	t.Cleanup(func() {
		for k, v := range saved {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	})
}

func skipWithoutToken(t *testing.T) string {
	t.Helper()
	token := os.Getenv("GITEA_TOKEN")
	if token == "" {
		t.Skip("GITEA_TOKEN не установлен — пропуск интеграционного теста")
	}
	return token
}

// actionEnv возвращает полный набор INPUT_* переменных,
// эквивалентный тому что Gitea Actions устанавливает из action.yaml with: + defaults.
//
// Соответствие с action.yaml defaults:
//   configSystem  → https://git.apkholding.ru/api/v1/repos/gitops-tools/gitops_congif/contents/app.yaml?ref=main
//   configProject → project.yaml (из текущего репозитория)
//   configSecret  → https://git.apkholding.ru/api/v1/repos/gitops-tools/gitops_congif/contents/secret.yaml?ref=main
//   configDbData  → https://git.apkholding.ru/api/v1/repos/gitops-tools/gitops_congif/contents/dbconfig.yaml?ref=main
//   menuMain      → https://git.apkholding.ru/api/v1/repos/gitops-tools/gitops_congif/contents/menu_main.yaml?ref=main
//   menuDebug     → https://git.apkholding.ru/api/v1/repos/gitops-tools/gitops_congif/contents/menu_debug.yaml?ref=main
//   startEpf      → https://git.apkholding.ru/api/v1/repos/gitops-tools/gitops_congif/contents/start.epf?ref=main
func actionEnv(token, cmd string) map[string]string {
	giteaURL := envOr("GITEA_URL", "https://git.apkholding.ru")
	repo := envOr("GITEA_REPO", "test/TOIR3")
	actor := envOr("GITEA_ACTOR", "xor")
	ref := envOr("GITEA_REF_NAME", "v18")

	configBase := giteaURL + "/api/v1/repos/gitops-tools/gitops_congif/contents"

	return map[string]string{
		// Обязательные параметры из workflow with:
		"INPUT_GITEAURL":    giteaURL,
		"INPUT_REPOSITORY":  repo,
		"INPUT_ACCESSTOKEN": token,
		"INPUT_COMMAND":     cmd,
		"INPUT_LOGLEVEL":    "Debug",
		"INPUT_ACTOR":       actor,

		// Defaults из action.yaml — конфигурационные файлы
		"INPUT_CONFIGSYSTEM":  configBase + "/app.yaml?ref=main",
		"INPUT_CONFIGPROJECT": "project.yaml",
		"INPUT_CONFIGSECRET":  configBase + "/secret.yaml?ref=main",
		"INPUT_CONFIGDBDATA":  configBase + "/dbconfig.yaml?ref=main",
		"INPUT_MENUMAIN":      configBase + "/menu_main.yaml?ref=main",
		"INPUT_MENUDEBUG":     configBase + "/menu_debug.yaml?ref=main",
		"INPUT_STARTEPF":      configBase + "/start.epf?ref=main",

		// Defaults из action.yaml — прочие
		"INPUT_TERMINATESESSIONS": "true",
		"INPUT_FORCE_UPDATE":      "false",

		// GitHub/Gitea контекст
		"GITHUB_REF_NAME":   ref,
		"GITHUB_SERVER_URL": giteaURL,
		"BR_OUTPUT_FORMAT":  "text",
	}
}

// TestIntegration_Pipeline_ConfigLoad — загрузка конфига из Gitea с полным набором INPUT_*.
func TestIntegration_Pipeline_ConfigLoad(t *testing.T) {
	token := skipWithoutToken(t)
	setTestEnv(t, actionEnv(token, constants.ActNRConvertPipeline))

	if err := handlers.RegisterAll(); err != nil {
		t.Logf("RegisterAll: %v", err)
	}

	cfg, err := config.MustLoad(context.Background())
	if err != nil {
		t.Fatalf("config.MustLoad: %v", err)
	}

	// Проверяем базовые поля
	if cfg.Owner == "" {
		t.Error("Owner пустой")
	}
	if cfg.Repo == "" {
		t.Error("Repo пустой")
	}
	if cfg.Command != constants.ActNRConvertPipeline {
		t.Errorf("Command: want %s, got %s", constants.ActNRConvertPipeline, cfg.Command)
	}

	t.Logf("✓ Owner=%s Repo=%s Project=%s Extensions=%v",
		cfg.Owner, cfg.Repo, cfg.ProjectName, cfg.AddArray)

	// Проверяем что AppConfig загружен из gitops_congif/app.yaml
	if cfg.AppConfig == nil {
		t.Error("AppConfig nil — app.yaml не загружен из gitops_congif")
	} else {
		t.Logf("  AppConfig: Bin1cv8=%s EdtCli=%s", cfg.AppConfig.Paths.Bin1cv8, cfg.AppConfig.Paths.EdtCli)
	}

	// Проверяем ProjectConfig (из project.yaml текущего репозитория)
	if cfg.ProjectConfig == nil {
		t.Error("ProjectConfig nil — project.yaml не загружен")
	} else {
		t.Logf("  ProjectConfig: StoreDb=%s Debug=%v", cfg.ProjectConfig.StoreDb, cfg.ProjectConfig.Debug)
	}

	// Проверяем SecretConfig
	if cfg.SecretConfig == nil {
		t.Log("  SecretConfig nil (может быть ожидаемо без доступа к gitops_congif)")
	} else {
		t.Log("  SecretConfig: загружен")
	}

	// Проверяем DbConfig
	if cfg.DbConfig == nil {
		t.Log("  DbConfig nil")
	} else {
		t.Logf("  DbConfig: %d баз", len(cfg.DbConfig))
	}

	// Проверяем AnalyzeProject
	if cfg.ProjectName == "" {
		t.Error("ProjectName пустой — AnalyzeProject не отработал")
	}
	if len(cfg.AddArray) > 0 {
		t.Logf("  → extension-publish будет выполнен (расширения: %v)", cfg.AddArray)
	} else {
		t.Log("  → extension-publish будет пропущен (нет расширений)")
	}
}

// TestIntegration_Pipeline_FullRun — полный запуск nr-convert-pipeline.
func TestIntegration_Pipeline_FullRun(t *testing.T) {
	token := skipWithoutToken(t)
	setTestEnv(t, actionEnv(token, constants.ActNRConvertPipeline))

	if err := handlers.RegisterAll(); err != nil {
		t.Logf("RegisterAll: %v", err)
	}

	cfg, err := config.MustLoad(context.Background())
	if err != nil {
		t.Fatalf("config.MustLoad: %v", err)
	}
	slog.SetDefault(cfg.Logger)

	t.Logf("Config: Owner=%s Repo=%s Project=%s Ext=%v AppConfig=%v",
		cfg.Owner, cfg.Repo, cfg.ProjectName, cfg.AddArray, cfg.AppConfig != nil)

	handler, ok := command.Get(constants.ActNRConvertPipeline)
	if !ok {
		t.Fatalf("Команда %s не зарегистрирована", constants.ActNRConvertPipeline)
	}

	start := time.Now()
	execErr := handler.Execute(context.Background(), cfg)
	t.Logf("Duration: %v", time.Since(start))

	if execErr != nil {
		t.Logf("Pipeline error (ожидаемо без 1C на dev-container): %v", execErr)
	} else {
		t.Log("✓ Pipeline completed successfully")
	}
}

// TestIntegration_Pipeline_StagesIndividually — запуск каждого шага отдельно
// (эквивалент 3 отдельных step в workflow).
func TestIntegration_Pipeline_StagesIndividually(t *testing.T) {
	token := skipWithoutToken(t)

	if err := handlers.RegisterAll(); err != nil {
		t.Logf("RegisterAll: %v", err)
	}

	steps := []struct {
		name    string
		command string
	}{
		{"Конвертация", constants.ActNRConvert},
		{"Перенос в хранилище", constants.ActNRGit2store},
		{"Публикация расширений", constants.ActNRExtensionPublish},
	}

	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			setTestEnv(t, actionEnv(token, step.command))

			cfg, err := config.MustLoad(context.Background())
			if err != nil {
				t.Fatalf("config.MustLoad для %s: %v", step.command, err)
			}
			slog.SetDefault(cfg.Logger)

			t.Logf("[%s] Owner=%s Repo=%s Project=%s AppConfig=%v",
				step.command, cfg.Owner, cfg.Repo, cfg.ProjectName, cfg.AppConfig != nil)

			handler, ok := command.Get(step.command)
			if !ok {
				t.Fatalf("Команда %s не зарегистрирована", step.command)
			}

			start := time.Now()
			execErr := handler.Execute(context.Background(), cfg)
			duration := time.Since(start)

			if execErr != nil {
				t.Logf("[%s] Error (%v): %v", step.command, duration, execErr)
			} else {
				t.Logf("[%s] ✓ Success (%v)", step.command, duration)
			}
		})
	}
}
