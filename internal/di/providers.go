package di

import (
	"context"
	"log/slog"
	"os"

	"github.com/Kargones/apk-ci/internal/adapter/onec"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/alerting"
	"github.com/Kargones/apk-ci/internal/pkg/logging"
	"github.com/Kargones/apk-ci/internal/pkg/metrics"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
)

// ProvideLogger создаёт Logger на основе LoggingConfig из Config.
// Использует logging.NewLogger() для создания SlogAdapter.
//
// Провайдер извлекает настройки из Config.LoggingConfig:
//   - Level: уровень логирования (debug, info, warn, error)
//   - Format: формат вывода (json, text)
//   - Output: куда выводить логи (stderr, file)
//   - FilePath, MaxSize, MaxBackups, MaxAge, Compress: параметры ротации файлов
//
// Если LoggingConfig == nil или поля пусты, используются значения по умолчанию:
//   - Level: "info"
//   - Format: "text"
//   - Output: "stderr" (backward compatible)
func ProvideLogger(cfg *config.Config) logging.Logger {
	logCfg := logging.DefaultConfig()

	// H6 fix: проверяем что значения не пустые перед присвоением
	if cfg != nil && cfg.LoggingConfig != nil {
		if cfg.LoggingConfig.Level != "" {
			logCfg.Level = cfg.LoggingConfig.Level
		}
		if cfg.LoggingConfig.Format != "" {
			logCfg.Format = cfg.LoggingConfig.Format
		}
		if cfg.LoggingConfig.Output != "" {
			logCfg.Output = cfg.LoggingConfig.Output
		}
		if cfg.LoggingConfig.FilePath != "" {
			logCfg.FilePath = cfg.LoggingConfig.FilePath
		}
		// M-9/Review #13: env-default гарантирует ненулевые значения из cleanenv.
		// При явном BR_LOG_MAX_SIZE=0 значение будет проигнорировано (используется default).
		// Это допустимо: размер 0 MB не имеет практического смысла для lumberjack.
		if cfg.LoggingConfig.MaxSize > 0 {
			logCfg.MaxSize = cfg.LoggingConfig.MaxSize
		}
		if cfg.LoggingConfig.MaxBackups > 0 {
			logCfg.MaxBackups = cfg.LoggingConfig.MaxBackups
		}
		if cfg.LoggingConfig.MaxAge > 0 {
			logCfg.MaxAge = cfg.LoggingConfig.MaxAge
		}
		// Compress: env-default:"true" гарантирует true по умолчанию.
		// Передаём значение из config всегда — false может быть задано явно.
		logCfg.Compress = cfg.LoggingConfig.Compress
	}

	return logging.NewLogger(logCfg)
}

// ProvideOutputWriter создаёт OutputWriter на основе BR_OUTPUT_FORMAT.
// Использует output.NewWriter() для создания JSONWriter или TextWriter.
//
// Провайдер читает переменную окружения BR_OUTPUT_FORMAT:
//   - "json": возвращает JSONWriter
//   - "text" или пустая строка: возвращает TextWriter (default)
//
// Не зависит от Config — формат вывода определяется переменной окружения
// для гибкости переключения формата без перезагрузки конфигурации.
func ProvideOutputWriter() output.Writer {
	format := os.Getenv("BR_OUTPUT_FORMAT")
	if format == "" {
		format = output.FormatText
	}
	return output.NewWriter(format)
}

// ProvideTraceID генерирует уникальный trace_id для корреляции логов.
// Использует tracing.GenerateTraceID() для криптографически безопасной генерации.
//
// Формат trace_id: 32-символьный hex string (16 байт).
// Пример: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"
//
// TraceID генерируется один раз при инициализации App
// и используется для корреляции всех логов в рамках одного запуска команды.
func ProvideTraceID() string {
	return tracing.GenerateTraceID()
}

// ProvideFactory создаёт OneCFactory на основе Config.
// Factory реализует Strategy pattern для выбора реализации операций 1C
// (1cv8/ibcmd/native) на основе config.AppConfig.Implementations.
//
// Использование:
//
//	factory := ProvideFactory(cfg)
//	exporter, err := factory.NewConfigExporter()
//	creator, err := factory.NewDatabaseCreator()
//
// Factory позволяет переключаться между реализациями без изменения кода.
func ProvideFactory(cfg *config.Config) *onec.Factory {
	return onec.NewFactory(cfg)
}

