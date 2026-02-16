package shadowrun

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHandler реализует command.Handler для тестов.
type mockHandler struct {
	name   string
	execFn func(ctx context.Context, cfg *config.Config) error
}

func (m *mockHandler) Name() string        { return m.name }
func (m *mockHandler) Description() string { return "mock handler" }
func (m *mockHandler) Execute(ctx context.Context, cfg *config.Config) error {
	if m.execFn != nil {
		return m.execFn(ctx, cfg)
	}
	return nil
}

func TestRunner_Execute_IdenticalResults(t *testing.T) {
	mapping := NewLegacyMapping()
	mapping.Register("nr-test-cmd", func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
		return nil
	})

	runner := NewRunner(mapping, slog.Default())
	handler := &mockHandler{
		name: "nr-test-cmd",
		execFn: func(ctx context.Context, cfg *config.Config) error {
			return nil
		},
	}

	result, _, nrErr := runner.Execute(context.Background(), &config.Config{}, handler)
	require.NoError(t, nrErr)
	require.NotNil(t, result)
	assert.True(t, result.Enabled)
	assert.True(t, result.Match)
	assert.Empty(t, result.Differences)
	assert.Empty(t, result.Warning)
	assert.Greater(t, result.NRDuration.Nanoseconds(), int64(0))
	assert.Greater(t, result.LegacyDuration.Nanoseconds(), int64(0))
}

func TestRunner_Execute_DifferentErrors(t *testing.T) {
	mapping := NewLegacyMapping()
	mapping.Register("nr-test-cmd", func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
		return nil // Legacy — успех
	})

	runner := NewRunner(mapping, slog.Default())
	handler := &mockHandler{
		name: "nr-test-cmd",
		execFn: func(ctx context.Context, cfg *config.Config) error {
			return errors.New("nr error") // NR — ошибка
		},
	}

	result, _, nrErr := runner.Execute(context.Background(), &config.Config{}, handler)
	require.Error(t, nrErr) // Основная ошибка — от NR
	require.NotNil(t, result)
	assert.True(t, result.Enabled)
	assert.False(t, result.Match)
	assert.NotEmpty(t, result.Differences)
	assert.Equal(t, "nr error", result.NRError)
	assert.Empty(t, result.LegacyError)
}

func TestRunner_Execute_NoLegacyMapping(t *testing.T) {
	mapping := NewLegacyMapping() // Пустой маппинг

	runner := NewRunner(mapping, slog.Default())
	handler := &mockHandler{
		name: "nr-version",
		execFn: func(ctx context.Context, cfg *config.Config) error {
			return nil
		},
	}

	result, _, nrErr := runner.Execute(context.Background(), &config.Config{}, handler)
	require.NoError(t, nrErr)
	require.NotNil(t, result)
	assert.True(t, result.Enabled)
	assert.True(t, result.Match) // Нет legacy — считаем match
	assert.Contains(t, result.Warning, "не найдена")
}

func TestRunner_Execute_BothErrors(t *testing.T) {
	mapping := NewLegacyMapping()
	mapping.Register("nr-test-cmd", func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
		return errors.New("legacy error")
	})

	runner := NewRunner(mapping, slog.Default())
	handler := &mockHandler{
		name: "nr-test-cmd",
		execFn: func(ctx context.Context, cfg *config.Config) error {
			return errors.New("nr error")
		},
	}

	result, _, nrErr := runner.Execute(context.Background(), &config.Config{}, handler)
	require.Error(t, nrErr)
	assert.Equal(t, "nr error", nrErr.Error())
	require.NotNil(t, result)
	assert.True(t, result.Enabled)
	// Оба вернули ошибку — exit code совпадает, но текст ошибок может различаться
	// В данном случае вывод пустой у обоих — match по output, но не по error message (не сравниваем текст)
	assert.Equal(t, "nr error", result.NRError)
	assert.Equal(t, "legacy error", result.LegacyError)
}

