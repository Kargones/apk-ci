package command

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kargones/apk-ci/internal/config"
)

// testHandler — тестовый обработчик для проверки DeprecatedBridge.
type testDeprecatedHandler struct {
	name       string
	executeErr error
	executeCnt int
}

func (h *testDeprecatedHandler) Name() string        { return h.name }
func (h *testDeprecatedHandler) Description() string { return "test: " + h.name }
func (h *testDeprecatedHandler) Execute(_ context.Context, _ *config.Config) error {
	h.executeCnt++
	return h.executeErr
}

// captureStderrWithCleanup перехватывает вывод в stderr для тестирования.
// Использует t.Cleanup() для гарантированного восстановления stderr даже при panic.
// Возвращает функцию для получения перехваченного вывода.
//
// ВНИМАНИЕ: Тесты, использующие эту функцию, НЕ могут быть параллельными (t.Parallel()),
// т.к. os.Stderr — глобальная переменная. Это архитектурное ограничение до рефакторинга
// на Writer interface (после Story 1.4).
func captureStderrWithCleanup(t *testing.T) func() string {
	t.Helper()
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// t.Cleanup гарантирует восстановление stderr даже при panic
	t.Cleanup(func() {
		os.Stderr = oldStderr
	})

	return func() string {
		_ = w.Close()
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		return buf.String()
	}
}

func TestDeprecatedBridge_Name(t *testing.T) {
	bridge := &DeprecatedBridge{
		actual:     &testDeprecatedHandler{name: "new-cmd"},
		deprecated: "old-cmd",
		newName:    "new-cmd",
	}

	assert.Equal(t, "old-cmd", bridge.Name())
}

// TestDeprecatedBridge_Execute_LogsWarning — проверка warning в stderr (AC1, AC2).
func TestDeprecatedBridge_Execute_LogsWarning(t *testing.T) {
	getOutput := captureStderrWithCleanup(t)

	bridge := &DeprecatedBridge{
		actual:     &testDeprecatedHandler{name: "new-cmd"},
		deprecated: "old-cmd",
		newName:    "new-cmd",
	}

	err := bridge.Execute(context.Background(), &config.Config{})

	output := getOutput()

	require.NoError(t, err)
	assert.Contains(t, output, "WARNING", "вывод должен содержать WARNING")
	assert.Contains(t, output, "old-cmd", "вывод должен содержать старое имя")
	assert.Contains(t, output, "deprecated", "вывод должен содержать слово deprecated")
	assert.Contains(t, output, "new-cmd", "вывод должен содержать новое имя")
}

// TestDeprecatedBridge_Execute_DelegatesToActual — проверка делегирования к actual (AC3).
func TestDeprecatedBridge_Execute_DelegatesToActual(t *testing.T) {
	// Redirect stderr to avoid test output pollution
	_ = captureStderrWithCleanup(t)

	handler := &testDeprecatedHandler{name: "new-cmd"}
	bridge := &DeprecatedBridge{
		actual:     handler,
		deprecated: "old-cmd",
		newName:    "new-cmd",
	}

	err := bridge.Execute(context.Background(), &config.Config{})

	require.NoError(t, err)
	assert.Equal(t, 1, handler.executeCnt, "actual handler должен быть вызван")
}

// TestDeprecatedBridge_Execute_PropagatesError — проверка проброса ошибки от actual (AC3).
func TestDeprecatedBridge_Execute_PropagatesError(t *testing.T) {
	_ = captureStderrWithCleanup(t)

	expectedErr := errors.New("test error")
	handler := &testDeprecatedHandler{name: "new-cmd", executeErr: expectedErr}
	bridge := &DeprecatedBridge{
		actual:     handler,
		deprecated: "old-cmd",
		newName:    "new-cmd",
	}

	err := bridge.Execute(context.Background(), &config.Config{})

	assert.Equal(t, expectedErr, err, "ошибка от actual должна пробрасываться")
}

// TestRegisterWithAlias_EmptyDeprecated — просто Register при пустом deprecated.
func TestRegisterWithAlias_EmptyDeprecated(t *testing.T) {
	clearRegistry()

	handler := &testDeprecatedHandler{name: "my-cmd"}
	RegisterWithAlias(handler, "")

	// Должен быть зарегистрирован только под основным именем
	h, ok := Get("my-cmd")
	require.True(t, ok, "команда должна быть зарегистрирована под основным именем")
	assert.Equal(t, handler, h)

	// Пустая строка не должна быть зарегистрирована как команда
	_, ok = Get("")
	assert.False(t, ok, "пустое имя не должно быть зарегистрировано")
}

