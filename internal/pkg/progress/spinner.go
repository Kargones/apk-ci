package progress

import (
	"fmt"
	"log/slog"
	"time"
)

// spinnerFrames — кадры анимации spinner (braille).
// spinnerFrames is effectively constant (initialized once, never modified).
// Cannot be const: Go does not support const slices.
var spinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

// nonTTYReportInterval — интервал вывода прогресса для non-TTY режима (LOW-1 fix).
const nonTTYReportInterval = 30 * time.Second

// SpinnerProgress реализует indeterminate progress (spinner) для операций с неизвестным total.
type SpinnerProgress struct {
	opts       Options
	startTime  time.Time
	message    string
	frameIndex int
	lastDraw   time.Time
	isTTY      bool
	lastReport time.Time    // для non-TTY режима
	log        *slog.Logger // для non-TTY режима (CRITICAL-2 consistency fix)
}

// NewSpinnerProgress создаёт новый spinner progress.
func NewSpinnerProgress(opts Options) *SpinnerProgress {
	isTTY := false
	if opts.Output != nil {
		isTTY = IsTTY(opts.Output)
	}
	return &SpinnerProgress{
		opts:  opts,
		isTTY: isTTY,
		log:   slog.Default(),
	}
}

// Start инициализирует spinner с начальным сообщением.
func (p *SpinnerProgress) Start(message string) {
	p.startTime = time.Now()
	p.message = message
	p.frameIndex = 0
	p.lastDraw = time.Time{}
	p.lastReport = time.Now()

	if p.isTTY {
		if p.opts.Output != nil {
			p.draw()
		}
	} else {
		// Non-TTY режим: используем slog (CRITICAL-2 consistency fix)
		p.log.Info("Операция начата", slog.String("message", message))
	}
}

// Update обновляет spinner (сообщение, следующий кадр анимации).
// MEDIUM-5 note: параметр current игнорируется намеренно — для indeterminate progress
// нет смысла отслеживать текущее значение, так как total неизвестен.
// Вызывающий код передаёт elapsed в миллисекундах, но spinner использует
// собственный расчёт elapsed через time.Since(startTime).
func (p *SpinnerProgress) Update(_ int64, message string) {
	if message != "" {
		p.message = message
	}

	if p.opts.Output == nil {
		return
	}

	if p.isTTY {
		// TTY режим — анимация spinner
		if p.opts.ThrottleInterval > 0 && time.Since(p.lastDraw) < p.opts.ThrottleInterval {
			return
		}
		p.lastDraw = time.Now()
		p.frameIndex = (p.frameIndex + 1) % len(spinnerFrames)
		p.draw()
	} else if time.Since(p.lastReport) >= nonTTYReportInterval {
		// Non-TTY режим — периодический вывод каждые 30 секунд через slog (CRITICAL-2 consistency fix)
		p.lastReport = time.Now()
		elapsed := time.Since(p.startTime).Round(time.Second)
		p.log.Info("Прогресс операции",
			slog.String("elapsed", FormatDuration(elapsed)),
			slog.String("message", p.message))
	}
}

// SetTotal устанавливает общее количество единиц работы.
// Для spinner это позволяет переключиться на determinate режим.
func (p *SpinnerProgress) SetTotal(total int64) {
	p.opts.Total = total
}

// Finish завершает spinner и выводит финальное сообщение.
func (p *SpinnerProgress) Finish() {
	duration := time.Since(p.startTime).Round(time.Second)

	if p.isTTY {
		if p.opts.Output != nil {
			// Очищаем строку и выводим финальный статус
			_, _ = fmt.Fprintf(p.opts.Output, "\r✓ %s (завершено за %s)\033[K\n", p.message, FormatDuration(duration)) //nolint:errcheck // terminal output
		}
	} else {
		// Non-TTY режим: используем slog (CRITICAL-2 consistency fix)
		p.log.Info("Операция завершена", slog.String("duration", FormatDuration(duration)))
	}
}

// draw отрисовывает текущее состояние spinner.
func (p *SpinnerProgress) draw() {
	if p.opts.Output == nil {
		return
	}

	elapsed := time.Since(p.startTime).Round(time.Second)
	frame := spinnerFrames[p.frameIndex]

	// Формат: ⠋ Restoring... (elapsed: 1m 30s)
	_, _ = fmt.Fprintf(p.opts.Output, "\r%c %s (время: %s)\033[K", frame, p.message, FormatDuration(elapsed)) //nolint:errcheck // terminal output
}
