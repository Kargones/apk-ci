package progress

import (
	"os"
	"time"
)

// DefaultThrottleInterval — интервал throttling по умолчанию (1 секунда).
const DefaultThrottleInterval = time.Second

// New создаёт подходящую реализацию Progress на основе окружения и Options.
// Логика выбора:
// 1. BR_SHOW_PROGRESS=false → NoopProgress
// 2. BR_OUTPUT_FORMAT=json && BR_PROGRESS_STREAM=true → JSONProgress
// 3. Total=0 (indeterminate) → SpinnerProgress
// 4. TTY → TTYProgress
// 5. Иначе → NonTTYProgress
func New(opts Options) Progress {
	// Устанавливаем дефолтный throttle если не задан
	if opts.ThrottleInterval == 0 {
		opts.ThrottleInterval = DefaultThrottleInterval
	}

	// Устанавливаем дефолтный output если не задан
	if opts.Output == nil {
		opts.Output = os.Stderr
	}

	// Проверяем явное отключение
	if os.Getenv("BR_SHOW_PROGRESS") == "false" {
		return &NoopProgress{}
	}

	// JSON output режим
	if os.Getenv("BR_OUTPUT_FORMAT") == "json" {
		// Если явно запрошен JSON progress streaming — используем JSONProgress
		if os.Getenv("BR_PROGRESS_STREAM") == "true" {
			return NewJSONProgress(opts)
		}
		// MEDIUM-4 fix: JSON output без PROGRESS_STREAM — отключаем progress,
		// чтобы не выводить текстовые сообщения в stderr которые могут конфликтовать с JSON парсингом
		return &NoopProgress{}
	}

	// Indeterminate — spinner
	if opts.Total == 0 {
		return NewSpinnerProgress(opts)
	}

	// Determinate — progress bar или log
	if IsTTY(opts.Output) {
		return NewTTYProgress(opts)
	}
	return NewNonTTYProgress(opts)
}

// NewIndeterminate создаёт progress для операций с неизвестной длительностью.
// M-4 fix: общий helper для handlers (dbupdatehandler, createtempdbhandler).
// Используется когда время операции непредсказуемо — показывает spinner.
func NewIndeterminate() Progress {
	// BR_SHOW_PROGRESS=false отключает progress bar
	if os.Getenv("BR_SHOW_PROGRESS") == "false" {
		return NewNoOp()
	}

	// Время операции непредсказуемо — используем SpinnerProgress (Total=0)
	opts := Options{
		Total:            0,         // indeterminate → SpinnerProgress
		Output:           os.Stderr, // важно: stderr, чтобы не ломать JSON output в stdout
		ShowETA:          false,     // нет ETA для неизвестной длительности
		ThrottleInterval: time.Second,
	}

	return New(opts)
}
