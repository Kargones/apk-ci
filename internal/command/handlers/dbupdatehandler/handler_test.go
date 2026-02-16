package dbupdatehandler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/onec"
	"github.com/Kargones/apk-ci/internal/adapter/onec/onectest"
	"github.com/Kargones/apk-ci/internal/adapter/onec/rac"
	"github.com/Kargones/apk-ci/internal/adapter/onec/rac/ractest"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/output"
)

// captureStdout перехватывает stdout во время выполнения функции и возвращает вывод
func captureStdout(fn func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

// TestDbUpdateHandler_Name проверяет возврат имени команды
func TestDbUpdateHandler_Name(t *testing.T) {
	h := &DbUpdateHandler{}
	want := constants.ActNRDbupdate
	if got := h.Name(); got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
}

// TestDbUpdateHandler_Description проверяет возврат описания команды
func TestDbUpdateHandler_Description(t *testing.T) {
	h := &DbUpdateHandler{}
	if desc := h.Description(); desc == "" {
		t.Error("Description() returned empty string")
	}
}

// TestDbUpdateHandler_Execute_MissingInfobaseName проверяет ошибку при отсутствии BR_INFOBASE_NAME
func TestDbUpdateHandler_Execute_MissingInfobaseName(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "nil config",
			cfg:  nil,
		},
		{
			name: "empty infobase name",
			cfg:  &config.Config{InfobaseName: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &DbUpdateHandler{}
			err := h.Execute(context.Background(), tt.cfg)
			if err == nil {
				t.Error("Execute() should return error for missing infobase name")
			}
			if err != nil && !strings.Contains(err.Error(), ErrDbUpdateValidation) {
				t.Errorf("Execute() error = %v, want error containing %q", err, ErrDbUpdateValidation)
			}
		})
	}
}

// TestDbUpdateHandler_Execute_DatabaseNotFound проверяет ошибку когда БД не найдена в конфигурации
func TestDbUpdateHandler_Execute_DatabaseNotFound(t *testing.T) {
	cfg := &config.Config{
		InfobaseName: "UnknownDB",
		DbConfig:     map[string]*config.DatabaseInfo{}, // Пустая конфигурация
	}

	h := &DbUpdateHandler{}

	output := captureStdout(func() {
		_ = h.Execute(context.Background(), cfg)
	})

	if !strings.Contains(output, "не найдена") {
		t.Errorf("Execute() should report database not found, got: %s", output)
	}
}

// TestDbUpdateHandler_Execute_SuccessfulUpdate проверяет успешное обновление с mock
func TestDbUpdateHandler_Execute_SuccessfulUpdate(t *testing.T) {
	mockClient := onectest.NewMockDatabaseUpdater()

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{
		oneCClient: mockClient,
	}

	var err error
	output := captureStdout(func() {
		err = h.Execute(context.Background(), cfg)
	})

	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}

	// Проверяем вывод
	if !strings.Contains(output, "Обновление завершено") {
		t.Errorf("Execute() output should contain success message, got: %s", output)
	}

	// Проверяем что mock был вызван
	if mockClient.UpdateDBCfgCallCount != 1 {
		t.Errorf("UpdateDBCfg called %d times, want 1", mockClient.UpdateDBCfgCallCount)
	}
}

// TestDbUpdateHandler_Execute_ExtensionUpdate проверяет обновление расширения (двойной вызов)
func TestDbUpdateHandler_Execute_ExtensionUpdate(t *testing.T) {
	mockClient := onectest.NewMockDatabaseUpdater()

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{
		oneCClient: mockClient,
	}

	// Устанавливаем расширение
	t.Setenv("BR_EXTENSION", "MyExtension")

	var err error
	output := captureStdout(func() {
		err = h.Execute(context.Background(), cfg)
	})

	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}

	// Проверяем что mock был вызван ДВАЖДЫ (особенность расширений)
	if mockClient.UpdateDBCfgCallCount != 2 {
		t.Errorf("UpdateDBCfg called %d times, want 2 for extension", mockClient.UpdateDBCfgCallCount)
	}

	// Проверяем что расширение указано в опциях
	if mockClient.LastUpdateOptions.Extension != "MyExtension" {
		t.Errorf("Extension = %q, want %q", mockClient.LastUpdateOptions.Extension, "MyExtension")
	}

	// Проверяем вывод содержит информацию о расширении
	if !strings.Contains(output, "Расширение") {
		t.Errorf("Execute() output should mention extension, got: %s", output)
	}
}

// TestDbUpdateHandler_Execute_UpdateError проверяет обработку ошибки обновления
func TestDbUpdateHandler_Execute_UpdateError(t *testing.T) {
	updateError := errors.New("update operation failed")
	mockClient := &onectest.MockDatabaseUpdater{
		UpdateDBCfgFunc: func(ctx context.Context, opts onec.UpdateOptions) (*onec.UpdateResult, error) {
			return nil, updateError
		},
	}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{
		oneCClient: mockClient,
	}

	output := captureStdout(func() {
		_ = h.Execute(context.Background(), cfg)
	})

	if !strings.Contains(output, "Ошибка") {
		t.Errorf("Execute() should report error, got: %s", output)
	}
}

// TestDbUpdateHandler_Execute_ExtensionSecondPassError проверяет ошибку ВТОРОГО прохода расширения
// M3 fix: добавлен тест для случая когда первый проход успешен, а второй — нет
func TestDbUpdateHandler_Execute_ExtensionSecondPassError(t *testing.T) {
	callCount := 0
	secondPassError := errors.New("second pass failed")
	mockClient := &onectest.MockDatabaseUpdater{
		UpdateDBCfgFunc: func(ctx context.Context, opts onec.UpdateOptions) (*onec.UpdateResult, error) {
			callCount++
			if callCount == 1 {
				// Первый проход — успех
				return &onec.UpdateResult{Success: true}, nil
			}
			// Второй проход — ошибка
			return nil, secondPassError
		},
	}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{
		oneCClient: mockClient,
	}

	// Устанавливаем расширение для активации двойного прохода
	t.Setenv("BR_EXTENSION", "TestExtension")

	var err error
	output := captureStdout(func() {
		err = h.Execute(context.Background(), cfg)
	})

	// Должна быть ошибка
	if err == nil {
		t.Error("Execute() should return error for second pass failure")
	}

	// Ошибка должна содержать код второго прохода (M2 fix)
	if err != nil && !strings.Contains(err.Error(), ErrDbUpdateSecondPassFailed) {
		t.Errorf("Execute() error = %v, want error containing %q", err, ErrDbUpdateSecondPassFailed)
	}

	// Вывод должен содержать информацию об ошибке
	if !strings.Contains(output, "Ошибка") {
		t.Errorf("Execute() output should contain error message, got: %s", output)
	}

	// Должно быть 2 вызова (первый успешный, второй с ошибкой)
	if callCount != 2 {
		t.Errorf("UpdateDBCfg called %d times, want 2", callCount)
	}
}

