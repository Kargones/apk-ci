package storebindhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/entity/one/convert"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConvertLoader — mock реализация ConvertLoader для тестов.
type mockConvertLoader struct {
	loadFromConfigFunc func(ctx *context.Context, l *slog.Logger, cfg *config.Config) (*convert.Config, error)
	storeBindFunc      func(cc *convert.Config, ctx *context.Context, l *slog.Logger, cfg *config.Config) error
}

func (m *mockConvertLoader) LoadFromConfig(ctx *context.Context, l *slog.Logger, cfg *config.Config) (*convert.Config, error) {
	if m.loadFromConfigFunc != nil {
		return m.loadFromConfigFunc(ctx, l, cfg)
	}
	// По умолчанию возвращаем минимальную конфигурацию
	return &convert.Config{
		StoreRoot: "tcp://example.com/store",
	}, nil
}

func (m *mockConvertLoader) StoreBind(cc *convert.Config, ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	if m.storeBindFunc != nil {
		return m.storeBindFunc(cc, ctx, l, cfg)
	}
	return nil
}

// newMockConvertLoaderSuccess создаёт mock, возвращающий успешные результаты.
func newMockConvertLoaderSuccess() *mockConvertLoader {
	return &mockConvertLoader{
		loadFromConfigFunc: func(_ *context.Context, _ *slog.Logger, _ *config.Config) (*convert.Config, error) {
			return &convert.Config{
				StoreRoot: "tcp://example.com/store",
			}, nil
		},
		storeBindFunc: func(_ *convert.Config, _ *context.Context, _ *slog.Logger, _ *config.Config) error {
			return nil
		},
	}
}

// newMockConvertLoaderLoadError создаёт mock, возвращающий ошибку при загрузке.
func newMockConvertLoaderLoadError(errMsg string) *mockConvertLoader {
	return &mockConvertLoader{
		loadFromConfigFunc: func(_ *context.Context, _ *slog.Logger, _ *config.Config) (*convert.Config, error) {
			return nil, fmt.Errorf("%s", errMsg)
		},
	}
}

// newMockConvertLoaderBindError создаёт mock, возвращающий ошибку при привязке.
func newMockConvertLoaderBindError(errMsg string) *mockConvertLoader {
	return &mockConvertLoader{
		loadFromConfigFunc: func(_ *context.Context, _ *slog.Logger, _ *config.Config) (*convert.Config, error) {
			return &convert.Config{
				StoreRoot: "tcp://example.com/store",
			}, nil
		},
		storeBindFunc: func(_ *convert.Config, _ *context.Context, _ *slog.Logger, _ *config.Config) error {
			return fmt.Errorf("%s", errMsg)
		},
	}
}

// === Тесты Name и Description ===

func TestStorebindHandler_Name(t *testing.T) {
	h := &StorebindHandler{}
	assert.Equal(t, "nr-storebind", h.Name())
	assert.Equal(t, constants.ActNRStorebind, h.Name())
}

func TestStorebindHandler_Description(t *testing.T) {
	h := &StorebindHandler{}
	desc := h.Description()
	assert.NotEmpty(t, desc)
	assert.Equal(t, "Привязка хранилища конфигурации к базе данных", desc)
}

// === AC-5: Registration и Deprecated Alias ===

func TestStorebindHandler_Registration(t *testing.T) {
	// init() уже вызван при импорте пакета — проверяем что handler зарегистрирован
	h, ok := command.Get("nr-storebind")
	require.True(t, ok, "handler nr-storebind должен быть зарегистрирован в registry")
	assert.Equal(t, constants.ActNRStorebind, h.Name())
}

