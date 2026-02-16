// Package main содержит точку входа для приложения apk-ci.
// Приложение предназначено для автоматизации процессов разработки и развертывания.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/command/shadowrun"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/di"
	"github.com/Kargones/apk-ci/internal/pkg/logging"
	"github.com/Kargones/apk-ci/internal/pkg/metrics"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	// NR-команды: blank import для self-registration через init()
	_ "github.com/Kargones/apk-ci/internal/command/handlers/converthandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/createstoreshandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/createtempdbhandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/deprecatedaudithandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/executeepfhandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/extensionpublishhandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/forcedisconnecthandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/git2storehandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/help"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/migratehandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/servicemodedisablehandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/servicemodeenablehandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/servicemodestatushandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/sonarqube/projectupdate"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/sonarqube/reportbranch"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/sonarqube/scanbranch"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/sonarqube/scanpr"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/store2dbhandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/storebindhandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/version"
)

// recordMetrics записывает результат выполнения команды и отправляет метрики в Pushgateway.
func recordMetrics(collector metrics.Collector, ctx context.Context, command, infobase string, start time.Time, success bool) {
	collector.RecordCommandEnd(command, infobase, time.Since(start), success)
	_ = collector.Push(ctx) // Ошибки push логируются внутри, не критичны
}

// executeShadowRun выполняет shadow-run: NR-команду и legacy-версию, сравнивает результаты.
// Возвращает exit code: 0 если результаты идентичны, 1 если есть различия.
// NR-результат всегда используется как основной (AC5).
func executeShadowRun(ctx context.Context, cfg *config.Config, l *slog.Logger,
	handler command.Handler, metricsCollector metrics.Collector, start time.Time) int {

	mapping := buildLegacyMapping()
	runner := shadowrun.NewRunner(mapping, l)

	l.Info("Shadow-run режим активирован", slog.String("command", cfg.Command))

	result, nrOutput, nrErr := runner.Execute(ctx, cfg, handler)

	// Записываем метрики
	recordMetrics(metricsCollector, ctx, cfg.Command, cfg.InfobaseName, start, nrErr == nil)

	// Выводим результат в зависимости от формата
	outputFormat := os.Getenv("BR_OUTPUT_FORMAT")
	if strings.EqualFold(outputFormat, "json") {
		// AC6: JSON содержит секцию shadow_run, объединённую с основным выводом NR
		fmt.Print(mergeShadowRunJSON(nrOutput, result))
	} else {
		// Текстовый вывод: NR output + shadow-run summary
		if nrOutput != "" {
			fmt.Print(nrOutput)
		}
		writeShadowRunTextSummary(os.Stdout, result)
	}

	// Приоритет exit code: NR-ошибка (8) > различия (1) > успех (0).
	if nrErr != nil {
		l.Error("Ошибка выполнения команды (shadow-run)",
			slog.String("command", cfg.Command),
			slog.String("error", nrErr.Error()),
			slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
		)
		return 8
	}

	// AC5: exit code = 0 если идентичны, 1 если различия
	if !result.Match {
		return 1
	}
	return 0
}

// mergeShadowRunJSON объединяет JSON-вывод NR-команды с секцией shadow_run.
func mergeShadowRunJSON(nrOutput string, result *shadowrun.ShadowRunResult) string {
	if result == nil {
		return nrOutput
	}

	trimmed := strings.TrimSpace(nrOutput)
	var base map[string]any
	if err := json.Unmarshal([]byte(trimmed), &base); err != nil {
		shadowJSON, marshalErr := json.MarshalIndent(map[string]any{
			"shadow_run": result.ToJSON(),
		}, "", "  ")
		if marshalErr != nil {
			return nrOutput
		}
		return nrOutput + string(shadowJSON) + "\n"
	}

	base["shadow_run"] = result.ToJSON()
	merged, err := json.MarshalIndent(base, "", "  ")
	if err != nil {
		return nrOutput
	}
	return string(merged) + "\n"
}

