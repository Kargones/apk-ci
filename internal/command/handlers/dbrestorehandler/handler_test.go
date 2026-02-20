package dbrestorehandler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/mssql"
	"github.com/Kargones/apk-ci/internal/adapter/mssql/mssqltest"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/output"
)

// captureStdout перехватывает stdout во время выполнения функции и возвращает вывод (M1 fix)
func captureStdout(fn func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

// TestDbRestoreHandler_Name проверяет возврат имени команды
func TestDbRestoreHandler_Name(t *testing.T) {
	h := &DbRestoreHandler{}
	want := constants.ActNRDbrestore
	if got := h.Name(); got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
}

// TestDbRestoreHandler_Description проверяет возврат описания команды
func TestDbRestoreHandler_Description(t *testing.T) {
	h := &DbRestoreHandler{}
	if desc := h.Description(); desc == "" {
		t.Error("Description() returned empty string")
	}
}

// TestDbRestoreHandler_Execute_MissingInfobaseName проверяет ошибку при отсутствии BR_INFOBASE_NAME
func TestDbRestoreHandler_Execute_MissingInfobaseName(t *testing.T) {
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
			h := &DbRestoreHandler{}
			err := h.Execute(context.Background(), tt.cfg)
			if err == nil {
				t.Error("Execute() should return error for missing infobase name")
			}
			if err != nil && !strings.Contains(err.Error(), ErrDbRestoreConfigMissing) {
				t.Errorf("Execute() error = %v, want error containing %q", err, ErrDbRestoreConfigMissing)
			}
		})
	}
}

// TestDbRestoreHandler_Execute_ProductionRestoreForbidden проверяет блокировку restore в production
func TestDbRestoreHandler_Execute_ProductionRestoreForbidden(t *testing.T) {
	// Конфигурация с production базой
	cfg := &config.Config{
		InfobaseName: "ProdDB",
		DbConfig: map[string]*config.DatabaseInfo{
			"ProdDB": {
				DbServer: "prod-server",
				Prod:     true, // PRODUCTION!
			},
		},
	}

	h := &DbRestoreHandler{}
	err := h.Execute(context.Background(), cfg)

	if err == nil {
		t.Fatal("Execute() должен вернуть ошибку для production базы")
	}
	if !strings.Contains(err.Error(), ErrDbRestoreProductionForbidden) {
		t.Errorf("Execute() error = %v, want error containing %q", err, ErrDbRestoreProductionForbidden)
	}
}

// TestDbRestoreHandler_Execute_SuccessfulRestore проверяет успешное восстановление с mock
func TestDbRestoreHandler_Execute_SuccessfulRestore(t *testing.T) {
	// Создаём mock клиент
	mockClient := &mssqltest.MockMSSQLClient{
		ConnectFunc: func(ctx context.Context) error {
			return nil
		},
		CloseFunc: func() error {
			return nil
		},
		RestoreFunc: func(ctx context.Context, opts mssql.RestoreOptions) error {
			// Проверяем, что параметры корректны
			if opts.DstDB != "TestDB" {
				t.Errorf("Restore() got DstDB = %q, want %q", opts.DstDB, "TestDB")
			}
			if opts.SrcDB != "ProdDB" {
				t.Errorf("Restore() got SrcDB = %q, want %q", opts.SrcDB, "ProdDB")
			}
			return nil
		},
		GetRestoreStatsFunc: func(ctx context.Context, opts mssql.StatsOptions) (*mssql.RestoreStats, error) {
			return &mssql.RestoreStats{
				AvgRestoreTimeSec: 180,
				MaxRestoreTimeSec: 600,
				HasData:           true,
			}, nil
		},
	}

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{
		mssqlClient: mockClient,
	}

	var err error
	output := captureStdout(func() {
		err = h.Execute(context.Background(), cfg)
	})

	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}

	// Проверяем вывод
	if !strings.Contains(output, "Восстановление завершено") {
		t.Errorf("Execute() output should contain success message, got: %s", output)
	}
}