// TestDbUpdateHandler_Execute_JSONOutput проверяет JSON формат вывода
// M1 fix: расширенная проверка Data полей
func TestDbUpdateHandler_Execute_JSONOutput(t *testing.T) {
	mockClient := onectest.NewMockDatabaseUpdater()

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{
		oneCClient: mockClient,
	}

	t.Setenv("BR_OUTPUT_FORMAT", "json")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}

	// Проверяем JSON формат
	var result output.Result
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Errorf("JSON unmarshal error: %v, output: %s", jsonErr, buf.String())
	}

	if result.Status != output.StatusSuccess {
		t.Errorf("JSON result.Status = %q, want %q", result.Status, output.StatusSuccess)
	}
	if result.Command != constants.ActNRDbupdate {
		t.Errorf("JSON result.Command = %q, want %q", result.Command, constants.ActNRDbupdate)
	}

	// M1 fix: проверяем содержимое Data
	if result.Data == nil {
		t.Error("JSON result.Data should not be nil")
	} else {
		dataMap, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Errorf("JSON result.Data type = %T, want map[string]interface{}", result.Data)
		} else {
			if infobaseName, exists := dataMap["infobase_name"]; !exists || infobaseName != "TestDB" {
				t.Errorf("JSON result.Data.infobase_name = %v, want %q", infobaseName, "TestDB")
			}
			if success, exists := dataMap["success"]; !exists || success != true {
				t.Errorf("JSON result.Data.success = %v, want true", success)
			}
		}
	}

	// Проверяем Metadata
	if result.Metadata == nil {
		t.Error("JSON result.Metadata should not be nil")
	} else {
		if result.Metadata.TraceID == "" {
			t.Error("JSON result.Metadata.TraceID should not be empty")
		}
		if result.Metadata.DurationMs < 0 {
			t.Errorf("JSON result.Metadata.DurationMs = %d, want >= 0", result.Metadata.DurationMs)
		}
		// L3 fix: проверяем APIVersion
		if result.Metadata.APIVersion != constants.APIVersion {
			t.Errorf("JSON result.Metadata.APIVersion = %q, want %q", result.Metadata.APIVersion, constants.APIVersion)
		}
	}
}

// TestDbUpdateHandler_Execute_ExplicitTimeout проверяет явный таймаут через BR_TIMEOUT_MIN
func TestDbUpdateHandler_Execute_ExplicitTimeout(t *testing.T) {
	var capturedTimeout time.Duration
	mockClient := &onectest.MockDatabaseUpdater{
		UpdateDBCfgFunc: func(ctx context.Context, opts onec.UpdateOptions) (*onec.UpdateResult, error) {
			capturedTimeout = opts.Timeout
			return &onec.UpdateResult{Success: true}, nil
		},
	}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{
		oneCClient: mockClient,
	}

	t.Setenv("BR_TIMEOUT_MIN", "45")

	captureStdout(func() {
		_ = h.Execute(context.Background(), cfg)
	})

	expectedTimeout := 45 * time.Minute
	if capturedTimeout != expectedTimeout {
		t.Errorf("Timeout = %v, want %v", capturedTimeout, expectedTimeout)
	}
}

// TestDbUpdateHandler_Execute_DefaultTimeout проверяет таймаут по умолчанию
func TestDbUpdateHandler_Execute_DefaultTimeout(t *testing.T) {
	var capturedTimeout time.Duration
	mockClient := &onectest.MockDatabaseUpdater{
		UpdateDBCfgFunc: func(ctx context.Context, opts onec.UpdateOptions) (*onec.UpdateResult, error) {
			capturedTimeout = opts.Timeout
			return &onec.UpdateResult{Success: true}, nil
		},
	}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{
		oneCClient: mockClient,
	}

	captureStdout(func() {
		_ = h.Execute(context.Background(), cfg)
	})

	expectedTimeout := 30 * time.Minute // defaultTimeout
	if capturedTimeout != expectedTimeout {
		t.Errorf("Timeout = %v, want %v (default)", capturedTimeout, expectedTimeout)
	}
}

// TestDbUpdateHandler_Execute_AutoDeps проверяет режим auto-deps
func TestDbUpdateHandler_Execute_AutoDeps(t *testing.T) {
	mockClient := onectest.NewMockDatabaseUpdater()

	// Создаём RAC mock
	racMock := ractest.NewMockRACClient()
	racMock.GetServiceModeStatusFunc = func(ctx context.Context, clusterUUID, infobaseUUID string) (*rac.ServiceModeStatus, error) {
		return &rac.ServiceModeStatus{Enabled: false}, nil
	}
	var enableCalled, disableCalled bool
	racMock.EnableServiceModeFunc = func(ctx context.Context, clusterUUID, infobaseUUID string, terminateSessions bool) error {
		enableCalled = true
		return nil
	}
	racMock.DisableServiceModeFunc = func(ctx context.Context, clusterUUID, infobaseUUID string) error {
		disableCalled = true
		return nil
	}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{
		oneCClient: mockClient,
		racClient:  racMock,
	}

	t.Setenv("BR_AUTO_DEPS", "true")

	var err error
	output := captureStdout(func() {
		err = h.Execute(context.Background(), cfg)
	})

	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}

	// Проверяем что service mode был включён и отключён
	if !enableCalled {
		t.Error("EnableServiceMode should be called when auto-deps is enabled")
	}
	if !disableCalled {
		t.Error("DisableServiceMode should be called after update when auto-deps is enabled")
	}

	// Проверяем вывод содержит auto-deps информацию
	if !strings.Contains(output, "Auto-deps") {
		t.Errorf("Execute() output should mention auto-deps, got: %s", output)
	}
}

// TestDbUpdateHandler_Execute_AutoDeps_AlreadyEnabled проверяет что service mode не отключается если был уже включён
func TestDbUpdateHandler_Execute_AutoDeps_AlreadyEnabled(t *testing.T) {
	mockClient := onectest.NewMockDatabaseUpdater()

	racMock := ractest.NewMockRACClient()
	// Service mode УЖЕ включён
	racMock.GetServiceModeStatusFunc = func(ctx context.Context, clusterUUID, infobaseUUID string) (*rac.ServiceModeStatus, error) {
		return &rac.ServiceModeStatus{Enabled: true}, nil
	}
	var disableCalled bool
	racMock.DisableServiceModeFunc = func(ctx context.Context, clusterUUID, infobaseUUID string) error {
		disableCalled = true
		return nil
	}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{
		oneCClient: mockClient,
		racClient:  racMock,
	}

	t.Setenv("BR_AUTO_DEPS", "true")

	captureStdout(func() {
		_ = h.Execute(context.Background(), cfg)
	})

	// Если service mode был уже включён, не должны его отключать
	if disableCalled {
		t.Error("DisableServiceMode should NOT be called when service mode was already enabled")
	}
}