// ProvideAlerter создаёт Alerter на основе AlertingConfig из Config.
// Использует alerting.NewAlerter() для создания multi-channel alerter или NopAlerter.
//
// Провайдер извлекает настройки из Config.AlertingConfig:
//   - Enabled: включён ли алертинг (по умолчанию false)
//   - RateLimitWindow: интервал rate limiting
//   - Email: конфигурация email канала
//   - Telegram: конфигурация telegram канала
//
// Если AlertingConfig == nil или Enabled=false, возвращает NopAlerter.
// При ошибке создания Alerter возвращает NopAlerter и логирует ошибку.
//
// TODO: Рефакторинг — объединить config.AlertingConfig и alerting.Config
// в единую структуру для устранения дублирования. Возможные варианты:
// 1. Использовать alerting.Config напрямую в config.AppConfig
// 2. Создать adapter interface для конвертации
// 3. Вынести общие типы в shared пакет
//
// TODO: Alerter инициализируется через DI, но НЕ интегрирован
// в command handlers. Необходимо добавить вызов alerter.Send() в error paths handlers.
// Без интеграции email/telegram/webhook/rules — мёртвый код.
// Реализовать в отдельной story (Epic 7 или отдельный task).
func ProvideAlerter(cfg *config.Config, logger logging.Logger) alerting.Alerter {
	// Если конфигурация отсутствует — возвращаем NopAlerter
	if cfg == nil || cfg.AlertingConfig == nil {
		return alerting.NewNopAlerter()
	}

	// L-2/Review #9: Три уровня валидации (defense-in-depth) — намеренная архитектура:
	// 1. config.validateAlertingConfig() — предварительная проверка обязательных полей при загрузке
	// 2. alerting.Config.Validate() — полная проверка (URL format, CRLF, Header Injection)
	// 3. Каждый канал (EmailConfig/TelegramConfig/WebhookConfig).Validate() — per-channel валидация

	// Конвертируем config.AlertingConfig в alerting.Config
	alertCfg := alerting.Config{
		Enabled:         cfg.AlertingConfig.Enabled,
		RateLimitWindow: cfg.AlertingConfig.RateLimitWindow,
		Email: alerting.EmailConfig{
			Enabled:         cfg.AlertingConfig.Email.Enabled,
			SMTPHost:        cfg.AlertingConfig.Email.SMTPHost,
			SMTPPort:        cfg.AlertingConfig.Email.SMTPPort,
			SMTPUser:        cfg.AlertingConfig.Email.SMTPUser,
			SMTPPassword:    cfg.AlertingConfig.Email.SMTPPassword,
			UseTLS:          cfg.AlertingConfig.Email.UseTLS,
			From:            cfg.AlertingConfig.Email.From,
			To:              cfg.AlertingConfig.Email.To,
			SubjectTemplate: cfg.AlertingConfig.Email.SubjectTemplate,
			Timeout:         cfg.AlertingConfig.Email.Timeout,
		},
		Telegram: alerting.TelegramConfig{
			Enabled:  cfg.AlertingConfig.Telegram.Enabled,
			BotToken: cfg.AlertingConfig.Telegram.BotToken,
			ChatIDs:  cfg.AlertingConfig.Telegram.ChatIDs,
			Timeout:  cfg.AlertingConfig.Telegram.Timeout,
		},
		Webhook: alerting.WebhookConfig{
			Enabled:    cfg.AlertingConfig.Webhook.Enabled,
			URLs:       cfg.AlertingConfig.Webhook.URLs,
			Headers:    cfg.AlertingConfig.Webhook.Headers,
			Timeout:    cfg.AlertingConfig.Webhook.Timeout,
			MaxRetries: cfg.AlertingConfig.Webhook.MaxRetries,
		},
	}

	// Конвертируем config.AlertRulesConfig в alerting.RulesConfig
	rulesCfg := alerting.RulesConfig{
		MinSeverity:       cfg.AlertingConfig.Rules.MinSeverity,
		ExcludeErrorCodes: cfg.AlertingConfig.Rules.ExcludeErrorCodes,
		IncludeErrorCodes: cfg.AlertingConfig.Rules.IncludeErrorCodes,
		ExcludeCommands:   cfg.AlertingConfig.Rules.ExcludeCommands,
		IncludeCommands:   cfg.AlertingConfig.Rules.IncludeCommands,
	}

	if len(cfg.AlertingConfig.Rules.ChannelOverrides) > 0 {
		rulesCfg.Channels = make(map[string]alerting.ChannelRulesConfig, len(cfg.AlertingConfig.Rules.ChannelOverrides))
		for name, ch := range cfg.AlertingConfig.Rules.ChannelOverrides {
			rulesCfg.Channels[name] = alerting.ChannelRulesConfig{
				MinSeverity:       ch.MinSeverity,
				ExcludeErrorCodes: ch.ExcludeErrorCodes,
				IncludeErrorCodes: ch.IncludeErrorCodes,
				ExcludeCommands:   ch.ExcludeCommands,
				IncludeCommands:   ch.IncludeCommands,
			}
		}
	}

	alerter, err := alerting.NewAlerter(alertCfg, rulesCfg, logger)
	if err != nil {
		logger.Error("ошибка создания Alerter, используется NopAlerter",
			slog.String("error", err.Error()),
		)
		return alerting.NewNopAlerter()
	}

	return alerter
}