func TestRunner_Execute_LegacyError_NRSuccess(t *testing.T) {
	mapping := NewLegacyMapping()
	mapping.Register("nr-test-cmd", func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
		return errors.New("legacy failed") // Legacy ошибка
	})

	runner := NewRunner(mapping, slog.Default())
	handler := &mockHandler{
		name: "nr-test-cmd",
		execFn: func(ctx context.Context, cfg *config.Config) error {
			return nil // NR успех
		},
	}

	result, _, nrErr := runner.Execute(context.Background(), &config.Config{}, handler)
	require.NoError(t, nrErr) // NR вернула nil — основной результат успешный
	require.NotNil(t, result)
	assert.False(t, result.Match) // Различия в error
	assert.NotEmpty(t, result.Differences)
}

func TestIsEnabled_BackwardCompatibility(t *testing.T) {
	// AC10: без BR_SHADOW_RUN — обычное поведение
	t.Setenv(constants.EnvShadowRun, "")
	assert.False(t, IsEnabled(), "пустое значение не должно активировать shadow-run")

	t.Setenv(constants.EnvShadowRun, "false")
	assert.False(t, IsEnabled(), "false не должно активировать shadow-run")

	// Только "true" (case-insensitive) активирует
	t.Setenv(constants.EnvShadowRun, "true")
	assert.True(t, IsEnabled())
}

func TestRunner_Execute_StateChangingWarning(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	mapping := NewLegacyMapping()
	mapping.Register("nr-enable-cmd", func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
		return nil
	})
	mapping.MarkStateChanging("nr-enable-cmd")

	runner := NewRunner(mapping, logger)
	handler := &mockHandler{
		name: "nr-enable-cmd",
		execFn: func(ctx context.Context, cfg *config.Config) error {
			return nil
		},
	}

	result, _, nrErr := runner.Execute(context.Background(), &config.Config{}, handler)
	require.NoError(t, nrErr)
	require.NotNil(t, result)
	assert.True(t, result.Match)

	// Проверяем что warning о state-changing залогирован
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "изменяет состояние")
	assert.Contains(t, logOutput, "nr-enable-cmd")
}

func TestRunner_Execute_NonStateChanging_NoWarning(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	mapping := NewLegacyMapping()
	mapping.Register("nr-status-cmd", func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
		return nil
	})
	// НЕ помечаем как state-changing

	runner := NewRunner(mapping, logger)
	handler := &mockHandler{
		name: "nr-status-cmd",
		execFn: func(ctx context.Context, cfg *config.Config) error {
			return nil
		},
	}

	_, _, nrErr := runner.Execute(context.Background(), &config.Config{}, handler)
	require.NoError(t, nrErr)

	// Проверяем что warning о state-changing НЕ залогирован
	logOutput := logBuf.String()
	assert.NotContains(t, logOutput, "изменяет состояние")
}

// TestCaptureExecution_PanicRecovery проверяет что captureExecution
// корректно восстанавливает os.Stdout при panic внутри fn().
// Review #30: Без defer-восстановления panic в fn() оставляет os.Stdout
// указывающим на закрытый pipe, что ломает весь последующий вывод.
func TestCaptureExecution_PanicRecovery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	origStdout := os.Stdout

	// Проверяем что panic пробрасывается и stdout восстанавливается
	assert.Panics(t, func() {
		_, _ = captureExecution(logger, func() error {
			panic("test panic in fn")
		})
	}, "panic должен пробрасываться из captureExecution")

	// Критическая проверка: os.Stdout должен быть восстановлен
	assert.Equal(t, origStdout, os.Stdout,
		"os.Stdout должен быть восстановлен после panic в fn()")
}

func TestShadowRunResult_ToJSON(t *testing.T) {
	result := &ShadowRunResult{
		Enabled:        true,
		Match:          false,
		NRDuration:     150_000_000, // 150ms
		LegacyDuration: 200_000_000, // 200ms
		Differences: []Difference{
			{Field: "error", NRValue: "err", LegacyValue: "<nil>"},
		},
		Warning: "",
		NRError: "err",
	}

	j := result.ToJSON()
	assert.True(t, j.Enabled)
	assert.False(t, j.Match)
	assert.Equal(t, int64(150), j.NRDurationMs)
	assert.Equal(t, int64(200), j.LegacyDurationMs)
	assert.Len(t, j.Differences, 1)
	assert.Equal(t, "err", j.NRError)
}