// TestDbUpdateHandler_Execute_ShowProgressDisabled проверяет отключение progress bar
func TestDbUpdateHandler_Execute_ShowProgressDisabled(t *testing.T) {
	mockClient := onectest.NewMockDatabaseUpdater()
	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{
		oneCClient: mockClient,
	}

	t.Setenv("BR_SHOW_PROGRESS", "false")

	var err error
	captureStdout(func() {
		err = h.Execute(context.Background(), cfg)
	})

	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}
}

// TestDbUpdateData_writeText проверяет текстовый вывод результата
func TestDbUpdateData_writeText(t *testing.T) {
	tests := []struct {
		name     string
		data     *DbUpdateData
		expected []string
	}{
		{
			name: "successful update",
			data: &DbUpdateData{
				InfobaseName: "TestDB",
				Success:      true,
				DurationMs:   5000,
				AutoDeps:     false,
			},
			expected: []string{
				"✅ Обновление завершено успешно",
				"TestDB",
			},
		},
		{
			name: "update with extension",
			data: &DbUpdateData{
				InfobaseName: "TestDB",
				Extension:    "MyExtension",
				Success:      true,
				DurationMs:   5000,
				AutoDeps:     true,
			},
			expected: []string{
				"Расширение: MyExtension",
				"Auto-deps: включён",
			},
		},
		{
			name: "failed update",
			data: &DbUpdateData{
				InfobaseName: "TestDB",
				Success:      false,
				Messages:     []string{"Ошибка конфигурации"},
				DurationMs:   1000,
			},
			expected: []string{
				"❌ Обновление завершено с ошибками",
				"Ошибка конфигурации",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.data.writeText(&buf)
			if err != nil {
				t.Errorf("writeText() error = %v", err)
			}

			output := buf.String()
			for _, part := range tt.expected {
				if !strings.Contains(output, part) {
					t.Errorf("writeText() output should contain %q, got: %s", part, output)
				}
			}
		})
	}
}