func TestStorebindHandler_DeprecatedAlias(t *testing.T) {
	// AC-5: Проверяем что deprecated alias "storebind" работает
	h, ok := command.Get("storebind")
	require.True(t, ok, "deprecated alias storebind должен быть зарегистрирован в registry")

	// Проверяем что это DeprecatedBridge
	dep, isDep := h.(command.Deprecatable)
	require.True(t, isDep, "handler должен реализовывать Deprecatable")
	assert.True(t, dep.IsDeprecated())
	assert.Equal(t, "nr-storebind", dep.NewName())
}

// === AC-1: Основной сценарий ===

func TestStorebindHandler_Execute_TextOutput_Success(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	mock := newMockConvertLoaderSuccess()
	h := &StorebindHandler{convertLoader: mock}
	cfg := &config.Config{
		InfobaseName: "TestBase",
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)
	// AC-4: Text output показывает человекочитаемую информацию
	assert.Contains(t, out, "Привязка хранилища: успешно")
	assert.Contains(t, out, "Информационная база: TestBase")
	assert.Contains(t, out, "Путь к хранилищу:")
}

func TestStorebindHandler_Execute_JSONOutput_Success(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mock := newMockConvertLoaderSuccess()
	h := &StorebindHandler{convertLoader: mock}
	cfg := &config.Config{
		InfobaseName: "TestBase",
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err, "stdout должен содержать валидный JSON")

	// AC-3: JSON output содержит необходимые поля
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "nr-storebind", result.Command)
	assert.NotNil(t, result.Data)

	// Проверяем поля data
	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok, "Data должен быть map")
	assert.Equal(t, true, dataMap["state_changed"])          // AC-6
	assert.Equal(t, "TestBase", dataMap["infobase_name"])    // AC-3
	assert.NotEmpty(t, dataMap["store_path"])                // AC-3
	_, hasDuration := dataMap["duration_ms"]                 // AC-3
	assert.True(t, hasDuration, "Data должен содержать duration_ms")

	// AC-3: metadata содержит duration_ms
	require.NotNil(t, result.Metadata)
	assert.NotEmpty(t, result.Metadata.TraceID)
	assert.GreaterOrEqual(t, result.Metadata.DurationMs, int64(0))
	assert.Equal(t, constants.APIVersion, result.Metadata.APIVersion)
}

// === AC-7: Error cases ===

func TestStorebindHandler_Execute_NoInfobase(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &StorebindHandler{}
	cfg := &config.Config{InfobaseName: ""}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.INFOBASE_MISSING")

	// Проверяем структурированный вывод ошибки
	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err, "stdout должен содержать валидный JSON ошибки")

	assert.Equal(t, "error", result.Status)
	assert.Equal(t, "nr-storebind", result.Command)
	require.NotNil(t, result.Error)
	assert.Equal(t, "CONFIG.INFOBASE_MISSING", result.Error.Code)
	assert.Contains(t, result.Error.Message, "BR_INFOBASE_NAME")
}

func TestStorebindHandler_Execute_NilConfig(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &StorebindHandler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.INFOBASE_MISSING")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
}

// === AC-9: ERR_STORE_BIND ===

func TestStorebindHandler_Execute_LoadConfigError(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mock := newMockConvertLoaderLoadError("database not found in config")
	h := &StorebindHandler{convertLoader: mock}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "ERR_CONFIG_LOAD")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	// Ошибка загрузки конфигурации использует ERR_CONFIG_LOAD (не ERR_STORE_BIND)
	assert.Equal(t, "ERR_CONFIG_LOAD", result.Error.Code)
	assert.Contains(t, result.Error.Message, "database not found in config")
}

func TestStorebindHandler_Execute_StoreBindError(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mock := newMockConvertLoaderBindError("store connection failed")
	h := &StorebindHandler{convertLoader: mock}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "ERR_STORE_BIND")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "ERR_STORE_BIND", result.Error.Code)
	assert.Contains(t, result.Error.Message, "store connection failed")
}

// === Text error output ===

