package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// summaryDivider — разделитель для summary блока в текстовом выводе.
// L-2 fix: вынесено в константу для избежания дублирования.
const summaryDivider = "══════════════════════════════════════════════════════"

// TextWriter форматирует Result в человекочитаемый текст.
type TextWriter struct{}

// NewTextWriter создаёт новый TextWriter.
func NewTextWriter() *TextWriter {
	return &TextWriter{}
}

// Write форматирует result в текст и записывает в w.
// Story 5-9 AC-2: Text output содержит визуальный summary блок.
func (t *TextWriter) Write(w io.Writer, result *Result) error {
	if result == nil {
		return nil
	}

	// Базовый формат: Command: status
	if _, err := fmt.Fprintf(w, "%s: %s\n", result.Command, result.Status); err != nil {
		return err
	}

	// Ошибка
	if result.Error != nil {
		if _, err := fmt.Fprintf(w, "Error [%s]: %s\n", result.Error.Code, result.Error.Message); err != nil {
			return err
		}
	}

	// Data — выводим как JSON если не пустое
	if result.Data != nil {
		dataJSON, err := json.MarshalIndent(result.Data, "", "  ")
		if err != nil {
			return fmt.Errorf("не удалось сериализовать Data: %w", err)
		}
		if _, err := fmt.Fprintf(w, "Data: %s\n", dataJSON); err != nil {
			return err
		}
	}

	// Story 5-9 AC-2: Summary блок (визуально отделён от основного содержимого)
	// M-4 fix: Не выводим summary для ошибок — это перегружает вывод.
	// Summary полезен только для успешных операций с метриками.
	if result.Status != StatusError {
		if err := t.writeSummary(w, result); err != nil {
			return err
		}
	}

	return nil
}

// writeSummary выводит summary блок в конце text output.
// Story 5-9 AC-2: Красивый summary визуально отделён двойной линией.
// Story 5-9 AC-6: Duration автоматически вычисляется из Metadata.DurationMs.
// Story 5-9 AC-8: Warnings отображаются с иконками.
func (t *TextWriter) writeSummary(w io.Writer, result *Result) error {
	if _, err := fmt.Fprintf(w, "\n%s\n", summaryDivider); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "📊 Сводка\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%s\n", summaryDivider); err != nil {
		return err
	}

	// Duration из Metadata
	// Story 5-9 AC-6: duration вычисляется из Metadata.DurationMs
	if result.Metadata != nil && result.Metadata.DurationMs > 0 {
		if _, err := fmt.Fprintf(w, "⏱️  Время выполнения: %s\n", formatDuration(result.Metadata.DurationMs)); err != nil {
			return err
		}
	}

	// Key Metrics
	// Story 5-9 AC-1: key_metrics отображается в summary
	if result.Summary != nil && len(result.Summary.KeyMetrics) > 0 {
		for _, m := range result.Summary.KeyMetrics {
			if m.Unit != "" {
				if _, err := fmt.Fprintf(w, "📈 %s: %s %s\n", m.Name, m.Value, m.Unit); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(w, "📈 %s: %s\n", m.Name, m.Value); err != nil {
					return err
				}
			}
		}
	}

	// Warnings
	// Story 5-9 AC-8: Warnings отображаются с иконками
	if result.Summary != nil && result.Summary.WarningsCount > 0 {
		if _, err := fmt.Fprintf(w, "\n⚠️  Предупреждений: %d\n", result.Summary.WarningsCount); err != nil {
			return err
		}
		for _, warn := range result.Summary.Warnings {
			if _, err := fmt.Fprintf(w, "   • %s\n", warn); err != nil {
				return err
			}
		}
	}

	if _, err := fmt.Fprintf(w, "%s\n", summaryDivider); err != nil {
		return err
	}

	return nil
}

// formatDuration форматирует duration в человекочитаемый вид.
// Story 5-9 AC-6: Поддерживает миллисекунды, секунды и минуты.
// M-2 fix: используем int64 для избежания overflow на 32-bit системах.
func formatDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dмс", ms)
	}
	sec := ms / 1000
	if sec < 60 {
		// Для секунд показываем десятичную часть.
		secFloat := float64(ms) / 1000
		return fmt.Sprintf("%.1fс", secFloat)
	}
	min := sec / 60
	secRem := sec % 60
	return fmt.Sprintf("%dм %dс", min, secRem)
}