// TestBuildConnectString проверяет построение строки подключения
func TestBuildConnectString(t *testing.T) {
	tests := []struct {
		name     string
		dbInfo   *config.DatabaseInfo
		cfg      *config.Config
		expected string
	}{
		{
			name: "basic connection string",
			dbInfo: &config.DatabaseInfo{
				OneServer: "1c-server",
			},
			cfg: &config.Config{
				InfobaseName: "TestDB",
			},
			expected: "/S 1c-server\\TestDB",
		},
		{
			name: "with user and password",
			dbInfo: &config.DatabaseInfo{
				OneServer: "1c-server",
			},
			cfg: &config.Config{
				InfobaseName: "TestDB",
				AppConfig: &config.AppConfig{
					Users: struct {
						Rac        string `yaml:"rac"`
						Db         string `yaml:"db"`
						Mssql      string `yaml:"mssql"`
						StoreAdmin string `yaml:"storeAdmin"`
					}{
						Db: "admin",
					},
				},
				SecretConfig: &config.SecretConfig{
					Passwords: struct {
						Rac                string `yaml:"rac"`
						Db                 string `yaml:"db"`
						Mssql              string `yaml:"mssql"`
						StoreAdminPassword string `yaml:"storeAdminPassword"`
						Smb                string `yaml:"smb"`
					}{
						Db: "secret123",
					},
				},
			},
			expected: "/S 1c-server\\TestDB /N admin /P secret123",
		},
		{
			name: "fallback to DbServer",
			dbInfo: &config.DatabaseInfo{
				DbServer: "sql-server",
			},
			cfg: &config.Config{
				InfobaseName: "TestDB",
			},
			expected: "/S sql-server\\TestDB",
		},
	}

	h := &DbUpdateHandler{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := h.buildConnectString(tt.dbInfo, tt.cfg)
			if got != tt.expected {
				t.Errorf("buildConnectString() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestBuildConnectString_NilConfigs проверяет построение строки подключения с nil конфигами (L1 fix)
func TestBuildConnectString_NilConfigs(t *testing.T) {
	h := &DbUpdateHandler{}

	// Тест с nil SecretConfig
	cfg := &config.Config{
		InfobaseName: "TestDB",
		AppConfig: &config.AppConfig{
			Users: struct {
				Rac        string `yaml:"rac"`
				Db         string `yaml:"db"`
				Mssql      string `yaml:"mssql"`
				StoreAdmin string `yaml:"storeAdmin"`
			}{
				Db: "admin",
			},
		},
		SecretConfig: nil, // nil SecretConfig
	}
	dbInfo := &config.DatabaseInfo{OneServer: "1c-server"}

	got := h.buildConnectString(dbInfo, cfg)
	expected := "/S 1c-server\\TestDB /N admin"
	if got != expected {
		t.Errorf("buildConnectString() with nil SecretConfig = %q, want %q", got, expected)
	}

	// Тест с nil AppConfig
	cfg2 := &config.Config{
		InfobaseName: "TestDB",
		AppConfig:    nil, // nil AppConfig
		SecretConfig: &config.SecretConfig{
			Passwords: struct {
				Rac                string `yaml:"rac"`
				Db                 string `yaml:"db"`
				Mssql              string `yaml:"mssql"`
				StoreAdminPassword string `yaml:"storeAdminPassword"`
				Smb                string `yaml:"smb"`
			}{
				Db: "secret",
			},
		},
	}

	got2 := h.buildConnectString(dbInfo, cfg2)
	expected2 := "/S 1c-server\\TestDB /P secret"
	if got2 != expected2 {
		t.Errorf("buildConnectString() with nil AppConfig = %q, want %q", got2, expected2)
	}
}

// TestDbUpdateHandler_Execute_ContextCancellation проверяет обработку отменённого контекста (M2 fix)
func TestDbUpdateHandler_Execute_ContextCancellation(t *testing.T) {
	// Mock который проверяет что контекст отменён
	mockClient := &onectest.MockDatabaseUpdater{
		UpdateDBCfgFunc: func(ctx context.Context, opts onec.UpdateOptions) (*onec.UpdateResult, error) {
			// Проверяем состояние контекста
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return &onec.UpdateResult{Success: true}, nil
		},
	}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{
		oneCClient: mockClient,
	}

	// Создаём уже отменённый контекст
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Сразу отменяем

	var err error
	output := captureStdout(func() {
		err = h.Execute(ctx, cfg)
	})

	// Должна быть ошибка из-за отменённого контекста
	if err == nil {
		t.Error("Execute() should return error for cancelled context")
	}

	// Вывод должен содержать информацию об ошибке
	if !strings.Contains(output, "Ошибка") && !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Execute() should report context cancellation error, got output: %s, err: %v", output, err)
	}
}

// TestDbUpdateHandler_Execute_ExtensionContextCancellation проверяет отмену контекста между проходами расширений
// M4 fix: теперь контекст проверяется ПЕРЕД вторым проходом, поэтому второй вызов не происходит
func TestDbUpdateHandler_Execute_ExtensionContextCancellation(t *testing.T) {
	callCount := 0
	// Mock который отменяет контекст после первого вызова
	ctx, cancel := context.WithCancel(context.Background())

	mockClient := &onectest.MockDatabaseUpdater{
		UpdateDBCfgFunc: func(innerCtx context.Context, opts onec.UpdateOptions) (*onec.UpdateResult, error) {
			callCount++
			if callCount == 1 {
				// Первый проход — успех, но отменяем контекст
				cancel()
				return &onec.UpdateResult{Success: true}, nil
			}
			// Второй проход — не должен выполняться благодаря M4 fix
			return &onec.UpdateResult{Success: true}, nil
		},
	}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{
		oneCClient: mockClient,
	}

	t.Setenv("BR_EXTENSION", "TestExtension")

	var err error
	captureStdout(func() {
		err = h.Execute(ctx, cfg)
	})

	// Должна быть ошибка из-за отменённого контекста перед вторым проходом (M4 fix)
	if err == nil {
		t.Error("Execute() should return error when context cancelled between passes")
	}

	// M4 fix: теперь должен быть только 1 вызов (контекст проверяется ДО второго вызова)
	if callCount != 1 {
		t.Errorf("UpdateDBCfg called %d times, want 1 (M4 fix: context checked before second pass)", callCount)
	}
}

// TestDbUpdateHandler_Execute_MissingBin1cv8 проверяет ошибку когда путь к 1cv8 не указан (H1 fix)
func TestDbUpdateHandler_Execute_MissingBin1cv8(t *testing.T) {
	mockClient := onectest.NewMockDatabaseUpdater()

	// Конфигурация без пути к 1cv8
	cfg := &config.Config{
		InfobaseName: "TestDB",
		WorkDir:      "/tmp/work",
		TmpDir:       "/tmp",
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "", // Пустой путь!
			},
		},
		DbConfig: map[string]*config.DatabaseInfo{
			"TestDB": {OneServer: "test-server"},
		},
	}

	h := &DbUpdateHandler{
		oneCClient: mockClient,
	}

	var err error
	output := captureStdout(func() {
		err = h.Execute(context.Background(), cfg)
	})

	if err == nil {
		t.Error("Execute() should return error for missing bin1cv8 path")
	}

	if !strings.Contains(err.Error(), ErrDbUpdateConfig) {
		t.Errorf("Execute() error = %v, want error containing %q", err, ErrDbUpdateConfig)
	}

	if !strings.Contains(output, "1cv8") {
		t.Errorf("Execute() output should mention 1cv8, got: %s", output)
	}
}

// TestDbUpdateHandler_Execute_SuccessFalseNoError проверяет поведение когда UpdateDBCfg возвращает Success=false без ошибки (M2 fix)
func TestDbUpdateHandler_Execute_SuccessFalseNoError(t *testing.T) {
	mockClient := &onectest.MockDatabaseUpdater{
		UpdateDBCfgFunc: func(ctx context.Context, opts onec.UpdateOptions) (*onec.UpdateResult, error) {
			// Платформа 1C может вернуть Success=false без ошибки
			return &onec.UpdateResult{
				Success:    false,
				Messages:   []string{"Нет изменений для обновления"},
				DurationMs: 500,
			}, nil
		},
	}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{
		oneCClient: mockClient,
	}

	var err error
	output := captureStdout(func() {
		err = h.Execute(context.Background(), cfg)
	})

	// Операция должна завершиться без ошибки (err == nil от mock)
	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}

	// Но вывод должен показывать неуспешный результат
	if !strings.Contains(output, "❌") && !strings.Contains(output, "ошибками") {
		t.Errorf("Execute() output should indicate failure, got: %s", output)
	}
}

// TestDbUpdateHandler_Execute_InvalidTimeout проверяет обработку невалидного BR_TIMEOUT_MIN (L2 fix)
func TestDbUpdateHandler_Execute_InvalidTimeout(t *testing.T) {
	tests := []struct {
		name            string
		timeoutValue    string
		expectedTimeout time.Duration
	}{
		{
			name:            "non-numeric value",
			timeoutValue:    "abc",
			expectedTimeout: 30 * time.Minute, // default
		},
		{
			name:            "negative value",
			timeoutValue:    "-5",
			expectedTimeout: 30 * time.Minute, // default
		},
		{
			name:            "zero value",
			timeoutValue:    "0",
			expectedTimeout: 30 * time.Minute, // default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedTimeout time.Duration
			mockClient := &onectest.MockDatabaseUpdater{
				UpdateDBCfgFunc: func(ctx context.Context, opts onec.UpdateOptions) (*onec.UpdateResult, error) {
					capturedTimeout = opts.Timeout
					return &onec.UpdateResult{Success: true}, nil
				},
			}

			cfg := createTestConfig("TestDB")

			h := &DbUpdateHandler{
				oneCClient: mockClient,
			}

			t.Setenv("BR_TIMEOUT_MIN", tt.timeoutValue)

			captureStdout(func() {
				_ = h.Execute(context.Background(), cfg)
			})

			if capturedTimeout != tt.expectedTimeout {
				t.Errorf("Timeout = %v, want %v (default) for invalid input %q",
					capturedTimeout, tt.expectedTimeout, tt.timeoutValue)
			}
		})
	}
}

