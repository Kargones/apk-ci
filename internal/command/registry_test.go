package command

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Kargones/apk-ci/internal/config"
)

// mockHandler — тестовый обработчик команды.
type mockHandler struct {
	name string
}

func (m *mockHandler) Name() string        { return m.name }
func (m *mockHandler) Description() string { return "mock: " + m.name }
func (m *mockHandler) Execute(_ context.Context, _ *config.Config) error {
	return nil
}

func TestRegister_Success(t *testing.T) {
	clearRegistry()

	h := &mockHandler{name: "test-command"}
	err := Register(h)
	assert.NoError(t, err)

	got, ok := Get("test-command")
	assert.True(t, ok, "команда должна быть найдена в реестре")
	assert.Equal(t, h, got, "должен вернуться тот же handler")
}

func TestRegister_Duplicate_ReturnsError(t *testing.T) {
	clearRegistry()

	h1 := &mockHandler{name: "dup-command"}
	h2 := &mockHandler{name: "dup-command"}

	err := Register(h1)
	assert.NoError(t, err)

	err = Register(h2)
	assert.EqualError(t, err, "command: duplicate handler registration for dup-command", "повторная регистрация должна вернуть ошибку")
}

func TestRegister_NilHandler_ReturnsError(t *testing.T) {
	clearRegistry()

	err := Register(nil)
	assert.EqualError(t, err, "command: nil handler",
		"nil handler должен вызвать panic")
}

func TestRegister_EmptyName_ReturnsError(t *testing.T) {
	clearRegistry()

	h := &mockHandler{name: ""}

	err := Register(h)
	assert.EqualError(t, err, "command: empty handler name",
		"пустое имя должно вызвать panic")
}

func TestGet_NotFound(t *testing.T) {
	clearRegistry()

	got, ok := Get("non-existent")
	assert.False(t, ok, "несуществующая команда должна вернуть false")
	assert.Nil(t, got, "несуществующая команда должна вернуть nil")
}

func TestGet_Found(t *testing.T) {
	clearRegistry()

	h := &mockHandler{name: "existing"}
	_ = Register(h)

	got, ok := Get("existing")
	assert.True(t, ok, "зарегистрированная команда должна быть найдена")
	assert.Equal(t, h, got, "должен вернуться зарегистрированный handler")
}

func TestConcurrentAccess(t *testing.T) {
	clearRegistry()

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Регистрируем команды из нескольких горутин
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			// Используем fmt.Sprintf для корректной генерации уникальных имён
			h := &mockHandler{name: fmt.Sprintf("concurrent-cmd-%d", idx)}
			_ = Register(h)
		}(i)
	}

	// Одновременно читаем из реестра
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			// Пробуем получить команду (может ещё не быть зарегистрирована из-за race)
			Get(fmt.Sprintf("concurrent-cmd-%d", idx))
		}(i)
	}

	wg.Wait()

	// Проверяем, что все команды были зарегистрированы
	for i := 0; i < numGoroutines; i++ {
		name := fmt.Sprintf("concurrent-cmd-%d", i)
		handler, found := Get(name)
		assert.True(t, found, "команда %s должна быть зарегистрирована после завершения всех горутин", name)
		assert.NotNil(t, handler, "handler для %s не должен быть nil", name)
	}
}

func TestMultipleRegistrations(t *testing.T) {
	clearRegistry()

	handlers := []*mockHandler{
		{name: "cmd-alpha"},
		{name: "cmd-beta"},
		{name: "cmd-gamma"},
	}

	for _, h := range handlers {
		_ = Register(h)
	}

	for _, h := range handlers {
		got, ok := Get(h.name)
		assert.True(t, ok, "команда %s должна быть найдена", h.name)
		assert.Equal(t, h, got, "команда %s должна вернуть правильный handler", h.name)
	}
}

