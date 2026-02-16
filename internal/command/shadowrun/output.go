package shadowrun

import "time"

// ShadowRunResult содержит результаты shadow-run для включения в вывод.
// В JSON-формате добавляется как поле "shadow_run" к основному Result.
type ShadowRunResult struct {
	// Enabled — shadow-run был активирован.
	Enabled bool `json:"enabled"`
	// Match — результаты NR и legacy совпадают.
	Match bool `json:"match"`
	// NRDuration — время выполнения NR-команды.
	NRDuration time.Duration `json:"nr_duration_ms"`
	// LegacyDuration — время выполнения legacy-команды (0 если не выполнялась).
	LegacyDuration time.Duration `json:"legacy_duration_ms"`
	// Differences — список расхождений (пусто если Match=true).
	Differences []Difference `json:"differences,omitempty"`
	// Warning — предупреждение (например, "legacy-версия не найдена").
	Warning string `json:"warning,omitempty"`
	// NRError — текст ошибки NR (пусто если нет ошибки).
	NRError string `json:"nr_error,omitempty"`
	// LegacyError — текст ошибки legacy (пусто если нет ошибки).
	LegacyError string `json:"legacy_error,omitempty"`
}

// ShadowRunResultJSON — структура для JSON-сериализации shadow_run секции.
// Дублирует ShadowRunResult с правильными JSON-тегами для duration (миллисекунды).
type ShadowRunResultJSON struct {
	Enabled          bool         `json:"enabled"`
	Match            bool         `json:"match"`
	NRDurationMs     int64        `json:"nr_duration_ms"`
	LegacyDurationMs int64        `json:"legacy_duration_ms"`
	Differences      []Difference `json:"differences,omitempty"`
	Warning          string       `json:"warning,omitempty"`
	NRError          string       `json:"nr_error,omitempty"`
	LegacyError      string       `json:"legacy_error,omitempty"`
}

// ToJSON конвертирует ShadowRunResult в JSON-сериализуемую структуру.
func (r *ShadowRunResult) ToJSON() ShadowRunResultJSON {
	return ShadowRunResultJSON{
		Enabled:          r.Enabled,
		Match:            r.Match,
		NRDurationMs:     r.NRDuration.Milliseconds(),
		LegacyDurationMs: r.LegacyDuration.Milliseconds(),
		Differences:      r.Differences,
		Warning:          r.Warning,
		NRError:          r.NRError,
		LegacyError:      r.LegacyError,
	}
}
