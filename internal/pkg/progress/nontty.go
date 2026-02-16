package progress

import (
	"log/slog"
	"time"
)

// NonTTYProgress реализует progress для non-TTY режима (CI/CD, pipes).
// Выводит сообщения в лог каждые 10% прогресса через slog (CRITICAL-2 fix).
type NonTTYProgress struct {
	opts                Options
	startTime           time.Time
	lastReportedPercent int
	message             string
	log                 *slog.Logger
}

// NewNonTTYProgress создаёт новый non-TTY progress.
func NewNonTTYProgress(opts Options) *NonTTYProgress {
	return &NonTTYProgress{
		opts: opts,
		log:  slog.Default(),
	}
}

// Start инициализирует progress с начальным сообщением.
func (p *NonTTYProgress) Start(message string) {
	p.startTime = time.Now()
	p.message = message
	p.lastReportedPercent = 0

	p.log.Info("Операция начата", slog.String("message", message))
}

// Update обновляет текущий прогресс и выводит в лог при пересечении 10% границы.
func (p *NonTTYProgress) Update(current int64, message string) {
	if message != "" {
		p.message = message
	}

	if p.opts.Total == 0 {
		return
	}

	percent := int(float64(current) / float64(p.opts.Total) * 100)
	if percent > 100 {
		percent = 100
	}

	// Выводим только при пересечении 10% границы
	reportThreshold := (percent / 10) * 10
	if reportThreshold > p.lastReportedPercent && reportThreshold > 0 && reportThreshold < 100 {
		p.lastReportedPercent = reportThreshold
		elapsed := time.Since(p.startTime).Round(time.Second)
		p.log.Info("Прогресс операции",
			slog.Int("percent", reportThreshold),
			slog.String("elapsed", FormatDuration(elapsed)),
			slog.String("message", p.message))
	}
}

// SetTotal устанавливает общее количество единиц работы.
func (p *NonTTYProgress) SetTotal(total int64) {
	p.opts.Total = total
}

// Finish завершает progress и выводит финальное сообщение.
func (p *NonTTYProgress) Finish() {
	duration := time.Since(p.startTime).Round(time.Second)
	p.log.Info("Операция завершена", slog.String("duration", FormatDuration(duration)))
}
