package help

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/testutil"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deprecatedTestHandler — тестовый handler для проверки deprecated в buildData.
type deprecatedTestHandler struct{}

func (h *deprecatedTestHandler) Name() string                                          { return "nr-test-deprecated-cmd" }
func (h *deprecatedTestHandler) Description() string                                   { return "тестовая deprecated команда" }
func (h *deprecatedTestHandler) Execute(_ context.Context, _ *config.Config) error { return nil }

// registerDeprecatedOnce гарантирует однократную регистрацию тестового deprecated handler.
var registerDeprecatedOnce sync.Once

// TestHelpHandler_Integration_DeprecatedInBuildData проверяет что deprecated команды
// из Registry отображаются в buildData() с пометкой Deprecated и NewName.
func TestHelpHandler_Integration_DeprecatedInBuildData(t *testing.T) {
	// Регистрируем deprecated handler однократно (сохраняется в глобальном registry).
	registerDeprecatedOnce.Do(func() {
		command.RegisterWithAlias(&deprecatedTestHandler{}, "old-test-deprecated-cmd")
	})

	data := buildData()

	// Ищем deprecated команду в NR-командах.
	var found bool
	for _, cmd := range data.NRCommands {
		if cmd.Name == "old-test-deprecated-cmd" {
			found = true
			assert.True(t, cmd.Deprecated, "deprecated команда должна быть помечена Deprecated=true")
			assert.Equal(t, "nr-test-deprecated-cmd", cmd.NewName,
				"deprecated команда должна указывать новое имя")
			break
		}
	}
	assert.True(t, found, "deprecated команда old-test-deprecated-cmd должна присутствовать в NR-командах")

	// Проверяем что текстовый вывод содержит пометку [deprecated →].
	var buf bytes.Buffer
	err := data.writeText(&buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "[deprecated → nr-test-deprecated-cmd]",
		"текстовый вывод должен содержать пометку deprecated")
}

// TestHelpHandler_Integration_EmptyCommand проверяет что пустой BR_COMMAND приводит к help.
// Логика перенаправления реализована в main.go, здесь проверяем что help handler
// зарегистрирован и доступен по ключу "help".
func TestHelpHandler_Integration_EmptyCommand(t *testing.T) {
	// Проверяем что help зарегистрирован в registry
	h, ok := command.Get("help")
	require.True(t, ok, "help должен быть зарегистрирован в registry")

	// Проверяем что пустая команда НЕ найдена (перенаправление делается в main.go)
	_, emptyOk := command.Get("")
	assert.False(t, emptyOk, "пустая команда не должна быть зарегистрирована в registry")

	// Выполняем help handler
	t.Setenv("BR_OUTPUT_FORMAT", "text")
	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(context.Background(), nil)
	})

	require.NoError(t, execErr)
	assert.Contains(t, out, "apk-ci")
}

// TestHelpHandler_Integration_Registration проверяет что help зарегистрирован в registry.
func TestHelpHandler_Integration_Registration(t *testing.T) {
	h, ok := command.Get(constants.ActHelp)
	require.True(t, ok, "help должен быть зарегистрирован в registry")
	assert.Equal(t, constants.ActHelp, h.Name())
	assert.Equal(t, "Вывод списка доступных команд", h.Description())
}

// TestHelpHandler_Integration_StdoutStderrSeparation проверяет что результат идёт в stdout.
func TestHelpHandler_Integration_StdoutStderrSeparation(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h, ok := command.Get(constants.ActHelp)
	require.True(t, ok)

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(context.Background(), nil)
	})

	require.NoError(t, execErr)

	// stdout должен содержать валидный JSON
	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err, "stdout должен содержать валидный JSON")
	assert.Equal(t, "success", result.Status)
}

// TestHelpHandler_Integration_TraceID проверяет что trace_id присутствует в metadata.
func TestHelpHandler_Integration_TraceID(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	traceID := tracing.GenerateTraceID()
	ctx := tracing.WithTraceID(context.Background(), traceID)

	h, ok := command.Get(constants.ActHelp)
	require.True(t, ok)

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.NoError(t, execErr)

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)

	require.NotNil(t, result.Metadata)
	assert.Equal(t, traceID, result.Metadata.TraceID, "trace_id должен совпадать с переданным в context")
	assert.Equal(t, constants.APIVersion, result.Metadata.APIVersion)
}