// TestDbRestoreHandler_Execute_AutoTimeout проверяет расчёт auto-timeout из статистики
func TestDbRestoreHandler_Execute_AutoTimeout(t *testing.T) {
	maxRestoreTimeSec := int64(600) // 10 минут
	expectedMinTimeout := time.Duration(float64(maxRestoreTimeSec)*1.7) * time.Second

	var capturedTimeout time.Duration
	var statsCallCount int
	mockClient := &mssqltest.MockMSSQLClient{
		ConnectFunc: func(ctx context.Context) error {
			return nil
		},
		CloseFunc: func() error {
			return nil
		},
		RestoreFunc: func(ctx context.Context, opts mssql.RestoreOptions) error {
			capturedTimeout = opts.Timeout
			return nil
		},
		GetRestoreStatsFunc: func(ctx context.Context, opts mssql.StatsOptions) (*mssql.RestoreStats, error) {
			statsCallCount++
			return &mssql.RestoreStats{
				AvgRestoreTimeSec: 180,
				MaxRestoreTimeSec: maxRestoreTimeSec,
				HasData:           true,
			}, nil
		},
	}

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{
		mssqlClient: mockClient,
	}

	// Устанавливаем BR_AUTO_TIMEOUT=true (используем t.Setenv для автоматической очистки)
	t.Setenv("BR_AUTO_TIMEOUT", "true")

	// Перехватываем stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}

	// Проверяем что GetRestoreStats был вызван (M2 fix)
	if statsCallCount == 0 {
		t.Error("GetRestoreStats() should be called when auto-timeout is enabled")
	}

	if capturedTimeout < expectedMinTimeout-time.Second || capturedTimeout > expectedMinTimeout+time.Second {
		t.Errorf("Auto-timeout = %v, want approximately %v", capturedTimeout, expectedMinTimeout)
	}
}

// TestDbRestoreHandler_Execute_ExplicitTimeout проверяет переопределение таймаута через BR_TIMEOUT_MIN
func TestDbRestoreHandler_Execute_ExplicitTimeout(t *testing.T) {
	var capturedTimeout time.Duration
	var statsCallCount int
	mockClient := &mssqltest.MockMSSQLClient{
		ConnectFunc: func(ctx context.Context) error {
			return nil
		},
		CloseFunc: func() error {
			return nil
		},
		RestoreFunc: func(ctx context.Context, opts mssql.RestoreOptions) error {
			capturedTimeout = opts.Timeout
			return nil
		},
		GetRestoreStatsFunc: func(ctx context.Context, opts mssql.StatsOptions) (*mssql.RestoreStats, error) {
			statsCallCount++
			// Статистика должна быть проигнорирована при явном таймауте
			return &mssql.RestoreStats{
				MaxRestoreTimeSec: 600,
				HasData:           true,
			}, nil
		},
	}

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{
		mssqlClient: mockClient,
	}

	// Устанавливаем явный таймаут (используем t.Setenv для автоматической очистки)
	t.Setenv("BR_TIMEOUT_MIN", "15")

	// Перехватываем stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}

	// При явном таймауте GetRestoreStats НЕ должен вызываться (оптимизация)
	if statsCallCount > 0 {
		t.Error("GetRestoreStats() should NOT be called when explicit timeout is set")
	}

	expectedTimeout := 15 * time.Minute
	if capturedTimeout != expectedTimeout {
		t.Errorf("Explicit timeout = %v, want %v", capturedTimeout, expectedTimeout)
	}
}

// TestDbRestoreHandler_Execute_JSONOutput проверяет JSON формат вывода
func TestDbRestoreHandler_Execute_JSONOutput(t *testing.T) {
	mockClient := mssqltest.NewMockMSSQLClient()

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{
		mssqlClient: mockClient,
	}

	// Используем t.Setenv для автоматической очистки
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	// Перехватываем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
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
	if result.Command != constants.ActNRDbrestore {
		t.Errorf("JSON result.Command = %q, want %q", result.Command, constants.ActNRDbrestore)
	}
}

// TestDbRestoreHandler_Execute_ConnectError проверяет обработку ошибки подключения (H2 fix)
func TestDbRestoreHandler_Execute_ConnectError(t *testing.T) {
	connectError := errors.New("connection refused")
	mockClient := &mssqltest.MockMSSQLClient{
		ConnectFunc: func(ctx context.Context) error {
			return connectError
		},
		CloseFunc: func() error {
			return nil
		},
	}

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{
		mssqlClient: mockClient,
	}

	// Перехватываем stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	if err == nil {
		t.Error("Execute() should return error on connect failure")
	}
	if !strings.Contains(err.Error(), ErrDbRestoreConnectFailed) {
		t.Errorf("Execute() error = %v, want error containing %q", err, ErrDbRestoreConnectFailed)
	}
}

