package shadowrun

import (
	"fmt"
	"strings"
)

// maxTruncateLen — максимальная длина строки (в рунах) при усечении вывода в diff-отчёте.
const maxTruncateLen = 500

// Difference описывает конкретное расхождение между NR и legacy результатами.
type Difference struct {
	// Field — название поля/аспекта где обнаружено различие.
	Field string `json:"field"`
	// NRValue — значение от NR-команды.
	NRValue string `json:"nr_value"`
	// LegacyValue — значение от legacy-команды.
	LegacyValue string `json:"legacy_value"`
}

// ComparisonResult содержит результат сравнения NR и legacy выполнений.
type ComparisonResult struct {
	// Match — true если результаты идентичны.
	Match bool `json:"match"`
	// Differences — список обнаруженных расхождений.
	Differences []Difference `json:"differences,omitempty"`
}

// CompareResults сравнивает результаты выполнения NR и legacy команд.
// Сравниваются: наличие ошибки (exit code) и вывод stdout.
func CompareResults(nrErr, legacyErr error, nrOutput, legacyOutput string) *ComparisonResult {
	result := &ComparisonResult{
		Match: true,
	}

	// Сравниваем ошибки: проверяем и наличие, и текст ошибки.
	// Review #31: Ранее сравнивались только nil/not-nil, что пропускало различия
	// в тексте ошибок (например, "timeout" vs "connection refused").
	nrErrStr := "<nil>"
	legacyErrStr := "<nil>"
	if nrErr != nil {
		nrErrStr = nrErr.Error()
	}
	if legacyErr != nil {
		legacyErrStr = legacyErr.Error()
	}

	if nrErrStr != legacyErrStr {
		result.Match = false
		result.Differences = append(result.Differences, Difference{
			Field:       "error",
			NRValue:     nrErrStr,
			LegacyValue: legacyErrStr,
		})
	}

	// Сравниваем stdout (нормализуем trailing whitespace)
	nrTrimmed := strings.TrimSpace(nrOutput)
	legacyTrimmed := strings.TrimSpace(legacyOutput)

	if nrTrimmed != legacyTrimmed {
		result.Match = false
		result.Differences = append(result.Differences, Difference{
			Field:       "output",
			NRValue:     truncate(nrTrimmed, maxTruncateLen),
			LegacyValue: truncate(legacyTrimmed, maxTruncateLen),
		})
	}

	return result
}

// FormatDiff формирует читаемый diff для текстового вывода.
func FormatDiff(comparison *ComparisonResult) string {
	if comparison.Match {
		return "Результаты идентичны"
	}

	var b strings.Builder
	b.WriteString("Обнаружены различия:\n")
	for _, d := range comparison.Differences {
		b.WriteString(fmt.Sprintf("  [%s]\n", d.Field))
		b.WriteString(fmt.Sprintf("    NR:     %s\n", d.NRValue))
		b.WriteString(fmt.Sprintf("    Legacy: %s\n", d.LegacyValue))
	}
	return b.String()
}

// truncate обрезает строку до maxLen рун (Unicode-символов), добавляя "..." при обрезке.
// Используется []rune для корректной обрезки UTF-8 строк (кириллица, emoji и др.).
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
