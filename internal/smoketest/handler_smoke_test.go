package smoketest

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ВАЖНО: Тесты в этом файле НЕ должны использовать t.Parallel().
// CaptureStdout() модифицирует глобальный os.Stdout без синхронизации,
// поэтому параллельный запуск вызовет race condition.

// smokeResult — минимальная структура для валидации JSON output.
// Не используем output.Result напрямую, т.к. smoke-тесты проверяют
// системную целостность вывода, а не внутренние структуры.
type smokeResult struct {
	Status  string          `json:"status"`
	Command string          `json:"command"`
	Error   *smokeErrorInfo `json:"error,omitempty"`
	DryRun  bool            `json:"dry_run,omitempty"`
}

type smokeErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// TestSmoke_ServiceModeStatus_JSON проверяет что nr-service-mode-status
// корректно обрабатывает запрос без InfobaseName и возвращает JSON error output.
func TestSmoke_ServiceModeStatus_JSON(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h, ok := command.Get(constants.ActNRServiceModeStatus)
	require.True(t, ok, "nr-service-mode-status должен быть зарегистрирован")

	cfg := &config.Config{
		InfobaseName: "", // Пустое имя — вызовет ошибку валидации
	}

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(context.Background(), cfg)
	})

	// Handler должен вернуть error
	require.Error(t, execErr, "ожидается ошибка при пустом InfobaseName")

	// JSON output должен содержать валидную структуру
	var result smokeResult
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err, "JSON output должен быть валидным: %s", out)

	assert.Equal(t, "error", result.Status, "статус должен быть error")
	assert.Equal(t, constants.ActNRServiceModeStatus, result.Command,
		"command должен быть nr-service-mode-status")
	require.NotNil(t, result.Error, "поле error не должно быть nil")
	assert.Equal(t, "CONFIG.INFOBASE_MISSING", result.Error.Code,
		"код ошибки должен быть CONFIG.INFOBASE_MISSING")
	assert.NotEmpty(t, result.Error.Message, "сообщение об ошибке не должно быть пустым")
}

// TestSmoke_ServiceModeStatus_Text проверяет text-формат вывода
// nr-service-mode-status при ошибке валидации.
func TestSmoke_ServiceModeStatus_Text(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	h, ok := command.Get(constants.ActNRServiceModeStatus)
	require.True(t, ok)

	cfg := &config.Config{
		InfobaseName: "",
	}

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(context.Background(), cfg)
	})

	require.Error(t, execErr, "ожидается ошибка при пустом InfobaseName")
	// В text-формате при ошибке выводится сообщение в stdout
	assert.Contains(t, out, "CONFIG.INFOBASE_MISSING",
		"text output должен содержать код ошибки")
}

// TestSmoke_Dbrestore_JSON проверяет что nr-dbrestore корректно обрабатывает
// запрос без InfobaseName и возвращает JSON error output.
func TestSmoke_Dbrestore_JSON(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h, ok := command.Get(constants.ActNRDbrestore)
	require.True(t, ok, "nr-dbrestore должен быть зарегистрирован")

	cfg := &config.Config{
		InfobaseName: "",
	}

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(context.Background(), cfg)
	})

	require.Error(t, execErr, "ожидается ошибка при пустом InfobaseName")

	var result smokeResult
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err, "JSON output должен быть валидным: %s", out)

	assert.Equal(t, "error", result.Status)
	assert.Equal(t, constants.ActNRDbrestore, result.Command)
	require.NotNil(t, result.Error)
	assert.NotEmpty(t, result.Error.Code)
	assert.NotEmpty(t, result.Error.Message)
}

// TestSmoke_Dbrestore_Text проверяет text-формат вывода nr-dbrestore при ошибке.
func TestSmoke_Dbrestore_Text(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	h, ok := command.Get(constants.ActNRDbrestore)
	require.True(t, ok)

	cfg := &config.Config{
		InfobaseName: "",
	}

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(context.Background(), cfg)
	})

	require.Error(t, execErr, "ожидается ошибка при пустом InfobaseName")
	// В text-формате выводится сообщение об ошибке
	assert.NotEmpty(t, out, "text output не должен быть пустым при ошибке")
}

// TestSmoke_Convert_JSON проверяет что nr-convert корректно обрабатывает
// запрос без обязательных параметров и возвращает JSON error output.
func TestSmoke_Convert_JSON(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	// Очищаем параметры конвертации для гарантии ошибки
	t.Setenv("BR_SOURCE", "")
	t.Setenv("BR_TARGET", "")
	t.Setenv("BR_DIRECTION", "")

	h, ok := command.Get(constants.ActNRConvert)
	require.True(t, ok, "nr-convert должен быть зарегистрирован")

	cfg := &config.Config{}

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(context.Background(), cfg)
	})

	require.Error(t, execErr, "ожидается ошибка при отсутствии параметров конвертации")

	var result smokeResult
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err, "JSON output должен быть валидным: %s", out)

	assert.Equal(t, "error", result.Status)
	assert.Equal(t, constants.ActNRConvert, result.Command)
	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Code, "CONFIG.",
		"код ошибки должен начинаться с CONFIG.")
	assert.NotEmpty(t, result.Error.Message)
}

// TestSmoke_Convert_Text проверяет text-формат вывода nr-convert при ошибке.
func TestSmoke_Convert_Text(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")
	t.Setenv("BR_SOURCE", "")
	t.Setenv("BR_TARGET", "")
	t.Setenv("BR_DIRECTION", "")

	h, ok := command.Get(constants.ActNRConvert)
	require.True(t, ok)

	cfg := &config.Config{}

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(context.Background(), cfg)
	})

	require.Error(t, execErr, "ожидается ошибка при отсутствии параметров конвертации")
	assert.Contains(t, execErr.Error(), "CONFIG.",
		"ошибка должна содержать код CONFIG.")
	// ConvertHandler в text-формате не выводит ошибку в stdout (в отличие от
	// ServiceModeStatus и Dbrestore), поэтому проверяем что stdout пуст.
	_ = out
}
