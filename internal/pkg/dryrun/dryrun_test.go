package dryrun

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kargones/apk-ci/internal/pkg/output"
)

func TestIsDryRun(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     bool
	}{
		{
			name:     "BR_DRY_RUN=true возвращает true",
			envValue: "true",
			want:     true,
		},
		{
			name:     "BR_DRY_RUN=false возвращает false",
			envValue: "false",
			want:     false,
		},
		{
			name:     "BR_DRY_RUN=1 возвращает true (консистентно с другими флагами)",
			envValue: "1",
			want:     true,
		},
		{
			name:     "BR_DRY_RUN пустой возвращает false",
			envValue: "",
			want:     false,
		},
		{
			name:     "BR_DRY_RUN=TRUE (uppercase) возвращает true",
			envValue: "TRUE",
			want:     true,
		},
		{
			name:     "BR_DRY_RUN=True (mixed case) возвращает true",
			envValue: "True",
			want:     true,
		},
		{
			name:     "BR_DRY_RUN=yes возвращает false (только true/TRUE/True/1)",
			envValue: "yes",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BR_DRY_RUN", tt.envValue)
			got := IsDryRun()
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestIsDryRun_Unset проверяет возврат false для unset переменной BR_DRY_RUN.
func TestIsDryRun_Unset(t *testing.T) {
	// Сохраняем текущее значение
	oldValue, wasSet := os.LookupEnv("BR_DRY_RUN")

	// Удаляем переменную
	_ = os.Unsetenv("BR_DRY_RUN")

	// Восстанавливаем после теста
	t.Cleanup(func() {
		if wasSet {
			_ = os.Setenv("BR_DRY_RUN", oldValue)
		} else {
			_ = os.Unsetenv("BR_DRY_RUN")
		}
	})

	// Проверяем что возвращает false для unset переменной
	got := IsDryRun()
	assert.False(t, got, "IsDryRun() должен возвращать false когда BR_DRY_RUN не установлена")
}

func TestBuildPlan(t *testing.T) {
	steps := []output.PlanStep{
		{
			Order:     1,
			Operation: "Тестовая операция",
			Parameters: map[string]any{
				"param1": "value1",
			},
			ExpectedChanges: []string{"Изменение 1"},
		},
	}

	plan := BuildPlan("test-command", steps)

	require.NotNil(t, plan)
	assert.Equal(t, "test-command", plan.Command)
	assert.Len(t, plan.Steps, 1)
	assert.True(t, plan.ValidationPassed)
	assert.Empty(t, plan.Summary)
}

func TestBuildPlanWithSummary(t *testing.T) {
	steps := []output.PlanStep{
		{
			Order:     1,
			Operation: "Операция",
		},
	}

	plan := BuildPlanWithSummary("test-command", steps, "Тестовое резюме")

	require.NotNil(t, plan)
	assert.Equal(t, "test-command", plan.Command)
	assert.Equal(t, "Тестовое резюме", plan.Summary)
	assert.True(t, plan.ValidationPassed)
}

func TestMaskPassword(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "пароль в конце строки",
			input:    "/S server\\base /N user /P secret123",
			expected: "/S server\\base /N user /P ***",
		},
		{
			name:     "пароль в середине строки",
			input:    "/S server\\base /N user /P secret123 /UC uc",
			expected: "/S server\\base /N user /P *** /UC uc",
		},
		{
			name:     "без пароля",
			input:    "/S server\\base /N user",
			expected: "/S server\\base /N user",
		},
		{
			name:     "пустая строка",
			input:    "",
			expected: "",
		},
		{
			name:     "только пароль",
			input:    " /P password",
			expected: " /P ***",
		},
		{
			name:     "пароль с пробелами после",
			input:    "/S srv /P pass /N usr /UC code",
			expected: "/S srv /P *** /N usr /UC code",
		},
		{
			name:     "lowercase /p (case-insensitive)",
			input:    "/S server\\base /N user /p secret123",
			expected: "/S server\\base /N user /p ***",
		},
		{
			name:     "несколько паролей в строке",
			input:    "/S srv1 /P pass1 /S srv2 /P pass2",
			expected: "/S srv1 /P *** /S srv2 /P ***",
		},
		{
			name:     "смешанный регистр /P и /p",
			input:    "/S srv /P pass1 /N usr /p pass2",
			expected: "/S srv /P *** /N usr /p ***",
		},
		// Тест для пароля содержащего / (слэш)
		{
			name:     "пароль со слэшем",
			input:    "/S srv /P pass/word123 /N usr",
			expected: "/S srv /P *** /N usr",
		},
		{
			name:     "пароль только из спецсимволов",
			input:    "/S srv /P @#$%^&*()_+! /N usr",
			expected: "/S srv /P *** /N usr",
		},
		// Тесты для quoted passwords
		{
			name:     "пароль в двойных кавычках",
			input:    `/S srv /P "my secret" /N usr`,
			expected: `/S srv /P *** /N usr`,
		},
		{
			name:     "пароль в одинарных кавычках",
			input:    `/S srv /P 'my secret' /N usr`,
			expected: `/S srv /P *** /N usr`,
		},
		{
			name:     "пароль в кавычках со спецсимволами",
			input:    `/S srv /P "pass/word with spaces" /N usr`,
			expected: `/S srv /P *** /N usr`,
		},
		// H-4 fix: тесты для /P в начале строки
		{
			name:     "пароль в начале строки",
			input:    "/P secret123 /S server\\base",
			expected: "/P *** /S server\\base",
		},
		{
			name:     "-P в начале строки",
			input:    "-P secret123 -S server",
			expected: "-P *** -S server",
		},
		{
			name:     "/P= в начале строки",
			input:    "/P=secret123 /S server",
			expected: "/P=*** /S server",
		},
		// H-3/L-1 fix: тесты для password= формата (connection strings)
		{
			name:     "password= формат connection string",
			input:    "Server=srv;Password=secret123;Database=db",
			expected: "Server=srv;Password=***;Database=db",
		},
		{
			name:     "Password= с заглавной буквы",
			input:    "Server=srv;Password=MySecret;Database=db",
			expected: "Server=srv;Password=***;Database=db",
		},
		{
			name:     "PASSWORD= uppercase",
			input:    "Server=srv;PASSWORD=secret;Database=db",
			expected: "Server=srv;PASSWORD=***;Database=db",
		},
		{
			name:     "password= в конце строки",
			input:    "Server=srv;Database=db;password=secret",
			expected: "Server=srv;Database=db;password=***",
		},
		// Review #35 fix: тесты для pwd= формата (Review #34 добавил regex, но тестов не было)
		{
			name:     "pwd= формат connection string",
			input:    "Server=srv;Pwd=secret123;Database=db",
			expected: "Server=srv;Pwd=***;Database=db",
		},
		{
			name:     "pwd= lowercase",
			input:    "Server=srv;pwd=mypassword;Database=db",
			expected: "Server=srv;pwd=***;Database=db",
		},
		{
			name:     "PWD= uppercase",
			input:    "Server=srv;PWD=secret;Database=db",
			expected: "Server=srv;PWD=***;Database=db",
		},
		{
			name:     "pwd= в конце строки",
			input:    "Server=srv;Database=db;pwd=secret",
			expected: "Server=srv;Database=db;pwd=***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskPassword(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestIsPlanOnly(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     bool
	}{
		{"BR_PLAN_ONLY=true возвращает true", "true", true},
		{"BR_PLAN_ONLY=false возвращает false", "false", false},
		{"BR_PLAN_ONLY=1 возвращает true", "1", true},
		{"BR_PLAN_ONLY пустой возвращает false", "", false},
		{"BR_PLAN_ONLY=TRUE (uppercase) возвращает true", "TRUE", true},
		{"BR_PLAN_ONLY=True (mixed case) возвращает true", "True", true},
		{"BR_PLAN_ONLY=yes возвращает false", "yes", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BR_PLAN_ONLY", tt.envValue)
			got := IsPlanOnly()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsVerbose(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     bool
	}{
		{"BR_VERBOSE=true возвращает true", "true", true},
		{"BR_VERBOSE=false возвращает false", "false", false},
		{"BR_VERBOSE=1 возвращает true", "1", true},
		{"BR_VERBOSE пустой возвращает false", "", false},
		{"BR_VERBOSE=TRUE (uppercase) возвращает true", "TRUE", true},
		{"BR_VERBOSE=True (mixed case) возвращает true", "True", true},
		{"BR_VERBOSE=yes возвращает false", "yes", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BR_VERBOSE", tt.envValue)
			got := IsVerbose()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEffectiveMode(t *testing.T) {
	tests := []struct {
		name     string
		dryRun   string
		planOnly string
		verbose  string
		want     string
	}{
		{"все выключены — normal", "", "", "", "normal"},
		{"только dry-run — dry-run", "true", "", "", "dry-run"},
		{"только plan-only — plan-only", "", "true", "", "plan-only"},
		{"только verbose — verbose", "", "", "true", "verbose"},
		{"dry-run перекрывает plan-only", "true", "true", "", "dry-run"},
		{"dry-run перекрывает verbose", "true", "", "true", "dry-run"},
		{"dry-run перекрывает все", "true", "true", "true", "dry-run"},
		{"plan-only перекрывает verbose", "", "true", "true", "plan-only"},
		{"dry-run=1 перекрывает plan-only=true", "1", "true", "", "dry-run"},
		{"plan-only=1 перекрывает verbose=1", "", "1", "1", "plan-only"},
		{"dry-run=false, plan-only=false, verbose=false — normal", "false", "false", "false", "normal"},
		{"dry-run=false, plan-only=true — plan-only", "false", "true", "", "plan-only"},
		{"dry-run=false, verbose=true — verbose", "false", "", "true", "verbose"},
		{"dry-run=TRUE, plan-only=TRUE — dry-run", "TRUE", "TRUE", "", "dry-run"},
		{"plan-only=TRUE, verbose=TRUE — plan-only", "", "TRUE", "TRUE", "plan-only"},
		{"dry-run=yes (недопустимо), plan-only=true — plan-only", "yes", "true", "", "plan-only"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BR_DRY_RUN", tt.dryRun)
			t.Setenv("BR_PLAN_ONLY", tt.planOnly)
			t.Setenv("BR_VERBOSE", tt.verbose)
			got := EffectiveMode()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWritePlanOnlyUnsupported(t *testing.T) {
	var buf bytes.Buffer
	err := WritePlanOnlyUnsupported(&buf, "nr-service-mode-status")
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "nr-service-mode-status")
	assert.Contains(t, buf.String(), "не поддерживает отображение плана операций")
}
