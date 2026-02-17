package converthandler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
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

// mockConverter — mock реализация Converter для тестов.
type mockConverter struct {
	convertFunc func(ctx context.Context, l *slog.Logger, cfg *config.Config, direction, pathIn, pathOut string) error
}

func (m *mockConverter) Convert(ctx context.Context, l *slog.Logger, cfg *config.Config, direction, pathIn, pathOut string) error {
	if m.convertFunc != nil {
		return m.convertFunc(ctx, l, cfg, direction, pathIn, pathOut)
	}
	return nil
}

// newTestAppConfig создаёт AppConfig для тестов.
func newTestAppConfig() *config.AppConfig {
	return &config.AppConfig{
		Paths: struct {
			Bin1cv8  string `yaml:"bin1cv8"`
			BinIbcmd string `yaml:"binIbcmd"`
			EdtCli   string `yaml:"edtCli"`
			Rac      string `yaml:"rac"`
		}{
			EdtCli: "/opt/1cedtcli/ring",
		},
	}
}

// === Тесты Name и Description ===

func TestConvertHandler_Name(t *testing.T) {
	h := &ConvertHandler{}
	assert.Equal(t, "nr-convert", h.Name())
	assert.Equal(t, constants.ActNRConvert, h.Name())
}

func TestConvertHandler_Description(t *testing.T) {
	h := &ConvertHandler{}
	desc := h.Description()
	assert.NotEmpty(t, desc)
	assert.Equal(t, "Конвертация между форматами EDT и XML", desc)
}

// === AC-6: Registration и Deprecated Alias ===

func TestConvertHandler_Registration(t *testing.T) {
	// RegisterCmd() вызван в TestMain — проверяем что handler зарегистрирован
	h, ok := command.Get("nr-convert")
	require.True(t, ok, "handler nr-convert должен быть зарегистрирован в registry")
	assert.Equal(t, constants.ActNRConvert, h.Name())
}

func TestConvertHandler_DeprecatedAlias(t *testing.T) {
	// AC-6: Проверяем что deprecated alias "convert" работает
	h, ok := command.Get("convert")
	require.True(t, ok, "deprecated alias convert должен быть зарегистрирован в registry")

	// Проверяем что это DeprecatedBridge
	dep, isDep := h.(command.Deprecatable)
	require.True(t, isDep, "handler должен реализовывать Deprecatable")
	assert.True(t, dep.IsDeprecated())
	assert.Equal(t, "nr-convert", dep.NewName())
}

// === AC-1: Конвертация EDT → XML ===

func TestConvertHandler_Execute_JSONOutput_Success_Edt2xml(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	// Создаём временный source директорий
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	t.Setenv("BR_SOURCE", sourceDir)
	t.Setenv("BR_TARGET", targetDir)
	t.Setenv("BR_DIRECTION", "edt2xml")

	mockConv := &mockConverter{}
	h := &ConvertHandler{converter: mockConv}

	cfg := &config.Config{
		AppConfig: newTestAppConfig(),
		TmpDir:    t.TempDir(),
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
	assert.Equal(t, "nr-convert", result.Command)
	assert.NotNil(t, result.Data)

	// Проверяем поля data
	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok, "Data должен быть map")
	assert.Equal(t, true, dataMap["state_changed"])       // AC-7
	assert.Equal(t, sourceDir, dataMap["source_path"])    // AC-4
	assert.Equal(t, targetDir, dataMap["target_path"])    // AC-4
	assert.Equal(t, "edt2xml", dataMap["direction"])      // AC-4
	assert.Equal(t, "1cedtcli", dataMap["tool_used"])     // AC-4
	_, hasDuration := dataMap["duration_ms"]              // AC-4
	assert.True(t, hasDuration, "Data должен содержать duration_ms")

	// AC-4: metadata
	require.NotNil(t, result.Metadata)
	assert.NotEmpty(t, result.Metadata.TraceID)
	assert.GreaterOrEqual(t, result.Metadata.DurationMs, int64(0))
	assert.Equal(t, constants.APIVersion, result.Metadata.APIVersion)
}

// === AC-2: Конвертация XML → EDT ===

