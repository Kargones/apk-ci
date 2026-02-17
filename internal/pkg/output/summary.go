// Package output предоставляет структуры для форматирования результатов команд.
package output

// SummaryInfo содержит сводку результатов выполнения команды.
// Используется для формирования summary блока в выводе.
// AC-4: Summary интегрирован в output.Result структуру.
// AC-5: Каждый handler может опционально предоставить свой SummaryData через Result.
type SummaryInfo struct {
	// KeyMetrics — ключевые метрики операции.
	// AC-1: key_metrics отображается в summary.
	KeyMetrics []KeyMetric `json:"key_metrics,omitempty"`

	// WarningsCount — количество предупреждений.
	// AC-3: warnings_count в JSON metadata.summary.
	WarningsCount int `json:"warnings_count"`

	// Warnings — список предупреждений (текстовых сообщений).
	// AC-8: Warnings отображаются в Text output с иконками.
	Warnings []string `json:"warnings,omitempty"`
}

// KeyMetric представляет одну ключевую метрику.
// AC-1: key_metrics включается в summary.
type KeyMetric struct {
	// Name — название метрики (например, "Файлов обработано").
	Name string `json:"name"`

	// Value — значение метрики (строка для гибкости: "15", "3.5MB", "2 из 10").
	Value string `json:"value"`

	// Unit — единица измерения (опционально: "шт", "МБ", "сек", "").
	Unit string `json:"unit,omitempty"`
}

// NewSummaryInfo создаёт новый SummaryInfo.
// AC-5: Helper функция для создания SummaryData.
func NewSummaryInfo() *SummaryInfo {
	return &SummaryInfo{
		KeyMetrics: make([]KeyMetric, 0),
		Warnings:   make([]string, 0),
	}
}

// AddMetric добавляет метрику в summary.
// AC-5: Метод для добавления ключевой метрики.
func (s *SummaryInfo) AddMetric(name, value, unit string) {
	s.KeyMetrics = append(s.KeyMetrics, KeyMetric{
		Name:  name,
		Value: value,
		Unit:  unit,
	})
}

// AddWarning добавляет предупреждение в summary.
// AC-8: Warnings накапливаются и отображаются в Text output.
func (s *SummaryInfo) AddWarning(msg string) {
	s.Warnings = append(s.Warnings, msg)
	s.WarningsCount++
}

// BuildBasicSummary создаёт базовый summary.
// Deprecated: since v1.0. Use NewSummaryInfo() instead. Remove in v2.0.0.
// AC-7: Если handler не предоставляет SummaryData — выводится базовый summary.
// Примечание: duration вычисляется из Metadata.DurationMs автоматически.
func BuildBasicSummary() *SummaryInfo {
	return NewSummaryInfo()
}
