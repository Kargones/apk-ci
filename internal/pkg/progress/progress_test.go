// Package progress предоставляет интерфейсы и реализации для отображения прогресса долгих операций.
package progress

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// TestProgressInterface проверяет, что все реализации соответствуют интерфейсу Progress.
func TestProgressInterface(_ *testing.T) {
	var _ Progress = &TTYProgress{}
	var _ Progress = &NonTTYProgress{}
	var _ Progress = &JSONProgress{}
	var _ Progress = &SpinnerProgress{}
	var _ Progress = &NoopProgress{}
}

// TestIsTTY проверяет детекцию терминала.
// Примечание: полноценное тестирование TTY невозможно в CI окружении,
// так как os.Stdout/os.Stderr не являются терминалами.
// Тест проверяет только non-TTY случаи (MEDIUM-5).
func TestIsTTY(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{
			name:     "bytes.Buffer не TTY",
			expected: false,
		},
		{
			name:     "nil writer не TTY",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "nil writer не TTY" {
				result := IsTTY(nil)
				if result != tt.expected {
					t.Errorf("IsTTY(nil) = %v, want %v", result, tt.expected)
				}
				return
			}
			var buf bytes.Buffer
			result := IsTTY(&buf)
			if result != tt.expected {
				t.Errorf("IsTTY() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestBRShowProgressEnvInFactory проверяет что BR_SHOW_PROGRESS=false в Factory возвращает NoopProgress.
// Это заменяет удалённый тест TestShouldShowProgress (MEDIUM-2 fix: ShouldShowProgress() был мёртвым кодом).
func TestBRShowProgressEnvInFactory(t *testing.T) {
	tests := []struct {
		name         string
		envShow      string
		expectedType string
	}{
		{
			name:         "BR_SHOW_PROGRESS=false возвращает NoopProgress",
			envShow:      "false",
			expectedType: "*progress.NoopProgress",
		},
		{
			name:         "BR_SHOW_PROGRESS не задан — обычный progress",
			envShow:      "",
			expectedType: "*progress.NonTTYProgress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldShow := os.Getenv("BR_SHOW_PROGRESS")
			t.Cleanup(func() {
				_ = os.Setenv("BR_SHOW_PROGRESS", oldShow)
			})

			if tt.envShow != "" {
				_ = os.Setenv("BR_SHOW_PROGRESS", tt.envShow)
			} else {
				_ = os.Unsetenv("BR_SHOW_PROGRESS")
			}

			var buf bytes.Buffer
			opts := Options{
				Total:  100,
				Output: &buf,
			}

			p := New(opts)
			gotType := getTypeName(p)
			if gotType != tt.expectedType {
				t.Errorf("New() returned %s, want %s", gotType, tt.expectedType)
			}
		})
	}
}

// TestOptionsDefaults проверяет значения по умолчанию для Options.
func TestOptionsDefaults(t *testing.T) {
	opts := Options{}

	if opts.Total != 0 {
		t.Errorf("Default Total should be 0, got %d", opts.Total)
	}
	if opts.ShowETA != false {
		t.Errorf("Default ShowETA should be false, got %v", opts.ShowETA)
	}
	if opts.ThrottleInterval != 0 {
		t.Errorf("Default ThrottleInterval should be 0, got %v", opts.ThrottleInterval)
	}
}

// TestTTYProgressFormat проверяет формат TTY progress bar.
func TestTTYProgressFormat(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Total:            100,
		Output:           &buf,
		ShowETA:          true,
		ThrottleInterval: 0, // отключаем throttling для тестов
	}

	p := NewTTYProgress(opts)
	p.Start("Тест операции")
	p.Update(45, "Обработка...")
	p.Finish()

	output := buf.String()
	// Проверяем наличие ключевых элементов формата
	if !strings.Contains(output, "45%") {
		t.Errorf("Output should contain '45%%', got: %s", output)
	}
	if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
		t.Errorf("Output should contain progress bar brackets, got: %s", output)
	}
}

// TestTTYProgressETA проверяет расчёт ETA.
// MEDIUM-3 fix: тест использует минимальный sleep для стабильности в CI.
func TestTTYProgressETA(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Total:            100,
		Output:           &buf,
		ShowETA:          true,
		ThrottleInterval: 0,
	}

	p := NewTTYProgress(opts)
	p.Start("Тест")
	// Симулируем прогресс с небольшой задержкой для расчёта ETA
	// Увеличен sleep для стабильности в нагруженных CI системах
	time.Sleep(200 * time.Millisecond)
	p.Update(50, "Тест")

	output := buf.String()
	// ETA должен быть в выводе когда ShowETA = true и current > 0
	if !strings.Contains(output, "ETA:") {
		t.Errorf("Output should contain 'ETA:', got: %s", output)
	}
}