// TestDbRestoreHandler_Execute_MinTimeoutOnEmptyStats проверяет минимальный таймаут при пустой статистике (M3 fix)
func TestDbRestoreHandler_Execute_MinTimeoutOnEmptyStats(t *testing.T) {
	var capturedTimeout time.Duration
	mockClient := &mssqltest.MockMSSQLClient{
		ConnectFunc: func(ctx context.Context) error {
			return nil
		},
		CloseFunc: func() error {
			return nil
		},
		RestoreFunc: func(ctx context.Context, opts mssql.RestoreOptions) error {
			capturedTimeout = opts.Timeout
			return nil
		},
		GetRestoreStatsFunc: func(ctx context.Context, opts mssql.StatsOptions) (*mssql.RestoreStats, error) {
			// Возвращаем пустую статистику
			return &mssql.RestoreStats{
				HasData: false,
			}, nil
		},
	}

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{
		mssqlClient: mockClient,
	}

	// Включаем auto-timeout
	t.Setenv("BR_AUTO_TIMEOUT", "true")

	// Перехватываем stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}

	// При пустой статистике должен использоваться минимальный таймаут (5 минут)
	expectedMinTimeout := 5 * time.Minute
	if capturedTimeout != expectedMinTimeout {
		t.Errorf("Timeout with empty stats = %v, want %v (minTimeout)", capturedTimeout, expectedMinTimeout)
	}
}

// TestDbRestoreHandler_Execute_GetRestoreStatsError проверяет обработку ошибки GetRestoreStats (H1 fix)
func TestDbRestoreHandler_Execute_GetRestoreStatsError(t *testing.T) {
	var capturedTimeout time.Duration
	statsError := errors.New("database connection lost")
	mockClient := &mssqltest.MockMSSQLClient{
		ConnectFunc: func(ctx context.Context) error {
			return nil
		},
		CloseFunc: func() error {
			return nil
		},
		RestoreFunc: func(ctx context.Context, opts mssql.RestoreOptions) error {
			capturedTimeout = opts.Timeout
			return nil
		},
		GetRestoreStatsFunc: func(ctx context.Context, opts mssql.StatsOptions) (*mssql.RestoreStats, error) {
			// Возвращаем ошибку
			return nil, statsError
		},
	}

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{
		mssqlClient: mockClient,
	}

	// Включаем auto-timeout
	t.Setenv("BR_AUTO_TIMEOUT", "true")

	// Перехватываем stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}

	// При ошибке GetRestoreStats должен использоваться минимальный таймаут (5 минут)
	expectedMinTimeout := 5 * time.Minute
	if capturedTimeout != expectedMinTimeout {
		t.Errorf("Timeout on stats error = %v, want %v (minTimeout)", capturedTimeout, expectedMinTimeout)
	}
}

// TestDbRestoreHandler_Execute_CalculatedTimeoutBelowMin проверяет граничное значение auto-timeout (H4 fix)
func TestDbRestoreHandler_Execute_CalculatedTimeoutBelowMin(t *testing.T) {
	var capturedTimeout time.Duration
	// Максимальное время восстановления 100 секунд * 1.7 = 170 секунд = 2.83 минуты < 5 минут
	maxRestoreTimeSec := int64(100)
	mockClient := &mssqltest.MockMSSQLClient{
		ConnectFunc: func(ctx context.Context) error {
			return nil
		},
		CloseFunc: func() error {
			return nil
		},
		RestoreFunc: func(ctx context.Context, opts mssql.RestoreOptions) error {
			capturedTimeout = opts.Timeout
			return nil
		},
		GetRestoreStatsFunc: func(ctx context.Context, opts mssql.StatsOptions) (*mssql.RestoreStats, error) {
			return &mssql.RestoreStats{
				AvgRestoreTimeSec: 50,
				MaxRestoreTimeSec: maxRestoreTimeSec,
				HasData:           true,
			}, nil
		},
	}

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{
		mssqlClient: mockClient,
	}

	// Включаем auto-timeout
	t.Setenv("BR_AUTO_TIMEOUT", "true")

	// Перехватываем stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}

	// Рассчитанный таймаут 170 секунд < 5 минут, должен использоваться minTimeout
	expectedMinTimeout := 5 * time.Minute
	if capturedTimeout != expectedMinTimeout {
		t.Errorf("Timeout when calculated < min = %v, want %v (minTimeout)", capturedTimeout, expectedMinTimeout)
	}
}

