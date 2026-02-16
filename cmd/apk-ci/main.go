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

	"github.com/Kargones/apk-ci/internal/app"
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
// M-6/Review #15: Выделена для устранения дублирования в legacy switch (9 мест → 1 helper).
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
	// При NR-ошибке exit code 8 сигнализирует CI/CD о реальном сбое команды,
	// даже если различие с legacy тоже обнаружено (match=false).
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
// Если NR output — валидный JSON, shadow_run добавляется как поле в корневой объект.
// Если NR output невалиден — shadow_run выводится отдельным JSON-объектом.
// Примечание: unmarshal/marshal через map[string]any не сохраняет порядок полей JSON.
// Согласно RFC 8259 порядок полей не гарантирован, поэтому это приемлемо.
func mergeShadowRunJSON(nrOutput string, result *shadowrun.ShadowRunResult) string {
	if result == nil {
		return nrOutput
	}

	trimmed := strings.TrimSpace(nrOutput)
	var base map[string]any
	if err := json.Unmarshal([]byte(trimmed), &base); err != nil {
		// NR output не является валидным JSON — выводим shadow_run отдельно
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
// Использует plain ASCII маркеры для совместимости с CI/CD и log parsers.
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
// C-1/Review #16: Вынесена из main() чтобы os.Exit() вызывался ПОСЛЕ отработки
// всех defer-ов (tracerShutdown, span.End). Без этого трейсы ошибочных выполнений
// терялись, потому что os.Exit() не выполняет defer.
func run() int {
	var err error
	ctx := context.Background()
	cfg, err := config.MustLoad()
	if err != nil || cfg == nil {
		fmt.Fprintf(os.Stderr, "Не удалось загрузить конфигурацию приложения: %v\n", err)
		return 5
	}
	l := cfg.Logger
	// Логирование информации о версии и коммите на уровне Debug
	l.Debug("Информация о сборке",
		slog.String("version", constants.Version),
		slog.String("commit_hash", constants.PreCommitHash),
	)

	// Пустая команда → help
	if cfg.Command == "" {
		cfg.Command = "help"
	}

	// Генерируем trace_id для корреляции логов (AC8)
	traceID := tracing.GenerateTraceID()
	// Добавляем trace_id в context для handlers
	ctx = tracing.WithTraceID(ctx, traceID)
	// Связываем с OTel span context — все span-ы будут использовать этот trace ID (AC8)
	ctx = tracing.ContextWithOTelTraceID(ctx, traceID)

	// TODO (M-3/Review Epic-6): Перейти на di.InitializeApp(cfg) вместо прямого вызова
	// ProvideMetricsCollector. Сейчас main.go не использует Wire DI (App struct),
	// что создаёт дублирование инициализации при будущей интеграции.

	// M-4/Review #17: Создаём один adapter для переиспользования в providers.
	logAdapter := logging.NewSlogAdapter(l)
	metricsCollector := di.ProvideMetricsCollector(cfg, logAdapter)

	// Инициализация OpenTelemetry трейсинга
	tracerShutdown := di.ProvideTracerProvider(cfg, logAdapter)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tracerShutdown(shutdownCtx); err != nil {
			// L-3/Review #17: Добавлен trace_id и command для корреляции ошибки shutdown.
			l.Error("ошибка завершения tracing",
				slog.String("error", err.Error()),
				slog.String("trace_id", traceID),
				slog.String("command", cfg.Command),
			)
		}
	}()

	// TODO (H-1, H-9/Review #16): Интегрировать Alerter в command handlers.
	// Alerter инициализируется через DI (App.Alerter), но ни один handler
	// не вызывает alerter.Send(). Вся система алертинга (~3000 строк: email, telegram,
	// webhook, rules) — мёртвый код без этой интеграции.
	// Требуется отдельная story для добавления alerter.Send() в error paths.

	// H-3/Review #17: Создаём root span ОДИН РАЗ для обоих путей (registry и legacy).
	// Ранее tracer/span дублировался в каждой ветке.
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

	// Проверяем Registry перед legacy switch
	if handler, ok := command.Get(cfg.Command); ok {
		l.Debug("Выполнение команды через registry", slog.String("command", cfg.Command))

		// Shadow-run: если BR_SHADOW_RUN=true — выполняем обе версии и сравниваем
		if shadowrun.IsEnabled() {
			return executeShadowRun(ctx, cfg, l, handler, metricsCollector, start)
		}

		// Обычное выполнение через registry
		execErr := handler.Execute(ctx, cfg)

		// Записываем завершение и отправляем метрики
		recordMetrics(metricsCollector, ctx, cfg.Command, cfg.InfobaseName, start, execErr == nil)

		if execErr != nil {
			l.Error("Ошибка выполнения команды",
				slog.String("command", cfg.Command),
				slog.String("error", execErr.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			return 8 // Согласованный exit code с legacy switch
		}
		return 0 // Успешное выполнение через registry
	}
	l.Debug("Команда не найдена в registry, fallback на legacy switch", slog.String("command", cfg.Command))

	// H-5/Review #17: recordMetrics вызывается ОДИН РАЗ через defer вместо дублирования
	// в каждом case. Переменная legacyExitCode определяет success/failure для метрик.
	legacyExitCode := 0
	defer func() {
		recordMetrics(metricsCollector, ctx, cfg.Command, cfg.InfobaseName, start, legacyExitCode == 0)
	}()

	switch cfg.Command {
	case constants.ActStore2db:
		err = app.Store2DbWithConfig(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка обновления хранилища",
				slog.String("Сообщение об ошибке", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			legacyExitCode = 7
			return legacyExitCode
		}
		l.Info("Обновление хранилища успешно завершено") // L-1/Review #9
	case constants.ActConvert:
		err = app.Convert(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка конвертации",
				slog.String("Сообщение об ошибке", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			legacyExitCode = 6
			return legacyExitCode
		}
		l.Info("Конвертация успешно завершена")
	// NOTE: ActGit2store ("git2store") обрабатывается через registry
	// как deprecated alias для nr-git2store (см. git2storehandler)
	case constants.ActDbupdate:
		err = app.DbUpdateWithConfig(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка выполнения DbUpdate",
				slog.String("dbName", cfg.InfobaseName),
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			legacyExitCode = 8
			return legacyExitCode
		}
		l.Info("DbUpdate успешно выполнен", "dbName", cfg.InfobaseName)
	case constants.ActDbrestore:
		err = app.DbRestoreWithConfig(&ctx, l, cfg, cfg.InfobaseName)
		if err != nil {
			l.Error("Ошибка выполнения DbRestore",
				slog.String("dbName", cfg.InfobaseName),
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			legacyExitCode = 8
			return legacyExitCode
		}
		l.Info("DbRestore успешно выполнен", "dbName", cfg.InfobaseName)
	case constants.ActionMenuBuildName:
		err = app.ActionMenuBuildWrapper(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка выполнения ActionMenuBuild",
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			legacyExitCode = 8
			return legacyExitCode
		}
		l.Info("ActionMenuBuild успешно выполнен")
	// NOTE: ActStoreBind ("storebind") обрабатывается через registry
	// как deprecated alias для nr-storebind (см. storebindhandler)
	// NOTE: ActCreateTempDb ("create-temp-db") обрабатывается через registry
	// как deprecated alias для nr-create-temp-db (см. createtempdbhandler)
	case constants.ActCreateStores:
		err = app.CreateStoresWrapper(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка создания хранилищ конфигурации",
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			legacyExitCode = 8
			return legacyExitCode
		}
		l.Info("Хранилища конфигурации успешно созданы")
	// NOTE: ActExecuteEpf ("execute-epf") обрабатывается через registry
	// как deprecated alias для nr-execute-epf (см. executeepfhandler)
	// NOTE: ActSQScanBranch ("sq-scan-branch") обрабатывается через registry
	// как deprecated alias для nr-sq-scan-branch (см. sonarqube/scanbranch)
	// NOTE: ActSQScanPR ("sq-scan-pr") обрабатывается через registry
	// как deprecated alias для nr-sq-scan-pr (см. sonarqube/scanpr)
	// NOTE: ActSQProjectUpdate ("sq-project-update") обрабатывается через registry
	// как deprecated alias для nr-sq-project-update (см. sonarqube/projectupdate)
	// NOTE: ActSQReportBranch ("sq-report-branch") обрабатывается через registry
	// как deprecated alias для nr-sq-report-branch (см. sonarqube/reportbranch)
	case constants.ActTestMerge:
		err = app.TestMerge(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка проверки конфликтов слияния",
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			legacyExitCode = 8
			return legacyExitCode
		}
		l.Info("Проверка конфликтов слияния успешно завершена")
	case constants.ActExtensionPublish:
		err = app.ExtensionPublish(&ctx, l, cfg)
		if err != nil {
			l.Error("Ошибка публикации расширения",
				slog.String("error", err.Error()),
				slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
			)
			legacyExitCode = 8
			return legacyExitCode
		}
		l.Info("Публикация расширения успешно завершена")
	default:
		l.Error("неизвестная команда",
			slog.String("BR_ACTION", cfg.Command),
			slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
		)
		legacyExitCode = 2
		return legacyExitCode
	}

	return 0
}