// TestNonTTYProgressReportsEvery10Percent проверяет что progress выводит каждые 10%.
// CRITICAL-2 fix: NonTTYProgress теперь использует slog, поэтому тест проверяет
// что lastReportedPercent обновляется корректно.
func TestNonTTYProgressReportsEvery10Percent(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Total:            100,
		Output:           &buf,
		ThrottleInterval: 0,
	}

	p := NewNonTTYProgress(opts)
	p.Start("Тест операции")

	// Проверяем что lastReportedPercent обновляется при каждом пороге 10%
	expectedThresholds := []int{10, 20, 30, 40, 50, 60, 70, 80, 90}

	for i, threshold := range expectedThresholds {
		// Обновляем до следующего порога
		p.Update(int64(threshold), "Обработка")
		if p.lastReportedPercent != threshold {
			t.Errorf("After Update(%d), lastReportedPercent = %d, want %d",
				threshold, p.lastReportedPercent, threshold)
		}

		// Проверяем что промежуточные значения не меняют порог
		if i < len(expectedThresholds)-1 {
			nextThreshold := expectedThresholds[i+1]
			for j := threshold + 1; j < nextThreshold; j++ {
				p.Update(int64(j), "Обработка")
				if p.lastReportedPercent != threshold {
					t.Errorf("After Update(%d), lastReportedPercent changed to %d, want %d",
						j, p.lastReportedPercent, threshold)
				}
			}
		}
	}
}

// TestJSONProgressValidJSON проверяет, что JSON Progress генерирует валидный JSON.
func TestJSONProgressValidJSON(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Total:            100,
		Output:           &buf,
		ThrottleInterval: 0,
	}

	p := NewJSONProgress(opts)
	p.Start("Тест операции")
	p.Update(50, "Обработка")
	p.Finish()

	// Каждая строка должна быть валидным JSON
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Errorf("Invalid JSON line: %s, error: %v", line, err)
		}
	}
}

// TestJSONEvents проверяет структуру JSON событий.
func TestJSONEvents(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Total:            100,
		Output:           &buf,
		ThrottleInterval: 0,
	}

	p := NewJSONProgress(opts)
	p.Start("Тест")
	p.Update(50, "Обработка")
	p.Finish()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 3 {
		t.Fatalf("Expected at least 3 JSON events, got %d", len(lines))
	}

	// Первое событие — progress_start
	var startEvent Event
	if err := json.Unmarshal([]byte(lines[0]), &startEvent); err != nil {
		t.Fatalf("Failed to parse start event: %v", err)
	}
	if startEvent.Type != "progress_start" {
		t.Errorf("First event should be 'progress_start', got '%s'", startEvent.Type)
	}

	// Событие progress
	var progressEvent Event
	if err := json.Unmarshal([]byte(lines[1]), &progressEvent); err != nil {
		t.Fatalf("Failed to parse progress event: %v", err)
	}
	if progressEvent.Type != "progress" {
		t.Errorf("Second event should be 'progress', got '%s'", progressEvent.Type)
	}
	// HIGH-1 fix: Percent теперь pointer
	if progressEvent.Percent == nil || *progressEvent.Percent != 50 {
		t.Errorf("Progress event percent should be 50, got %v", progressEvent.Percent)
	}

	// Последнее событие — progress_end
	var endEvent Event
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &endEvent); err != nil {
		t.Fatalf("Failed to parse end event: %v", err)
	}
	if endEvent.Type != "progress_end" {
		t.Errorf("Last event should be 'progress_end', got '%s'", endEvent.Type)
	}
}

