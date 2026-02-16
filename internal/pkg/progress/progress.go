// Package progress предоставляет интерфейсы и реализации для отображения прогресса долгих операций.
// Поддерживает несколько режимов: TTY progress bar, non-TTY логирование, JSON streaming и spinner.
package progress

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Progress определяет интерфейс для отображения прогресса операций.
type Progress interface {
	// Start инициализирует progress с начальным сообщением.
	Start(message string)
	// Update обновляет текущий прогресс.
	// current — текущее значение, message — опциональное сообщение.
	Update(current int64, message string)
	// Finish завершает progress с финальным статусом.
	Finish()
	// SetTotal устанавливает общее количество (если стало известно).
	SetTotal(total int64)
}

// Options конфигурирует progress bar.
type Options struct {
	// Total — общее количество единиц работы (0 = indeterminate)
	Total int64
	// Output — куда выводить (обычно os.Stderr)
	Output io.Writer
	// ShowETA — показывать ли расчётное время завершения
	ShowETA bool
	// ThrottleInterval — минимальный интервал между обновлениями
	ThrottleInterval time.Duration
}

// Event описывает JSON событие прогресса для streaming вывода.
type Event struct {
	Type       string `json:"type"`                  // "progress_start", "progress", "progress_end"
	Percent    *int   `json:"percent,omitempty"`     // 0-100 (pointer для omitempty при неизвестном значении)
	ETASeconds *int64 `json:"eta_seconds,omitempty"` // оставшееся время в секундах (pointer: nil = неизвестно)
	Message    string `json:"message,omitempty"`     // текстовое сообщение
	DurationMs int64  `json:"duration_ms,omitempty"` // для progress_end — общее время выполнения
}

// IsTTY проверяет, является ли writer терминалом.
func IsTTY(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		fi, err := f.Stat()
		if err != nil {
			return false
		}
		return (fi.Mode() & os.ModeCharDevice) != 0
	}
	return false
}

// FormatDuration форматирует duration в читаемый вид (1h 7m 30s, 5m 30s, 45s).
// Экспортирована для использования во всех реализациях progress.
// M-2 fix (Review #9): добавлена поддержка часов для долгих операций.
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	if d < 0 {
		return "0s" // защита от отрицательных значений (MEDIUM-2 fix)
	}

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	// M-2 fix: поддержка часов для операций > 1 часа
	if d >= time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		seconds := int(d.Seconds()) % 60

		if minutes == 0 && seconds == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		if seconds == 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		if minutes == 0 {
			return fmt.Sprintf("%dh %ds", hours, seconds)
		}
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}

	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60

	if seconds == 0 {
		return fmt.Sprintf("%dm", minutes)
	}

	return fmt.Sprintf("%dm %ds", minutes, seconds)
}
