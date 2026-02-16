package output

import "io"

// Writer определяет интерфейс для форматирования результатов команд.
// Реализации: JSONWriter, TextWriter.
type Writer interface {
	// Write форматирует result и записывает в w.
	// Возвращает ошибку если сериализация или запись не удались.
	Write(w io.Writer, result *Result) error
}
