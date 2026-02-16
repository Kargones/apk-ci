package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSummaryInfo проверяет создание SummaryInfo.
func TestNewSummaryInfo(t *testing.T) {
	s := NewSummaryInfo()

	require.NotNil(t, s)
	assert.NotNil(t, s.KeyMetrics)
	assert.NotNil(t, s.Warnings)
	assert.Empty(t, s.KeyMetrics)
	assert.Empty(t, s.Warnings)
	assert.Equal(t, 0, s.WarningsCount)
}

// TestSummaryInfo_AddMetric проверяет добавление метрики.
// AC-5: Метод AddMetric добавляет метрику с name, value, unit.
func TestSummaryInfo_AddMetric(t *testing.T) {
	s := NewSummaryInfo()
	s.AddMetric("Файлов обработано", "15", "шт")

	require.Len(t, s.KeyMetrics, 1)
	assert.Equal(t, "Файлов обработано", s.KeyMetrics[0].Name)
	assert.Equal(t, "15", s.KeyMetrics[0].Value)
	assert.Equal(t, "шт", s.KeyMetrics[0].Unit)
}

// TestSummaryInfo_AddMetric_Multiple проверяет добавление нескольких метрик.
func TestSummaryInfo_AddMetric_Multiple(t *testing.T) {
	s := NewSummaryInfo()
	s.AddMetric("Файлов", "10", "шт")
	s.AddMetric("Размер", "3.5", "МБ")
	s.AddMetric("Время", "2.1", "сек")

	require.Len(t, s.KeyMetrics, 3)
	assert.Equal(t, "Файлов", s.KeyMetrics[0].Name)
	assert.Equal(t, "Размер", s.KeyMetrics[1].Name)
	assert.Equal(t, "Время", s.KeyMetrics[2].Name)
}

// TestSummaryInfo_AddMetric_EmptyUnit проверяет метрику без единицы измерения.
func TestSummaryInfo_AddMetric_EmptyUnit(t *testing.T) {
	s := NewSummaryInfo()
	s.AddMetric("Состояние", "активно", "")

	require.Len(t, s.KeyMetrics, 1)
	assert.Equal(t, "", s.KeyMetrics[0].Unit)
}

// TestSummaryInfo_AddWarning проверяет добавление предупреждения.
// AC-8: Warnings накапливаются и счётчик увеличивается.
func TestSummaryInfo_AddWarning(t *testing.T) {
	s := NewSummaryInfo()
	s.AddWarning("Некоторые файлы пропущены")

	assert.Equal(t, 1, s.WarningsCount)
	require.Len(t, s.Warnings, 1)
	assert.Equal(t, "Некоторые файлы пропущены", s.Warnings[0])
}

// TestSummaryInfo_AddWarning_Multiple проверяет добавление нескольких предупреждений.
func TestSummaryInfo_AddWarning_Multiple(t *testing.T) {
	s := NewSummaryInfo()
	s.AddWarning("Warning 1")
	s.AddWarning("Warning 2")
	s.AddWarning("Warning 3")

	assert.Equal(t, 3, s.WarningsCount)
	require.Len(t, s.Warnings, 3)
}

// TestBuildBasicSummary проверяет создание базового summary.
// AC-7: Базовый summary — пустой SummaryInfo.
func TestBuildBasicSummary(t *testing.T) {
	s := BuildBasicSummary()

	require.NotNil(t, s)
	assert.NotNil(t, s.KeyMetrics)
	assert.NotNil(t, s.Warnings)
	assert.Empty(t, s.KeyMetrics)
	assert.Empty(t, s.Warnings)
	assert.Equal(t, 0, s.WarningsCount)
}

// TestSummaryInfo_JSONSerialization проверяет JSON сериализацию SummaryInfo.
// AC-3: JSON output содержит key_metrics, warnings_count, warnings.
func TestSummaryInfo_JSONSerialization(t *testing.T) {
	s := NewSummaryInfo()
	s.AddMetric("Файлов", "5", "шт")
	s.AddWarning("Test warning")

	data, err := json.Marshal(s)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, float64(1), parsed["warnings_count"])
	assert.NotNil(t, parsed["key_metrics"])
	assert.NotNil(t, parsed["warnings"])

	// Проверяем key_metrics
	metrics := parsed["key_metrics"].([]any)
	require.Len(t, metrics, 1)
	metric := metrics[0].(map[string]any)
	assert.Equal(t, "Файлов", metric["name"])
	assert.Equal(t, "5", metric["value"])
	assert.Equal(t, "шт", metric["unit"])
}

