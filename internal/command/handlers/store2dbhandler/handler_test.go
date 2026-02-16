package store2dbhandler

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
	loadFromConfigFunc func(ctx context.Context, l *slog.Logger, cfg *config.Config) (*convert.Config, error)
	storeBindFunc      func(cc *convert.Config, ctx context.Context, l *slog.Logger, cfg *config.Config) error
}

func (m *mockConvertLoader) LoadFromConfig(ctx context.Context, l *slog.Logger, cfg *config.Config) (*convert.Config, error) {
	if m.loadFromConfigFunc != nil {
		return m.loadFromConfigFunc(ctx, l, cfg)
	}
	// По умолчанию возвращаем минимальную конфигурацию
	return &convert.Config{
		StoreRoot: "tcp://example.com/store",
	}, nil
}

func (m *mockConvertLoader) StoreBind(cc *convert.Config, ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	if m.storeBindFunc != nil {
		return m.storeBindFunc(cc, ctx, l, cfg)
	}
	return nil
}

// newMockConvertLoaderSuccess создаёт mock, возвращающий успешные результаты.
func newMockConvertLoaderSuccess() *mockConvertLoader {
	return &mockConvertLoader{
		loadFromConfigFunc: func(_ context.Context, _ *slog.Logger, _ *config.Config) (*convert.Config, error) {
			return &convert.Config{
				StoreRoot: "tcp://example.com/store",
			}, nil
		},
		storeBindFunc: func(_ *convert.Config, _ context.Context, _ *slog.Logger, _ *config.Config) error {
			return nil
		},
	}
}

// newMockConvertLoaderLoadError создаёт mock, возвращающий ошибку при загрузке.
func newMockConvertLoaderLoadError(errMsg string) *mockConvertLoader {
	return &mockConvertLoader{
		loadFromConfigFunc: func(_ context.Context, _ *slog.Logger, _ *config.Config) (*convert.Config, error) {
			return nil, fmt.Errorf("%s", errMsg)
		},
	}
}

// newMockConvertLoaderBindError создаёт mock, возвращающий ошибку при привязке.
func newMockConvertLoaderBindError(errMsg string) *mockConvertLoader {
	return &mockConvertLoader{
		loadFromConfigFunc: func(_ context.Context, _ *slog.Logger, _ *config.Config) (*convert.Config, error) {
			return &convert.Config{
				StoreRoot: "tcp://example.com/store",
			}, nil
		},
		storeBindFunc: func(_ *convert.Config, _ context.Context, _ *slog.Logger, _ *config.Config) error {
			return fmt.Errorf("%s", errMsg)
		},
	}
}

func TestStore2DbHandler_Name(t *testing.T) {
	h := &Store2DbHandler{}
	assert.Equal(t, "nr-store2db", h.Name())
	assert.Equal(t, constants.ActNRStore2db, h.Name())
}

func TestStore2DbHandler_Description(t *testing.T) {
	h := &Store2DbHandler{}
	desc := h.Description()
	assert.NotEmpty(t, desc)
	assert.Equal(t, "Загрузка конфигурации из хранилища в базу данных", desc)
}

func TestStore2DbHandler_Registration(t *testing.T) {
	// init() уже вызван при импорте пакета — проверяем что handler зарегистрирован
	h, ok := command.Get("nr-store2db")
	require.True(t, ok, "handler nr-store2db должен быть зарегистрирован в registry")
	assert.Equal(t, constants.ActNRStore2db, h.Name())
}

func TestStore2DbHandler_DeprecatedAlias(t *testing.T) {
	// AC-6: Проверяем что deprecated alias "store2db" работает
	h, ok := command.Get("store2db")
	require.True(t, ok, "deprecated alias store2db должен быть зарегистрирован в registry")

	// Проверяем что это DeprecatedBridge
	dep, isDep := h.(command.Deprecatable)
	require.True(t, isDep, "handler должен реализовывать Deprecatable")
	assert.True(t, dep.IsDeprecated())
	assert.Equal(t, "nr-store2db", dep.NewName())
}

// === AC-1: Основной сценарий ===

