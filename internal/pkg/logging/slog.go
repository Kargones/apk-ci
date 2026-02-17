package logging

import "log/slog"

// SlogAdapter реализует Logger interface используя slog из stdlib.
// Это основная production реализация логгера.
type SlogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter создаёт новый SlogAdapter с указанным slog.Logger.
// Для создания с конфигурацией используйте NewLogger().
// При nil logger используется slog.Default() с предупреждением.
func NewSlogAdapter(logger *slog.Logger) *SlogAdapter {
	if logger == nil {
		logger = slog.Default()
		logger.Warn("logging: nil slog.Logger passed to NewSlogAdapter, using default")
	}
	return &SlogAdapter{logger: logger}
}

// Debug записывает сообщение уровня DEBUG.
func (s *SlogAdapter) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}

// Info записывает сообщение уровня INFO.
func (s *SlogAdapter) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

// Warn записывает сообщение уровня WARN.
func (s *SlogAdapter) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}

// Error записывает сообщение уровня ERROR.
func (s *SlogAdapter) Error(msg string, args ...any) {
	s.logger.Error(msg, args...)
}

// With возвращает новый Logger с добавленными атрибутами.
func (s *SlogAdapter) With(args ...any) Logger {
	return &SlogAdapter{logger: s.logger.With(args...)}
}