// TestSpinnerProgressIndeterminate проверяет работу spinner для неизвестного total.
// CRITICAL-2 update: SpinnerProgress в non-TTY режиме использует slog вместо buffer,
// поэтому тест проверяет что методы вызываются без паники.
func TestSpinnerProgressIndeterminate(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Total:            0, // indeterminate
		Output:           &buf,
		ThrottleInterval: 0,
	}

	p := NewSpinnerProgress(opts)

	// Проверяем что все методы работают без паники
	t.Run("Start работает", func(t *testing.T) {
		p.Start("Тест операции")
	})

	t.Run("Update работает", func(t *testing.T) {
		p.Update(0, "Обработка")
	})

	t.Run("SetTotal работает", func(t *testing.T) {
		p.SetTotal(100)
	})

	t.Run("Finish работает", func(t *testing.T) {
		p.Finish()
	})

	// В non-TTY режиме вывод идёт через slog, а не в buffer
	// Поэтому buffer может быть пустым — это нормально
	// Проверяем только что isTTY определён корректно для bytes.Buffer
	if p.isTTY {
		t.Error("bytes.Buffer should not be detected as TTY")
	}
}

// TestFactoryReturnsCorrectImplementation проверяет, что Factory выбирает правильную реализацию.
func TestFactoryReturnsCorrectImplementation(t *testing.T) {
	tests := []struct {
		name         string
		envShow      string
		envFormat    string
		envProgress  string
		total        int64
		expectedType string
	}{
		{
			name:         "BR_SHOW_PROGRESS=false возвращает NoopProgress",
			envShow:      "false",
			total:        100,
			expectedType: "*progress.NoopProgress",
		},
		{
			name:         "JSON output с PROGRESS_STREAM возвращает JSONProgress",
			envFormat:    "json",
			envProgress:  "true",
			total:        100,
			expectedType: "*progress.JSONProgress",
		},
		{
			name:         "JSON output без PROGRESS_STREAM возвращает NoopProgress (MEDIUM-4 fix)",
			envFormat:    "json",
			envProgress:  "",
			total:        100,
			expectedType: "*progress.NoopProgress",
		},
		{
			name:         "Total=0 возвращает SpinnerProgress для non-TTY",
			total:        0,
			expectedType: "*progress.SpinnerProgress",
		},
		{
			name:         "Обычный случай возвращает NonTTYProgress",
			total:        100,
			expectedType: "*progress.NonTTYProgress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сохраняем и очищаем переменные окружения
			oldShow := os.Getenv("BR_SHOW_PROGRESS")
			oldFormat := os.Getenv("BR_OUTPUT_FORMAT")
			oldProgress := os.Getenv("BR_PROGRESS_STREAM")
			t.Cleanup(func() {
				_ = os.Setenv("BR_SHOW_PROGRESS", oldShow)
				_ = os.Setenv("BR_OUTPUT_FORMAT", oldFormat)
				_ = os.Setenv("BR_PROGRESS_STREAM", oldProgress)
			})

			if tt.envShow != "" {
				_ = os.Setenv("BR_SHOW_PROGRESS", tt.envShow)
			} else {
				_ = os.Unsetenv("BR_SHOW_PROGRESS")
			}
			if tt.envFormat != "" {
				_ = os.Setenv("BR_OUTPUT_FORMAT", tt.envFormat)
			} else {
				_ = os.Unsetenv("BR_OUTPUT_FORMAT")
			}
			if tt.envProgress != "" {
				_ = os.Setenv("BR_PROGRESS_STREAM", tt.envProgress)
			} else {
				_ = os.Unsetenv("BR_PROGRESS_STREAM")
			}

			var buf bytes.Buffer
			opts := Options{
				Total:  tt.total,
				Output: &buf,
			}

			p := New(opts)
			gotType := getTypeName(p)
			if gotType != tt.expectedType {
				t.Errorf("New() returned %s, want %s", gotType, tt.expectedType)
			}
		})
	}
}

// getTypeName возвращает имя типа для сравнения в тестах.
func getTypeName(p Progress) string {
	switch p.(type) {
	case *NoopProgress:
		return "*progress.NoopProgress"
	case *JSONProgress:
		return "*progress.JSONProgress"
	case *SpinnerProgress:
		return "*progress.SpinnerProgress"
	case *TTYProgress:
		return "*progress.TTYProgress"
	case *NonTTYProgress:
		return "*progress.NonTTYProgress"
	default:
		return "unknown"
	}
}

