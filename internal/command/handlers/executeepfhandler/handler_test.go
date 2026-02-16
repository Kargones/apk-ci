package executeepfhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEpfExecutor — mock реализация EpfExecutor для тестов.
type mockEpfExecutor struct {
	executeFunc func(ctx context.Context, cfg *config.Config) error
}

func (m *mockEpfExecutor) Execute(ctx context.Context, cfg *config.Config) error {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, cfg)
	}
	return nil
}

// newMockEpfExecutorSuccess создаёт mock, возвращающий успешный результат.
func newMockEpfExecutorSuccess() *mockEpfExecutor {
	return &mockEpfExecutor{
		executeFunc: func(_ context.Context, _ *config.Config) error {
			return nil
		},
	}
}

// newMockEpfExecutorError создаёт mock, возвращающий ошибку.
func newMockEpfExecutorError(errMsg string) *mockEpfExecutor {
	return &mockEpfExecutor{
		executeFunc: func(_ context.Context, _ *config.Config) error {
			return fmt.Errorf("%s", errMsg)
		},
	}
}

// === AC-1: Name и Description ===

func TestExecuteEpfHandler_Name(t *testing.T) {
	h := &ExecuteEpfHandler{}
	assert.Equal(t, "nr-execute-epf", h.Name())
	assert.Equal(t, constants.ActNRExecuteEpf, h.Name())
}

func TestExecuteEpfHandler_Description(t *testing.T) {
	h := &ExecuteEpfHandler{}
	desc := h.Description()
	assert.NotEmpty(t, desc)
	assert.Equal(t, "Выполнение внешней обработки 1C (.epf)", desc)
}

// === Compile-time interface check (AC-8) ===

func TestExecuteEpfHandler_ImplementsHandler(t *testing.T) {
	var h command.Handler = &ExecuteEpfHandler{}
	assert.NotNil(t, h)
	assert.Equal(t, constants.ActNRExecuteEpf, h.Name())
}

// === Registration ===

func TestExecuteEpfHandler_Registration(t *testing.T) {
	// init() уже вызван при импорте пакета — проверяем что handler зарегистрирован
	h, ok := command.Get("nr-execute-epf")
	require.True(t, ok, "handler nr-execute-epf должен быть зарегистрирован в registry")
	assert.Equal(t, constants.ActNRExecuteEpf, h.Name())
}

// === AC-6: Deprecated alias ===

func TestExecuteEpfHandler_DeprecatedAlias(t *testing.T) {
	// Проверяем что deprecated alias "execute-epf" работает
	h, ok := command.Get("execute-epf")
	require.True(t, ok, "deprecated alias execute-epf должен быть зарегистрирован в registry")

	// Проверяем что это DeprecatedBridge
	dep, isDep := h.(command.Deprecatable)
	require.True(t, isDep, "handler должен реализовывать Deprecatable")
	assert.True(t, dep.IsDeprecated())
	assert.Equal(t, "nr-execute-epf", dep.NewName())
}

// === AC-2, AC-3: Success cases ===

func TestExecuteEpfHandler_Execute_TextOutput_Success(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")
	t.Setenv("BR_EPF_TIMEOUT", "")

	mock := newMockEpfExecutorSuccess()
	h := &ExecuteEpfHandler{executor: mock}
	cfg := &config.Config{
		StartEpf:     "https://gitea.example.com/api/v1/repos/owner/repo/raw/path/file.epf",
		InfobaseName: "TestBase",
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)
	// AC-5: Text output показывает человекочитаемую информацию
	assert.Contains(t, out, "Внешняя обработка выполнена успешно")
	assert.Contains(t, out, "Файл:")
	assert.Contains(t, out, "База: TestBase")
	assert.Contains(t, out, "Время выполнения:")
}

