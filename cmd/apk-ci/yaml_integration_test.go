//go:build integration

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
	"github.com/Kargones/apk-ci/internal/di"
	"github.com/Kargones/apk-ci/internal/pkg/logging"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// allEnvKeys — все переменные окружения, которые могут влиять на конфигурацию.
var allEnvKeys = []string{
	"INPUT_ACTOR", "INPUT_COMMAND", "INPUT_LOGLEVEL", "INPUT_DBNAME",
	"INPUT_REPOSITORY", "INPUT_GITEAURL", "INPUT_ACCESSTOKEN",
	"INPUT_CONFIGSYSTEM", "INPUT_CONFIGPROJECT", "INPUT_CONFIGSECRET",
	"INPUT_CONFIGDBDATA", "INPUT_TERMINATESESSIONS", "INPUT_FORCE_UPDATE",
	"INPUT_ISSUENUMBER", "INPUT_MENUMAIN", "INPUT_MENUDEBUG",
	"INPUT_STARTEPF", "INPUT_BRANCHFORSCAN", "INPUT_COMMITHASH",
	"WorkDir", "TmpDir", "RepPath", "Connect_String",
	"BR_ACTOR", "BR_ENV", "BR_COMMAND", "BR_INFOBASE_NAME",
	"BR_TERMINATE_SESSIONS", "BR_FORCE_UPDATE", "BR_ISSUE_NUMBER",
	"BR_START_EPF", "BR_ACCESS_TOKEN", "BR_CONFIG_SYSTEM",
	"BR_CONFIG_PROJECT", "BR_CONFIG_SECRET", "BR_CONFIG_DBDATA",
	"BR_CONFIG_MENU_MAIN", "BR_CONFIG_MENU_DEBUG",
	"BR_OUTPUT_FORMAT", "BR_SOURCE", "BR_TARGET", "BR_DIRECTION",
	"GITHUB_REF_NAME", "GITHUB_SERVER_URL", "GITHUB_REPOSITORY",
	"GIT_TIMEOUT",
}

// setFullActionEnv устанавливает полный набор переменных окружения,
// эквивалентный запуску через Gitea Actions с action.yaml@v0.0.4.
//
// Параметры соответствуют defaults из action.yaml:
//   configSystem  → gitops-tools/gitops_congif/app.yaml
//   configProject → project.yaml (из текущего репозитория)
//   configSecret  → gitops-tools/gitops_congif/secret.yaml
//   configDbData  → gitops-tools/gitops_congif/dbconfig.yaml
//   menuMain/Debug → gitops-tools/gitops_congif/menu_*.yaml
//   startEpf      → gitops-tools/gitops_congif/start.epf
func setFullActionEnv(t *testing.T, cmd string) {
	t.Helper()
	saved := saveEnvironment(allEnvKeys)
	t.Cleanup(func() { restoreEnvironment(saved) })
	clearTestEnv()

	giteaURL := envOrDef("GITEA_URL", "https://git.apkholding.ru")
	token := os.Getenv("GITEA_TOKEN")
	if token == "" {
		t.Skip("GITEA_TOKEN не установлен")
	}
	repo := envOrDef("GITEA_REPO", "test/TOIR3")
	actor := envOrDef("GITEA_ACTOR", "xor")
	ref := envOrDef("GITEA_REF_NAME", "v18")
	configBase := giteaURL + "/api/v1/repos/gitops-tools/gitops_congif/contents"

	vars := map[string]string{
		"INPUT_GITEAURL":            giteaURL,
		"INPUT_REPOSITORY":          repo,
		"INPUT_ACCESSTOKEN":         token,
		"INPUT_COMMAND":             cmd,
		"INPUT_LOGLEVEL":            "Debug",
		"INPUT_ACTOR":               actor,
		"INPUT_CONFIGSYSTEM":        configBase + "/app.yaml?ref=main",
		"INPUT_CONFIGPROJECT":       "project.yaml",
		"INPUT_CONFIGSECRET":        configBase + "/secret.yaml?ref=main",
		"INPUT_CONFIGDBDATA":        configBase + "/dbconfig.yaml?ref=main",
		"INPUT_MENUMAIN":            configBase + "/menu_main.yaml?ref=main",
		"INPUT_MENUDEBUG":           configBase + "/menu_debug.yaml?ref=main",
		"INPUT_STARTEPF":            configBase + "/start.epf?ref=main",
		"INPUT_TERMINATESESSIONS":   "true",
		"INPUT_FORCE_UPDATE":        "false",
		"GITHUB_REF_NAME":           ref,
		"GITHUB_SERVER_URL":         giteaURL,
		"BR_OUTPUT_FORMAT":          "text",
	}
	for k, v := range vars {
		os.Setenv(k, v)
	}
}

