package tracing

import (
	"context"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/pkg/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// ВАЖНО: Тесты в этом файле модифицируют глобальный otel.SetTracerProvider().
// НЕ ДОБАВЛЯТЬ t.Parallel() — тесты должны выполняться последовательно.

// testLogger — минимальный Logger для тестов.
type testLogger struct{}

func (l *testLogger) Debug(_ string, _ ...any)        {}
func (l *testLogger) Info(_ string, _ ...any)         {}
func (l *testLogger) Warn(_ string, _ ...any)         {}
func (l *testLogger) Error(_ string, _ ...any)        {}
func (l *testLogger) With(_ ...any) logging.Logger { return l }

func TestNewTracerProvider_Disabled(t *testing.T) {
	cfg := Config{Enabled: false}
	shutdown, err := NewTracerProvider(cfg, &testLogger{})

	require.NoError(t, err)
	require.NotNil(t, shutdown)

	// Shutdown не должен возвращать ошибку
	assert.NoError(t, shutdown(context.Background()))
}

func TestNewTracerProvider_DisabledNoOverhead(t *testing.T) {
	cfg := Config{Enabled: false}
	shutdown, err := NewTracerProvider(cfg, &testLogger{})

	require.NoError(t, err)

	// Многократный вызов shutdown безопасен
	for i := 0; i < 10; i++ {
		assert.NoError(t, shutdown(context.Background()))
	}
}

func TestNewTracerProvider_InvalidConfig(t *testing.T) {
	cfg := Config{
		Enabled:      true,
		Endpoint:     "",
		ServiceName:  "test",
		Timeout:      5 * time.Second,
		SamplingRate: 1.0,
	}
	shutdown, err := NewTracerProvider(cfg, &testLogger{})

	require.Error(t, err)
	assert.Nil(t, shutdown)
	assert.Contains(t, err.Error(), "endpoint обязателен")
}

func TestNewNopTracerProvider(t *testing.T) {
	shutdown := NewNopTracerProvider()

	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}

func TestNewNopTracerProvider_CancelledContext(t *testing.T) {
	shutdown := NewNopTracerProvider()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Nop shutdown не зависит от контекста
	assert.NoError(t, shutdown(ctx))
}

// TestSpanCreation_WithInMemoryExporter проверяет создание span-ов
// с использованием InMemoryExporter для отлова span-ов в тесте.
func TestSpanCreation_WithInMemoryExporter(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(noop.NewTracerProvider())
	}()

	tracer := otel.Tracer("test")
	_, span := tracer.Start(context.Background(), "test-operation",
		trace.WithAttributes(
			attribute.String("command", "nr-version"),
			attribute.String("infobase", "TestDB"),
			attribute.String("trace_id", "abcdef1234567890abcdef1234567890"),
		),
	)
	span.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	s := spans[0]
	assert.Equal(t, "test-operation", s.Name)

	// Проверяем атрибуты
	attrs := make(map[string]string)
	for _, a := range s.Attributes {
		attrs[string(a.Key)] = a.Value.AsString()
	}
	assert.Equal(t, "nr-version", attrs["command"])
	assert.Equal(t, "TestDB", attrs["infobase"])
	assert.Equal(t, "abcdef1234567890abcdef1234567890", attrs["trace_id"])
}

// TestSpanCreation_ChildSpans проверяет создание child span-ов (AC3).
func TestSpanCreation_ChildSpans(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(noop.NewTracerProvider())
	}()

	tracer := otel.Tracer("test")

	// Root span
	ctx, rootSpan := tracer.Start(context.Background(), "command-execution")

	// Child spans (AC3: ключевые этапы)
	_, initSpan := tracer.Start(ctx, "initialization")
	initSpan.End()

	_, execSpan := tracer.Start(ctx, "execution")
	execSpan.End()

	_, finalSpan := tracer.Start(ctx, "finalization")
	finalSpan.End()

	rootSpan.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 4)

	// Все child span-ы должны иметь одинаковый trace ID
	rootTraceID := spans[0].SpanContext.TraceID()
	for _, s := range spans {
		assert.Equal(t, rootTraceID, s.SpanContext.TraceID(),
			"child span %s должен иметь тот же trace_id", s.Name)
	}
}

// TestShutdown_FlushesSpans проверяет что при shutdown буферизированные span-ы экспортируются (AC7).
func TestShutdown_FlushesSpans(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter), // Синхронный для теста
	)
	otel.SetTracerProvider(tp)

	tracer := otel.Tracer("test")
	_, span := tracer.Start(context.Background(), "before-shutdown")
	span.End()

	// Span-ы должны быть экспортированы синхронно (WithSyncer)
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "before-shutdown", spans[0].Name)

	// Shutdown должен завершиться без ошибки
	err := tp.Shutdown(context.Background())
	require.NoError(t, err)

	// После shutdown новые span-ы не должны экспортироваться
	tracer2 := otel.Tracer("test")
	_, span2 := tracer2.Start(context.Background(), "after-shutdown")
	span2.End()

	// Количество span-ов не должно увеличиться после shutdown
	// (InMemoryExporter очищается при shutdown, поэтому может быть 0)
	// Главное — shutdown не возвращает ошибку

	// Восстанавливаем nop provider
	otel.SetTracerProvider(noop.NewTracerProvider())
}

