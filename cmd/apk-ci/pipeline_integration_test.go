//go:build integration

// Интеграционный тест nr-convert-pipeline.
// Эмулирует полный запуск эквивалентный Gitea Actions workflow:
//
//   uses: gitops-tools/apk-ci-bin@v0.0.4
//   with:
//     command: "nr-convert-pipeline" / "nr-convert" / "nr-git2store" / "nr-extension-publish"
//     giteaURL / repository / accessToken / logLevel / actor
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

func baseEnv(token string) map[string]string {
	return map[string]string{
		"INPUT_GITEAURL":    envOr("GITEA_URL", "https://git.apkholding.ru"),
		"INPUT_REPOSITORY":  envOr("GITEA_REPO", "test/TOIR3"),
		"INPUT_ACCESSTOKEN": token,
		"INPUT_LOGLEVEL":    "Debug",
		"INPUT_ACTOR":       envOr("GITEA_ACTOR", "xor"),
		"GITHUB_REF_NAME":   envOr("GITEA_REF_NAME", "v18"),
		"BR_OUTPUT_FORMAT":  "text",
	}
}

// TestIntegration_Pipeline_ConfigLoad — быстрый smoke: загрузка конфига из Gitea.
func TestIntegration_Pipeline_ConfigLoad(t *testing.T) {
	token := skipWithoutToken(t)
	env := baseEnv(token)
	env["INPUT_COMMAND"] = constants.ActNRConvertPipeline
	setTestEnv(t, env)

	if err := handlers.RegisterAll(); err != nil {
		t.Logf("RegisterAll: %v", err)
	}

	cfg, err := config.MustLoad(context.Background())
	if err != nil {
		t.Fatalf("config.MustLoad: %v", err)
	}

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

	if cfg.AppConfig != nil {
		t.Logf("  Bin1cv8=%s EdtCli=%s", cfg.AppConfig.Paths.Bin1cv8, cfg.AppConfig.Paths.EdtCli)
	}
	if cfg.ProjectConfig != nil {
		t.Logf("  StoreDb=%s", cfg.ProjectConfig.StoreDb)
	}
}

// TestIntegration_Pipeline_FullRun — полный запуск nr-convert-pipeline.
func TestIntegration_Pipeline_FullRun(t *testing.T) {
	token := skipWithoutToken(t)
	env := baseEnv(token)
	env["INPUT_COMMAND"] = constants.ActNRConvertPipeline
	setTestEnv(t, env)

	if err := handlers.RegisterAll(); err != nil {
		t.Logf("RegisterAll: %v", err)
	}

	cfg, err := config.MustLoad(context.Background())
	if err != nil {
		t.Fatalf("config.MustLoad: %v", err)
	}
	slog.SetDefault(cfg.Logger)

	handler, ok := command.Get(constants.ActNRConvertPipeline)
	if !ok {
		t.Fatalf("Команда %s не зарегистрирована", constants.ActNRConvertPipeline)
	}

	start := time.Now()
	execErr := handler.Execute(context.Background(), cfg)
	t.Logf("Duration: %v", time.Since(start))

	if execErr != nil {
		t.Logf("Pipeline error (ожидаемо без 1C): %v", execErr)
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
			env := baseEnv(token)
			env["INPUT_COMMAND"] = step.command
			setTestEnv(t, env)

			cfg, err := config.MustLoad(context.Background())
			if err != nil {
				t.Fatalf("config.MustLoad для %s: %v", step.command, err)
			}
			slog.SetDefault(cfg.Logger)

			t.Logf("[%s] Owner=%s Repo=%s Project=%s",
				step.command, cfg.Owner, cfg.Repo, cfg.ProjectName)

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
