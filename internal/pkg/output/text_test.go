package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTextWriter(t *testing.T) {
	writer := NewTextWriter()
	assert.NotNil(t, writer)
}

func TestTextWriter_ImplementsWriter(_ *testing.T) {
	var _ Writer = (*TextWriter)(nil)
}

// TestTextWriter_Write_Success проверяет вывод успешного результата.
// Story 5-9: Теперь включает summary блок с duration.
func TestTextWriter_Write_Success(t *testing.T) {
	result := &Result{
		Status:  StatusSuccess,
		Command: "test-command",
		Metadata: &Metadata{
			DurationMs: 150,
			APIVersion: "v1",
		},
	}

	writer := NewTextWriter()
	var buf bytes.Buffer
	err := writer.Write(&buf, result)
	require.NoError(t, err)

	output := buf.String()
	// Базовый вывод
	assert.Contains(t, output, "test-command: success")
	// Story 5-9: Summary блок
	assert.Contains(t, output, "📊 Сводка")
	assert.Contains(t, output, "⏱️  Время выполнения: 150мс")
	assert.Contains(t, output, "══════════════════════════════════════════════════════")
}

// TestTextWriter_Write_Error проверяет вывод ошибочного результата.
// M-4 fix: Summary блок НЕ выводится для ошибок — это перегружает вывод.
func TestTextWriter_Write_Error(t *testing.T) {
	result := &Result{
		Status:  StatusError,
		Command: "test-command",
		Error: &ErrorInfo{
			Code:    "CONFIG.LOAD_FAILED",
			Message: "не удалось загрузить конфигурацию",
		},
		Metadata: &Metadata{
			DurationMs: 50,
			APIVersion: "v1",
		},
	}

	writer := NewTextWriter()
	var buf bytes.Buffer
	err := writer.Write(&buf, result)
	require.NoError(t, err)

	output := buf.String()
	// Базовый вывод с ошибкой
	assert.Contains(t, output, "test-command: error")
	assert.Contains(t, output, "Error [CONFIG.LOAD_FAILED]: не удалось загрузить конфигурацию")
	// M-4 fix: Summary блок НЕ выводится для ошибок
	assert.NotContains(t, output, "📊 Сводка")
	assert.NotContains(t, output, "⏱️  Время выполнения")
}

// TestTextWriter_Write_Minimal проверяет минимальный результат без metadata.
// Story 5-9: Summary блок выводится, но без duration.
func TestTextWriter_Write_Minimal(t *testing.T) {
	result := &Result{
		Status:  StatusSuccess,
		Command: "test-command",
	}

	writer := NewTextWriter()
	var buf bytes.Buffer
	err := writer.Write(&buf, result)
	require.NoError(t, err)

	output := buf.String()
	// Базовый вывод
	assert.Contains(t, output, "test-command: success")
	// Story 5-9: Summary блок выводится, но без duration
	assert.Contains(t, output, "📊 Сводка")
	// Нет duration
	assert.NotContains(t, output, "⏱️")
}

// TestTextWriter_Write_NoDuration проверяет вывод при нулевом duration.
// Story 5-9: При DurationMs == 0 duration не выводится.
func TestTextWriter_Write_NoDuration(t *testing.T) {
	result := &Result{
		Status:  StatusSuccess,
		Command: "test-command",
		Metadata: &Metadata{
			DurationMs: 0,
			APIVersion: "v1",
		},
	}

	writer := NewTextWriter()
	var buf bytes.Buffer
	err := writer.Write(&buf, result)
	require.NoError(t, err)

	output := buf.String()
	// Duration не должен выводиться когда равен 0
	assert.NotContains(t, output, "⏱️")
	// Summary блок всё равно есть
	assert.Contains(t, output, "📊 Сводка")
}

// TestTextWriter_Write_WithData проверяет вывод с данными.
// Story 5-9: Data выводится перед summary блоком.
func TestTextWriter_Write_WithData(t *testing.T) {
	result := &Result{
		Status:  StatusSuccess,
		Command: "test-command",
		Data:    map[string]string{"version": "1.0.0"},
	}

	writer := NewTextWriter()
	var buf bytes.Buffer
	err := writer.Write(&buf, result)
	require.NoError(t, err)

	output := buf.String()
	// Data выводится
	assert.Contains(t, output, "Data: {")
	assert.Contains(t, output, "\"version\": \"1.0.0\"")
	// Story 5-9: Summary блок в конце
	assert.Contains(t, output, "📊 Сводка")
}

func TestTextWriter_Write_NilResult(t *testing.T) {
	writer := NewTextWriter()
	var buf bytes.Buffer
	err := writer.Write(&buf, nil)
	require.NoError(t, err)

	// nil result не должен ничего выводить
	assert.Equal(t, "", buf.String())
}
