package command

import (
	"regexp"
	"sort"
	"sync"
)

var (
	// registry хранит зарегистрированные обработчики команд.
	// Ключ — имя команды, значение — обработчик.
	registry = make(map[string]Handler)
	// mu обеспечивает потокобезопасный доступ к registry.
	mu sync.RWMutex
	// commandNamePattern валидирует формат имени команды (kebab-case).
	// Допустимы: буквы a-z, цифры 0-9, дефис. Должно начинаться с буквы.
	// Review #34 fix: запрещён завершающий дефис и двойные дефисы (strict kebab-case).
	commandNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
)

// Register регистрирует обработчик команды в глобальном реестре.
// Вызывается из RegisterCmd() функций пакетов-обработчиков.
//
// Паникует если:
//   - h == nil (programming error)
//   - h.Name() == "" (programming error)
//   - h.Name() не соответствует формату kebab-case (programming error)
//   - команда с таким именем уже зарегистрирована (programming error)
//
// Формат имени команды: kebab-case (a-z, 0-9, дефис), начинается с буквы.
// Примеры валидных имён: "convert", "service-mode-status", "nr-version".
//
// Пример использования:
//
//	func RegisterCmd() {
//	    command.Register(&MyHandler{})
//	}
func Register(h Handler) {
	if h == nil {
		panic("command: nil handler")
	}
	name := h.Name()
	if name == "" {
		panic("command: empty handler name")
	}
	if !commandNamePattern.MatchString(name) {
		panic("command: invalid handler name format (must be kebab-case): " + name)
	}

	mu.Lock()
	defer mu.Unlock()

	if _, exists := registry[name]; exists {
		panic("command: duplicate handler registration for " + name)
	}
	registry[name] = h
}

// Get возвращает обработчик команды по имени.
// Возвращает (nil, false) если команда не зарегистрирована.
func Get(name string) (Handler, bool) {
	mu.RLock()
	defer mu.RUnlock()
	h, ok := registry[name]
	return h, ok
}

// All возвращает копию всех зарегистрированных обработчиков.
// Используется для отладки и диагностики.
// Возвращает новую map, изменения которой не влияют на registry.
func All() map[string]Handler {
	mu.RLock()
	defer mu.RUnlock()
	result := make(map[string]Handler, len(registry))
	for k, v := range registry {
		result[k] = v
	}
	return result
}

// Names возвращает отсортированный список имён всех зарегистрированных команд.
// Используется для отладки и диагностики.
// M4 fix: результат всегда отсортирован по алфавиту для детерминированного вывода.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RegisterWithAlias регистрирует обработчик под его основным именем и
// дополнительно под deprecated именем (если указано).
//
// При вызове deprecated имени пользователь получит warning в stderr с рекомендацией
// перехода на новое имя команды. Команда будет выполнена через actual handler.
//
// Если deprecated пустой — просто вызывает Register(h).
// Если deprecated не пустой:
//   - Регистрирует handler под его Name()
//   - Создаёт DeprecatedBridge и регистрирует под deprecated именем
//
// Паникует если:
//   - h == nil (programming error)
//   - deprecated == h.Name() (бессмысленная регистрация)
//
// Пример использования:
//
//	func RegisterCmd() {
//	    // Регистрирует "nr-version" и "version" (deprecated)
//	    command.RegisterWithAlias(&VersionHandler{}, "version")
//	}
func RegisterWithAlias(h Handler, deprecated string) {
	if h == nil {
		panic("command: nil handler")
	}

	// Регистрируем под основным именем.
	// NOTE: Register() захватывает и освобождает mu внутри.
	// Это безопасно, т.к. функция вызывается только в init() — однопоточно.
	// Между Register() и повторным захватом mu ниже есть теоретическое окно,
	// но в init() конкуренции нет.
	Register(h)

	// Если deprecated указан — создаём bridge
	if deprecated != "" {
		if deprecated == h.Name() {
			panic("command: deprecated name cannot be same as handler name: " + deprecated)
		}
		bridge := &DeprecatedBridge{
			actual:     h,
			deprecated: deprecated,
			newName:    h.Name(),
		}
		// Регистрируем bridge под deprecated именем.
		// Используем прямой доступ к registry, т.к. Register() проверяет kebab-case,
		// а deprecated имена могут быть legacy (например, с подчёркиваниями).
		mu.Lock()
		defer mu.Unlock()
		if _, exists := registry[deprecated]; exists {
			panic("command: duplicate handler registration for " + deprecated)
		}
		registry[deprecated] = bridge
	}
}

// Info содержит информацию о зарегистрированной команде
// с данными о deprecated-алиасе для rollback-маппинга.
type Info struct {
	// Name — основное имя команды (например, "nr-service-mode-status").
	Name string
	// DeprecatedAlias — deprecated-алиас команды (например, "service-mode-status").
	// Пустая строка если алиас отсутствует.
	DeprecatedAlias string
}

// ListAllWithAliases возвращает информацию обо всех зарегистрированных командах
// с их deprecated-алиасами. Deprecated bridges не включаются как отдельные записи —
// вместо этого их алиасы указываются в поле DeprecatedAlias основной команды.
// Результат отсортирован по имени команды.
func ListAllWithAliases() []Info {
	mu.RLock()
	defer mu.RUnlock()

	// Сначала собираем маппинг deprecated-алиасов: newName → deprecatedName
	aliasMap := make(map[string]string)
	for _, h := range registry {
		if bridge, ok := h.(*DeprecatedBridge); ok {
			aliasMap[bridge.newName] = bridge.deprecated
		}
	}

	// Собираем информацию о командах (исключая deprecated bridges).
	// Capacity = общее количество минус bridges для точного выделения.
	result := make([]Info, 0, len(registry)-len(aliasMap))
	for name, h := range registry {
		if _, isBridge := h.(*DeprecatedBridge); isBridge {
			continue
		}
		info := Info{
			Name:            name,
			DeprecatedAlias: aliasMap[name],
		}
		result = append(result, info)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// clearRegistry очищает реестр. Используется только в тестах
// для обеспечения изоляции между тестами.
func clearRegistry() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]Handler)
}