// TestDbUpdateHandler_Execute_AutoDeps_EnableError проверяет обработку ошибки EnableServiceMode (L2 fix)
func TestDbUpdateHandler_Execute_AutoDeps_EnableError(t *testing.T) {
	mockClient := onectest.NewMockDatabaseUpdater()

	enableError := errors.New("RAC connection failed")
	racMock := ractest.NewMockRACClient()
	racMock.GetServiceModeStatusFunc = func(ctx context.Context, clusterUUID, infobaseUUID string) (*rac.ServiceModeStatus, error) {
		return &rac.ServiceModeStatus{Enabled: false}, nil
	}
	racMock.EnableServiceModeFunc = func(ctx context.Context, clusterUUID, infobaseUUID string, terminateSessions bool) error {
		return enableError
	}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{
		oneCClient: mockClient,
		racClient:  racMock,
	}

	t.Setenv("BR_AUTO_DEPS", "true")

	var err error
	output := captureStdout(func() {
		err = h.Execute(context.Background(), cfg)
	})

	// Операция должна завершиться успешно (auto-deps отключается при ошибке RAC)
	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}

	// Проверяем что обновление прошло, но без auto-deps
	if !strings.Contains(output, "Обновление завершено") {
		t.Errorf("Execute() output should contain success message, got: %s", output)
	}

	// auto-deps не должен быть указан в выводе (т.к. был отключён из-за ошибки)
	if strings.Contains(output, "Auto-deps: включён") {
		t.Errorf("Execute() output should NOT contain auto-deps enabled when RAC failed, got: %s", output)
	}
}

// TestDbUpdateHandler_Execute_MessagesLimit проверяет что сообщения обрезаются при превышении лимита (M2 fix)
func TestDbUpdateHandler_Execute_MessagesLimit(t *testing.T) {
	// Создаём mock который возвращает много сообщений (больше maxMessages=100)
	var manyMessages []string
	for i := 0; i < 150; i++ {
		manyMessages = append(manyMessages, "Сообщение")
	}

	mockClient := &onectest.MockDatabaseUpdater{
		UpdateDBCfgFunc: func(ctx context.Context, opts onec.UpdateOptions) (*onec.UpdateResult, error) {
			return &onec.UpdateResult{
				Success:    true,
				Messages:   manyMessages,
				DurationMs: 100,
			}, nil
		},
	}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{
		oneCClient: mockClient,
	}

	// Устанавливаем расширение для активации двойного прохода (удваивает сообщения)
	t.Setenv("BR_EXTENSION", "TestExtension")

	var err error
	captureStdout(func() {
		err = h.Execute(context.Background(), cfg)
	})

	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}

	// Успешное завершение означает что лимит сообщений сработал без паники
	// (тестируем что код не падает при большом количестве сообщений)
}

// TestGetOrCreateRacClient_NilAppConfig проверяет создание RAC клиента с nil AppConfig (H2 fix)
func TestGetOrCreateRacClient_NilAppConfig(t *testing.T) {
	h := &DbUpdateHandler{}

	cfg := &config.Config{
		InfobaseName: "TestDB",
		AppConfig:    nil, // nil AppConfig
		DbConfig: map[string]*config.DatabaseInfo{
			"TestDB": {OneServer: "test-server"},
		},
	}
	dbInfo := cfg.DbConfig["TestDB"]

	// Не должно паниковать
	client := h.getOrCreateRacClient(cfg, dbInfo, nil)

	// Должен вернуть nil т.к. нет пути к RAC
	if client != nil {
		t.Error("getOrCreateRacClient() should return nil when AppConfig is nil")
	}
}

// TestGetOrCreateRacClient_EmptyRacPath проверяет создание RAC клиента с пустым путём (H2 fix)
func TestGetOrCreateRacClient_EmptyRacPath(t *testing.T) {
	h := &DbUpdateHandler{}

	cfg := &config.Config{
		InfobaseName: "TestDB",
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Rac: "", // Пустой путь
			},
		},
		DbConfig: map[string]*config.DatabaseInfo{
			"TestDB": {OneServer: "test-server"},
		},
	}
	dbInfo := cfg.DbConfig["TestDB"]

	// Не должно паниковать
	client := h.getOrCreateRacClient(cfg, dbInfo, nil)

	// Должен вернуть nil
	if client != nil {
		t.Error("getOrCreateRacClient() should return nil when RAC path is empty")
	}
}

// TestGetOrCreateRacClient_NoServer проверяет создание RAC клиента без сервера (H2 fix)
func TestGetOrCreateRacClient_NoServer(t *testing.T) {
	h := &DbUpdateHandler{}

	cfg := &config.Config{
		InfobaseName: "TestDB",
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Rac: "/opt/1cv8/rac",
			},
		},
		DbConfig: map[string]*config.DatabaseInfo{
			"TestDB": {
				OneServer: "", // Нет сервера
				DbServer:  "", // Нет fallback
			},
		},
	}
	dbInfo := cfg.DbConfig["TestDB"]

	// Не должно паниковать
	client := h.getOrCreateRacClient(cfg, dbInfo, nil)

	// Должен вернуть nil
	if client != nil {
		t.Error("getOrCreateRacClient() should return nil when server is not configured")
	}
}

// TestGetOrCreateRacClient_UsesExistingMock проверяет что mock клиент используется если предоставлен
func TestGetOrCreateRacClient_UsesExistingMock(t *testing.T) {
	racMock := ractest.NewMockRACClient()

	h := &DbUpdateHandler{
		racClient: racMock,
	}

	cfg := createTestConfig("TestDB")
	dbInfo := cfg.DbConfig["TestDB"]

	client := h.getOrCreateRacClient(cfg, dbInfo, nil)

	// Должен вернуть существующий mock
	if client != racMock {
		t.Error("getOrCreateRacClient() should return existing mock client")
	}
}

// ==== DRY-RUN TESTS ====

