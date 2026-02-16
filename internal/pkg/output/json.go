package output

import (
	"encoding/json"
	"io"
)

// JSONWriter форматирует Result в JSON.
// Использует encoding/json с отступами для читаемости.
type JSONWriter struct{}

// NewJSONWriter создаёт новый JSONWriter.
func NewJSONWriter() *JSONWriter {
	return &JSONWriter{}
}

// Write сериализует result в JSON и записывает в w.
// Story 5-9 AC-3: Summary копируется в Metadata.Summary для JSON output.
// Метод не мутирует входной result — создаётся shallow copy для сериализации.
func (j *JSONWriter) Write(w io.Writer, result *Result) error {
	if result == nil {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}

	// Создаём shallow copy result для сериализации без мутации оригинала.
	// H-1 fix: метод Write() не должен иметь side-effects на входных данных.
	output := *result

	// Копируем Summary в Metadata.Summary для JSON структуры.
	// Story 5-9 AC-3: JSON output содержит metadata.summary.
	if result.Summary != nil && result.Metadata != nil {
		// Создаём копию Metadata чтобы не мутировать оригинал.
		metaCopy := *result.Metadata
		metaCopy.Summary = result.Summary
		output.Metadata = &metaCopy
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(&output)
}