func TestExecuteEpfHandler_Execute_JSONOutput_Success(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mock := newMockEpfExecutorSuccess()
	h := &ExecuteEpfHandler{executor: mock}
	cfg := &config.Config{
		StartEpf:     "https://gitea.example.com/api/v1/repos/owner/repo/raw/path/file.epf",
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

	// AC-4: JSON output содержит необходимые поля
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "nr-execute-epf", result.Command)
	assert.NotNil(t, result.Data)

	// Проверяем поля data
	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok, "Data должен быть map")
	// AC-10: state_changed: true
	assert.Equal(t, true, dataMap["state_changed"])
	assert.Equal(t, "TestBase", dataMap["infobase_name"])
	assert.Contains(t, dataMap["epf_path"].(string), "file.epf")

	// AC-4: duration_ms в data
	durationMs, ok := dataMap["duration_ms"]
	require.True(t, ok, "Data должен содержать duration_ms")
	assert.GreaterOrEqual(t, durationMs.(float64), float64(0))

	// Metadata
	require.NotNil(t, result.Metadata)
	assert.NotEmpty(t, result.Metadata.TraceID)
	assert.GreaterOrEqual(t, result.Metadata.DurationMs, int64(0))
	assert.Equal(t, constants.APIVersion, result.Metadata.APIVersion)
}

// === AC-2: Validation errors ===

func TestExecuteEpfHandler_Execute_ValidationError_MissingEpfPath(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &ExecuteEpfHandler{}
	cfg := &config.Config{
		StartEpf:     "", // Missing
		InfobaseName: "TestBase",
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "ERR_EXECUTE_EPF_VALIDATION")
	assert.Contains(t, execErr.Error(), "BR_EPF_PATH")

	// Проверяем структурированный вывод ошибки
	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err, "stdout должен содержать валидный JSON ошибки")

	assert.Equal(t, "error", result.Status)
	assert.Equal(t, "nr-execute-epf", result.Command)
	require.NotNil(t, result.Error)
	assert.Equal(t, "ERR_EXECUTE_EPF_VALIDATION", result.Error.Code)
	assert.Contains(t, result.Error.Message, "BR_EPF_PATH")
}

func TestExecuteEpfHandler_Execute_ValidationError_MissingInfobaseName(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &ExecuteEpfHandler{}
	cfg := &config.Config{
		StartEpf:     "https://example.com/file.epf",
		InfobaseName: "", // Missing
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "ERR_EXECUTE_EPF_VALIDATION")
	assert.Contains(t, execErr.Error(), "BR_INFOBASE_NAME")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "ERR_EXECUTE_EPF_VALIDATION", result.Error.Code)
	assert.Contains(t, result.Error.Message, "BR_INFOBASE_NAME")
}

func TestExecuteEpfHandler_Execute_ValidationError_InvalidURL(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &ExecuteEpfHandler{}
	cfg := &config.Config{
		StartEpf:     "/local/path/file.epf", // Not a URL
		InfobaseName: "TestBase",
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "ERR_EXECUTE_EPF_VALIDATION")
	assert.Contains(t, execErr.Error(), "некорректный URL")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "ERR_EXECUTE_EPF_VALIDATION", result.Error.Code)
}

func TestExecuteEpfHandler_Execute_NilConfig(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &ExecuteEpfHandler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "ERR_EXECUTE_EPF_VALIDATION")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
}

// === AC-7: Execution error ===

func TestExecuteEpfHandler_Execute_ExecutionError(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mock := newMockEpfExecutorError("ошибка запуска 1С:Enterprise")
	h := &ExecuteEpfHandler{executor: mock}
	cfg := &config.Config{
		StartEpf:     "https://example.com/file.epf",
		InfobaseName: "TestBase",
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "ERR_EXECUTE_EPF_EXECUTION")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "ERR_EXECUTE_EPF_EXECUTION", result.Error.Code)
	assert.Contains(t, result.Error.Message, "ошибка запуска 1С:Enterprise")
}

