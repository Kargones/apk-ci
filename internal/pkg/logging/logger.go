// Package logging предоставляет интерфейс и реализации для структурированного логирования.
package logging

// Logger определяет интерфейс для структурированного логирования.
// Реализации: SlogAdapter (использует slog из stdlib).
//
// Все методы принимают сообщение и опциональные key-value пары:
//
//	logger.Info("Команда выполнена", "command", cmd, "duration_ms", 150)
//
// ВАЖНО: Logger пишет ТОЛЬКО в stderr, никогда в stdout.
// Это критично для корректной работы с OutputWriter.
type Logger interface {
	// Debug записывает сообщение уровня DEBUG.
	// Используется для детальной диагностики.
	Debug(msg string, args ...any)

	// Info записывает сообщение уровня INFO.
	// Используется для значимых событий (старт/стоп, успешные операции).
	Info(msg string, args ...any)

	// Warn записывает сообщение уровня WARN.
	// Используется для recoverable issues, deprecated usage.
	Warn(msg string, args ...any)

	// Error записывает сообщение уровня ERROR.
	// Используется для ошибок требующих внимания.
	Error(msg string, args ...any)

	// With возвращает новый Logger с добавленными атрибутами.
	// Атрибуты будут включены во все последующие записи.
	//
	//	logger.With("trace_id", traceID).Info("Операция началась")
	With(args ...any) Logger
}