func TestStore2DbHandler_Execute_TextOutput_Success(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	mock := newMockConvertLoaderSuccess()
	h := &Store2DbHandler{convertLoader: mock}
	cfg := &config.Config{
		InfobaseName: "TestBase",
		AddArray:     []string{"Extension1", "Extension2"},
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)
	// AC-5: Text output показывает человекочитаемую информацию
	assert.Contains(t, out, "Загрузка конфигурации из хранилища: успешно")
	assert.Contains(t, out, "Информационная база: TestBase")
	assert.Contains(t, out, "Версия хранилища: latest")
	assert.Contains(t, out, "Основная конфигурация: загружена")
	// AC-7: Расширения отображаются
	assert.Contains(t, out, "Extension1")
	assert.Contains(t, out, "Extension2")
}

func TestStore2DbHandler_Execute_JSONOutput_Success(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mock := newMockConvertLoaderSuccess()
	h := &Store2DbHandler{convertLoader: mock}
	cfg := &config.Config{
		InfobaseName: "TestBase",
		AddArray:     []string{"Ext1"},
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

	// AC-4: JSON output содержит необходимые поля
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "nr-store2db", result.Command)
	assert.NotNil(t, result.Data)

	// Проверяем поля data
	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok, "Data должен быть map")
	assert.Equal(t, true, dataMap["state_changed"])
	assert.Equal(t, "TestBase", dataMap["infobase_name"])
	assert.Equal(t, "latest", dataMap["store_version"])
	assert.Equal(t, true, dataMap["main_config_loaded"])

	// H-1 fix: AC-4 требует duration_ms в data
	durationMs, ok := dataMap["duration_ms"]
	require.True(t, ok, "Data должен содержать duration_ms (AC-4)")
	assert.GreaterOrEqual(t, durationMs.(float64), float64(0))

	// AC-7: extensions_loaded содержит расширения
	extRaw, ok := dataMap["extensions_loaded"]
	require.True(t, ok, "Data должен содержать extensions_loaded")
	extensions, ok := extRaw.([]any)
	require.True(t, ok, "extensions_loaded должен быть массивом")
	assert.Len(t, extensions, 1)

	ext0, ok := extensions[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Ext1", ext0["name"])
	assert.Equal(t, true, ext0["success"])

	// AC-4: metadata содержит duration_ms
	require.NotNil(t, result.Metadata)
	assert.NotEmpty(t, result.Metadata.TraceID)
	assert.GreaterOrEqual(t, result.Metadata.DurationMs, int64(0))
	assert.Equal(t, constants.APIVersion, result.Metadata.APIVersion)
}

// === AC-2: BR_STORE_VERSION ===

func TestStore2DbHandler_Execute_WithStoreVersion(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	t.Setenv("BR_STORE_VERSION", "42")

	mock := newMockConvertLoaderSuccess()
	h := &Store2DbHandler{convertLoader: mock}
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
	assert.Equal(t, "42", dataMap["store_version"])
}

func TestStore2DbHandler_Execute_LatestStoreVersion(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	t.Setenv("BR_STORE_VERSION", "latest")

	mock := newMockConvertLoaderSuccess()
	h := &Store2DbHandler{convertLoader: mock}
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
	assert.Equal(t, "latest", dataMap["store_version"])
}

// === AC-3: Error cases ===

func TestStore2DbHandler_Execute_NoInfobase(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &Store2DbHandler{}
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
	assert.Equal(t, "nr-store2db", result.Command)
	require.NotNil(t, result.Error)
	assert.Equal(t, "CONFIG.INFOBASE_MISSING", result.Error.Code)
	assert.Contains(t, result.Error.Message, "BR_INFOBASE_NAME")
}

func TestStore2DbHandler_Execute_NilConfig(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &Store2DbHandler{}
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

// === AC-10: ERR_STORE_OP ===

func TestStore2DbHandler_Execute_LoadConfigError(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mock := newMockConvertLoaderLoadError("database not found in config")
	h := &Store2DbHandler{convertLoader: mock}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "ERR_STORE_OP")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "ERR_STORE_OP", result.Error.Code)
	assert.Contains(t, result.Error.Message, "database not found in config")
}

func TestStore2DbHandler_Execute_StoreBindError(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mock := newMockConvertLoaderBindError("store connection failed")
	h := &Store2DbHandler{convertLoader: mock}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "ERR_STORE_OP")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "ERR_STORE_OP", result.Error.Code)
	assert.Contains(t, result.Error.Message, "store connection failed")
}

