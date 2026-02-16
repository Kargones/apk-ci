package shadowrun

import (
	"context"
	"log/slog"

	"github.com/Kargones/apk-ci/internal/config"
)

// LegacyFunc определяет тип legacy-функции из internal/app/.
// Все legacy-функции принимают *context.Context, *slog.Logger, *config.Config
// и возвращают error. Для функций с дополнительными параметрами используются обёртки.
//
// TODO(H-7): *context.Context — антипаттерн Go (передача указателя на интерфейс).
// Legacy-функции в internal/app/ используют *context.Context из-за исторического API.
// При миграции на v2.0.0 заменить на context.Context (без указателя).
type LegacyFunc func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error

// LegacyMapping хранит маппинг NR-команд на legacy-функции.
// Ключ — имя NR-команды (например, "nr-service-mode-status"),
// значение — обёртка legacy-функции с унифицированной сигнатурой.
type LegacyMapping struct {
	mapping       map[string]LegacyFunc
	stateChanging map[string]bool
}

// NewLegacyMapping создаёт пустой маппинг.
// Регистрация legacy-функций происходит через Register().
func NewLegacyMapping() *LegacyMapping {
	return &LegacyMapping{
		mapping:       make(map[string]LegacyFunc),
		stateChanging: make(map[string]bool),
	}
}

// Register добавляет маппинг NR-команды на legacy-функцию.
func (m *LegacyMapping) Register(nrCommand string, legacyFn LegacyFunc) {
	m.mapping[nrCommand] = legacyFn
}

// Get возвращает legacy-функцию для NR-команды.
// Возвращает (nil, false) если маппинг не найден.
func (m *LegacyMapping) Get(nrCommand string) (LegacyFunc, bool) {
	fn, ok := m.mapping[nrCommand]
	return fn, ok
}

// HasMapping возвращает true если для NR-команды существует legacy-маппинг.
func (m *LegacyMapping) HasMapping(nrCommand string) bool {
	_, ok := m.mapping[nrCommand]
	return ok
}

// RegisteredCommands возвращает список NR-команд с зарегистрированным маппингом.
func (m *LegacyMapping) RegisteredCommands() []string {
	commands := make([]string, 0, len(m.mapping))
	for cmd := range m.mapping {
		commands = append(commands, cmd)
	}
	return commands
}

// MarkStateChanging помечает команды как изменяющие состояние (enable, disable, dbrestore и т.д.).
// Shadow-run выведет предупреждение при выполнении таких команд,
// т.к. legacy-версия может вызвать побочные эффекты повторно.
func (m *LegacyMapping) MarkStateChanging(commands ...string) {
	for _, cmd := range commands {
		m.stateChanging[cmd] = true
	}
}

// IsStateChanging возвращает true если команда помечена как изменяющая состояние.
func (m *LegacyMapping) IsStateChanging(command string) bool {
	return m.stateChanging[command]
}