// TestRegistryPath_AC5 тестирует AC5: логика выбора пути (registry vs legacy).
// Этот тест проверяет, что Get корректно различает зарегистрированные
// и незарегистрированные команды, что является основой для логирования в main.go.
func TestRegistryPath_AC5(t *testing.T) {
	clearRegistry()

	// Регистрируем тестовую команду
	registeredCmd := "test-registry-path"
	h := &mockHandler{name: registeredCmd}
	_ = Register(h)

	tests := []struct {
		name         string
		command      string
		expectFound  bool
		expectedPath string // "registry" или "legacy"
	}{
		{
			name:         "зарегистрированная команда использует registry путь",
			command:      registeredCmd,
			expectFound:  true,
			expectedPath: "registry",
		},
		{
			name:         "незарегистрированная команда использует legacy путь",
			command:      "unknown-command",
			expectFound:  false,
			expectedPath: "legacy",
		},
		{
			name:         "пустая команда использует legacy путь",
			command:      "",
			expectFound:  false,
			expectedPath: "legacy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, found := Get(tt.command)

			if tt.expectFound {
				assert.True(t, found, "команда %s должна быть найдена (путь: %s)", tt.command, tt.expectedPath)
				assert.NotNil(t, handler, "handler для %s не должен быть nil", tt.command)
			} else {
				assert.False(t, found, "команда %s не должна быть найдена (путь: %s)", tt.command, tt.expectedPath)
				assert.Nil(t, handler, "handler для %s должен быть nil", tt.command)
			}
		})
	}
}

// TestRegister_InvalidNameFormat_Panics тестирует валидацию формата имени команды.
func TestRegister_InvalidNameFormat_ReturnsError(t *testing.T) {
	tests := []struct {
		name        string
		handlerName string
		wantError   string
	}{
		{
			name:        "имя с пробелами",
			handlerName: "my command",
			wantError:   "command: invalid handler name format (must be kebab-case): my command",
		},
		{
			name:        "имя с заглавными буквами",
			handlerName: "MyCommand",
			wantError:   "command: invalid handler name format (must be kebab-case): MyCommand",
		},
		{
			name:        "имя начинается с цифры",
			handlerName: "1command",
			wantError:   "command: invalid handler name format (must be kebab-case): 1command",
		},
		{
			name:        "имя начинается с дефиса",
			handlerName: "-command",
			wantError:   "command: invalid handler name format (must be kebab-case): -command",
		},
		{
			name:        "имя с подчёркиванием",
			handlerName: "my_command",
			wantError:   "command: invalid handler name format (must be kebab-case): my_command",
		},
		{
			name:        "имя со спецсимволами",
			handlerName: "cmd!@#",
			wantError:   "command: invalid handler name format (must be kebab-case): cmd!@#",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearRegistry()
			h := &mockHandler{name: tt.handlerName}
			err := Register(h)
			assert.EqualError(t, err, tt.wantError,
				"некорректный формат имени должен вернуть ошибку")
		})
	}
}

// TestRegister_ValidNameFormats тестирует допустимые форматы имён команд.
func TestRegister_ValidNameFormats(t *testing.T) {
	validNames := []string{
		"convert",
		"service-mode-status",
		"nr-version",
		"sq-scan-branch",
		"a",
		"a1",
		"command123",
		"my-long-command-name-with-numbers-123",
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			clearRegistry()
			h := &mockHandler{name: name}
			// Не должно паниковать
			assert.NotPanics(t, func() {
				_ = Register(h)
			}, "валидное имя %s не должно вызывать panic", name)

			// Проверяем, что зарегистрировалось
			got, ok := Get(name)
			assert.True(t, ok, "команда %s должна быть найдена", name)
			assert.Equal(t, h, got)
		})
	}
}

// TestAll возвращает копию всех зарегистрированных обработчиков.
func TestAll(t *testing.T) {
	clearRegistry()

	// Регистрируем несколько команд
	h1 := &mockHandler{name: "cmd-alpha"}
	h2 := &mockHandler{name: "cmd-beta"}
	h3 := &mockHandler{name: "cmd-gamma"}
	_ = Register(h1)
	_ = Register(h2)
	_ = Register(h3)

	all := All()

	// Проверяем размер
	assert.Len(t, all, 3, "должно быть 3 команды")

	// Проверяем содержимое
	assert.Equal(t, h1, all["cmd-alpha"])
	assert.Equal(t, h2, all["cmd-beta"])
	assert.Equal(t, h3, all["cmd-gamma"])

	// Проверяем, что это копия (изменения не влияют на registry)
	delete(all, "cmd-alpha")
	_, ok := Get("cmd-alpha")
	assert.True(t, ok, "удаление из копии не должно влиять на registry")
}

