package progress

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// JSONProgress реализует JSON streaming progress для автоматизации.
// Выводит события в формате JSON-lines.
type JSONProgress struct {
	opts      Options
	encoder   *json.Encoder
	startTime time.Time
	lastEmit  time.Time
}

// NewJSONProgress создаёт новый JSON progress.
func NewJSONProgress(opts Options) *JSONProgress {
	var encoder *json.Encoder
	if opts.Output != nil {
		encoder = json.NewEncoder(opts.Output)
	}
	return &JSONProgress{
		opts:    opts,
		encoder: encoder,
	}
}

// Start инициализирует progress и выводит событие progress_start.
func (p *JSONProgress) Start(message string) {
	p.startTime = time.Now()
	p.lastEmit = time.Time{} // сбрасываем для первого события

	if p.encoder != nil {
		event := Event{
			Type:    "progress_start",
			Message: message,
		}
		if encErr := p.encoder.Encode(event); encErr != nil {
			fmt.Fprintf(os.Stderr, "progress: failed to encode event: %v\n", encErr) //nolint:errcheck // writing to stderr
		}
	}
}

// Update обновляет текущий прогресс и выводит событие progress.
func (p *JSONProgress) Update(current int64, message string) {
	// Throttling — не выводим чаще заданного интервала
	if p.opts.ThrottleInterval > 0 && time.Since(p.lastEmit) < p.opts.ThrottleInterval {
		return
	}
	p.lastEmit = time.Now()

	if p.encoder == nil {
		return
	}

	// HIGH-1 fix: используем pointers для корректного omitempty
	var percentPtr *int
	var etaPtr *int64

	if p.opts.Total > 0 {
		percent := int(float64(current) / float64(p.opts.Total) * 100)
		if percent > 100 {
			percent = 100
		}
		percentPtr = &percent

		// ETA вычисляем только если есть прогресс
		if current > 0 {
			eta := p.calculateETASeconds(current)
			if eta > 0 {
				etaPtr = &eta
			}
		}
	}

	event := Event{
		Type:       "progress",
		Percent:    percentPtr,
		ETASeconds: etaPtr,
		Message:    message,
	}
	if err := p.encoder.Encode(event); err != nil {
		fmt.Fprintf(os.Stderr, "progress: encode error: %v\n", err) //nolint:errcheck // writing to stderr
	}
}

// SetTotal устанавливает общее количество единиц работы.
func (p *JSONProgress) SetTotal(total int64) {
	p.opts.Total = total
}

// Finish завершает progress и выводит событие progress_end.
func (p *JSONProgress) Finish() {
	if p.encoder == nil {
		return
	}

	duration := time.Since(p.startTime)

	event := Event{
		Type:       "progress_end",
		DurationMs: duration.Milliseconds(),
	}
	if err := p.encoder.Encode(event); err != nil {
		fmt.Fprintf(os.Stderr, "progress: encode error: %v\n", err) //nolint:errcheck // writing to stderr
	}
}

// calculateETASeconds вычисляет оставшееся время в секундах.
func (p *JSONProgress) calculateETASeconds(current int64) int64 {
	if current == 0 || p.opts.Total == 0 {
		return 0
	}

	remainingWork := p.opts.Total - current

	// MEDIUM-2 fix: защита от отрицательного ETA (если current > Total)
	if remainingWork <= 0 {
		return 0
	}

	elapsed := time.Since(p.startTime)
	remaining := time.Duration(float64(elapsed) / float64(current) * float64(remainingWork))

	return int64(remaining.Seconds())
}
