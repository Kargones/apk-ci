package createstoreshandler

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
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStoreCreator — mock реализация StoreCreator для тестов.
// H-3 fix: mock теперь создаёт директории для расширений, чтобы проверка os.Stat проходила.
type mockStoreCreator struct {
	createStoresFunc func(l *slog.Logger, cfg *config.Config, storeRoot string, dbConnectString string, arrayAdd []string) error
	// createDirs — если true, создаёт директории main и add/<ext> в storeRoot
	createDirs bool
}

func (m *mockStoreCreator) CreateStores(l *slog.Logger, cfg *config.Config, storeRoot string, dbConnectString string, arrayAdd []string) error {
	if m.createStoresFunc != nil {
		return m.createStoresFunc(l, cfg, storeRoot, dbConnectString, arrayAdd)
	}
	// H-3 fix: создаём директории для успешной проверки os.Stat
	if m.createDirs {
		// Создаём main store
		mainPath := storeRoot + "/main"
		if err := os.MkdirAll(mainPath, 0o755); err != nil {
			return err
		}
		// Создаём директории расширений
		for _, ext := range arrayAdd {
			extPath := storeRoot + "/add/" + ext
			if err := os.MkdirAll(extPath, 0o755); err != nil {
				return err
			}
		}
	}
	return nil
}

// mockTempDbCreator — mock реализация TempDbCreator для тестов.
type mockTempDbCreator struct {
	createTempDbFunc func(ctx *context.Context, l *slog.Logger, cfg *config.Config) (string, error)
}

func (m *mockTempDbCreator) CreateTempDb(ctx *context.Context, l *slog.Logger, cfg *config.Config) (string, error) {
	if m.createTempDbFunc != nil {
		return m.createTempDbFunc(ctx, l, cfg)
	}
	return "/Srvr=\"test-server\";Ref=\"test-db\"", nil
}

// newTestAppConfig создаёт AppConfig для тестов с указанным путём к 1cv8.
func newTestAppConfig(bin1cv8 string) *config.AppConfig {
	return &config.AppConfig{
		Paths: struct {
			Bin1cv8  string `yaml:"bin1cv8"`
			BinIbcmd string `yaml:"binIbcmd"`
			EdtCli   string `yaml:"edtCli"`
			Rac      string `yaml:"rac"`
		}{
			Bin1cv8: bin1cv8,
		},
	}
}

// === Тесты Name и Description ===

func TestCreateStoresHandler_Name(t *testing.T) {
	h := &CreateStoresHandler{}
	assert.Equal(t, "nr-create-stores", h.Name())
	assert.Equal(t, constants.ActNRCreateStores, h.Name())
}

func TestCreateStoresHandler_Description(t *testing.T) {
	h := &CreateStoresHandler{}
	desc := h.Description()
	assert.NotEmpty(t, desc)
	assert.Equal(t, "Инициализация хранилищ конфигурации для проекта", desc)
}

// === AC-5: Registration и Deprecated Alias ===

func TestCreateStoresHandler_Registration(t *testing.T) {
	// init() уже вызван при импорте пакета — проверяем что handler зарегистрирован
	h, ok := command.Get("nr-create-stores")
	require.True(t, ok, "handler nr-create-stores должен быть зарегистрирован в registry")
	assert.Equal(t, constants.ActNRCreateStores, h.Name())
}

func TestCreateStoresHandler_DeprecatedAlias(t *testing.T) {
	// AC-5: Проверяем что deprecated alias "create-stores" работает
	h, ok := command.Get("create-stores")
	require.True(t, ok, "deprecated alias create-stores должен быть зарегистрирован в registry")

	// Проверяем что это DeprecatedBridge
	dep, isDep := h.(command.Deprecatable)
	require.True(t, isDep, "handler должен реализовывать Deprecatable")
	assert.True(t, dep.IsDeprecated())
	assert.Equal(t, "nr-create-stores", dep.NewName())
}

// === AC-1: Основной сценарий (основная конфигурация + расширения) ===

