package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatConstants(t *testing.T) {
	assert.Equal(t, "json", FormatJSON)
	assert.Equal(t, "text", FormatText)
}

func TestNewWriter_JSON(t *testing.T) {
	writer := NewWriter(FormatJSON)
	assert.NotNil(t, writer)

	_, ok := writer.(*JSONWriter)
	assert.True(t, ok, "должен вернуть JSONWriter для формата json")
}

func TestNewWriter_Text(t *testing.T) {
	writer := NewWriter(FormatText)
	assert.NotNil(t, writer)

	_, ok := writer.(*TextWriter)
	assert.True(t, ok, "должен вернуть TextWriter для формата text")
}

func TestNewWriter_Default(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{"пустая строка", ""},
		{"неизвестный формат", "xml"},
		{"неизвестный формат yaml", "yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := NewWriter(tt.format)
			assert.NotNil(t, writer)

			_, ok := writer.(*TextWriter)
			assert.True(t, ok, "должен вернуть TextWriter по умолчанию для: %s", tt.format)
		})
	}
}

// TestNewWriter_CaseInsensitive проверяет case-insensitive обработку форматов (M2 fix).
func TestNewWriter_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		expectJSON bool
	}{
		{"JSON uppercase", "JSON", true},
		{"Json mixed", "Json", true},
		{"TEXT uppercase", "TEXT", false},
		{"Text mixed", "Text", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := NewWriter(tt.format)
			assert.NotNil(t, writer)

			if tt.expectJSON {
				_, ok := writer.(*JSONWriter)
				assert.True(t, ok, "должен вернуть JSONWriter для: %s", tt.format)
			} else {
				_, ok := writer.(*TextWriter)
				assert.True(t, ok, "должен вернуть TextWriter для: %s", tt.format)
			}
		})
	}
}