func TestExecuteEpfHandler_Execute_DownloadError(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	// Mock возвращающий ошибку скачивания (текст из enterprise.EpfExecutor.downloadEpfFile)
	mock := newMockEpfExecutorError("ошибка получения данных .epf файла: connection refused")
	h := &ExecuteEpfHandler{executor: mock}
	cfg := &config.Config{
		StartEpf:     "https://example.com/file.epf",
		InfobaseName: "TestBase",
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "ERR_EXECUTE_EPF_DOWNLOAD")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "ERR_EXECUTE_EPF_DOWNLOAD", result.Error.Code)
	assert.Contains(t, result.Error.Message, "ошибка получения данных .epf файла")
}

func TestExecuteEpfHandler_Execute_TempFileError(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	// Mock возвращающий ошибку создания временного файла
	mock := newMockEpfExecutorError("ошибка создания временного файла: permission denied")
	h := &ExecuteEpfHandler{executor: mock}
	cfg := &config.Config{
		StartEpf:     "https://example.com/file.epf",
		InfobaseName: "TestBase",
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "ERR_EXECUTE_EPF_DOWNLOAD")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "ERR_EXECUTE_EPF_DOWNLOAD", result.Error.Code)
}

// === AC-3: Custom timeout ===

// newMockEpfExecutorWithTimeoutCheck создаёт mock, проверяющий timeout в context.
func newMockEpfExecutorWithTimeoutCheck(expectedMin, expectedMax time.Duration) *mockEpfExecutor {
	return &mockEpfExecutor{
		executeFunc: func(ctx context.Context, _ *config.Config) error {
			deadline, ok := ctx.Deadline()
			if !ok {
				return fmt.Errorf("context должен иметь deadline")
			}
			remaining := time.Until(deadline)
			if remaining < expectedMin || remaining > expectedMax {
				return fmt.Errorf("timeout должен быть в диапазоне %v-%v, получено: %v",
					expectedMin, expectedMax, remaining)
			}
			return nil
		},
	}
}

func TestExecuteEpfHandler_Execute_CustomTimeout(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	t.Setenv("BR_EPF_TIMEOUT", "60")

	// Mock проверяющий что context имеет timeout ~60 секунд
	mock := newMockEpfExecutorWithTimeoutCheck(59*time.Second, 61*time.Second)

	h := &ExecuteEpfHandler{executor: mock}
	cfg := &config.Config{
		StartEpf:     "https://example.com/file.epf",
		InfobaseName: "TestBase",
	}
	ctx := context.Background()

	var execErr error
	_ = testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)
}

func TestExecuteEpfHandler_Execute_InvalidTimeout_UsesDefault(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	t.Setenv("BR_EPF_TIMEOUT", "invalid")

	// Mock проверяющий что используется default timeout ~300 секунд
	mock := newMockEpfExecutorWithTimeoutCheck(299*time.Second, 301*time.Second)

	h := &ExecuteEpfHandler{executor: mock}
	cfg := &config.Config{
		StartEpf:     "https://example.com/file.epf",
		InfobaseName: "TestBase",
	}
	ctx := context.Background()

	var execErr error
	_ = testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)
}

func TestExecuteEpfHandler_Execute_ZeroTimeout_UsesDefault(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	t.Setenv("BR_EPF_TIMEOUT", "0")

	// Mock проверяющий что используется default timeout ~300 секунд (t > 0 условие)
	mock := newMockEpfExecutorWithTimeoutCheck(299*time.Second, 301*time.Second)

	h := &ExecuteEpfHandler{executor: mock}
	cfg := &config.Config{
		StartEpf:     "https://example.com/file.epf",
		InfobaseName: "TestBase",
	}
	ctx := context.Background()

	var execErr error
	_ = testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)
}

func TestExecuteEpfHandler_Execute_NegativeTimeout_UsesDefault(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	t.Setenv("BR_EPF_TIMEOUT", "-10")

	// Mock проверяющий что используется default timeout ~300 секунд (t > 0 условие)
	mock := newMockEpfExecutorWithTimeoutCheck(299*time.Second, 301*time.Second)

	h := &ExecuteEpfHandler{executor: mock}
	cfg := &config.Config{
		StartEpf:     "https://example.com/file.epf",
		InfobaseName: "TestBase",
	}
	ctx := context.Background()

	var execErr error
	_ = testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)
}