func envOrDef(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// executeCommand воспроизводит логику run() из main.go для одной команды:
// RegisterAll → MustLoad → Get handler → Execute.
// Возвращает exit code аналогично run().
func executeCommand(t *testing.T, ctx context.Context, cmdName string) int {
	t.Helper()

	// Регистрация handlers (idempotent — повторная регистрация возвращает ошибку, но это ОК)
	_ = handlers.RegisterAll()

	// Загрузка конфигурации (как в main.go)
	cfg, err := config.MustLoad(ctx)
	if err != nil || cfg == nil {
		t.Logf("[%s] config.MustLoad error: %v", cmdName, err)
		return 5
	}
	l := cfg.Logger
	slog.SetDefault(l)

	l.Debug("Информация о сборке",
		slog.String("version", constants.Version),
		slog.String("commit_hash", constants.PreCommitHash),
	)

	if cfg.Command == "" {
		cfg.Command = "help"
	}

	// Trace ID
	traceID := tracing.GenerateTraceID()
	ctx = tracing.WithTraceID(ctx, traceID)
	ctx = tracing.ContextWithOTelTraceID(ctx, traceID)

	// Metrics + Alerter + Tracing (как в main.go)
	logAdapter := logging.NewSlogAdapter(l)
	metricsCollector := di.ProvideMetricsCollector(cfg, logAdapter)
	alerter := di.ProvideAlerter(cfg, logAdapter)
	cfg.Alerter = alerter

	tracerShutdown := di.ProvideTracerProvider(cfg, logAdapter)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tracerShutdown(shutdownCtx)
	}()

	tracer := otel.Tracer("apk-ci")
	ctx, span := tracer.Start(ctx, cfg.Command,
		trace.WithAttributes(
			attribute.String("command", cfg.Command),
			attribute.String("infobase", cfg.InfobaseName),
			attribute.String("trace_id", traceID),
		),
	)
	defer span.End()

	metricsCollector.RecordCommandStart(cfg.Command, cfg.InfobaseName)
	start := time.Now()

	// Получаем handler из registry
	handler, ok := command.Get(cfg.Command)
	if !ok {
		t.Logf("[%s] команда не найдена в registry", cfg.Command)
		recordMetrics(ctx, metricsCollector, cfg.Command, cfg.InfobaseName, start, false)
		return 2
	}

	t.Logf("[%s] Owner=%s Repo=%s Project=%s Extensions=%v",
		cfg.Command, cfg.Owner, cfg.Repo, cfg.ProjectName, cfg.AddArray)

	// Выполнение
	execErr := handler.Execute(ctx, cfg)
	recordMetrics(ctx, metricsCollector, cfg.Command, cfg.InfobaseName, start, execErr == nil)

	duration := time.Since(start)
	if execErr != nil {
		t.Logf("[%s] ✗ Error (%v): %v", cfg.Command, duration, execErr)
		return 8
	}
	t.Logf("[%s] ✓ Success (%v)", cfg.Command, duration)
	return 0
}

// TestMain_WithRealYamlFile последовательно выполняет полные цепочки
// convert → git2store → extension-publish, эквивалентные workflow:
//
//   step 1: apk-ci command=nr-convert      (конвертация EDT→XML)
//   step 2: apk-ci command=nr-git2store    (перенос в хранилище 1C)
//   step 3: apk-ci command=nr-extension-publish (публикация расширений)
//
// Каждый шаг:
//  - устанавливает полный набор INPUT_* переменных (как action.yaml)
//  - загружает конфигурацию через config.MustLoad (app.yaml, project.yaml, secret.yaml, dbconfig.yaml)
//  - анализирует проект (AnalyzeProject → ProjectName + Extensions)
//  - выполняет команду через command registry
//
// Между шагами выполняется очистка /tmp/4del (как в workflow).
func TestMain_WithRealYamlFile(t *testing.T) {
	steps := []struct {
		name    string
		command string
	}{
		{"1. Конвертация (nr-convert)", constants.ActNRConvert},
		{"2. Перенос в хранилище (nr-git2store)", constants.ActNRGit2store},
		{"3. Публикация расширений (nr-extension-publish)", constants.ActNRExtensionPublish},
	}

	for i, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			// Устанавливаем окружение для текущей команды
			setFullActionEnv(t, step.command)

			ctx := context.Background()
			exitCode := executeCommand(t, ctx, step.command)

			t.Logf("[%s] exit code: %d", step.command, exitCode)

			// Очистка между шагами (эмуляция workflow step)
			if i < len(steps)-1 {
				cleanupDir := "/tmp/4del"
				if info, err := os.Stat(cleanupDir); err == nil && info.IsDir() {
					entries, _ := os.ReadDir(cleanupDir)
					t.Logf("Очистка %s (%d элементов)", cleanupDir, len(entries))
					// Не удаляем реально в тесте — только логируем
				}
			}
		})
	}
}
