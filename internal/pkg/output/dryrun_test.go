package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDryRunPlan_WriteText(t *testing.T) {
	tests := []struct {
		name     string
		plan     *DryRunPlan
		contains []string
	}{
		{
			name: "пустой план без шагов",
			plan: &DryRunPlan{
				Command:          "test-command",
				Steps:            []PlanStep{},
				ValidationPassed: true,
			},
			contains: []string{
				"=== DRY RUN ===",
				"Команда: test-command",
				"Валидация: ✅ Пройдена",
				"План выполнения:",
				"=== END DRY RUN ===",
			},
		},
		{
			name: "план с nil Steps",
			plan: &DryRunPlan{
				Command:          "test-command",
				Steps:            nil,
				ValidationPassed: false,
			},
			contains: []string{
				"=== DRY RUN ===",
				"Команда: test-command",
				"Валидация: ❌ Не пройдена",
				"=== END DRY RUN ===",
			},
		},
		{
			name: "базовый план с шагами",
			plan: &DryRunPlan{
				Command: "test-command",
				Steps: []PlanStep{
					{
						Order:     1,
						Operation: "Тестовая операция",
						Parameters: map[string]any{
							"param1": "value1",
						},
						ExpectedChanges: []string{"Изменение 1"},
					},
				},
				ValidationPassed: true,
			},
			contains: []string{
				"=== DRY RUN ===",
				"Команда: test-command",
				"Валидация: ✅ Пройдена",
				"1. Тестовая операция",
				"param1: value1",
				"Ожидаемые изменения:",
				"- Изменение 1",
				"=== END DRY RUN ===",
			},
		},
		{
			name: "план с пропущенным шагом",
			plan: &DryRunPlan{
				Command: "test-command",
				Steps: []PlanStep{
					{
						Order:      1,
						Operation:  "Пропущенная операция",
						Skipped:    true,
						SkipReason: "auto-deps отключён",
					},
				},
				ValidationPassed: true,
			},
			contains: []string{
				"=== DRY RUN ===",
				"1. [SKIP] Пропущенная операция — auto-deps отключён",
				"=== END DRY RUN ===",
			},
		},
		{
			name: "план с summary",
			plan: &DryRunPlan{
				Command:          "test-command",
				Steps:            []PlanStep{},
				Summary:          "Тестовое резюме операции",
				ValidationPassed: true,
			},
			contains: []string{
				"=== DRY RUN ===",
				"Итого: Тестовое резюме операции",
				"=== END DRY RUN ===",
			},
		},
		{
			name: "план с непройденной валидацией",
			plan: &DryRunPlan{
				Command:          "test-command",
				Steps:            []PlanStep{},
				ValidationPassed: false,
			},
			contains: []string{
				"=== DRY RUN ===",
				"Валидация: ❌ Не пройдена",
				"=== END DRY RUN ===",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.plan.WriteText(&buf)
			require.NoError(t, err)

			output := buf.String()
			for _, expected := range tt.contains {
				assert.Contains(t, output, expected, "Ожидалось наличие: %s", expected)
			}
		})
	}
}