// TestAll_Empty возвращает пустую map если registry пуст.
func TestAll_Empty(t *testing.T) {
	clearRegistry()

	all := All()
	assert.NotNil(t, all, "All() должен вернуть не nil")
	assert.Empty(t, all, "All() должен вернуть пустую map")
}

// TestNames возвращает список имён всех зарегистрированных команд.
func TestNames(t *testing.T) {
	clearRegistry()

	// Регистрируем несколько команд
	_ = Register(&mockHandler{name: "cmd-alpha"})
	_ = Register(&mockHandler{name: "cmd-beta"})
	_ = Register(&mockHandler{name: "cmd-gamma"})

	names := Names()

	assert.Len(t, names, 3, "должно быть 3 имени")
	assert.Contains(t, names, "cmd-alpha")
	assert.Contains(t, names, "cmd-beta")
	assert.Contains(t, names, "cmd-gamma")
}

// TestNames_Empty возвращает пустой slice если registry пуст.
func TestNames_Empty(t *testing.T) {
	clearRegistry()

	names := Names()
	assert.NotNil(t, names, "Names() должен вернуть не nil")
	assert.Empty(t, names, "Names() должен вернуть пустой slice")
}

// TestListAllWithAliases_ReturnsCompleteMapping проверяет что ListAllWithAliases
// возвращает полный маппинг команд с их deprecated-алиасами.
func TestListAllWithAliases_ReturnsCompleteMapping(t *testing.T) {
	clearRegistry()

	// Регистрируем команды: с алиасом и без
	h1 := &mockHandler{name: "nr-cmd-one"}
	_ = RegisterWithAlias(h1, "cmd-one")

	h2 := &mockHandler{name: "nr-cmd-two"}
	_ = RegisterWithAlias(h2, "cmd-two")

	h3 := &mockHandler{name: "nr-standalone"}
	_ = Register(h3)

	result := ListAllWithAliases()

	// Должно быть 3 записи (bridges не включаются как отдельные)
	assert.Len(t, result, 3, "должно быть 3 команды (без deprecated bridges)")

	// Проверяем отсортированность
	assert.Equal(t, "nr-cmd-one", result[0].Name)
	assert.Equal(t, "nr-cmd-two", result[1].Name)
	assert.Equal(t, "nr-standalone", result[2].Name)

	// Проверяем алиасы
	assert.Equal(t, "cmd-one", result[0].DeprecatedAlias)
	assert.Equal(t, "cmd-two", result[1].DeprecatedAlias)
	assert.Empty(t, result[2].DeprecatedAlias, "standalone команда не должна иметь алиас")

}

// TestListAllWithAliases_EmptyRegistry проверяет поведение с пустым реестром.
func TestListAllWithAliases_EmptyRegistry(t *testing.T) {
	clearRegistry()

	result := ListAllWithAliases()

	assert.NotNil(t, result, "должен вернуть не nil")
	assert.Empty(t, result, "должен вернуть пустой slice")
}

// TestConcurrentReadWrite тестирует одновременное чтение и запись одного ключа.
// Это дополнительный race condition тест.
func TestConcurrentReadWrite(t *testing.T) {
	clearRegistry()

	const targetCmd = "concurrent-rw-cmd"
	var wg sync.WaitGroup

	// Запускаем горутину записи
	wg.Add(1)
	go func() {
		defer wg.Done()
		h := &mockHandler{name: targetCmd}
		_ = Register(h)
	}()

	// Запускаем несколько горутин чтения
	const readers = 50
	wg.Add(readers)
	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			// Команда может быть найдена или нет в зависимости от timing
			_, _ = Get(targetCmd)
		}()
	}

	wg.Wait()

	// После завершения всех горутин команда должна быть зарегистрирована
	handler, found := Get(targetCmd)
	assert.True(t, found, "команда должна быть зарегистрирована после всех операций")
	assert.NotNil(t, handler, "handler не должен быть nil")
}