// TestSummaryInfo_JSONOmitEmpty проверяет omitempty для пустых полей.
func TestSummaryInfo_JSONOmitEmpty(t *testing.T) {
	s := &SummaryInfo{} // Без инициализации слайсов

	data, err := json.Marshal(s)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// key_metrics и warnings не должны присутствовать (omitempty)
	_, hasMetrics := parsed["key_metrics"]
	_, hasWarnings := parsed["warnings"]
	assert.False(t, hasMetrics, "key_metrics should be omitted when empty")
	assert.False(t, hasWarnings, "warnings should be omitted when empty")

	// warnings_count всегда присутствует (int без omitempty)
	assert.Equal(t, float64(0), parsed["warnings_count"])
}

// TestJSONWriter_WithSummary проверяет JSON вывод с Summary в metadata.
// AC-3: JSON output: metadata.summary object содержит key_metrics, warnings_count.
func TestJSONWriter_WithSummary(t *testing.T) {
	summary := NewSummaryInfo()
	summary.AddMetric("Processed", "10", "шт")
	summary.AddWarning("Test warning")

	result := &Result{
		Status:  StatusSuccess,
		Command: "test-cmd",
		Data:    map[string]any{"key": "value"},
		Summary: summary,
		Metadata: &Metadata{
			DurationMs: 1500,
			TraceID:    "trace123",
			APIVersion: "v1",
		},
	}

	var buf bytes.Buffer
	err := NewJSONWriter().Write(&buf, result)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	// Проверяем metadata.summary
	metadata := parsed["metadata"].(map[string]any)
	metaSummary := metadata["summary"].(map[string]any)

	assert.Equal(t, float64(1), metaSummary["warnings_count"])
	assert.NotNil(t, metaSummary["key_metrics"])
	assert.NotNil(t, metaSummary["warnings"])
}

// TestJSONWriter_WithoutSummary проверяет JSON вывод без Summary (backward compatible).
// AC-10: Существующие handlers не требуют изменений.
func TestJSONWriter_WithoutSummary(t *testing.T) {
	result := &Result{
		Status:  StatusSuccess,
		Command: "test-cmd",
		Data:    map[string]any{"key": "value"},
		// Summary == nil
		Metadata: &Metadata{
			DurationMs: 500,
			APIVersion: "v1",
		},
	}

	var buf bytes.Buffer
	err := NewJSONWriter().Write(&buf, result)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	// metadata не должен содержать summary
	metadata := parsed["metadata"].(map[string]any)
	_, hasSummary := metadata["summary"]
	assert.False(t, hasSummary, "summary should not be present when nil")
}

// TestJSONWriter_NoMetadata проверяет JSON вывод без Metadata.
// M-3: Edge-case тест — Summary есть, но Metadata nil.
func TestJSONWriter_NoMetadata(t *testing.T) {
	summary := NewSummaryInfo()
	summary.AddMetric("Test", "1", "")

	result := &Result{
		Status:  StatusSuccess,
		Command: "test-cmd",
		Summary: summary,
		// Metadata == nil
	}

	var buf bytes.Buffer
	err := NewJSONWriter().Write(&buf, result)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	// Summary не должен появляться в root (json:"-")
	_, hasSummary := parsed["summary"]
	assert.False(t, hasSummary, "summary should not be in root JSON")

	// metadata отсутствует
	_, hasMetadata := parsed["metadata"]
	assert.False(t, hasMetadata, "metadata should not be present when nil")
}

// TestJSONWriter_NoMutation проверяет что Write() не мутирует входной result.
// H-1: JSONWriter.Write() не должен иметь side-effects.
func TestJSONWriter_NoMutation(t *testing.T) {
	summary := NewSummaryInfo()
	summary.AddMetric("Test", "1", "")

	metadata := &Metadata{
		DurationMs: 100,
		APIVersion: "v1",
		// Summary изначально nil
	}

	result := &Result{
		Status:   StatusSuccess,
		Command:  "test-cmd",
		Summary:  summary,
		Metadata: metadata,
	}

	var buf bytes.Buffer
	err := NewJSONWriter().Write(&buf, result)
	require.NoError(t, err)

	// Проверяем что оригинальный Metadata.Summary остался nil
	assert.Nil(t, result.Metadata.Summary, "original Metadata.Summary should not be mutated")
}