// TestRegisterWithAlias_CreatesDeprecatedBridge — оба имени зарегистрированы (AC1).
func TestRegisterWithAlias_CreatesDeprecatedBridge(t *testing.T) {
	clearRegistry()

	handler := &testDeprecatedHandler{name: "new-cmd"}
	RegisterWithAlias(handler, "old-cmd")

	// Основное имя — оригинальный handler
	h1, ok := Get("new-cmd")
	require.True(t, ok, "команда должна быть зарегистрирована под основным именем")
	assert.Equal(t, handler, h1)

	// Deprecated имя — DeprecatedBridge
	h2, ok := Get("old-cmd")
	require.True(t, ok, "команда должна быть зарегистрирована под deprecated именем")

	bridge, isBridge := h2.(*DeprecatedBridge)
	require.True(t, isBridge, "old-cmd должен быть DeprecatedBridge")
	assert.Equal(t, handler, bridge.actual, "actual должен указывать на оригинальный handler")
	assert.Equal(t, "old-cmd", bridge.deprecated, "deprecated должен содержать старое имя")
	assert.Equal(t, "new-cmd", bridge.newName, "newName должен содержать новое имя")
}

// TestRegisterWithAlias_SameNamePanics — panic если deprecated == Name().
func TestRegisterWithAlias_SameNamePanics(t *testing.T) {
	clearRegistry()

	handler := &testDeprecatedHandler{name: "same-name"}

	assert.PanicsWithValue(t,
		"command: deprecated name cannot be same as handler name: same-name",
		func() {
			RegisterWithAlias(handler, "same-name")
		},
		"регистрация с одинаковым deprecated и основным именем должна вызвать panic")
}

// TestRegisterWithAlias_NilHandlerPanics — panic для nil handler.
func TestRegisterWithAlias_NilHandlerPanics(t *testing.T) {
	clearRegistry()

	assert.PanicsWithValue(t, "command: nil handler", func() {
		RegisterWithAlias(nil, "old-cmd")
	}, "nil handler должен вызвать panic")
}

// TestRegisterWithAlias_DuplicateDeprecatedPanics — panic при дублировании deprecated имени.
func TestRegisterWithAlias_DuplicateDeprecatedPanics(t *testing.T) {
	clearRegistry()

	h1 := &testDeprecatedHandler{name: "cmd-one"}
	h2 := &testDeprecatedHandler{name: "cmd-two"}

	RegisterWithAlias(h1, "legacy-name")

	assert.PanicsWithValue(t,
		"command: duplicate handler registration for legacy-name",
		func() {
			RegisterWithAlias(h2, "legacy-name")
		},
		"повторная регистрация deprecated имени должна вызвать panic")
}

// TestDeprecatedBridge_MultipleExecutions_MultipleWarnings — warning на каждый вызов (AC4).
func TestDeprecatedBridge_MultipleExecutions_MultipleWarnings(t *testing.T) {
	getOutput := captureStderrWithCleanup(t)

	handler := &testDeprecatedHandler{name: "new-cmd"}
	bridge := &DeprecatedBridge{
		actual:     handler,
		deprecated: "old-cmd",
		newName:    "new-cmd",
	}

	// Execute дважды
	_ = bridge.Execute(context.Background(), &config.Config{})
	_ = bridge.Execute(context.Background(), &config.Config{})

	output := getOutput()

	// Warning должен появиться дважды
	warningCount := strings.Count(output, "WARNING")
	assert.Equal(t, 2, warningCount, "warning должен выводиться при каждом вызове")
	assert.Equal(t, 2, handler.executeCnt, "actual должен быть вызван дважды")
}

// TestDeprecatedBridge_Execute_WarningFormat проверяет точный формат warning сообщения.
func TestDeprecatedBridge_Execute_WarningFormat(t *testing.T) {
	getOutput := captureStderrWithCleanup(t)

	bridge := &DeprecatedBridge{
		actual:     &testDeprecatedHandler{name: "nr-version"},
		deprecated: "version",
		newName:    "nr-version",
	}

	_ = bridge.Execute(context.Background(), &config.Config{})

	output := getOutput()

	expectedWarning := "WARNING: command 'version' is deprecated, use 'nr-version' instead\n"
	assert.Equal(t, expectedWarning, output, "формат warning должен соответствовать спецификации")
}