func TestConvertHandler_Execute_JSONOutput_Success_Xml2edt(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	t.Setenv("BR_SOURCE", sourceDir)
	t.Setenv("BR_TARGET", targetDir)
	t.Setenv("BR_DIRECTION", "xml2edt")

	mockConv := &mockConverter{}
	h := &ConvertHandler{converter: mockConv}

	cfg := &config.Config{
		AppConfig: newTestAppConfig(),
		TmpDir:    t.TempDir(),
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
	assert.Equal(t, "xml2edt", dataMap["direction"]) // AC-2
	assert.Equal(t, true, dataMap["state_changed"])  // AC-7
}

// === AC-5: Text output ===

func TestConvertHandler_Execute_TextOutput_Success(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	t.Setenv("BR_SOURCE", sourceDir)
	t.Setenv("BR_TARGET", targetDir)
	t.Setenv("BR_DIRECTION", "edt2xml")

	mockConv := &mockConverter{}
	h := &ConvertHandler{converter: mockConv}

	cfg := &config.Config{
		AppConfig: newTestAppConfig(),
		TmpDir:    t.TempDir(),
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)

	// AC-5: Text output показывает человекочитаемую информацию
	assert.Contains(t, out, "Конвертация: успешно")
	assert.Contains(t, out, "Сводка:")
	assert.Contains(t, out, "Направление: edt2xml")
	assert.Contains(t, out, "Исходный путь:")
	assert.Contains(t, out, "Целевой путь:")
	assert.Contains(t, out, "Инструмент: 1cedtcli")
	assert.Contains(t, out, "Длительность:")
}

// === Error cases: нет BR_SOURCE ===

func TestConvertHandler_Execute_NoSource(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	t.Setenv("BR_SOURCE", "")
	t.Setenv("BR_TARGET", "/tmp/target")
	t.Setenv("BR_DIRECTION", "edt2xml")

	h := &ConvertHandler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.SOURCE_MISSING")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "CONFIG.SOURCE_MISSING", result.Error.Code)
}

// === Error cases: нет BR_TARGET ===

func TestConvertHandler_Execute_NoTarget(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	sourceDir := t.TempDir()
	t.Setenv("BR_SOURCE", sourceDir)
	t.Setenv("BR_TARGET", "")
	t.Setenv("BR_DIRECTION", "edt2xml")

	h := &ConvertHandler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.TARGET_MISSING")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "CONFIG.TARGET_MISSING", result.Error.Code)
}

// === Error cases: нет BR_DIRECTION ===

func TestConvertHandler_Execute_NoDirection(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	sourceDir := t.TempDir()
	t.Setenv("BR_SOURCE", sourceDir)
	t.Setenv("BR_TARGET", "/tmp/target")
	t.Setenv("BR_DIRECTION", "")

	h := &ConvertHandler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.DIRECTION_MISSING")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "CONFIG.DIRECTION_MISSING", result.Error.Code)
}

// === Error cases: недопустимый BR_DIRECTION ===

func TestConvertHandler_Execute_InvalidDirection(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	sourceDir := t.TempDir()
	t.Setenv("BR_SOURCE", sourceDir)
	t.Setenv("BR_TARGET", "/tmp/target")
	t.Setenv("BR_DIRECTION", "invalid-direction")

	h := &ConvertHandler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.DIRECTION_INVALID")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "CONFIG.DIRECTION_INVALID", result.Error.Code)
	assert.Contains(t, result.Error.Message, "invalid-direction")
	assert.Contains(t, result.Error.Message, "edt2xml")
	assert.Contains(t, result.Error.Message, "xml2edt")
}

// === Error cases: source path не существует ===

func TestConvertHandler_Execute_SourceNotExists(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	t.Setenv("BR_SOURCE", "/nonexistent/path/that/does/not/exist")
	t.Setenv("BR_TARGET", "/tmp/target")
	t.Setenv("BR_DIRECTION", "edt2xml")

	h := &ConvertHandler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "ERR_SOURCE_NOT_FOUND")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "ERR_SOURCE_NOT_FOUND", result.Error.Code)
	assert.Contains(t, result.Error.Message, "/nonexistent/path")
}

// === Error cases: ошибка конвертации ===

func TestConvertHandler_Execute_ConvertError(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	t.Setenv("BR_SOURCE", sourceDir)
	t.Setenv("BR_TARGET", targetDir)
	t.Setenv("BR_DIRECTION", "edt2xml")

	mockConv := &mockConverter{
		convertFunc: func(_ context.Context, _ *slog.Logger, _ *config.Config, _, _, _ string) error {
			return fmt.Errorf("EDT CLI завершился с ошибкой")
		},
	}
	h := &ConvertHandler{converter: mockConv}

	cfg := &config.Config{
		AppConfig: newTestAppConfig(),
		TmpDir:    t.TempDir(),
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "ERR_CONVERT")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "ERR_CONVERT", result.Error.Code)
	assert.Contains(t, result.Error.Message, "EDT CLI завершился с ошибкой")
}