// TestDbRestoreHandler_Execute_RestoreError проверяет обработку ошибки восстановления
func TestDbRestoreHandler_Execute_RestoreError(t *testing.T) {
	restoreError := errors.New("restore operation failed")
	mockClient := &mssqltest.MockMSSQLClient{
		ConnectFunc: func(ctx context.Context) error {
			return nil
		},
		CloseFunc: func() error {
			return nil
		},
		RestoreFunc: func(ctx context.Context, opts mssql.RestoreOptions) error {
			return restoreError
		},
		GetRestoreStatsFunc: func(ctx context.Context, opts mssql.StatsOptions) (*mssql.RestoreStats, error) {
			return &mssql.RestoreStats{HasData: false}, nil
		},
	}

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{
		mssqlClient: mockClient,
	}

	// Перехватываем stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	if err == nil {
		t.Error("Execute() should return error on restore failure")
	}
	if !strings.Contains(err.Error(), ErrDbRestoreRestoreFailed) {
		t.Errorf("Execute() error = %v, want error containing %q", err, ErrDbRestoreRestoreFailed)
	}
}

// TestIsProductionDatabase проверяет функцию определения production базы
func TestIsProductionDatabase(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		dbName   string
		expected bool
	}{
		{
			name:     "nil config",
			cfg:      nil,
			dbName:   "TestDB",
			expected: false,
		},
		{
			name:     "nil DbConfig",
			cfg:      &config.Config{},
			dbName:   "TestDB",
			expected: false,
		},
		{
			name: "production database",
			cfg: &config.Config{
				DbConfig: map[string]*config.DatabaseInfo{
					"ProdDB": {Prod: true},
				},
			},
			dbName:   "ProdDB",
			expected: true,
		},
		{
			name: "test database",
			cfg: &config.Config{
				DbConfig: map[string]*config.DatabaseInfo{
					"TestDB": {Prod: false},
				},
			},
			dbName:   "TestDB",
			expected: false,
		},
		{
			name: "unknown database",
			cfg: &config.Config{
				DbConfig: map[string]*config.DatabaseInfo{
					"OtherDB": {Prod: true},
				},
			},
			dbName:   "UnknownDB",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isProductionDatabase(tt.cfg, tt.dbName)
			if got != tt.expected {
				t.Errorf("isProductionDatabase() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestDbRestoreData_writeText проверяет текстовый вывод результата
func TestDbRestoreData_writeText(t *testing.T) {
	data := &DbRestoreData{
		SrcServer:   "src-server",
		SrcDB:       "SrcDB",
		DstServer:   "dst-server",
		DstDB:       "DstDB",
		DurationMs:  5000,                                  // 5 секунд в миллисекундах
		TimeoutMs:   int64(10 * time.Minute / time.Millisecond), // 10 минут в миллисекундах
		AutoTimeout: true,
	}

	var buf bytes.Buffer
	err := data.writeText(&buf)
	if err != nil {
		t.Errorf("writeText() error = %v", err)
	}

	output := buf.String()
	expectedParts := []string{
		"Восстановление завершено",
		"Источник: src-server/SrcDB",
		"Назначение: dst-server/DstDB",
		"авто: да",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("writeText() output should contain %q, got: %s", part, output)
		}
	}
}

// TestDetermineSrcAndDstServers проверяет определение серверов
func TestDetermineSrcAndDstServers(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *config.Config
		dstDB      string
		wantSrcDB  string
		wantSrcSrv string
		wantDstSrv string
		wantErr    bool
		errContain string // ожидаемая подстрока в ошибке
	}{
		{
			name:    "nil config",
			cfg:     nil,
			dstDB:   "TestDB",
			wantErr: true,
		},
		{
			name: "successful determination",
			cfg: &config.Config{
				DbConfig: map[string]*config.DatabaseInfo{
					"ProdDB": {DbServer: "prod-sql-server", Prod: true},
					"TestDB": {DbServer: "test-sql-server", Prod: false},
				},
				ProjectConfig: &config.ProjectConfig{
					Prod: map[string]struct {
						DbName     string                 `yaml:"dbName"`
						AddDisable []string               `yaml:"add-disable"`
						Related    map[string]interface{} `yaml:"related"`
					}{
						"ProdDB": {
							Related: map[string]interface{}{
								"TestDB": nil,
							},
						},
					},
				},
			},
			dstDB:      "TestDB",
			wantSrcDB:  "ProdDB",
			wantSrcSrv: "prod-sql-server",
			wantDstSrv: "test-sql-server",
			wantErr:    false,
		},
		{
			name: "production database not found",
			cfg: &config.Config{
				DbConfig: map[string]*config.DatabaseInfo{
					"UnrelatedDB": {DbServer: "other-server"},
				},
				ProjectConfig: &config.ProjectConfig{
					Prod: map[string]struct {
						DbName     string                 `yaml:"dbName"`
						AddDisable []string               `yaml:"add-disable"`
						Related    map[string]interface{} `yaml:"related"`
					}{},
				},
			},
			dstDB:   "TestDB",
			wantErr: true,
		},
		{
			name: "same server forbidden - защита от восстановления на тот же сервер",
			cfg: &config.Config{
				DbConfig: map[string]*config.DatabaseInfo{
					"ProdDB": {DbServer: "same-server", Prod: true},
					"TestDB": {DbServer: "same-server", Prod: false}, // ТОТ ЖЕ СЕРВЕР!
				},
				ProjectConfig: &config.ProjectConfig{
					Prod: map[string]struct {
						DbName     string                 `yaml:"dbName"`
						AddDisable []string               `yaml:"add-disable"`
						Related    map[string]interface{} `yaml:"related"`
					}{
						"ProdDB": {
							Related: map[string]interface{}{
								"TestDB": nil,
							},
						},
					},
				},
			},
			dstDB:      "TestDB",
			wantErr:    true,
			errContain: "восстановление на тот же сервер",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcDB, srcServer, dstServer, err := determineSrcAndDstServers(tt.cfg, tt.dstDB)

			if (err != nil) != tt.wantErr {
				t.Errorf("determineSrcAndDstServers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContain != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("determineSrcAndDstServers() error = %v, want error containing %q", err, tt.errContain)
				}
				return
			}

			if !tt.wantErr {
				if srcDB != tt.wantSrcDB {
					t.Errorf("srcDB = %q, want %q", srcDB, tt.wantSrcDB)
				}
				if srcServer != tt.wantSrcSrv {
					t.Errorf("srcServer = %q, want %q", srcServer, tt.wantSrcSrv)
				}
				if dstServer != tt.wantDstSrv {
					t.Errorf("dstServer = %q, want %q", dstServer, tt.wantDstSrv)
				}
			}
		})
	}
}

// ==== DRY-RUN TESTS ====

// TestDbRestoreHandler_DryRun_Success проверяет успешный dry-run с построением плана.
// AC-1: При BR_DRY_RUN=true команды возвращают план действий БЕЗ выполнения.
func TestDbRestoreHandler_DryRun_Success(t *testing.T) {
	// Создаём mock который FAIL-ит при любом вызове
	// AC-8: В dry-run режиме НЕ выполняются SQL запросы
	mockClient := &FailOnCallMock{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{mssqlClient: mockClient}

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
		"Команда: nr-dbrestore",
		"Валидация: ✅ Пройдена",
		"План выполнения:",
		"Проверка production флага",
		"Подключение к MSSQL серверу",
		"Восстановление базы данных",
		"=== END DRY RUN ===",
	}

	for _, part := range expectedParts {
		if !strings.Contains(capturedOutput, part) {
			t.Errorf("DryRun output should contain %q, got: %s", part, capturedOutput)
		}
	}
}

// TestDbRestoreHandler_DryRun_JSONOutput проверяет JSON формат dry-run вывода.
// AC-3: JSON output имеет поле "dry_run": true и структуру "plan": {...}.
func TestDbRestoreHandler_DryRun_JSONOutput(t *testing.T) {
	mockClient := &FailOnCallMock{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{mssqlClient: mockClient}

	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	// Перехватываем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	if err != nil {
		t.Errorf("DryRun Execute() unexpected error = %v", err)
	}

	// Проверяем JSON формат
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
		if result.Plan.Command != constants.ActNRDbrestore {
			t.Errorf("Plan.Command = %q, want %q", result.Plan.Command, constants.ActNRDbrestore)
		}
		if !result.Plan.ValidationPassed {
			t.Error("Plan.ValidationPassed should be true")
		}
		if len(result.Plan.Steps) == 0 {
			t.Error("Plan.Steps should not be empty")
		}
	}

	if result.Command != constants.ActNRDbrestore {
		t.Errorf("result.Command = %q, want %q", result.Command, constants.ActNRDbrestore)
	}
}

