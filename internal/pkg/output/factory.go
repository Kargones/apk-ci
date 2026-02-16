package output

import "strings"

// FormatJSON и FormatText — поддерживаемые форматы вывода.
const (
	FormatJSON = "json"
	FormatText = "text"
)

// NewWriter создаёт Writer по указанному формату.
// Поддерживаемые форматы: "json", "text" (case-insensitive).
// При неизвестном формате возвращает TextWriter (default).
func NewWriter(format string) Writer {
	// M2 fix: нормализуем case для user-friendly поведения
	switch strings.ToLower(format) {
	case FormatJSON:
		return NewJSONWriter()
	case FormatText:
		return NewTextWriter()
	default:
		// По умолчанию — текстовый формат для человекочитаемости
		return NewTextWriter()
	}
}
