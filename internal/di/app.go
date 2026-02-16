package di

import (
	"context"

	"github.com/Kargones/apk-ci/internal/adapter/onec"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/pkg/alerting"
	"github.com/Kargones/apk-ci/internal/pkg/logging"
	"github.com/Kargones/apk-ci/internal/pkg/metrics"
	"github.com/Kargones/apk-ci/internal/pkg/output"
)

// App содержит инициализированные зависимости приложения.
// Создаётся через Wire DI в InitializeApp().
//
// Все поля инициализируются через провайдеры в providers.go.
// При добавлении новых зависимостей:
// 1. Добавить поле в App struct
// 2. Создать провайдер в providers.go
// 3. Добавить провайдер в ProviderSet в wire.go
// 4. Перегенерировать wire_gen.go: go generate ./internal/di/...
type App struct {
	// Config содержит конфигурацию приложения.
	// Передаётся извне через InitializeApp().
	Config *config.Config

	// Logger предоставляет структурированное логирование.
	// Создаётся через ProvideLogger на основе LoggingConfig.
	Logger logging.Logger

	// OutputWriter форматирует результаты команд.
	// Создаётся через ProvideOutputWriter на основе BR_OUTPUT_FORMAT.
	OutputWriter output.Writer

	// TraceID содержит уникальный идентификатор для корреляции логов.
	// Генерируется через ProvideTraceID.
	TraceID string

	// OneCFactory создаёт реализации операций 1C на основе конфигурации.
	// Создаётся через ProvideFactory.
	// Реализует Strategy pattern для выбора инструмента (1cv8/ibcmd/native).
	OneCFactory *onec.Factory

	// Alerter отправляет алерты при критических ошибках.
	// Создаётся через ProvideAlerter на основе AlertingConfig.
	// Если алертинг отключён — используется NopAlerter.
	Alerter alerting.Alerter

	// MetricsCollector собирает и отправляет метрики в Prometheus Pushgateway.
	// Создаётся через ProvideMetricsCollector на основе MetricsConfig.
	// Если метрики отключены — используется NopCollector.
	MetricsCollector metrics.Collector

	// TracerShutdown завершает OTel TracerProvider и отправляет буферизированные span-ы.
	// Создаётся через ProvideTracerProvider на основе TracingConfig.
	// Если трейсинг отключён — nop function (нулевой overhead).
	TracerShutdown func(context.Context) error
}