// TestDbRestoreHandler_DryRun_NoMockCalls проверяет что mock НЕ вызывается в dry-run.
// AC-8: В dry-run режиме НЕ выполняются SQL запросы.
func TestDbRestoreHandler_DryRun_NoMockCalls(t *testing.T) {
	// FailOnCallMock упадёт если вызван любой метод
	mockClient := &FailOnCallMock{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{mssqlClient: mockClient}

	t.Setenv("BR_DRY_RUN", "true")

	// Перехватываем stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	// Если mock был вызван — тест упал бы в FailOnCallMock
	if err != nil {
		t.Errorf("DryRun Execute() unexpected error = %v", err)
	}
}

// TestDbRestoreHandler_DryRun_ValidationError проверяет что в dry-run режиме
// ошибки валидации возвращают exit code != 0.
// AC-6: exit code != 0 если план невалиден (например, база не найдена, production база).
// ПРИМЕЧАНИЕ: Валидация происходит ДО вызова executeDryRun() — это корректное поведение.
// Если валидация не прошла, план просто не создаётся, и команда возвращает ошибку.
func TestDbRestoreHandler_DryRun_ValidationError(t *testing.T) {
	// Конфигурация с production базой — восстановление запрещено (даже в dry-run)
	cfg := &config.Config{
		InfobaseName: "ProdDB",
		DbConfig: map[string]*config.DatabaseInfo{
			"ProdDB": {
				DbServer: "prod-server",
				Prod:     true, // PRODUCTION!
			},
		},
	}

	h := &DbRestoreHandler{}

	t.Setenv("BR_DRY_RUN", "true")

	// Перехватываем stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	// AC-6: должна быть ошибка валидации
	if err == nil {
		t.Error("DryRun Execute() should return error for production database")
	}
	if !strings.Contains(err.Error(), ErrDbRestoreProductionForbidden) {
		t.Errorf("Error should contain %q, got: %v", ErrDbRestoreProductionForbidden, err)
	}
}

// TestDbRestoreHandler_DryRun_PlanContainsCorrectData проверяет содержимое плана.
// AC-2: Plan содержит операции, параметры, ожидаемые изменения.
func TestDbRestoreHandler_DryRun_PlanContainsCorrectData(t *testing.T) {
	mockClient := &FailOnCallMock{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{mssqlClient: mockClient}

	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	// Перехватываем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("DryRun Execute() unexpected error = %v", err)
	}

	var result output.Result
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Fatalf("JSON unmarshal error: %v", jsonErr)
	}

	plan := result.Plan
	if plan == nil {
		t.Fatal("Plan should not be nil")
	}

	// Проверяем количество шагов
	if len(plan.Steps) != 3 {
		t.Errorf("Plan should have 3 steps, got %d", len(plan.Steps))
	}

	// Проверяем шаг 3 — восстановление
	if len(plan.Steps) >= 3 {
		step3 := plan.Steps[2]
		if step3.Operation != "Восстановление базы данных" {
			t.Errorf("Step 3 operation = %q, want %q", step3.Operation, "Восстановление базы данных")
		}
		// AC-2: проверяем параметры
		if step3.Parameters["src_server"] != "prod-sql-server" {
			t.Errorf("src_server = %v, want %v", step3.Parameters["src_server"], "prod-sql-server")
		}
		if step3.Parameters["dst_db"] != "TestDB" {
			t.Errorf("dst_db = %v, want %v", step3.Parameters["dst_db"], "TestDB")
		}
		// AC-2: проверяем ожидаемые изменения
		if len(step3.ExpectedChanges) == 0 {
			t.Error("Step 3 ExpectedChanges should not be empty")
		}
	}

	// Проверяем summary
	if plan.Summary == "" {
		t.Error("Plan.Summary should not be empty")
	}
}