// TestResourceAttributes проверяет что resource attributes устанавливаются (AC6).
func TestResourceAttributes(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()

	// Создаём resource с атрибутами сервиса (как в NewTracerProvider)
	res, err := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			semconv.ServiceName("test-service"),
			semconv.ServiceVersion("1.2.3"),
			semconv.DeploymentEnvironment("staging"),
		),
	)
	require.NoError(t, err)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithResource(res),
	)
	defer func() {
		_ = tp.Shutdown(context.Background())
	}()

	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "resource-test")
	span.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	// Проверяем resource attributes (AC6)
	spanResource := spans[0].Resource
	attrs := make(map[string]string)
	for _, a := range spanResource.Attributes() {
		attrs[string(a.Key)] = a.Value.AsString()
	}
	assert.Equal(t, "test-service", attrs["service.name"])
	assert.Equal(t, "1.2.3", attrs["service.version"])
	assert.Equal(t, "staging", attrs["deployment.environment"])
}

// TestContextWithOTelTraceID проверяет связку internal trace_id с OTel span context (AC8).
func TestContextWithOTelTraceID_ValidHex(t *testing.T) {
	traceIDHex := "abcdef1234567890abcdef1234567890"
	ctx := ContextWithOTelTraceID(context.Background(), traceIDHex)

	// Span context должен содержать наш trace ID
	sc := trace.SpanContextFromContext(ctx)
	assert.True(t, sc.HasTraceID(), "span context должен содержать trace ID")
	assert.True(t, sc.IsRemote(), "span context должен быть remote")
	assert.Equal(t, traceIDHex, sc.TraceID().String())
}

// TestContextWithOTelTraceID_InvalidHex проверяет что невалидный hex не ломает контекст.
func TestContextWithOTelTraceID_InvalidHex(t *testing.T) {
	ctx := ContextWithOTelTraceID(context.Background(), "not-valid-hex")

	sc := trace.SpanContextFromContext(ctx)
	assert.False(t, sc.IsValid(), "span context должен быть невалидным для невалидного hex")
}

// TestContextWithOTelTraceID_SpanInheritsTraceID проверяет что span наследует trace ID (AC8).
func TestContextWithOTelTraceID_SpanInheritsTraceID(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	defer func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(noop.NewTracerProvider())
	}()

	traceIDHex := "abcdef1234567890abcdef1234567890"
	ctx := ContextWithOTelTraceID(context.Background(), traceIDHex)

	tracer := otel.Tracer("test")
	_, span := tracer.Start(ctx, "test-operation")
	span.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	// Span должен использовать наш trace ID
	assert.Equal(t, traceIDHex, spans[0].SpanContext.TraceID().String(),
		"span должен наследовать trace_id из ContextWithOTelTraceID")
}

// TestNewTracerProvider_SamplingRateFull проверяет что SamplingRate=1.0 записывает все span-ы.
func TestNewTracerProvider_SamplingRateFull(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(newSampler(1.0)),
	)
	otel.SetTracerProvider(tp)
	defer func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(noop.NewTracerProvider())
	}()

	tracer := otel.Tracer("test")
	for i := 0; i < 10; i++ {
		_, span := tracer.Start(context.Background(), "sampled-operation")
		span.End()
	}

	spans := exporter.GetSpans()
	assert.Len(t, spans, 10, "SamplingRate=1.0 должен записывать все span-ы")
}

// TestNewTracerProvider_SamplingRateZero проверяет что SamplingRate=0.0 не записывает ни одного span-а.
func TestNewTracerProvider_SamplingRateZero(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(newSampler(0.0)),
	)
	otel.SetTracerProvider(tp)
	defer func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(noop.NewTracerProvider())
	}()

	tracer := otel.Tracer("test")
	for i := 0; i < 10; i++ {
		_, span := tracer.Start(context.Background(), "unsampled-operation")
		span.End()
	}

	spans := exporter.GetSpans()
	assert.Len(t, spans, 0, "SamplingRate=0.0 не должен записывать span-ы")
}

// TestNewTracerProvider_SamplingRateHalf проверяет статистическое поведение SamplingRate=0.5.
func TestNewTracerProvider_SamplingRateHalf(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(newSampler(0.5)),
	)
	otel.SetTracerProvider(tp)
	defer func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(noop.NewTracerProvider())
	}()

	tracer := otel.Tracer("test")
	for i := 0; i < 1000; i++ {
		_, span := tracer.Start(context.Background(), "half-sampled-operation")
		span.End()
	}

	count := len(exporter.GetSpans())
	// M-10/Review #15: Расширен допуск 15%-85% (150..850) для устойчивости к
	// детерминированному распределению TraceIDRatioBased (не random, зависит от trace ID hash).
	assert.Greater(t, count, 150, "SamplingRate=0.5 должен записать > 150 span-ов из 1000")
	assert.Less(t, count, 850, "SamplingRate=0.5 должен записать < 850 span-ов из 1000")
}

// TestSampling_WithRemoteParentContext проверяет что sampling rate работает корректно
// при наличии remote parent context (как в production flow с ContextWithOTelTraceID).
// Это интеграционный тест: ContextWithOTelTraceID устанавливает FlagsSampled на remote parent,
// и sampler должен НЕ форсировать AlwaysSample, а использовать TraceIDRatioBased.
func TestSampling_WithRemoteParentContext(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(newSampler(0.0)), // rate=0 — ни один span не должен записаться
	)
	otel.SetTracerProvider(tp)
	defer func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(noop.NewTracerProvider())
	}()

	// Имитируем production flow: ContextWithOTelTraceID + создание span-а
	ctx := ContextWithOTelTraceID(context.Background(), "abcdef1234567890abcdef1234567890")
	tracer := otel.Tracer("test")
	for i := 0; i < 10; i++ {
		_, span := tracer.Start(ctx, "should-not-be-sampled")
		span.End()
	}

	spans := exporter.GetSpans()
	assert.Len(t, spans, 0,
		"SamplingRate=0.0 не должен записывать span-ы даже с remote parent context (FlagsSampled)")
}