// === Text error output ===

func TestConvertHandler_Execute_TextErrorOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")
	t.Setenv("BR_SOURCE", "")
	t.Setenv("BR_TARGET", "/tmp/target")
	t.Setenv("BR_DIRECTION", "edt2xml")

	h := &ConvertHandler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.")

	// Для text формата ошибка НЕ выводится в stdout — main.go логирует через logger
	assert.NotContains(t, out, `"status"`, "Текстовый формат НЕ должен содержать JSON")
}

// === Data structures tests ===

func TestConvertData_writeText(t *testing.T) {
	tests := []struct {
		name     string
		data     *ConvertData
		contains []string
	}{
		{
			name: "Success_Edt2xml",
			data: &ConvertData{
				StateChanged: true,
				SourcePath:   "/path/to/edt",
				TargetPath:   "/path/to/xml",
				Direction:    "edt2xml",
				ToolUsed:     "1cedtcli",
				DurationMs:   5000,
			},
			contains: []string{
				"успешно",
				"Сводка:",
				"Направление: edt2xml",
				"Исходный путь: /path/to/edt",
				"Целевой путь: /path/to/xml",
				"Инструмент: 1cedtcli",
				"Длительность: 5000 мс",
			},
		},
		{
			name: "Success_Xml2edt",
			data: &ConvertData{
				StateChanged: true,
				SourcePath:   "/path/to/xml",
				TargetPath:   "/path/to/edt",
				Direction:    "xml2edt",
				ToolUsed:     "1cedtcli",
				DurationMs:   3000,
			},
			contains: []string{
				"успешно",
				"Направление: xml2edt",
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

// === AC-11: Progress logging ===

func TestConvertHandler_Execute_ProgressLogs(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	t.Setenv("BR_SOURCE", sourceDir)
	t.Setenv("BR_TARGET", targetDir)
	t.Setenv("BR_DIRECTION", "edt2xml")

	// Перехватываем slog для проверки progress сообщений
	var logBuf bytes.Buffer
	testLogger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	oldDefault := slog.Default()
	slog.SetDefault(testLogger)
	defer slog.SetDefault(oldDefault)

	mockConv := &mockConverter{}
	h := &ConvertHandler{converter: mockConv}

	cfg := &config.Config{
		AppConfig: newTestAppConfig(),
		TmpDir:    t.TempDir(),
	}
	ctx := context.Background()

	_ = testutil.CaptureStdout(t, func() {
		_ = h.Execute(ctx, cfg)
	})

	logOutput := logBuf.String()

	// AC-11: Progress отображается: validating → preparing → converting → completing
	assert.Contains(t, logOutput, "validating: проверка параметров", "Progress log должен содержать 'validating'")
	assert.Contains(t, logOutput, "preparing: подготовка к конвертации", "Progress log должен содержать 'preparing'")
	assert.Contains(t, logOutput, "converting: выполнение конвертации", "Progress log должен содержать 'converting'")
	assert.Contains(t, logOutput, "completing: завершение операции", "Progress log должен содержать 'completing'")
}

// === Compile-time interface check (AC-8) ===

func TestConvertHandler_ImplementsHandler(t *testing.T) {
	// Этот тест документирует что ConvertHandler реализует command.Handler
	// Реальная проверка происходит через var _ command.Handler = (*ConvertHandler)(nil) в handler.go
	var h command.Handler = &ConvertHandler{}
	assert.NotNil(t, h)
	assert.Equal(t, constants.ActNRConvert, h.Name())
}

// === H-2 fix: Path traversal validation ===

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Valid absolute path",
			path:    "/tmp/test/source",
			wantErr: false,
		},
		{
			name:    "Valid absolute path with spaces",
			path:    "/tmp/test path/source",
			wantErr: false,
		},
		{
			name:    "Path traversal attack",
			path:    "/tmp/../etc/passwd",
			wantErr: true,
		},
		{
			name:    "Relative path",
			path:    "relative/path",
			wantErr: true,
		},
		{
			name:    "Current directory",
			path:    ".",
			wantErr: true,
		},
		{
			name:    "Parent directory attack",
			path:    "/home/user/../../etc/shadow",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path)
			if tt.wantErr {
				assert.Error(t, err, "validatePath(%q) должен вернуть ошибку", tt.path)
			} else {
				assert.NoError(t, err, "validatePath(%q) не должен вернуть ошибку", tt.path)
			}
		})
	}
}