// FailOnCallMock — mock который падает при любом вызове методов.
// Используется для проверки что в dry-run методы NOT вызываются.
type FailOnCallMock struct {
	t *testing.T
}

func (m *FailOnCallMock) Connect(ctx context.Context) error {
	m.t.Fatal("Connect() не должен вызываться в dry-run режиме")
	return nil
}

func (m *FailOnCallMock) Close() error {
	m.t.Fatal("Close() не должен вызываться в dry-run режиме")
	return nil
}

func (m *FailOnCallMock) Restore(ctx context.Context, opts mssql.RestoreOptions) error {
	m.t.Fatal("Restore() не должен вызываться в dry-run режиме")
	return nil
}

func (m *FailOnCallMock) GetRestoreStats(ctx context.Context, opts mssql.StatsOptions) (*mssql.RestoreStats, error) {
	m.t.Fatal("GetRestoreStats() не должен вызываться в dry-run режиме")
	return nil, nil
}

func (m *FailOnCallMock) GetBackupSize(ctx context.Context, database string) (int64, error) {
	m.t.Fatal("GetBackupSize() не должен вызываться в dry-run режиме")
	return 0, nil
}

func (m *FailOnCallMock) Ping(ctx context.Context) error {
	m.t.Fatal("Ping() не должен вызываться в dry-run режиме")
	return nil
}

