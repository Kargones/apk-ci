package progress

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// barWidth — ширина progress bar в символах.
const barWidth = 30

// TTYProgress реализует интерактивный progress bar для терминала.
type TTYProgress struct {
	mu        sync.Mutex
	opts      Options
	startTime time.Time
	current   int64
	lastDraw  time.Time
	message   string
}

// NewTTYProgress создаёт новый TTY progress bar.
func NewTTYProgress(opts Options) *TTYProgress {
	return &TTYProgress{
		opts: opts,
	}
}

// Start инициализирует progress bar с начальным сообщением.
func (p *TTYProgress) Start(message string) {
	p.startTime = time.Now()
	p.message = message
	p.current = 0
	p.lastDraw = time.Time{} // сбрасываем для первого вывода
}

// Update обновляет текущий прогресс и перерисовывает bar.
func (p *TTYProgress) Update(current int64, message string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = current
	if message != "" {
		p.message = message
	}

	// Throttling — не обновляем чаще заданного интервала
	if p.opts.ThrottleInterval > 0 && time.Since(p.lastDraw) < p.opts.ThrottleInterval {
		return
	}
	p.lastDraw = time.Now()

	p.draw()
}

// SetTotal устанавливает общее количество единиц работы.
func (p *TTYProgress) SetTotal(total int64) {
	p.opts.Total = total
}

// Finish завершает progress bar и выводит финальный статус.
func (p *TTYProgress) Finish() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Финальный вывод на 100%
	p.current = p.opts.Total
	p.draw()

	// Переход на новую строку
	if p.opts.Output != nil {
		_, _ = fmt.Fprintln(p.opts.Output) //nolint:errcheck // terminal output
	}
}

// draw отрисовывает текущее состояние progress bar.
func (p *TTYProgress) draw() {
	if p.opts.Output == nil {
		return
	}

	percent := 0
	if p.opts.Total > 0 {
		percent = int(float64(p.current) / float64(p.opts.Total) * 100)
		if percent > 100 {
			percent = 100
		}
	}

	bar := p.renderBar(percent)

	// Формат: [=====>    ] 45% | ETA: 2m 30s | Restoring...
	line := fmt.Sprintf("\r%s %d%%", bar, percent)

	if p.opts.ShowETA && p.opts.Total > 0 && p.current > 0 {
		eta := p.calculateETA()
		line += fmt.Sprintf(" | ETA: %s", eta)
	}

	if p.message != "" {
		line += fmt.Sprintf(" | %s", p.message)
	}

	// Очистка до конца строки (ANSI escape)
	line += "\033[K"

	_, _ = fmt.Fprint(p.opts.Output, line) //nolint:errcheck // terminal output
}

// renderBar создаёт визуальное представление progress bar.
// MEDIUM-1 fix: при percent=0 показываем пустой bar без стрелки.
func (p *TTYProgress) renderBar(percent int) string {
	filled := percent * barWidth / 100
	if filled > barWidth {
		filled = barWidth
	}

	var bar strings.Builder
	bar.WriteString("[")

	for i := 0; i < barWidth; i++ {
		switch {
		case i < filled:
			bar.WriteString("=")
		case i == filled && filled > 0 && filled < barWidth:
			// MEDIUM-1 fix: стрелка только если есть прогресс (filled > 0)
			bar.WriteString(">")
		default:
			bar.WriteString(" ")
		}
	}

	bar.WriteString("]")
	return bar.String()
}

// calculateETA вычисляет оставшееся время на основе прошедшего времени и текущего прогресса.
func (p *TTYProgress) calculateETA() string {
	if p.current == 0 {
		return "вычисляется..."
	}

	elapsed := time.Since(p.startTime)
	remainingWork := p.opts.Total - p.current

	// MEDIUM-2 fix: защита от отрицательного remaining (если current > Total)
	if remainingWork <= 0 {
		return "<1s"
	}

	remaining := time.Duration(float64(elapsed) / float64(p.current) * float64(remainingWork))

	// Округляем до секунд
	remaining = remaining.Round(time.Second)

	if remaining < time.Second {
		return "<1s"
	}

	return FormatDuration(remaining)
}