// ProvideMetricsCollector создаёт Collector на основе MetricsConfig из Config.
// Если MetricsConfig == nil или Enabled=false, возвращает NopCollector.
//
// Провайдер извлекает настройки из Config.MetricsConfig:
//   - Enabled: включены ли метрики (по умолчанию false)
//   - PushgatewayURL: URL Prometheus Pushgateway
//   - JobName: имя job для группировки метрик
//   - Timeout: таймаут HTTP запросов
//   - InstanceLabel: переопределение instance label (или hostname)
//
// При ошибке создания Collector возвращает NopCollector и логирует ошибку.
func ProvideMetricsCollector(cfg *config.Config, logger logging.Logger) metrics.Collector {
	// Если конфигурация отсутствует — возвращаем NopCollector
	if cfg == nil || cfg.MetricsConfig == nil {
		return metrics.NewNopCollector()
	}

	// Конвертируем config.MetricsConfig в metrics.Config
	metricsCfg := metrics.Config{
		Enabled:        cfg.MetricsConfig.Enabled,
		PushgatewayURL: cfg.MetricsConfig.PushgatewayURL,
		JobName:        cfg.MetricsConfig.JobName,
		Timeout:        cfg.MetricsConfig.Timeout,
		InstanceLabel:  cfg.MetricsConfig.InstanceLabel,
	}

	collector, err := metrics.NewCollector(metricsCfg, logger)
	if err != nil {
		logger.Error("ошибка создания MetricsCollector, используется NopCollector",
			slog.String("error", err.Error()),
		)
		return metrics.NewNopCollector()
	}

	return collector
}

// ProvideTracerProvider создаёт и инициализирует OTel TracerProvider.
// Возвращает shutdown function для graceful завершения.
// Если TracingConfig == nil или Enabled=false, возвращает nop shutdown.
// При ошибке создания TracerProvider возвращает nop shutdown и логирует ошибку.
func ProvideTracerProvider(cfg *config.Config, logger logging.Logger) func(context.Context) error {
	// Если конфигурация отсутствует — возвращаем nop shutdown
	if cfg == nil || cfg.TracingConfig == nil {
		return tracing.NewNopTracerProvider()
	}

	// Конвертируем config.TracingConfig в tracing.Config
	tracingCfg := tracing.Config{
		Enabled:      cfg.TracingConfig.Enabled,
		Endpoint:     cfg.TracingConfig.Endpoint,
		ServiceName:  cfg.TracingConfig.ServiceName,
		Version:      constants.Version,
		Environment:  cfg.TracingConfig.Environment,
		Insecure:     cfg.TracingConfig.Insecure,
		Timeout:      cfg.TracingConfig.Timeout,
		SamplingRate: cfg.TracingConfig.SamplingRate,
	}

	shutdown, err := tracing.NewTracerProvider(tracingCfg, logger)
	if err != nil {
		logger.Error("ошибка инициализации tracing, используется nop provider",
			slog.String("error", err.Error()),
		)
		return tracing.NewNopTracerProvider()
	}

	return shutdown
}