// TestNoopProgressDoesNothing проверяет, что NoopProgress ничего не делает.
// MEDIUM-6 fix: тест проверяет что методы вызываются без паники и не имеют побочных эффектов.
func TestNoopProgressDoesNothing(t *testing.T) {
	p := &NoopProgress{}

	// Проверяем что все методы вызываются без паники
	t.Run("Start не паникует", func(t *testing.T) {
		p.Start("Тест")
	})

	t.Run("Update не паникует", func(t *testing.T) {
		p.Update(50, "Тест")
	})

	t.Run("SetTotal не паникует", func(t *testing.T) {
		p.SetTotal(100)
	})

	t.Run("Finish не паникует", func(t *testing.T) {
		p.Finish()
	})

	// Проверяем что несколько вызовов подряд безопасны
	t.Run("Множественные вызовы безопасны", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			p.Start("Тест")
			p.Update(int64(i*10), "Обработка")
			p.SetTotal(100)
			p.Finish()
		}
	})
}

// TestTTYProgressETACalculation проверяет корректность расчёта ETA (MEDIUM-3 fix).
func TestTTYProgressETACalculation(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Total:            100,
		Output:           &buf,
		ShowETA:          true,
		ThrottleInterval: 0,
	}

	p := NewTTYProgress(opts)
	p.Start("Тест")

	// Симулируем ситуацию: прошло 1 секунда, выполнено 50% работы
	// ETA должен быть примерно 1 секунда (50% осталось, скорость 50%/сек)
	p.startTime = time.Now().Add(-1 * time.Second) // "прошла" 1 секунда
	p.Update(50, "Тест")

	output := buf.String()

	// ETA должен быть в диапазоне 0-2 секунды (с погрешностью на выполнение теста)
	// Проверяем что ETA содержит "1s" или "<1s" или "2s"
	hasValidETA := strings.Contains(output, "ETA: 1s") ||
		strings.Contains(output, "ETA: <1s") ||
		strings.Contains(output, "ETA: 2s") ||
		strings.Contains(output, "ETA: 0s")

	if !hasValidETA {
		t.Errorf("ETA should be around 1s for 50%% progress after 1s elapsed, got: %s", output)
	}
}

// TestTTYProgressRenderBarZeroPercent проверяет что при 0%% bar пустой (MEDIUM-1 fix).
func TestTTYProgressRenderBarZeroPercent(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Total:            100,
		Output:           &buf,
		ShowETA:          false,
		ThrottleInterval: 0,
	}

	p := NewTTYProgress(opts)
	p.Start("Тест")
	p.Update(0, "")

	output := buf.String()

	// При 0% не должно быть стрелки '>' — только пробелы внутри скобок
	// Ожидаем: "[                              ]" (30 пробелов)
	if strings.Contains(output, "[>") {
		t.Errorf("At 0%% progress, bar should not have '>' arrow, got: %s", output)
	}
}

// TestFormatDuration проверяет форматирование длительности.
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "Секунды",
			duration: 45 * time.Second,
			expected: "45s",
		},
		{
			name:     "Минуты без секунд",
			duration: 2 * time.Minute,
			expected: "2m",
		},
		{
			name:     "Минуты с секундами",
			duration: 2*time.Minute + 30*time.Second,
			expected: "2m 30s",
		},
		{
			name:     "Отрицательная длительность (MEDIUM-2 fix)",
			duration: -5 * time.Second,
			expected: "0s",
		},
		{
			name:     "Нулевая длительность",
			duration: 0,
			expected: "0s",
		},
		// M-2 fix (Review #9): тесты для поддержки часов
		{
			name:     "Ровно 1 час",
			duration: 1 * time.Hour,
			expected: "1h",
		},
		{
			name:     "1 час и минуты",
			duration: 1*time.Hour + 7*time.Minute,
			expected: "1h 7m",
		},
		{
			name:     "1 час, минуты и секунды",
			duration: 1*time.Hour + 7*time.Minute + 30*time.Second,
			expected: "1h 7m 30s",
		},
		{
			name:     "1 час и секунды (без минут)",
			duration: 1*time.Hour + 45*time.Second,
			expected: "1h 45s",
		},
		{
			name:     "Несколько часов",
			duration: 3*time.Hour + 15*time.Minute + 20*time.Second,
			expected: "3h 15m 20s",
		},
		{
			name:     "24+ часов (долгая операция)",
			duration: 25*time.Hour + 30*time.Minute,
			expected: "25h 30m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatDuration(%v) = %s, want %s", tt.duration, result, tt.expected)
			}
		})
	}
}
