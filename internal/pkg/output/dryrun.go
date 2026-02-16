// Package output предоставляет структуры и интерфейсы для форматирования
// результатов команд в JSON и текстовом формате.
package output

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// DryRunPlan содержит план операций для dry-run режима.
// Используется для предварительного просмотра действий без реального выполнения.
type DryRunPlan struct {
	// Command — имя команды
	Command string `json:"command"`
	// Steps — шаги плана
	Steps []PlanStep `json:"steps"`
	// Summary — краткое описание плана
	Summary string `json:"summary,omitempty"`
	// ValidationPassed — прошла ли валидация
	ValidationPassed bool `json:"validation_passed"`
}

// PlanStep описывает один шаг плана.
type PlanStep struct {
	// Order — порядковый номер шага
	Order int `json:"order"`
	// Operation — название операции
	Operation string `json:"operation"`
	// Parameters — параметры операции (ключ-значение)
	Parameters map[string]any `json:"parameters"`
	// ExpectedChanges — ожидаемые изменения
	ExpectedChanges []string `json:"expected_changes,omitempty"`
	// Skipped — пропущен ли шаг (например, auto-deps=false)
	Skipped bool `json:"skipped,omitempty"`
	// SkipReason — причина пропуска
	SkipReason string `json:"skip_reason,omitempty"`
}

// WriteText выводит план в человекочитаемом формате.
// Формат соответствует AC-4: заголовок "=== DRY RUN ===" и структурированный план.
func (p *DryRunPlan) WriteText(w io.Writer) error {
	return p.writeTextInternal(w, "=== DRY RUN ===", "=== END DRY RUN ===")
}

// WritePlanText выводит план в человекочитаемом формате для plan-only и verbose режимов.
// Формат идентичен WriteText, но с заголовком "=== OPERATION PLAN ===" вместо "=== DRY RUN ===".
// Story 7.3 AC-2, AC-5.
func (p *DryRunPlan) WritePlanText(w io.Writer) error {
	return p.writeTextInternal(w, "=== OPERATION PLAN ===", "=== END OPERATION PLAN ===")
}

// writeTextInternal — общая логика текстового вывода плана.
// Используется в WriteText и WritePlanText с разными заголовками/подвалами.
func (p *DryRunPlan) writeTextInternal(w io.Writer, header, footer string) error {
	if _, err := fmt.Fprintf(w, "\n%s\n", header); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Команда: %s\n", p.Command); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Валидация: %s\n\n", boolToStatus(p.ValidationPassed)); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "План выполнения:\n"); err != nil {
		return err
	}

	for _, step := range p.Steps {
		if step.Skipped {
			if _, err := fmt.Fprintf(w, "  %d. [SKIP] %s — %s\n", step.Order, step.Operation, step.SkipReason); err != nil {
				return err
			}
			continue
		}

		if _, err := fmt.Fprintf(w, "  %d. %s\n", step.Order, step.Operation); err != nil {
			return err
		}

		// M-3 fix: явная проверка nil Parameters перед обработкой
		if len(step.Parameters) > 0 {
			// Сортируем ключи для детерминированного вывода
			keys := make([]string, 0, len(step.Parameters))
			for k := range step.Parameters {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				// M-5 fix: экранируем значения для предотвращения log injection
				if _, err := fmt.Fprintf(w, "      %s: %s\n", k, sanitizeValue(step.Parameters[k])); err != nil {
					return err
				}
			}
		}

		if len(step.ExpectedChanges) > 0 {
			if _, err := fmt.Fprintf(w, "      Ожидаемые изменения:\n"); err != nil {
				return err
			}
			for _, change := range step.ExpectedChanges {
				if _, err := fmt.Fprintf(w, "        - %s\n", change); err != nil {
					return err
				}
			}
		}
	}

	if p.Summary != "" {
		if _, err := fmt.Fprintf(w, "\nИтого: %s\n", p.Summary); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "%s\n", footer); err != nil {
		return err
	}

	return nil
}

// boolToStatus конвертирует boolean в человекочитаемый статус.
func boolToStatus(b bool) string {
	if b {
		return "✅ Пройдена"
	}
	return "❌ Не пройдена"
}

// sanitizeValue экранирует специальные символы в значении для безопасного вывода.
// M-5 fix: предотвращает log injection через ANSI escape sequences и управляющие символы.
// H-1 fix: полностью удаляет ANSI escape sequences (не только первый символ).
func sanitizeValue(v any) string {
	s := fmt.Sprintf("%v", v)
	// Удаляем ANSI escape sequences (например, \x1b[31m для красного цвета)
	// и другие управляющие символы (кроме пробелов и переносов строк)
	var result strings.Builder
	inEscapeSeq := false
	for _, r := range s {
		// Обработка ANSI escape sequence: \x1b[...m или \x1b[...;...m
		if r == '\x1b' {
			inEscapeSeq = true
			continue
		}
		if inEscapeSeq {
			// ANSI escape sequences заканчиваются буквой (обычно 'm' для цветов)
			// Формат: ESC [ <params> <letter>
			// Пропускаем все символы до завершающей буквы
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscapeSeq = false
			}
			continue
		}

		switch {
		case r == '\n' || r == '\t':
			// Заменяем переносы строк и табы на пробелы для однострочного вывода
			result.WriteRune(' ')
		case r < 32 || r == 127:
			// Пропускаем управляющие символы (0-31 и DEL)
			continue
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}