func TestConvertHandler_Execute_SourcePathTraversal(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	// Path traversal в source — должен быть отклонён ДО вызова конвертации
	t.Setenv("BR_SOURCE", "/tmp/../etc/passwd")
	t.Setenv("BR_TARGET", "/tmp/target")
	t.Setenv("BR_DIRECTION", "edt2xml")

	h := &ConvertHandler{}
	cfg := &config.Config{
		AppConfig: newTestAppConfig(),
		TmpDir:    t.TempDir(),
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	// Path traversal должен быть обнаружен
	assert.Contains(t, execErr.Error(), "ERR_SOURCE")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
}

func TestConvertHandler_Execute_TargetPathTraversal(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	sourceDir := t.TempDir()
	t.Setenv("BR_SOURCE", sourceDir)
	// Path traversal в target — должен быть отклонён
	t.Setenv("BR_TARGET", "/tmp/../../../etc/shadow")
	t.Setenv("BR_DIRECTION", "edt2xml")

	h := &ConvertHandler{}
	cfg := &config.Config{
		AppConfig: newTestAppConfig(),
		TmpDir:    t.TempDir(),
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "ERR_TARGET_INVALID")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "ERR_TARGET_INVALID", result.Error.Code)
}

func TestConvertHandler_Execute_RelativeSourcePath(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	// Относительный путь должен быть отклонён — сначала ошибка NOT_FOUND,
	// но если файл случайно существует, то ERR_SOURCE_INVALID
	t.Setenv("BR_SOURCE", "relative/path/source")
	t.Setenv("BR_TARGET", "/tmp/target")
	t.Setenv("BR_DIRECTION", "edt2xml")

	h := &ConvertHandler{}
	cfg := &config.Config{
		AppConfig: newTestAppConfig(),
		TmpDir:    t.TempDir(),
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	// Относительный путь либо не существует (NOT_FOUND), либо отклоняется (INVALID)
	assert.True(t,
		strings.Contains(execErr.Error(), "ERR_SOURCE_NOT_FOUND") ||
			strings.Contains(execErr.Error(), "ERR_SOURCE_INVALID"),
		"Ошибка должна содержать ERR_SOURCE_NOT_FOUND или ERR_SOURCE_INVALID")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
}

// === M-3 fix: Tool from config ===

func TestConvertHandler_Execute_ToolFromConfig(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	t.Setenv("BR_SOURCE", sourceDir)
	t.Setenv("BR_TARGET", targetDir)
	t.Setenv("BR_DIRECTION", "edt2xml")

	mockConv := &mockConverter{}
	h := &ConvertHandler{converter: mockConv}

	// Конфигурация с кастомным инструментом
	cfg := &config.Config{
		AppConfig: newTestAppConfig(),
		TmpDir:    t.TempDir(),
		ImplementationsConfig: &config.ImplementationsConfig{
			ConfigExport: "1cv8", // Используем 1cv8 вместо 1cedtcli
		},
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
	// M-3: Инструмент должен браться из конфигурации
	assert.Equal(t, "1cv8", dataMap["tool_used"])
}

// === M-2 fix: Timeout handling ===

func TestConvertHandler_Execute_Timeout(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	t.Setenv("BR_SOURCE", sourceDir)
	t.Setenv("BR_TARGET", targetDir)
	t.Setenv("BR_DIRECTION", "edt2xml")

	// Mock конвертер который блокируется до отмены контекста
	mockConv := &mockConverter{
		convertFunc: func(ctx context.Context, _ *slog.Logger, _ *config.Config, _, _, _ string) error {
			<-ctx.Done() // Ждём отмены контекста
			return ctx.Err()
		},
	}
	h := &ConvertHandler{converter: mockConv}

	// Очень короткий timeout через конфиг
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			EdtTimeout: 1 * time.Millisecond,
		},
		TmpDir: t.TempDir(),
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "ERR_CONVERT_TIMEOUT")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "ERR_CONVERT_TIMEOUT", result.Error.Code)
}
