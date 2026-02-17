// Package main содержит точку входа для приложения apk-ci.
// Приложение предназначено для автоматизации процессов разработки и развертывания.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/alerting"
	"github.com/Kargones/apk-ci/internal/di"
	"github.com/Kargones/apk-ci/internal/pkg/logging"
	"github.com/Kargones/apk-ci/internal/pkg/metrics"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/Kargones/apk-ci/internal/command/handlers"
)

// recordMetrics записывает результат выполнения команды и отправляет метрики в Pushgateway.
func recordMetrics(collector metrics.Collector, ctx context.Context, command, infobase string, start time.Time, success bool) {
	collector.RecordCommandEnd(command, infobase, time.Since(start), success)
	if pushErr := collector.Push(ctx); pushErr != nil {
		slog.Warn("failed to push metrics", slog.String("error", pushErr.Error()))
	}
}

func main() {
	os.Exit(run())
}

// run содержит основную логику приложения и возвращает exit code.
// Все команды маршрутизируются через command registry.
func run() int {
	// Explicit handler registration (replaces init()-based blank imports).
	handlers.RegisterAll()

	var err error
	ctx := context.Background()
	cfg, err := config.MustLoad()
	if err != nil || cfg == nil {
		fmt.Fprintf(os.Stderr, "Не удалось загрузить конфигурацию приложения: %v\n", err)
		return 5
	}
	l := cfg.Logger
	slog.SetDefault(l)
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

	// Инициализация alerter для отправки алертов при ошибках
	alerter := di.ProvideAlerter(cfg, logAdapter)
	cfg.Alerter = alerter

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

		if sendErr := alerter.Send(ctx, alerting.Alert{
			ErrorCode: "UNKNOWN_COMMAND",
			Message:   fmt.Sprintf("Неизвестная команда: %s", cfg.Command),
			Command:   cfg.Command,
			Infobase:  cfg.InfobaseName,
			TraceID:   traceID,
			Timestamp: time.Now(),
			Severity:  alerting.SeverityWarning,
		}); sendErr != nil {
			l.Warn("failed to send alert", slog.String("error", sendErr.Error()))
		}

		return 2
	}

	l.Debug("Выполнение команды через registry", slog.String("command", cfg.Command))

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

		// Отправляем алерт о неудачном выполнении команды
		if sendErr := alerter.Send(ctx, alerting.Alert{
			ErrorCode: "COMMAND_FAILED",
			Message:   fmt.Sprintf("Команда %s завершилась с ошибкой: %s", cfg.Command, execErr.Error()),
			Command:   cfg.Command,
			Infobase:  cfg.InfobaseName,
			TraceID:   traceID,
			Timestamp: time.Now(),
			Severity:  alerting.SeverityCritical,
		}); sendErr != nil {
			l.Warn("failed to send alert", slog.String("error", sendErr.Error()))
		}

		return 8
	}
	return 0
}