// ==== PLAN-ONLY TESTS (Story 7.3) ====

// TestDbRestoreHandler_PlanOnly_TextOutput проверяет текстовый вывод plan-only режима.
// Story 7.3 AC-1: При BR_PLAN_ONLY=true команда выводит план без выполнения.
// Story 7.3 AC-2: Заголовок "=== OPERATION PLAN ===" (не "=== DRY RUN ===").
func TestDbRestoreHandler_PlanOnly_TextOutput(t *testing.T) {
	// FailOnCallMock гарантирует что SQL запросы НЕ выполняются
	mockClient := &FailOnCallMock{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{mssqlClient: mockClient}

	t.Setenv("BR_PLAN_ONLY", "true")

	var capturedOutput string
	var err error
	capturedOutput = captureStdout(func() {
		err = h.Execute(context.Background(), cfg)
	})

	// Plan-only не должен возвращать ошибку для валидной конфигурации
	if err != nil {
		t.Errorf("PlanOnly Execute() unexpected error = %v", err)
	}

	// Проверяем наличие заголовков plan-only (НЕ dry-run)
	expectedParts := []string{
		"=== OPERATION PLAN ===",
		"Команда: nr-dbrestore",
		"Валидация: ✅ Пройдена",
		"=== END OPERATION PLAN ===",
	}

	for _, part := range expectedParts {
		if !strings.Contains(capturedOutput, part) {
			t.Errorf("PlanOnly output should contain %q, got: %s", part, capturedOutput)
		}
	}

	// НЕ должно быть заголовка DRY RUN
	if strings.Contains(capturedOutput, "=== DRY RUN ===") {
		t.Errorf("PlanOnly output should NOT contain '=== DRY RUN ===', got: %s", capturedOutput)
	}
}

// TestDbRestoreHandler_PlanOnly_JSONOutput проверяет JSON вывод plan-only режима.
// Story 7.3 AC-6: JSON output содержит plan_only: true и plan: {...}.
func TestDbRestoreHandler_PlanOnly_JSONOutput(t *testing.T) {
	mockClient := &FailOnCallMock{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{mssqlClient: mockClient}

	t.Setenv("BR_PLAN_ONLY", "true")
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
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

	// Plan не должен быть nil
	if result.Plan == nil {
		t.Error("JSON result.Plan should not be nil")
	} else {
		if result.Plan.Command != constants.ActNRDbrestore {
			t.Errorf("Plan.Command = %q, want %q", result.Plan.Command, constants.ActNRDbrestore)
		}
		if len(result.Plan.Steps) == 0 {
			t.Error("Plan.Steps should not be empty")
		}
	}

	// dry_run НЕ должен быть true
	if result.DryRun {
		t.Error("JSON result.DryRun should be false in plan-only mode")
	}
}

// TestDbRestoreHandler_Priority_DryRunOverPlanOnly проверяет приоритет:
// BR_DRY_RUN > BR_PLAN_ONLY. Если оба заданы, должен сработать dry-run.
func TestDbRestoreHandler_Priority_DryRunOverPlanOnly(t *testing.T) {
	mockClient := &FailOnCallMock{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{mssqlClient: mockClient}

	// Устанавливаем оба флага
	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_PLAN_ONLY", "true")

	var capturedOutput string
	var err error
	capturedOutput = captureStdout(func() {
		err = h.Execute(context.Background(), cfg)
	})

	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}

	// Должен быть DRY RUN (приоритет выше)
	if !strings.Contains(capturedOutput, "=== DRY RUN ===") {
		t.Errorf("Output should contain '=== DRY RUN ===' (priority over plan-only), got: %s", capturedOutput)
	}

	// НЕ должно быть OPERATION PLAN
	if strings.Contains(capturedOutput, "=== OPERATION PLAN ===") {
		t.Errorf("Output should NOT contain '=== OPERATION PLAN ===' when dry-run has priority, got: %s", capturedOutput)
	}
}

// ==== VERBOSE / PRIORITY TESTS (Story 7.3) ====

// TestDbRestoreHandler_Verbose_TextOutput проверяет verbose режим: план выводится ПЕРЕД выполнением.
// Story 7.3 AC-4: В verbose режиме сначала выводится план, затем выполняется операция.
func TestDbRestoreHandler_Verbose_TextOutput(t *testing.T) {
	mockClient := mssqltest.NewMockMSSQLClient()

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{mssqlClient: mockClient}

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
	if !strings.Contains(capturedOutput, "Восстановление завершено") {
		t.Errorf("Verbose output should contain 'Восстановление завершено', got: %s", capturedOutput)
	}
}

// TestDbRestoreHandler_Verbose_JSONOutput проверяет JSON вывод в verbose режиме.
// Story 7.3 AC-7: verbose JSON включает план в результат.
func TestDbRestoreHandler_Verbose_JSONOutput(t *testing.T) {
	mockClient := mssqltest.NewMockMSSQLClient()

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{mssqlClient: mockClient}

	t.Setenv("BR_VERBOSE", "true")
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := h.Execute(context.Background(), cfg)

	_ = w.Close()
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

// TestDbRestoreHandler_Priority_DryRunOverVerbose проверяет приоритет dry-run над verbose.
// AC-9: dry-run имеет высший приоритет над verbose.
func TestDbRestoreHandler_Priority_DryRunOverVerbose(t *testing.T) {
	mockClient := &FailOnCallMock{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{mssqlClient: mockClient}

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

// TestDbRestoreHandler_Priority_PlanOnlyOverVerbose проверяет приоритет plan-only над verbose.
// Plan-only останавливает выполнение (показывает план, не выполняет).
// Verbose показывает план и выполняет. Plan-only имеет приоритет.
func TestDbRestoreHandler_Priority_PlanOnlyOverVerbose(t *testing.T) {
	mockClient := &FailOnCallMock{t: t}

	cfg := createTestConfig("TestDB")

	h := &DbRestoreHandler{mssqlClient: mockClient}

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
	if strings.Contains(capturedOutput, "Восстановление завершено") {
		t.Errorf("Output should NOT contain 'Восстановление завершено' (plan-only, no execution), got: %s", capturedOutput)
	}
}

// createTestConfig создаёт тестовую конфигурацию для указанной базы данных
func createTestConfig(dbName string) *config.Config {
	return &config.Config{
		InfobaseName: dbName,
		Actor:        "test-user",
		AppConfig: &config.AppConfig{
			Users: struct {
				Rac        string `yaml:"rac"`
				Db         string `yaml:"db"`
				Mssql      string `yaml:"mssql"`
				StoreAdmin string `yaml:"storeAdmin"`
			}{
				Mssql: "test-mssql-user",
			},
			Dbrestore: struct {
				Database    string `yaml:"database"`
				Timeout     string `yaml:"timeout"`
				Autotimeout bool   `yaml:"autotimeout"`
			}{
				Database:    "master",
				Timeout:     "30s",
				Autotimeout: true,
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
				Mssql: "test-password",
			},
		},
		DbConfig: map[string]*config.DatabaseInfo{
			"ProdDB": {
				DbServer:  "prod-sql-server",
				OneServer: "prod-1c-server",
				Prod:      true,
			},
			dbName: {
				DbServer:  "test-sql-server",
				OneServer: "test-1c-server",
				Prod:      false,
			},
		},
		ProjectConfig: &config.ProjectConfig{
			Prod: map[string]struct {
				DbName     string                 `yaml:"dbName"`
				AddDisable []string               `yaml:"add-disable"`
				Related    map[string]interface{} `yaml:"related"`
			}{
				"ProdDB": {
					Related: map[string]interface{}{
						dbName: nil,
					},
				},
			},
		},
	}
}

