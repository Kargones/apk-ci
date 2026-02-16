package logging

// NopLogger — реализация Logger, которая ничего не делает.
// Используется в тестах для отключения логирования.
type NopLogger struct{}

// NewNopLogger создаёт Logger, который игнорирует все сообщения.
// Полезен для unit-тестов где логирование не важно.
func NewNopLogger() Logger {
	return &NopLogger{}
}

// Debug ничего не делает.
func (n *NopLogger) Debug(_ string, _ ...any) {}

// Info ничего не делает.
func (n *NopLogger) Info(_ string, _ ...any) {}

// Warn ничего не делает.
func (n *NopLogger) Warn(_ string, _ ...any) {}

// Error ничего не делает.
func (n *NopLogger) Error(_ string, _ ...any) {}

// With возвращает тот же NopLogger (no-op).
// ПРИМЕЧАНИЕ: В отличие от SlogAdapter.With(), который создаёт новый объект,
// NopLogger возвращает себя. Это корректное поведение для no-op логгера,
// т.к. атрибуты всё равно игнорируются. Не нарушает LSP в данном контексте.
func (n *NopLogger) With(_ ...any) Logger {
	return n
}