func TestStorebindHandler_Execute_TextErrorOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	h := &StorebindHandler{}
	cfg := &config.Config{InfobaseName: ""}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.INFOBASE_MISSING")

	// Для text формата ошибка НЕ выводится в stdout — main.go логирует через logger
	// stdout должен быть пустым (или без JSON)
	assert.NotContains(t, out, `"status"`, "Текстовый формат НЕ должен содержать JSON")
}

// === Data structures tests ===

func TestStorebindData_writeText(t *testing.T) {
	tests := []struct {
		name     string
		data     *StorebindData
		contains []string
	}{
		{
			name: "Success",
			data: &StorebindData{
				StateChanged: true,
				InfobaseName: "MyBase",
				StorePath:    "tcp://store.example.com/main",
				DurationMs:   1500,
			},
			contains: []string{
				"успешно",
				"MyBase",
				"tcp://store.example.com/main",
			},
		},
		{
			name: "NoChanges",
			data: &StorebindData{
				StateChanged: false,
				InfobaseName: "AlreadyBoundBase",
				StorePath:    "",
				DurationMs:   0,
			},
			contains: []string{
				"без изменений",
				"AlreadyBoundBase",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := testutil.CaptureStdout(t, func() {
				_ = tt.data.writeText(os.Stdout)
			})

			for _, substr := range tt.contains {
				assert.Contains(t, out, substr)
			}
		})
	}
}

// === AC-10: Progress logging ===

func TestStorebindHandler_Execute_ProgressLogs(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	// Перехватываем slog для проверки progress сообщений
	var logBuf bytes.Buffer
	testLogger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	oldDefault := slog.Default()
	slog.SetDefault(testLogger)
	defer slog.SetDefault(oldDefault)

	mock := newMockConvertLoaderSuccess()
	h := &StorebindHandler{convertLoader: mock}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	_ = testutil.CaptureStdout(t, func() {
		_ = h.Execute(ctx, cfg)
	})

	logOutput := logBuf.String()

	// AC-10: Progress отображается: validating → connecting → binding
	assert.Contains(t, logOutput, "validating: проверка параметров", "Progress log должен содержать 'validating'")
	assert.Contains(t, logOutput, "connecting: загрузка конфигурации подключения", "Progress log должен содержать 'connecting'")
	assert.Contains(t, logOutput, "binding: привязка базы данных к хранилищу", "Progress log должен содержать 'binding'")
}

// === Compile-time interface check (AC-7: 5.7) ===

func TestStorebindHandler_ImplementsHandler(t *testing.T) {
	// Этот тест документирует что StorebindHandler реализует command.Handler
	// Реальная проверка происходит через var _ command.Handler = (*StorebindHandler)(nil) в handler.go
	var h command.Handler = &StorebindHandler{}
	assert.NotNil(t, h)
	assert.Equal(t, constants.ActNRStorebind, h.Name())
}

// === BR_STORE_PATH из env ===

func TestStorebindHandler_Execute_WithStorePath(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	t.Setenv("BR_STORE_PATH", "tcp://custom.server/store")

	mock := newMockConvertLoaderSuccess()
	h := &StorebindHandler{convertLoader: mock}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)

	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "tcp://custom.server/store", dataMap["store_path"])
}

// === Edge case: пустой store_path ===

func TestStorebindHandler_Execute_EmptyStorePath(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	// BR_STORE_PATH не установлен

	// Mock с пустым StoreRoot
	mock := &mockConvertLoader{
		loadFromConfigFunc: func(_ *context.Context, _ *slog.Logger, _ *config.Config) (*convert.Config, error) {
			return &convert.Config{
				StoreRoot: "", // Пустой путь
			}, nil
		},
		storeBindFunc: func(_ *convert.Config, _ *context.Context, _ *slog.Logger, _ *config.Config) error {
			return nil
		},
	}
	h := &StorebindHandler{convertLoader: mock}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)

	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok)
	// store_path будет пустой строкой когда оба источника пустые
	assert.Equal(t, "", dataMap["store_path"])
}