// TestRegisterWithAlias_LegacyNameFormat тестирует что deprecated имена могут иметь legacy формат.
func TestRegisterWithAlias_LegacyNameFormat(t *testing.T) {
	clearRegistry()

	handler := &testDeprecatedHandler{name: "nr-service-mode"}
	// Legacy имя с подчёркиванием (не kebab-case)
	RegisterWithAlias(handler, "service_mode")

	// Оба имени должны быть зарегистрированы
	h1, ok := Get("nr-service-mode")
	require.True(t, ok, "основное имя должно быть зарегистрировано")
	assert.Equal(t, handler, h1)

	h2, ok := Get("service_mode")
	require.True(t, ok, "legacy имя должно быть зарегистрировано")

	bridge, isBridge := h2.(*DeprecatedBridge)
	require.True(t, isBridge, "legacy имя должно быть DeprecatedBridge")
	assert.Equal(t, "service_mode", bridge.deprecated)
}

// TestDeprecatedBridge_Execute_ContextCancelled проверяет поведение при отменённом context.
func TestDeprecatedBridge_Execute_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем context сразу

	handler := &testDeprecatedHandler{name: "new-cmd"}
	bridge := &DeprecatedBridge{
		actual:     handler,
		deprecated: "old-cmd",
		newName:    "new-cmd",
	}

	err := bridge.Execute(ctx, &config.Config{})

	assert.ErrorIs(t, err, context.Canceled, "должен вернуть context.Canceled")
	assert.Equal(t, 0, handler.executeCnt, "actual handler не должен быть вызван при отменённом context")
}

// TestDeprecatedBridge_Execute_ContextDeadlineExceeded проверяет поведение при истёкшем deadline.
func TestDeprecatedBridge_Execute_ContextDeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 0) // Истекает мгновенно
	defer cancel()

	handler := &testDeprecatedHandler{name: "new-cmd"}
	bridge := &DeprecatedBridge{
		actual:     handler,
		deprecated: "old-cmd",
		newName:    "new-cmd",
	}

	err := bridge.Execute(ctx, &config.Config{})

	assert.ErrorIs(t, err, context.DeadlineExceeded, "должен вернуть context.DeadlineExceeded")
	assert.Equal(t, 0, handler.executeCnt, "actual handler не должен быть вызван при истёкшем deadline")
}

// TestDeprecatedBridge_ImplementsHandler — compile-time проверка реализации интерфейса.
// Тест не имеет runtime assertions.
func TestDeprecatedBridge_ImplementsHandler(_ *testing.T) {
	var _ Handler = (*DeprecatedBridge)(nil)
}

// TestRegisterWithAlias_SpecialCharsInDeprecated тестирует deprecated имена со спецсимволами.
// Deprecated имена не проходят kebab-case валидацию, поэтому могут содержать любые символы.
func TestRegisterWithAlias_SpecialCharsInDeprecated(t *testing.T) {
	tests := []struct {
		name           string
		handlerName    string
		deprecatedName string
	}{
		{
			name:           "deprecated с точками",
			handlerName:    "nr-config",
			deprecatedName: "config.load",
		},
		{
			name:           "deprecated с двоеточием",
			handlerName:    "nr-status",
			deprecatedName: "status:check",
		},
		{
			name:           "deprecated со слешем",
			handlerName:    "nr-deploy",
			deprecatedName: "deploy/prod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearRegistry()

			handler := &testDeprecatedHandler{name: tt.handlerName}

			// Не должно паниковать — deprecated имена могут быть любыми
			assert.NotPanics(t, func() {
				RegisterWithAlias(handler, tt.deprecatedName)
			}, "спецсимволы в deprecated имени не должны вызывать panic")

			// Проверяем регистрацию
			h1, ok := Get(tt.handlerName)
			require.True(t, ok, "основное имя должно быть зарегистрировано")
			assert.Equal(t, handler, h1)

			h2, ok := Get(tt.deprecatedName)
			require.True(t, ok, "deprecated имя должно быть зарегистрировано")

			bridge, isBridge := h2.(*DeprecatedBridge)
			require.True(t, isBridge, "должен быть DeprecatedBridge")
			assert.Equal(t, tt.deprecatedName, bridge.deprecated)
		})
	}
}