// === Text error output ===

func TestExecuteEpfHandler_Execute_TextErrorOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	h := &ExecuteEpfHandler{}
	cfg := &config.Config{
		StartEpf:     "", // Missing
		InfobaseName: "TestBase",
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "ERR_EXECUTE_EPF_VALIDATION")

	// Текстовый формат НЕ должен содержать JSON
	assert.NotContains(t, out, `"status"`, "Текстовый формат НЕ должен содержать JSON")
}

// === Data structures tests ===

func TestExecuteEpfData_writeText(t *testing.T) {
	tests := []struct {
		name     string
		data     *ExecuteEpfData
		contains []string
	}{
		{
			name: "Success case",
			data: &ExecuteEpfData{
				StateChanged: true,
				EpfPath:      "https://example.com/file.epf",
				InfobaseName: "MyBase",
				DurationMs:   1234,
			},
			contains: []string{
				"Внешняя обработка выполнена успешно",
				"Файл: https://example.com/file.epf",
				"База: MyBase",
				"Время выполнения: 1234 мс",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := testutil.CaptureStdout(t, func() {
				// writeText использует os.Stdout внутри CaptureStdout
				err := tt.data.writeText(os.Stdout)
				require.NoError(t, err)
			})

			for _, substr := range tt.contains {
				assert.Contains(t, out, substr)
			}
		})
	}
}

// === URL validation tests ===

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com/file.epf", true},
		{"http://example.com/file.epf", true},
		{"https://gitea.example.com/api/v1/repos/owner/repo/raw/path/file.epf", true},
		{"/local/path/file.epf", false},
		{"file.epf", false},
		{"ftp://example.com/file.epf", false},
		{"", false},
		{"http://", false},
		{"https:/", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := isValidURL(tt.url)
			assert.Equal(t, tt.expected, result, "URL: %s", tt.url)
		})
	}
}

// === Safe URL logging tests ===

func TestSafeLogURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "Short URL without query — unchanged",
			url:      "https://example.com/file.epf",
			expected: "https://example.com/file.epf",
		},
		{
			name:     "URL with access_token — query masked",
			url:      "https://example.com/file.epf?access_token=secret123",
			expected: "https://example.com/file.epf?...",
		},
		{
			name:     "URL with multiple query params — all masked",
			url:      "https://gitea.example.com/api/v1/repos/owner/repo/raw/file.epf?token=abc&ref=main",
			expected: "https://gitea.example.com/api/v1/repos/owner/repo/raw/file.epf?...",
		},
		{
			name:     "Long URL without query — truncated",
			url:      "https://gitea.example.com/api/v1/repos/owner/repo/raw/very/long/path/to/file/with/many/directories/and/subdirectories/file.epf",
			expected: "https://gitea.example.com/api/v1/repos/owner/repo/raw/very/long/path/to/file/with/many/directories/a...",
		},
		// Review #35 fix: тесты для userinfo (credentials в URL)
		{
			name:     "URL with userinfo — credentials removed",
			url:      "https://user:password@example.com/file.epf",
			expected: "https://example.com/file.epf",
		},
		{
			name:     "URL with userinfo and query — both sanitized",
			url:      "https://admin:secret123@gitea.example.com/api/v1/repos/file.epf?token=abc",
			expected: "https://gitea.example.com/api/v1/repos/file.epf?...",
		},
		{
			name:     "URL with user only (no password) — user removed",
			url:      "https://deploy-token@example.com/file.epf",
			expected: "https://example.com/file.epf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := safeLogURL(tt.url)
			assert.Equal(t, tt.expected, result)
			// Гарантируем что результат не превышает 103 символа (100 + "...")
			assert.LessOrEqual(t, len(result), 103)
			// Гарантируем что access_token не попадает в лог
			assert.NotContains(t, result, "secret")
			assert.NotContains(t, result, "token=")
		})
	}
}

// === DefaultTimeout constant test ===

func TestDefaultTimeout(t *testing.T) {
	assert.Equal(t, 300*time.Second, DefaultTimeout)
}