func TestBoolToStatus(t *testing.T) {
	tests := []struct {
		name string
		b    bool
		want string
	}{
		{
			name: "true возвращает ✅ Пройдена",
			b:    true,
			want: "✅ Пройдена",
		},
		{
			name: "false возвращает ❌ Не пройдена",
			b:    false,
			want: "❌ Не пройдена",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := boolToStatus(tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestDryRunPlan_WriteText_ManySteps проверяет план с большим количеством шагов.
func TestDryRunPlan_WriteText_ManySteps(t *testing.T) {
	// Создаём план с 20 шагами
	var steps []PlanStep
	for i := 1; i <= 20; i++ {
		steps = append(steps, PlanStep{
			Order:     i,
			Operation: "Операция " + string(rune('A'+i-1)),
			Parameters: map[string]any{
				"step_number": i,
				"description": "Тестовый параметр",
			},
			ExpectedChanges: []string{
				"Изменение для шага",
			},
		})
	}

	plan := &DryRunPlan{
		Command:          "complex-command",
		Steps:            steps,
		Summary:          "Комплексная операция с 20 шагами",
		ValidationPassed: true,
	}

	var buf bytes.Buffer
	err := plan.WriteText(&buf)
	require.NoError(t, err)

	output := buf.String()

	// Проверяем что все шаги присутствуют
	for i := 1; i <= 20; i++ {
		assert.Contains(t, output, "Операция "+string(rune('A'+i-1)))
	}

	// Проверяем общую структуру
	assert.Contains(t, output, "=== DRY RUN ===")
	assert.Contains(t, output, "=== END DRY RUN ===")
	assert.Contains(t, output, "Комплексная операция с 20 шагами")
}

// TestSanitizeValue проверяет экранирование специальных символов.
// H-2 fix: добавлены тесты для sanitizeValue().
func TestSanitizeValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "обычная строка без изменений",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "строка с переносом заменяется на пробел",
			input:    "hello\nworld",
			expected: "hello world",
		},
		{
			name:     "строка с табом заменяется на пробел",
			input:    "hello\tworld",
			expected: "hello world",
		},
		{
			name:     "ANSI escape sequence для красного цвета полностью удаляется",
			input:    "\x1b[31mred text\x1b[0m",
			expected: "red text",
		},
		{
			name:     "ANSI escape sequence для жирного текста",
			input:    "\x1b[1mbold\x1b[0m",
			expected: "bold",
		},
		{
			name:     "множественные ANSI escape sequences",
			input:    "\x1b[31m\x1b[1mred bold\x1b[0m\x1b[0m",
			expected: "red bold",
		},
		{
			name:     "ANSI с параметрами через точку с запятой",
			input:    "\x1b[38;5;196mcolored\x1b[0m",
			expected: "colored",
		},
		{
			name:     "управляющие символы удаляются",
			input:    "hello\x00\x01\x02world",
			expected: "helloworld",
		},
		{
			name:     "DEL символ (127) удаляется",
			input:    "hello\x7fworld",
			expected: "helloworld",
		},
		{
			name:     "число конвертируется в строку",
			input:    12345,
			expected: "12345",
		},
		{
			name:     "boolean конвертируется в строку",
			input:    true,
			expected: "true",
		},
		{
			name:     "пустая строка остаётся пустой",
			input:    "",
			expected: "",
		},
		{
			name:     "только ANSI escape удаляется полностью",
			input:    "\x1b[31m",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeValue(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDryRunPlan_WritePlanText(t *testing.T) {
	tests := []struct {
		name        string
		plan        *DryRunPlan
		contains    []string
		notContains []string
	}{
		{
			name: "заголовок OPERATION PLAN вместо DRY RUN",
			plan: &DryRunPlan{
				Command:          "test-command",
				Steps:            []PlanStep{},
				ValidationPassed: true,
			},
			contains: []string{
				"=== OPERATION PLAN ===",
				"Команда: test-command",
				"Валидация: ✅ Пройдена",
				"План выполнения:",
				"=== END OPERATION PLAN ===",
			},
			notContains: []string{
				"=== DRY RUN ===",
				"=== END DRY RUN ===",
			},
		},
		{
			name: "план с шагами и summary",
			plan: &DryRunPlan{
				Command: "nr-dbupdate",
				Steps: []PlanStep{
					{
						Order:     1,
						Operation: "Обновление конфигурации БД",
						Parameters: map[string]any{
							"database": "TestDB",
							"timeout":  "5m",
						},
						ExpectedChanges: []string{"Структура БД будет обновлена"},
					},
				},
				Summary:          "Обновление TestDB",
				ValidationPassed: true,
			},
			contains: []string{
				"=== OPERATION PLAN ===",
				"1. Обновление конфигурации БД",
				"database: TestDB",
				"timeout: 5m",
				"Ожидаемые изменения:",
				"- Структура БД будет обновлена",
				"Итого: Обновление TestDB",
				"=== END OPERATION PLAN ===",
			},
			notContains: []string{
				"=== DRY RUN ===",
			},
		},
		{
			name: "план с пропущенным шагом",
			plan: &DryRunPlan{
				Command: "test-command",
				Steps: []PlanStep{
					{
						Order:      1,
						Operation:  "Пропущенная операция",
						Skipped:    true,
						SkipReason: "не требуется",
					},
				},
				ValidationPassed: true,
			},
			contains: []string{
				"=== OPERATION PLAN ===",
				"1. [SKIP] Пропущенная операция — не требуется",
				"=== END OPERATION PLAN ===",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.plan.WritePlanText(&buf)
			require.NoError(t, err)

			output := buf.String()
			for _, expected := range tt.contains {
				assert.Contains(t, output, expected, "Ожидалось наличие: %s", expected)
			}
			for _, notExpected := range tt.notContains {
				assert.NotContains(t, output, notExpected, "Не ожидалось наличие: %s", notExpected)
			}
		})
	}
}

func TestDryRunPlan_WriteText_MultipleSteps(t *testing.T) {
	plan := &DryRunPlan{
		Command: "nr-dbrestore",
		Steps: []PlanStep{
			{
				Order:     1,
				Operation: "Проверка production флага",
				Parameters: map[string]any{
					"database":      "TestDB",
					"is_production": false,
				},
				ExpectedChanges: []string{"Нет изменений — только валидация"},
			},
			{
				Order:     2,
				Operation: "Подключение к MSSQL серверу",
				Parameters: map[string]any{
					"server":   "test-server",
					"database": "master",
				},
				ExpectedChanges: []string{"Установка соединения с сервером"},
			},
			{
				Order:     3,
				Operation: "Восстановление базы данных",
				Parameters: map[string]any{
					"src_server": "prod-server",
					"src_db":     "ProdDB",
					"dst_server": "test-server",
					"dst_db":     "TestDB",
					"timeout":    "5m0s",
				},
				ExpectedChanges: []string{
					"База TestDB будет восстановлена из prod-server/ProdDB",
					"Все данные в целевой базе будут перезаписаны",
				},
			},
		},
		Summary:          "Восстановление prod-server/ProdDB → test-server/TestDB",
		ValidationPassed: true,
	}

	var buf bytes.Buffer
	err := plan.WriteText(&buf)
	require.NoError(t, err)

	output := buf.String()

	// Проверяем структуру вывода
	assert.Contains(t, output, "=== DRY RUN ===")
	assert.Contains(t, output, "Команда: nr-dbrestore")
	assert.Contains(t, output, "Валидация: ✅ Пройдена")
	assert.Contains(t, output, "План выполнения:")
	assert.Contains(t, output, "1. Проверка production флага")
	assert.Contains(t, output, "2. Подключение к MSSQL серверу")
	assert.Contains(t, output, "3. Восстановление базы данных")
	assert.Contains(t, output, "Итого: Восстановление prod-server/ProdDB → test-server/TestDB")
	assert.Contains(t, output, "=== END DRY RUN ===")
}