// TestDbUpdateHandler_DryRun_Success проверяет успешный dry-run с построением плана.
// AC-1: При BR_DRY_RUN=true команды возвращают план действий БЕЗ выполнения.
func TestDbUpdateHandler_DryRun_Success(t *testing.T) {
	// Создаём mock который FAIL-ит при любом вызове
	// AC-8: В dry-run режиме НЕ вызываются 1cv8/ibcmd, RAC операции
	mockClient := &FailOnCallMockUpdater{t: t}
	mockRac := &FailOnCallMockRAC{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{oneCClient: mockClient, racClient: mockRac}

	t.Setenv("BR_DRY_RUN", "true")

	var capturedOutput string
	var err error
	capturedOutput = captureStdout(func() {
		err = h.Execute(context.Background(), cfg)
	})

	// AC-5: exit code = 0 если план валиден
	if err != nil {
		t.Errorf("DryRun Execute() unexpected error = %v", err)
	}

	// AC-4: Text output форматирует план человекочитаемо с заголовком "=== DRY RUN ==="
	expectedParts := []string{
		"=== DRY RUN ===",
		"Команда: nr-dbupdate",
		"Валидация: ✅ Пройдена",
		"План выполнения:",
		"Валидация конфигурации",
		"Обновление структуры базы данных",
		"=== END DRY RUN ===",
	}

	for _, part := range expectedParts {
		if !strings.Contains(capturedOutput, part) {
			t.Errorf("DryRun output should contain %q, got: %s", part, capturedOutput)
		}
	}
}

// TestDbUpdateHandler_DryRun_JSONOutput проверяет JSON формат dry-run вывода.
// AC-3: JSON output имеет поле "dry_run": true и структуру "plan": {...}.
func TestDbUpdateHandler_DryRun_JSONOutput(t *testing.T) {
	mockClient := &FailOnCallMockUpdater{t: t}
	mockRac := &FailOnCallMockRAC{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{oneCClient: mockClient, racClient: mockRac}

	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	if err != nil {
		t.Errorf("DryRun Execute() unexpected error = %v", err)
	}

	var result output.Result
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Errorf("JSON unmarshal error: %v, output: %s", jsonErr, buf.String())
	}

	// AC-3: dry_run: true
	if !result.DryRun {
		t.Error("JSON result.DryRun should be true")
	}

	// AC-3: plan: {...}
	if result.Plan == nil {
		t.Error("JSON result.Plan should not be nil")
	} else {
		if result.Plan.Command != constants.ActNRDbupdate {
			t.Errorf("Plan.Command = %q, want %q", result.Plan.Command, constants.ActNRDbupdate)
		}
		if !result.Plan.ValidationPassed {
			t.Error("Plan.ValidationPassed should be true")
		}
	}
}

// TestDbUpdateHandler_DryRun_NoMockCalls проверяет что mock НЕ вызывается в dry-run.
// AC-8: В dry-run режиме НЕ вызываются 1cv8/ibcmd, RAC операции.
func TestDbUpdateHandler_DryRun_NoMockCalls(t *testing.T) {
	mockClient := &FailOnCallMockUpdater{t: t}
	mockRac := &FailOnCallMockRAC{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{oneCClient: mockClient, racClient: mockRac}

	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_AUTO_DEPS", "true") // RAC тоже не должен вызываться

	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("DryRun Execute() unexpected error = %v", err)
	}
}

// TestDbUpdateHandler_DryRun_WithExtension проверяет dry-run с расширением.
func TestDbUpdateHandler_DryRun_WithExtension(t *testing.T) {
	mockClient := &FailOnCallMockUpdater{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{oneCClient: mockClient}

	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_EXTENSION", "MyExtension")
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	if err != nil {
		t.Errorf("DryRun Execute() unexpected error = %v", err)
	}

	var result output.Result
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Fatalf("JSON unmarshal error: %v", jsonErr)
	}

	// Проверяем что расширение упоминается в summary
	if result.Plan == nil {
		t.Fatal("Plan should not be nil")
	}
	if !strings.Contains(result.Plan.Summary, "MyExtension") {
		t.Errorf("Plan.Summary should contain extension name, got: %s", result.Plan.Summary)
	}
}

// TestDbUpdateHandler_DryRun_WithAutoDeps проверяет dry-run с auto-deps.
func TestDbUpdateHandler_DryRun_WithAutoDeps(t *testing.T) {
	mockClient := &FailOnCallMockUpdater{t: t}
	mockRac := &FailOnCallMockRAC{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{oneCClient: mockClient, racClient: mockRac}

	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_AUTO_DEPS", "true")

	var capturedOutput string
	capturedOutput = captureStdout(func() {
		_ = h.Execute(context.Background(), cfg)
	})

	// Проверяем что шаги service mode включены в план
	if !strings.Contains(capturedOutput, "сервисного режима") {
		t.Errorf("DryRun with auto-deps should mention service mode, got: %s", capturedOutput)
	}
}

// TestDbUpdateHandler_DryRun_PasswordMasked проверяет маскирование пароля в плане.
// SECURITY: пароли НЕ должны появляться в dry-run плане.
func TestDbUpdateHandler_DryRun_PasswordMasked(t *testing.T) {
	mockClient := &FailOnCallMockUpdater{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{oneCClient: mockClient}

	t.Setenv("BR_DRY_RUN", "true")

	var capturedOutput string
	capturedOutput = captureStdout(func() {
		_ = h.Execute(context.Background(), cfg)
	})

	// Пароль "test-password" НЕ должен появиться в выводе
	if strings.Contains(capturedOutput, "test-password") {
		t.Error("DryRun output should NOT contain password")
	}

	// Маска "***" должна присутствовать
	if !strings.Contains(capturedOutput, "***") {
		t.Error("DryRun output should contain masked password (***)")
	}
}

// TestDbUpdateHandler_DryRun_ValidationError проверяет что dry-run возвращает ошибку валидации.
// AC-6: При ошибке валидации возвращается error с описанием проблемы.
func TestDbUpdateHandler_DryRun_ValidationError(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		wantErrCode string
	}{
		{
			name:        "nil config",
			cfg:         nil,
			wantErrCode: ErrDbUpdateValidation,
		},
		{
			name:        "empty infobase name",
			cfg:         &config.Config{InfobaseName: ""},
			wantErrCode: ErrDbUpdateValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &FailOnCallMockUpdater{t: t}
			h := &DbUpdateHandler{oneCClient: mockClient}

			t.Setenv("BR_DRY_RUN", "true")

			// Перехватываем stdout
			oldStdout := os.Stdout
			_, w, _ := os.Pipe()
			os.Stdout = w

			err := h.Execute(context.Background(), tt.cfg)

			w.Close()
			os.Stdout = oldStdout

			// AC-6: Должна быть ошибка валидации
			if err == nil {
				t.Error("DryRun Execute() should return error for invalid config")
			}

			if err != nil && !strings.Contains(err.Error(), tt.wantErrCode) {
				t.Errorf("DryRun Execute() error = %v, want error containing %q", err, tt.wantErrCode)
			}
		})
	}
}

// TestMaskPassword тестируется в internal/pkg/dryrun/dryrun_test.go
// Функция перемещена в общий пакет dryrun.MaskPassword()

// FailOnCallMockUpdater — mock который падает при любом вызове.
type FailOnCallMockUpdater struct {
	t *testing.T
}

func (m *FailOnCallMockUpdater) UpdateDBCfg(ctx context.Context, opts onec.UpdateOptions) (*onec.UpdateResult, error) {
	m.t.Fatal("UpdateDBCfg() не должен вызываться в dry-run режиме")
	return nil, nil
}

// FailOnCallMockRAC — mock который падает при любом вызове RAC.
type FailOnCallMockRAC struct {
	t *testing.T
}

func (m *FailOnCallMockRAC) GetClusterInfo(ctx context.Context) (*rac.ClusterInfo, error) {
	m.t.Fatal("GetClusterInfo() не должен вызываться в dry-run режиме")
	return nil, nil
}

func (m *FailOnCallMockRAC) GetInfobaseInfo(ctx context.Context, clusterUUID, infobaseName string) (*rac.InfobaseInfo, error) {
	m.t.Fatal("GetInfobaseInfo() не должен вызываться в dry-run режиме")
	return nil, nil
}

func (m *FailOnCallMockRAC) GetServiceModeStatus(ctx context.Context, clusterUUID, infobaseUUID string) (*rac.ServiceModeStatus, error) {
	m.t.Fatal("GetServiceModeStatus() не должен вызываться в dry-run режиме")
	return nil, nil
}

func (m *FailOnCallMockRAC) EnableServiceMode(ctx context.Context, clusterUUID, infobaseUUID string, terminateSessions bool) error {
	m.t.Fatal("EnableServiceMode() не должен вызываться в dry-run режиме")
	return nil
}

func (m *FailOnCallMockRAC) DisableServiceMode(ctx context.Context, clusterUUID, infobaseUUID string) error {
	m.t.Fatal("DisableServiceMode() не должен вызываться в dry-run режиме")
	return nil
}

func (m *FailOnCallMockRAC) GetSessions(ctx context.Context, clusterUUID, infobaseUUID string) ([]rac.SessionInfo, error) {
	m.t.Fatal("GetSessions() не должен вызываться в dry-run режиме")
	return nil, nil
}

func (m *FailOnCallMockRAC) TerminateSession(ctx context.Context, clusterUUID, sessionUUID string) error {
	m.t.Fatal("TerminateSession() не должен вызываться в dry-run режиме")
	return nil
}

func (m *FailOnCallMockRAC) TerminateAllSessions(ctx context.Context, clusterUUID, infobaseUUID string) error {
	m.t.Fatal("TerminateAllSessions() не должен вызываться в dry-run режиме")
	return nil
}

func (m *FailOnCallMockRAC) VerifyServiceMode(ctx context.Context, clusterUUID, infobaseUUID string, expectedEnabled bool) error {
	m.t.Fatal("VerifyServiceMode() не должен вызываться в dry-run режиме")
	return nil
}

// createTestConfig создаёт тестовую конфигурацию для указанной базы данных
func createTestConfig(dbName string) *config.Config {
	return &config.Config{
		InfobaseName: dbName,
		WorkDir:      "/tmp/work",
		TmpDir:       "/tmp",
		AppConfig: &config.AppConfig{
			Paths: struct {
				Bin1cv8  string `yaml:"bin1cv8"`
				BinIbcmd string `yaml:"binIbcmd"`
				EdtCli   string `yaml:"edtCli"`
				Rac      string `yaml:"rac"`
			}{
				Bin1cv8: "/opt/1cv8/bin/1cv8",
				Rac:     "/opt/1cv8/rac",
			},
			Users: struct {
				Rac        string `yaml:"rac"`
				Db         string `yaml:"db"`
				Mssql      string `yaml:"mssql"`
				StoreAdmin string `yaml:"storeAdmin"`
			}{
				Db: "test-user",
			},
		},
		SecretConfig: &config.SecretConfig{
			Passwords: struct {
				Rac                string `yaml:"rac"`
				Db                 string `yaml:"db"`
				Mssql              string `yaml:"mssql"`
				StoreAdminPassword string `yaml:"storeAdminPassword"`
				Smb                string `yaml:"smb"`
			}{
				Db: "test-password",
			},
		},
		DbConfig: map[string]*config.DatabaseInfo{
			dbName: {
				OneServer: "test-1c-server",
				DbServer:  "test-sql-server",
				Prod:      false,
			},
		},
	}
}

// ==== PLAN-ONLY / VERBOSE / PRIORITY TESTS ====

// TestDbUpdateHandler_PlanOnly_TextOutput проверяет текстовый вывод в режиме plan-only.
// Story 7.3 AC-2: Text output с заголовком "=== OPERATION PLAN ===".
func TestDbUpdateHandler_PlanOnly_TextOutput(t *testing.T) {
	mockClient := &FailOnCallMockUpdater{t: t}
	mockRac := &FailOnCallMockRAC{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{oneCClient: mockClient, racClient: mockRac}

	t.Setenv("BR_PLAN_ONLY", "true")

	var err error
	capturedOutput := captureStdout(func() {
		err = h.Execute(context.Background(), cfg)
	})

	if err != nil {
		t.Errorf("PlanOnly Execute() unexpected error = %v", err)
	}

	// Проверяем заголовок plan-only (НЕ dry-run)
	if !strings.Contains(capturedOutput, "=== OPERATION PLAN ===") {
		t.Errorf("PlanOnly output should contain '=== OPERATION PLAN ===', got: %s", capturedOutput)
	}
	if !strings.Contains(capturedOutput, "=== END OPERATION PLAN ===") {
		t.Errorf("PlanOnly output should contain '=== END OPERATION PLAN ===', got: %s", capturedOutput)
	}

	// Проверяем имя команды
	if !strings.Contains(capturedOutput, constants.ActNRDbupdate) {
		t.Errorf("PlanOnly output should contain command name %q, got: %s", constants.ActNRDbupdate, capturedOutput)
	}
}

// TestDbUpdateHandler_PlanOnly_JSONOutput проверяет JSON вывод в режиме plan-only.
// Story 7.3 AC-6: JSON output содержит plan_only: true.
func TestDbUpdateHandler_PlanOnly_JSONOutput(t *testing.T) {
	mockClient := &FailOnCallMockUpdater{t: t}
	mockRac := &FailOnCallMockRAC{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{oneCClient: mockClient, racClient: mockRac}

	t.Setenv("BR_PLAN_ONLY", "true")
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	if err != nil {
		t.Errorf("PlanOnly Execute() unexpected error = %v", err)
	}

	var result output.Result
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Errorf("JSON unmarshal error: %v, output: %s", jsonErr, buf.String())
	}

	// AC-6: plan_only: true
	if !result.PlanOnly {
		t.Error("JSON result.PlanOnly should be true")
	}

	// plan-only НЕ является dry-run
	if result.DryRun {
		t.Error("JSON result.DryRun should be false for plan-only mode")
	}

	// План должен присутствовать
	if result.Plan == nil {
		t.Fatal("JSON result.Plan should not be nil")
	}

	if result.Plan.Command != constants.ActNRDbupdate {
		t.Errorf("Plan.Command = %q, want %q", result.Plan.Command, constants.ActNRDbupdate)
	}
}

// TestDbUpdateHandler_PlanOnly_NoExecution проверяет что plan-only НЕ выполняет реальные операции.
// Также проверяет что RAC не вызывается даже при BR_AUTO_DEPS=true.
func TestDbUpdateHandler_PlanOnly_NoExecution(t *testing.T) {
	mockClient := &FailOnCallMockUpdater{t: t}
	mockRac := &FailOnCallMockRAC{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{oneCClient: mockClient, racClient: mockRac}

	t.Setenv("BR_PLAN_ONLY", "true")
	t.Setenv("BR_AUTO_DEPS", "true")

	// Перехватываем stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	w.Close()
	os.Stdout = oldStdout

	// Если mock-и не упали через t.Fatal — значит ни один не был вызван
	if err != nil {
		t.Errorf("PlanOnly Execute() unexpected error = %v", err)
	}
}

// TestDbUpdateHandler_Verbose_TextOutput проверяет verbose режим: план выводится ПЕРЕД выполнением.
// Story 7.3 AC-4: В verbose режиме сначала выводится план, затем выполняется операция.
func TestDbUpdateHandler_Verbose_TextOutput(t *testing.T) {
	mockClient := onectest.NewMockDatabaseUpdater()

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{oneCClient: mockClient}

	t.Setenv("BR_VERBOSE", "true")

	var err error
	capturedOutput := captureStdout(func() {
		err = h.Execute(context.Background(), cfg)
	})

	if err != nil {
		t.Errorf("Verbose Execute() unexpected error = %v", err)
	}

	// Проверяем что план выведен перед выполнением
	if !strings.Contains(capturedOutput, "=== OPERATION PLAN ===") {
		t.Errorf("Verbose output should contain '=== OPERATION PLAN ===', got: %s", capturedOutput)
	}

	// Проверяем что реальное выполнение произошло
	if !strings.Contains(capturedOutput, "Обновление завершено") {
		t.Errorf("Verbose output should contain 'Обновление завершено', got: %s", capturedOutput)
	}
}

// TestDbUpdateHandler_Verbose_JSONOutput проверяет JSON вывод в verbose режиме.
// Story 7.3 AC-7: verbose JSON включает план в результат.
func TestDbUpdateHandler_Verbose_JSONOutput(t *testing.T) {
	mockClient := onectest.NewMockDatabaseUpdater()

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{oneCClient: mockClient}

	t.Setenv("BR_VERBOSE", "true")
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	if err != nil {
		t.Errorf("Verbose Execute() unexpected error = %v", err)
	}

	var result output.Result
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Errorf("JSON unmarshal error: %v, output: %s", jsonErr, buf.String())
	}

	// Verbose включает план в JSON результат
	if result.Plan == nil {
		t.Error("Verbose JSON result.Plan should not be nil")
	}

	// Verbose — не plan-only и не dry-run
	if result.PlanOnly {
		t.Error("Verbose JSON result.PlanOnly should be false")
	}
	if result.DryRun {
		t.Error("Verbose JSON result.DryRun should be false")
	}

	// Реальное выполнение должно произойти
	if result.Status != output.StatusSuccess {
		t.Errorf("Verbose JSON result.Status = %q, want %q", result.Status, output.StatusSuccess)
	}

	// Data должна присутствовать (реальное выполнение)
	if result.Data == nil {
		t.Error("Verbose JSON result.Data should not be nil (real execution happened)")
	}
}

// TestDbUpdateHandler_Priority_DryRunOverPlanOnly проверяет приоритет dry-run над plan-only.
// AC-9: dry-run имеет высший приоритет — при наличии обоих флагов выполняется dry-run.
func TestDbUpdateHandler_Priority_DryRunOverPlanOnly(t *testing.T) {
	mockClient := &FailOnCallMockUpdater{t: t}
	mockRac := &FailOnCallMockRAC{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{oneCClient: mockClient, racClient: mockRac}

	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_PLAN_ONLY", "true")

	var err error
	capturedOutput := captureStdout(func() {
		err = h.Execute(context.Background(), cfg)
	})

	if err != nil {
		t.Errorf("Priority Execute() unexpected error = %v", err)
	}

	// Должен быть dry-run заголовок, НЕ plan-only
	if !strings.Contains(capturedOutput, "=== DRY RUN ===") {
		t.Errorf("Output should contain '=== DRY RUN ===' (dry-run priority), got: %s", capturedOutput)
	}
	if strings.Contains(capturedOutput, "=== OPERATION PLAN ===") {
		t.Errorf("Output should NOT contain '=== OPERATION PLAN ===' when dry-run active, got: %s", capturedOutput)
	}
}

// TestDbUpdateHandler_Priority_DryRunOverVerbose проверяет приоритет dry-run над verbose.
// AC-9: dry-run имеет высший приоритет над verbose.
func TestDbUpdateHandler_Priority_DryRunOverVerbose(t *testing.T) {
	mockClient := &FailOnCallMockUpdater{t: t}
	mockRac := &FailOnCallMockRAC{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{oneCClient: mockClient, racClient: mockRac}

	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_VERBOSE", "true")

	var err error
	capturedOutput := captureStdout(func() {
		err = h.Execute(context.Background(), cfg)
	})

	if err != nil {
		t.Errorf("Priority Execute() unexpected error = %v", err)
	}

	// Должен быть dry-run заголовок, НЕ operation plan
	if !strings.Contains(capturedOutput, "=== DRY RUN ===") {
		t.Errorf("Output should contain '=== DRY RUN ===' (dry-run priority), got: %s", capturedOutput)
	}
	if strings.Contains(capturedOutput, "=== OPERATION PLAN ===") {
		t.Errorf("Output should NOT contain '=== OPERATION PLAN ===' when dry-run active, got: %s", capturedOutput)
	}
}

// TestDbUpdateHandler_Priority_PlanOnlyOverVerbose проверяет приоритет plan-only над verbose.
// Plan-only останавливает выполнение (показывает план, не выполняет).
// Verbose показывает план и выполняет. Plan-only имеет приоритет.
func TestDbUpdateHandler_Priority_PlanOnlyOverVerbose(t *testing.T) {
	mockClient := &FailOnCallMockUpdater{t: t}
	mockRac := &FailOnCallMockRAC{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbUpdateHandler{oneCClient: mockClient, racClient: mockRac}

	t.Setenv("BR_PLAN_ONLY", "true")
	t.Setenv("BR_VERBOSE", "true")

	var err error
	capturedOutput := captureStdout(func() {
		err = h.Execute(context.Background(), cfg)
	})

	if err != nil {
		t.Errorf("Priority Execute() unexpected error = %v", err)
	}

	// Должен быть plan-only заголовок
	if !strings.Contains(capturedOutput, "=== OPERATION PLAN ===") {
		t.Errorf("Output should contain '=== OPERATION PLAN ===', got: %s", capturedOutput)
	}

	// НЕ должно быть реального выполнения
	if strings.Contains(capturedOutput, "Обновление завершено") {
		t.Errorf("Output should NOT contain 'Обновление завершено' (plan-only, no execution), got: %s", capturedOutput)
	}
}