// === Text error output ===

func TestStore2DbHandler_Execute_TextErrorOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	h := &Store2DbHandler{}
	cfg := &config.Config{InfobaseName: ""}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.INFOBASE_MISSING")

	// M-2 fix: для text формата ошибка НЕ выводится в stdout — main.go логирует через logger
	// stdout должен быть пустым (или без JSON)
	assert.NotContains(t, out, `"status"`, "Текстовый формат НЕ должен содержать JSON")
}

// === AC-7: Extensions ===

func TestStore2DbHandler_Execute_NoExtensions(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mock := newMockConvertLoaderSuccess()
	h := &Store2DbHandler{convertLoader: mock}
	cfg := &config.Config{
		InfobaseName: "TestBase",
		AddArray:     []string{}, // Нет расширений
	}
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

	// M-1 fix: с omitempty пустой массив НЕ сериализуется в JSON
	_, ok = dataMap["extensions_loaded"]
	assert.False(t, ok, "extensions_loaded не должен присутствовать при пустом массиве (omitempty)")
}

func TestStore2DbHandler_Execute_TextOutput_NoExtensions(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	mock := newMockConvertLoaderSuccess()
	h := &Store2DbHandler{convertLoader: mock}
	cfg := &config.Config{
		InfobaseName: "TestBase",
		AddArray:     []string{},
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)
	assert.Contains(t, out, "Загрузка конфигурации из хранилища: успешно")
	assert.Contains(t, out, "Основная конфигурация: загружена")
	// Секция "Расширения:" не должна отображаться если нет расширений
	assert.NotContains(t, out, "Расширения:")
}

// === Data structures tests ===

func TestStore2DbData_writeText(t *testing.T) {
	tests := []struct {
		name     string
		data     *Store2DbData
		contains []string
	}{
		{
			name: "Success with extensions",
			data: &Store2DbData{
				StateChanged:     true,
				InfobaseName:     "MyBase",
				StoreVersion:     "123",
				MainConfigLoaded: true,
				ExtensionsLoaded: []ExtensionLoadResult{
					{Name: "Ext1", Success: true},
					{Name: "Ext2", Success: false, Error: "не найдено"},
				},
			},
			contains: []string{
				"успешно",
				"MyBase",
				"123",
				"загружена",
				"Ext1: загружена",
				"Ext2: ошибка (не найдено)",
			},
		},
		{
			name: "Main config error",
			data: &Store2DbData{
				StateChanged:     false,
				InfobaseName:     "ErrorBase",
				StoreVersion:     "latest",
				MainConfigLoaded: false,
			},
			contains: []string{
				"ошибка",
				"ErrorBase",
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

func TestBoolToStatus(t *testing.T) {
	assert.Equal(t, "загружена", boolToStatus(true))
	assert.Equal(t, "ошибка", boolToStatus(false))
}

// === AC-3: Progress logging (H-4 fix) ===

func TestStore2DbHandler_Execute_ProgressLogs(t *testing.T) {
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
	h := &Store2DbHandler{convertLoader: mock}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	_ = testutil.CaptureStdout(t, func() {
		_ = h.Execute(ctx, cfg)
	})

	logOutput := logBuf.String()

	// AC-3: Progress отображается: connecting → loading → applying
	assert.Contains(t, logOutput, "connecting", "Progress log должен содержать 'connecting'")
	assert.Contains(t, logOutput, "loading", "Progress log должен содержать 'loading'")
	assert.Contains(t, logOutput, "applying", "Progress log должен содержать 'applying'")
}

// === Compile-time interface check (M-3) ===

func TestStore2DbHandler_ImplementsHandler(t *testing.T) {
	// Этот тест документирует что Store2DbHandler реализует command.Handler
	// Реальная проверка происходит через var _ command.Handler = (*Store2DbHandler)(nil) в handler.go
	var h command.Handler = &Store2DbHandler{}
	assert.NotNil(t, h)
	assert.Equal(t, constants.ActNRStore2db, h.Name())
}

// === Constant test (M-4) ===

func TestStoreVersionLatest(t *testing.T) {
	// Проверяем что константа используется вместо magic string
	assert.Equal(t, "latest", storeVersionLatest)
}