// TestTextWriter_WithSummary проверяет текстовый вывод с Summary.
// AC-2: Text output содержит визуальный summary блок.
func TestTextWriter_WithSummary(t *testing.T) {
	summary := NewSummaryInfo()
	summary.AddMetric("Processed", "5", "")
	summary.AddWarning("Test warning")

	result := &Result{
		Status:  StatusSuccess,
		Command: "test-cmd",
		Summary: summary,
		Metadata: &Metadata{
			DurationMs: 1500,
			APIVersion: "v1",
		},
	}

	var buf bytes.Buffer
	err := NewTextWriter().Write(&buf, result)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "📊 Сводка")
	assert.Contains(t, output, "⏱️  Время выполнения: 1.5с")
	assert.Contains(t, output, "📈 Processed: 5")
	assert.Contains(t, output, "⚠️  Предупреждений: 1")
	assert.Contains(t, output, "• Test warning")
}

// TestTextWriter_WithSummary_MetricWithUnit проверяет вывод метрики с единицей измерения.
func TestTextWriter_WithSummary_MetricWithUnit(t *testing.T) {
	summary := NewSummaryInfo()
	summary.AddMetric("Размер", "3.5", "МБ")

	result := &Result{
		Status:  StatusSuccess,
		Command: "test-cmd",
		Summary: summary,
		Metadata: &Metadata{
			DurationMs: 100,
			APIVersion: "v1",
		},
	}

	var buf bytes.Buffer
	err := NewTextWriter().Write(&buf, result)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "📈 Размер: 3.5 МБ")
}

// TestTextWriter_WithWarnings проверяет вывод предупреждений.
// AC-8: Warnings отображаются с иконками.
func TestTextWriter_WithWarnings(t *testing.T) {
	summary := NewSummaryInfo()
	summary.AddWarning("Warning one")
	summary.AddWarning("Warning two")

	result := &Result{
		Status:  StatusSuccess,
		Command: "test-cmd",
		Summary: summary,
		Metadata: &Metadata{
			DurationMs: 200,
			APIVersion: "v1",
		},
	}

	var buf bytes.Buffer
	err := NewTextWriter().Write(&buf, result)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "⚠️  Предупреждений: 2")
	assert.Contains(t, output, "• Warning one")
	assert.Contains(t, output, "• Warning two")
}

// TestTextWriter_NoSummary_BackwardCompatible проверяет backward compatibility.
// AC-7: Если Summary == nil, выводится базовый summary только с duration.
// AC-10: Существующие handlers не требуют изменений.
func TestTextWriter_NoSummary_BackwardCompatible(t *testing.T) {
	result := &Result{
		Status:  StatusSuccess,
		Command: "test-cmd",
		// Summary == nil
		Metadata: &Metadata{
			DurationMs: 500,
			APIVersion: "v1",
		},
	}

	var buf bytes.Buffer
	err := NewTextWriter().Write(&buf, result)
	require.NoError(t, err)

	output := buf.String()
	// Summary блок выводится с duration
	assert.Contains(t, output, "📊 Сводка")
	assert.Contains(t, output, "⏱️  Время выполнения: 500мс")
	// Но нет key_metrics и warnings
	assert.NotContains(t, output, "📈")
	assert.NotContains(t, output, "⚠️")
}

// TestTextWriter_DurationFormatting проверяет форматирование duration.
// AC-6: Summary автоматически вычисляет duration из Metadata.DurationMs.
func TestTextWriter_DurationFormatting(t *testing.T) {
	tests := []struct {
		name       string
		durationMs int64
		expected   string
	}{
		{
			name:       "milliseconds",
			durationMs: 500,
			expected:   "500мс",
		},
		{
			name:       "seconds",
			durationMs: 2500,
			expected:   "2.5с",
		},
		{
			name:       "minutes",
			durationMs: 125000, // 2м 5с
			expected:   "2м 5с",
		},
		{
			name:       "exact_second",
			durationMs: 1000,
			expected:   "1.0с",
		},
		{
			name:       "exact_minute",
			durationMs: 60000,
			expected:   "1м 0с",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{
				Status:  StatusSuccess,
				Command: "test-cmd",
				Metadata: &Metadata{
					DurationMs: tt.durationMs,
					APIVersion: "v1",
				},
			}

			var buf bytes.Buffer
			err := NewTextWriter().Write(&buf, result)
			require.NoError(t, err)

			assert.Contains(t, buf.String(), tt.expected)
		})
	}
}

// TestTextWriter_NoMetadata проверяет вывод без Metadata.
func TestTextWriter_NoMetadata(t *testing.T) {
	result := &Result{
		Status:  StatusSuccess,
		Command: "test-cmd",
		// Metadata == nil
	}

	var buf bytes.Buffer
	err := NewTextWriter().Write(&buf, result)
	require.NoError(t, err)

	output := buf.String()
	// Summary блок выводится, но без duration
	assert.Contains(t, output, "📊 Сводка")
	assert.NotContains(t, output, "⏱️")
}