func TestCreateStoresHandler_Execute_JSONOutput_Success_WithExtensions(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	// H-3 fix: используем t.TempDir() и createDirs: true для создания реальных директорий
	tmpDir := t.TempDir()
	mockStore := &mockStoreCreator{createDirs: true}
	mockTempDb := &mockTempDbCreator{}

	h := &CreateStoresHandler{
		storeCreator:  mockStore,
		tempDbCreator: mockTempDb,
	}

	cfg := &config.Config{
		Owner:     "test-owner",
		Repo:      "test-repo",
		TmpDir:    tmpDir,
		AppConfig: newTestAppConfig("/opt/1cv8/bin/1cv8"),
		AddArray:  []string{"ExtA", "ExtB"}, // AC-2: расширения из cfg.AddArray
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
	assert.Equal(t, "nr-create-stores", result.Command)
	assert.NotNil(t, result.Data)

	// Проверяем поля data
	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok, "Data должен быть map")
	assert.Equal(t, true, dataMap["state_changed"])        // AC-6
	assert.NotEmpty(t, dataMap["store_root"])              // AC-3, AC-11
	assert.NotEmpty(t, dataMap["main_store_path"])         // AC-3
	_, hasDuration := dataMap["duration_ms"]               // AC-3
	assert.True(t, hasDuration, "Data должен содержать duration_ms")

	// AC-3: extension_stores[]
	extStores, hasExt := dataMap["extension_stores"]
	assert.True(t, hasExt, "Data должен содержать extension_stores")
	extList, ok := extStores.([]any)
	require.True(t, ok, "extension_stores должен быть массивом")
	assert.Len(t, extList, 2, "Должно быть 2 расширения")

	// Проверяем первое расширение
	ext0, ok := extList[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ExtA", ext0["name"])
	assert.Equal(t, true, ext0["success"])
	assert.NotEmpty(t, ext0["path"])

	// M-3 fix: Проверяем второе расширение
	ext1, ok := extList[1].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ExtB", ext1["name"])
	assert.Equal(t, true, ext1["success"])
	assert.NotEmpty(t, ext1["path"])

	// AC-3: metadata содержит duration_ms
	require.NotNil(t, result.Metadata)
	assert.NotEmpty(t, result.Metadata.TraceID)
	assert.GreaterOrEqual(t, result.Metadata.DurationMs, int64(0))
	assert.Equal(t, constants.APIVersion, result.Metadata.APIVersion)
}

// === AC-3: Без расширений ===

func TestCreateStoresHandler_Execute_JSONOutput_Success_NoExtensions(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	tmpDir := t.TempDir()
	mockStore := &mockStoreCreator{createDirs: true}
	mockTempDb := &mockTempDbCreator{}

	h := &CreateStoresHandler{
		storeCreator:  mockStore,
		tempDbCreator: mockTempDb,
	}

	cfg := &config.Config{
		Owner:     "test-owner",
		Repo:      "test-repo",
		TmpDir:    tmpDir,
		AppConfig: newTestAppConfig("/opt/1cv8/bin/1cv8"),
		AddArray:  []string{}, // Нет расширений
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
	assert.Equal(t, true, dataMap["state_changed"])
	assert.NotEmpty(t, dataMap["main_store_path"])

	// extension_stores должен быть пустым или nil (omitempty)
	extStores, hasExt := dataMap["extension_stores"]
	if hasExt && extStores != nil {
		extList, ok := extStores.([]any)
		require.True(t, ok)
		assert.Empty(t, extList, "extension_stores должен быть пустым когда нет расширений")
	}
}

// === AC-4: Text output ===

func TestCreateStoresHandler_Execute_TextOutput_Success(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	tmpDir := t.TempDir()
	mockStore := &mockStoreCreator{createDirs: true}
	mockTempDb := &mockTempDbCreator{}

	h := &CreateStoresHandler{
		storeCreator:  mockStore,
		tempDbCreator: mockTempDb,
	}

	cfg := &config.Config{
		Owner:     "test-owner",
		Repo:      "test-repo",
		TmpDir:    tmpDir,
		AppConfig: newTestAppConfig("/opt/1cv8/bin/1cv8"),
		AddArray:  []string{"MyExt"},
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)

	// AC-4: Text output показывает человекочитаемую информацию
	assert.Contains(t, out, "Создание хранилищ: успешно")
	assert.Contains(t, out, "Сводка:")
	assert.Contains(t, out, "Корневой путь:")
	assert.Contains(t, out, "Основное хранилище:")
	assert.Contains(t, out, "Хранилища расширений:")
	assert.Contains(t, out, "MyExt")
}

// === AC-7/AC-9: Error cases ===

func TestCreateStoresHandler_Execute_NilConfig(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &CreateStoresHandler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.MISSING")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "CONFIG.MISSING", result.Error.Code)
}

func TestCreateStoresHandler_Execute_NoBin1cv8(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &CreateStoresHandler{}
	cfg := &config.Config{
		Owner:     "test",
		Repo:      "test",
		TmpDir:    "/tmp",
		AppConfig: newTestAppConfig(""), // Bin1cv8 не указан
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.BIN1CV8_MISSING")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "CONFIG.BIN1CV8_MISSING", result.Error.Code)
}

func TestCreateStoresHandler_Execute_NilAppConfig(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &CreateStoresHandler{}
	cfg := &config.Config{
		Owner:     "test",
		Repo:      "test",
		TmpDir:    "/tmp",
		AppConfig: nil, // AppConfig nil
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.BIN1CV8_MISSING")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
}

func TestCreateStoresHandler_Execute_NoTmpDir(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &CreateStoresHandler{}
	cfg := &config.Config{
		Owner:     "test",
		Repo:      "test",
		AppConfig: newTestAppConfig("/opt/1cv8/bin/1cv8"),
		// TmpDir не указан
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.TMPDIR_MISSING")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
}

func TestCreateStoresHandler_Execute_NoOwner(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &CreateStoresHandler{}
	cfg := &config.Config{
		Repo:      "test",
		TmpDir:    "/tmp",
		AppConfig: newTestAppConfig("/opt/1cv8/bin/1cv8"),
		// Owner не указан
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.OWNER_MISSING")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
}

func TestCreateStoresHandler_Execute_NoRepo(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &CreateStoresHandler{}
	cfg := &config.Config{
		Owner:     "test",
		TmpDir:    "/tmp",
		AppConfig: newTestAppConfig("/opt/1cv8/bin/1cv8"),
		// Repo не указан
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.REPO_MISSING")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
}

// === AC-9: ERR_TEMP_DB ===

func TestCreateStoresHandler_Execute_TempDbError(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mockTempDb := &mockTempDbCreator{
		createTempDbFunc: func(_ *context.Context, _ *slog.Logger, _ *config.Config) (string, error) {
			return "", fmt.Errorf("не удалось создать временную БД")
		},
	}

	h := &CreateStoresHandler{
		tempDbCreator: mockTempDb,
	}

	cfg := &config.Config{
		Owner:     "test",
		Repo:      "test",
		TmpDir:    "/tmp",
		AppConfig: newTestAppConfig("/opt/1cv8/bin/1cv8"),
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "ERR_TEMP_DB")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "ERR_TEMP_DB", result.Error.Code)
	assert.Contains(t, result.Error.Message, "не удалось создать временную БД")
}

// === AC-9: ERR_STORE_CREATE ===

func TestCreateStoresHandler_Execute_StoreCreateError(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mockTempDb := &mockTempDbCreator{}
	mockStore := &mockStoreCreator{
		createStoresFunc: func(_ *slog.Logger, _ *config.Config, _ string, _ string, _ []string) error {
			return fmt.Errorf("ошибка создания хранилища конфигурации")
		},
	}

	h := &CreateStoresHandler{
		storeCreator:  mockStore,
		tempDbCreator: mockTempDb,
	}

	cfg := &config.Config{
		Owner:     "test",
		Repo:      "test",
		TmpDir:    "/tmp",
		AppConfig: newTestAppConfig("/opt/1cv8/bin/1cv8"),
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "ERR_STORE_CREATE")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "ERR_STORE_CREATE", result.Error.Code)
}

// === Text error output ===

func TestCreateStoresHandler_Execute_TextErrorOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	h := &CreateStoresHandler{}
	cfg := &config.Config{} // Пустая конфигурация
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.")

	// Для text формата ошибка НЕ выводится в stdout — main.go логирует через logger
	assert.NotContains(t, out, `"status"`, "Текстовый формат НЕ должен содержать JSON")
}

// === Data structures tests ===

func TestCreateStoresData_writeText(t *testing.T) {
	tests := []struct {
		name     string
		data     *CreateStoresData
		contains []string
	}{
		{
			name: "Success_WithExtensions",
			data: &CreateStoresData{
				StateChanged:  true,
				StoreRoot:     "/tmp/store_20260204/owner/repo",
				MainStorePath: "/tmp/store_20260204/owner/repo/main",
				ExtensionStores: []ExtensionStoreResult{
					{Name: "ExtA", Path: "/tmp/store_20260204/owner/repo/add/ExtA", Success: true},
					{Name: "ExtB", Path: "/tmp/store_20260204/owner/repo/add/ExtB", Success: true},
				},
				DurationMs: 5000,
			},
			contains: []string{
				"успешно",
				"Сводка:",
				"Корневой путь:",
				"/tmp/store_20260204/owner/repo",
				"Основное хранилище:",
				"Хранилища расширений:",
				"ExtA",
				"ExtB",
				"✓",
			},
		},
		{
			name: "Success_NoExtensions",
			data: &CreateStoresData{
				StateChanged:    true,
				StoreRoot:       "/tmp/store/main",
				MainStorePath:   "/tmp/store/main",
				ExtensionStores: []ExtensionStoreResult{},
				DurationMs:      1000,
			},
			contains: []string{
				"успешно",
				"Хранилища расширений: нет",
			},
		},
		{
			name: "WithFailedExtension",
			data: &CreateStoresData{
				StateChanged:  true,
				StoreRoot:     "/tmp/store",
				MainStorePath: "/tmp/store/main",
				ExtensionStores: []ExtensionStoreResult{
					{Name: "ExtOk", Path: "/tmp/store/add/ExtOk", Success: true},
					{Name: "ExtFail", Path: "/tmp/store/add/ExtFail", Success: false, Error: "connection timeout"},
				},
				DurationMs: 3000,
			},
			contains: []string{
				"✓ ExtOk",
				"✗ ExtFail",
				"Ошибка: connection timeout",
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

func TestCreateStoresHandler_Execute_ProgressLogs(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	// Перехватываем slog для проверки progress сообщений
	var logBuf bytes.Buffer
	testLogger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	oldDefault := slog.Default()
	slog.SetDefault(testLogger)
	defer slog.SetDefault(oldDefault)

	tmpDir := t.TempDir()
	mockStore := &mockStoreCreator{createDirs: true}
	mockTempDb := &mockTempDbCreator{}

	h := &CreateStoresHandler{
		storeCreator:  mockStore,
		tempDbCreator: mockTempDb,
	}

	cfg := &config.Config{
		Owner:     "test",
		Repo:      "test",
		TmpDir:    tmpDir,
		AppConfig: newTestAppConfig("/opt/1cv8/bin/1cv8"),
		AddArray:  []string{"TestExt"},
	}
	ctx := context.Background()

	_ = testutil.CaptureStdout(t, func() {
		_ = h.Execute(ctx, cfg)
	})

	logOutput := logBuf.String()

	// AC-10: Progress отображается: validating → creating_temp_db → creating_main_store → creating_extension_stores
	// H-1 fix: теперь логируется creating_extension_stores ПЕРЕД созданием (подготовка)
	// и creating_extension_store ПОСЛЕ создания каждого расширения (результат)
	assert.Contains(t, logOutput, "validating: проверка параметров", "Progress log должен содержать 'validating'")
	assert.Contains(t, logOutput, "creating_temp_db: создание временной базы данных", "Progress log должен содержать 'creating_temp_db'")
	assert.Contains(t, logOutput, "creating_main_store: создание основного хранилища", "Progress log должен содержать 'creating_main_store'")
	assert.Contains(t, logOutput, "creating_extension_stores: подготовка к созданию хранилищ расширений", "Progress log должен содержать 'creating_extension_stores'")
	assert.Contains(t, logOutput, "creating_extension_store: хранилище расширения создано", "Progress log должен содержать результат создания расширения")
}

// === H-3 fix: проверка что при отсутствии директории расширения возвращается ошибка ===

func TestCreateStoresHandler_Execute_ExtensionDirNotCreated(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	tmpDir := t.TempDir()
	// Mock который создаёт только main, но НЕ создаёт директории расширений
	mockStore := &mockStoreCreator{
		createStoresFunc: func(_ *slog.Logger, _ *config.Config, storeRoot string, _ string, _ []string) error {
			// Создаём только main
			mainPath := storeRoot + "/main"
			return os.MkdirAll(mainPath, 0o755)
		},
	}
	mockTempDb := &mockTempDbCreator{}

	h := &CreateStoresHandler{
		storeCreator:  mockStore,
		tempDbCreator: mockTempDb,
	}

	cfg := &config.Config{
		Owner:     "test-owner",
		Repo:      "test-repo",
		TmpDir:    tmpDir,
		AppConfig: newTestAppConfig("/opt/1cv8/bin/1cv8"),
		AddArray:  []string{"MissingExt"},
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	// H-3 fix: теперь возвращается ошибка если расширение не создано
	require.Error(t, execErr, "Execute должен вернуть ошибку если расширение не создано")
	assert.Contains(t, execErr.Error(), "ERR_STORE_CREATE")
	assert.Contains(t, execErr.Error(), "MissingExt")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)

	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "ERR_STORE_CREATE", result.Error.Code)
	assert.Contains(t, result.Error.Message, "MissingExt")
}

// === H-2 fix: проверка что при отсутствии main директории возвращается ошибка ===

func TestCreateStoresHandler_Execute_MainDirNotCreated(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	tmpDir := t.TempDir()
	// Mock который НЕ создаёт никаких директорий
	mockStore := &mockStoreCreator{
		createStoresFunc: func(_ *slog.Logger, _ *config.Config, _ string, _ string, _ []string) error {
			// Не создаём ничего, но возвращаем success
			return nil
		},
	}
	mockTempDb := &mockTempDbCreator{}

	h := &CreateStoresHandler{
		storeCreator:  mockStore,
		tempDbCreator: mockTempDb,
	}

	cfg := &config.Config{
		Owner:     "test-owner",
		Repo:      "test-repo",
		TmpDir:    tmpDir,
		AppConfig: newTestAppConfig("/opt/1cv8/bin/1cv8"),
		AddArray:  []string{}, // Без расширений чтобы проверить только main
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	// H-2 fix: ошибка если main не создан
	require.Error(t, execErr, "Execute должен вернуть ошибку если main хранилище не создано")
	assert.Contains(t, execErr.Error(), "ERR_STORE_CREATE")
	assert.Contains(t, execErr.Error(), "основное хранилище не создано")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)

	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "ERR_STORE_CREATE", result.Error.Code)
}

// === Compile-time interface check (AC-7) ===

func TestCreateStoresHandler_ImplementsHandler(t *testing.T) {
	// Этот тест документирует что CreateStoresHandler реализует command.Handler
	// Реальная проверка происходит через var _ command.Handler = (*CreateStoresHandler)(nil) в handler.go
	var h command.Handler = &CreateStoresHandler{}
	assert.NotNil(t, h)
	assert.Equal(t, constants.ActNRCreateStores, h.Name())
}