// writeShadowRunTextSummary выводит текстовый summary shadow-run результата.
func writeShadowRunTextSummary(w io.Writer, result *shadowrun.ShadowRunResult) {
	if result == nil {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "--- Shadow-run сравнение ---")
	if result.Warning != "" {
		fmt.Fprintf(w, "[WARNING] %s\n", result.Warning)
	}
	if result.Match {
		fmt.Fprintln(w, "[OK] Результаты идентичны")
	} else {
		fmt.Fprintln(w, "[DIFF] Обнаружены различия:")
		for _, d := range result.Differences {
			fmt.Fprintf(w, "  [%s] NR: %s | Legacy: %s\n", d.Field, d.NRValue, d.LegacyValue)
		}
	}
	fmt.Fprintf(w, "[TIME] NR: %s | Legacy: %s\n",
		result.NRDuration.Round(time.Millisecond),
		result.LegacyDuration.Round(time.Millisecond))
	fmt.Fprintln(w, "----------------------------")
}

func main() {
	os.Exit(run())
}

// run содержит основную логику приложения и возвращает exit code.
// Все команды маршрутизируются через command registry.
func run() int {
	var err error
	ctx := context.Background()
	cfg, err := config.MustLoad()
	if err != nil || cfg == nil {
		fmt.Fprintf(os.Stderr, "Не удалось загрузить конфигурацию приложения: %v\n", err)
		return 5
	}
	l := cfg.Logger
	l.Debug("Информация о сборке",
		slog.String("version", constants.Version),
		slog.String("commit_hash", constants.PreCommitHash),
	)

	// Пустая команда → help
	if cfg.Command == "" {
		cfg.Command = "help"
	}

	// Генерируем trace_id для корреляции логов
	traceID := tracing.GenerateTraceID()
	ctx = tracing.WithTraceID(ctx, traceID)
	ctx = tracing.ContextWithOTelTraceID(ctx, traceID)

	logAdapter := logging.NewSlogAdapter(l)
	metricsCollector := di.ProvideMetricsCollector(cfg, logAdapter)

	// Инициализация OpenTelemetry трейсинга
	tracerShutdown := di.ProvideTracerProvider(cfg, logAdapter)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tracerShutdown(shutdownCtx); err != nil {
			l.Error("ошибка завершения tracing",
				slog.String("error", err.Error()),
				slog.String("trace_id", traceID),
				slog.String("command", cfg.Command),
			)
		}
	}()

	// Создаём root span
	tracer := otel.Tracer("apk-ci")
	ctx, span := tracer.Start(ctx, cfg.Command,
		trace.WithAttributes(
			attribute.String("command", cfg.Command),
			attribute.String("infobase", cfg.InfobaseName),
			attribute.String("trace_id", traceID),
		),
	)
	defer span.End()

	// Записываем начало выполнения команды
	metricsCollector.RecordCommandStart(cfg.Command, cfg.InfobaseName)
	start := time.Now()

	// Все команды маршрутизируются через registry
	handler, ok := command.Get(cfg.Command)
	if !ok {
		l.Error("неизвестная команда",
			slog.String("BR_ACTION", cfg.Command),
			slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
		)
		recordMetrics(metricsCollector, ctx, cfg.Command, cfg.InfobaseName, start, false)
		return 2
	}

	l.Debug("Выполнение команды через registry", slog.String("command", cfg.Command))

	// Shadow-run: если BR_SHADOW_RUN=true — выполняем обе версии и сравниваем
	if shadowrun.IsEnabled() {
		return executeShadowRun(ctx, cfg, l, handler, metricsCollector, start)
	}

	// Выполнение через registry
	execErr := handler.Execute(ctx, cfg)

	// Записываем завершение и отправляем метрики
	recordMetrics(metricsCollector, ctx, cfg.Command, cfg.InfobaseName, start, execErr == nil)

	if execErr != nil {
		l.Error("Ошибка выполнения команды",
			slog.String("command", cfg.Command),
			slog.String("error", execErr.Error()),
			slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
		)
		return 8
	}
	return 0
}
