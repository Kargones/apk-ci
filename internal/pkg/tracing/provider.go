package tracing

import (
	"context"
	"net/url"

	"github.com/Kargones/apk-ci/internal/pkg/logging"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// NewTracerProvider создаёт и настраивает OTel TracerProvider.
// Если трейсинг выключен, возвращает nop shutdown function.
// При включённом трейсинге:
// 1. Создаёт OTLP HTTP exporter
// 2. Настраивает BatchSpanProcessor для асинхронного экспорта
// 3. Устанавливает resource attributes (service.name, version, environment)
// 4. Регистрирует TracerProvider глобально через otel.SetTracerProvider()
// 5. Возвращает shutdown function для graceful завершения
func NewTracerProvider(cfg Config, logger logging.Logger) (func(context.Context) error, error) {
	if !cfg.Enabled {
		logger.Debug("трейсинг выключен, используется nop provider")
		return NewNopTracerProvider(), nil
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	ctx := context.Background()

	// Создать resource с атрибутами сервиса.
	// Используем NewSchemaless для избежания конфликта Schema URL
	// между resource.Default() (SDK schema) и semconv v1.26.0.
	res, err := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.Version),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, err
	}

	// Извлечь host:port из endpoint URL.
	// otlptracehttp.WithEndpoint() принимает только host:port, без path.
	// Путь (например /v1/traces) задаётся через WithURLPath() при необходимости.
	endpointHost := cfg.Endpoint
	if u, parseErr := url.Parse(cfg.Endpoint); parseErr == nil && u.Host != "" {
		endpointHost = u.Host
	}

	// Создать OTLP HTTP exporter
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpointHost),
		otlptracehttp.WithTimeout(cfg.Timeout),
	}
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	// Создать sampler на основе SamplingRate
	sampler := newSampler(cfg.SamplingRate)

	// Создать TracerProvider с BatchSpanProcessor и sampler
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Регистрируем глобально.
	// TODO: Global state — otel.SetTracerProvider() без sync.Once.
	// Для CLI допустимо (однократный вызов), но тесты не могут использовать t.Parallel().
	otel.SetTracerProvider(tp)

	logger.Info("OpenTelemetry трейсинг инициализирован",
		"endpoint", cfg.Endpoint,
		"service_name", cfg.ServiceName,
		"environment", cfg.Environment,
		"sampling_rate", cfg.SamplingRate,
	)

	return tp.Shutdown, nil
}

// ContextWithOTelTraceID создаёт контекст с OTel remote span context,
// содержащим указанный trace ID (AC8).
// Это связывает существующий internal trace_id (из GenerateTraceID())
// с OTel distributed tracing — все span-ы, созданные из этого контекста,
// будут использовать тот же trace ID.
// Если traceIDHex невалидный — возвращает оригинальный контекст без изменений.
func ContextWithOTelTraceID(ctx context.Context, traceIDHex string) context.Context {
	traceID, err := trace.TraceIDFromHex(traceIDHex)
	if err != nil {
		return ctx
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	return trace.ContextWithRemoteSpanContext(ctx, sc)
}

// newSampler создаёт sampler на основе SamplingRate.
// Используется ParentBased wrapper для корректного distributed tracing:
//   - root spans (без parent): решение по TraceIDRatioBased
//   - remote parent (sampled): также TraceIDRatioBased — не форсирует AlwaysSample,
//     потому что ContextWithOTelTraceID устанавливает FlagsSampled на remote parent,
//     и дефолтный ParentBased(RemoteParentSampled=AlwaysSample) игнорировал бы rate
//   - local parent: наследует решение parent span-а (дефолтное поведение)
//
// M-5/Review #17: Это нестандартное поведение (OTel convention: remote parent sampled →
// AlwaysSample). Сделано намеренно для CLI, где ContextWithOTelTraceID устанавливает
// FlagsSampled на все запросы. При интеграции с внешними tracing системами может
// потребоваться возврат к стандартному поведению.
func newSampler(rate float64) sdktrace.Sampler {
	return sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(rate),
		sdktrace.WithRemoteParentSampled(sdktrace.TraceIDRatioBased(rate)),
	)
}
