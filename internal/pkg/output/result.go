// Package output предоставляет структуры и интерфейсы для форматирования
// результатов команд в JSON и текстовом формате.
package output

// StatusSuccess и StatusError — возможные значения поля Status в Result.
const (
	StatusSuccess = "success"
	StatusError   = "error"
)

// Result представляет структурированный результат выполнения команды.
// Используется для сериализации в JSON (BR_OUTPUT_FORMAT=json)
// или для формирования человекочитаемого вывода (BR_OUTPUT_FORMAT=text).
type Result struct {
	// Status содержит статус выполнения: "success" или "error".
	Status string `json:"status"`

	// Command содержит имя выполненной команды.
	Command string `json:"command"`

	// Data содержит command-specific payload.
	// Для каждой команды определяется свой типизированный struct.
	Data any `json:"data,omitempty"`

	// Error содержит информацию об ошибке (только при status="error").
	Error *ErrorInfo `json:"error,omitempty"`

	// Metadata содержит метаданные выполнения.
	Metadata *Metadata `json:"metadata,omitempty"`

	// DryRun указывает что результат — это dry-run план, а не реальное выполнение.
	// AC-3: В JSON output имеет поле "dry_run": true.
	DryRun bool `json:"dry_run,omitempty"`

	// PlanOnly указывает что результат — это plan-only отображение плана.
	// Story 7.3 AC-6: JSON output содержит "plan_only": true.
	PlanOnly bool `json:"plan_only,omitempty"`

	// Plan содержит план операций для dry-run режима.
	// AC-3: JSON output содержит структуру "plan": {...}.
	Plan *DryRunPlan `json:"plan,omitempty"`

	// Summary содержит сводку с ключевыми метриками (опционально).
	// Story 5-9 AC-4: Summary интегрирован в output.Result структуру.
	// Story 5-9 AC-7: Если nil — выводится базовый summary только с duration.
	// H-2 fix: json:"-" — Summary не сериализуется напрямую в Result,
	// а копируется в Metadata.Summary через JSONWriter (AC-3).
	Summary *SummaryInfo `json:"-"`
}

// ErrorInfo содержит информацию об ошибке в структурированном виде.
// Code — машиночитаемый код ошибки (например, "CONFIG.LOAD_FAILED").
// Message — человекочитаемое описание ошибки.
// ВАЖНО: Message НЕ ДОЛЖЕН содержать секреты!
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Metadata содержит метаданные выполнения команды.
type Metadata struct {
	// DurationMs — время выполнения команды в миллисекундах.
	DurationMs int64 `json:"duration_ms"`

	// TraceID — идентификатор трассировки для корреляции логов.
	// Заполняется через tracing.TraceIDFromContext(ctx) при формировании результата.
	TraceID string `json:"trace_id,omitempty"`

	// APIVersion — версия формата API для backward compatibility.
	// Текущая версия: "v1".
	APIVersion string `json:"api_version"`

	// Summary содержит сводку результатов для JSON output.
	// Story 5-9 AC-3: JSON output: metadata.summary object.
	// Заполняется из Result.Summary при сериализации в JSONWriter.
	Summary *SummaryInfo `json:"summary,omitempty"`
}
