// Package dryrun предоставляет функции для работы с dry-run режимом.
// В dry-run режиме команды возвращают план действий без реального выполнения.
package dryrun

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/output"
)

// IsDryRun проверяет включён ли dry-run режим.
// Возвращает true если переменная окружения BR_DRY_RUN равна "true" или "1".
// AC-1: При BR_DRY_RUN=true команды возвращают план действий БЕЗ выполнения.
// L-3 fix: полностью case-insensitive проверка через strings.EqualFold.
func IsDryRun() bool {
	val := os.Getenv(constants.EnvDryRun)
	return strings.EqualFold(val, "true") || val == "1"
}

// IsPlanOnly проверяет включён ли plan-only режим.
// Возвращает true если переменная окружения BR_PLAN_ONLY равна "true" или "1".
// Story 7.3 AC-1: BR_PLAN_ONLY=true активирует отображение плана без выполнения.
func IsPlanOnly() bool {
	val := os.Getenv(constants.EnvPlanOnly)
	return strings.EqualFold(val, "true") || val == "1"
}

// IsVerbose проверяет включён ли verbose режим.
// Возвращает true если переменная окружения BR_VERBOSE равна "true" или "1".
// Story 7.3 AC-4: BR_VERBOSE=true активирует предпросмотр плана перед выполнением.
func IsVerbose() bool {
	val := os.Getenv(constants.EnvVerbose)
	return strings.EqualFold(val, "true") || val == "1"
}

// EffectiveMode возвращает текущий приоритетный режим выполнения.
// Приоритет: "dry-run" > "plan-only" > "verbose" > "normal".
// Story 7.3 AC-11: BR_DRY_RUN перекрывает BR_PLAN_ONLY и BR_VERBOSE.
func EffectiveMode() string {
	if IsDryRun() {
		return "dry-run"
	}
	if IsPlanOnly() {
		return "plan-only"
	}
	if IsVerbose() {
		return "verbose"
	}
	return "normal"
}

// WritePlanOnlyUnsupported выводит предупреждение для команд без поддержки плана.
// Story 7.3 AC-8: plan-only для команд без dry-run выводит warning и exit code = 0.
// Review #32: всегда возвращает nil — ошибка записи в stdout не должна приводить
// к ненулевому exit code для информационного сообщения.
func WritePlanOnlyUnsupported(w io.Writer, command string) error {
	fmt.Fprintf(w, "Команда %s не поддерживает отображение плана операций\n", command) //nolint:errcheck // best-effort output
	return nil
}

// BuildPlan создаёт план операций для dry-run режима.
// AC-2: Plan содержит операции, параметры, ожидаемые изменения.
func BuildPlan(command string, steps []output.PlanStep) *output.DryRunPlan {
	return &output.DryRunPlan{
		Command:          command,
		Steps:            steps,
		ValidationPassed: true,
	}
}

// BuildPlanWithSummary создаёт план операций с кратким описанием.
func BuildPlanWithSummary(command string, steps []output.PlanStep, summary string) *output.DryRunPlan {
	return &output.DryRunPlan{
		Command:          command,
		Steps:            steps,
		Summary:          summary,
		ValidationPassed: true,
	}
}

// passwordRegexes — скомпилированные регулярные выражения для поиска паролей.
// H-2 fix: поддерживаем несколько форматов паролей для полного покрытия.
// H-4 fix: /P и -P работают и в начале строки (опциональный пробел перед ними).
// passwordRegexes is effectively constant (compiled once, never modified).
// Cannot be const because Go does not support const slices.
var passwordRegexes = []*regexp.Regexp{
	// Формат: /P password или /P "password" или /P 'password' (может быть в начале строки)
	regexp.MustCompile(`(?i)(^|[ ])(/P )("[^"]*"|'[^']*'|[^\s]+)`),
	// Формат: /P=password (без пробела, может быть в начале строки)
	regexp.MustCompile(`(?i)(^|[ ])(/P=)("[^"]*"|'[^']*'|[^\s]+)`),
	// Формат: -P password или -P "password" (может быть в начале строки)
	regexp.MustCompile(`(?i)(^|[ ])(-P )("[^"]*"|'[^']*'|[^\s]+)`),
	// Формат: -P=password (может быть в начале строки)
	regexp.MustCompile(`(?i)(^|[ ])(-P=)("[^"]*"|'[^']*'|[^\s]+)`),
	// Формат: password= (generic для connection strings)
	regexp.MustCompile(`(?i)(password=)([^;]+)`),
	// Review #34 fix: pwd= формат (сокращённый вариант в connection strings)
	regexp.MustCompile(`(?i)(pwd=)([^;]+)`),
}

// MaskPassword маскирует пароль в connect string.
// SECURITY: пароли НЕ должны появляться в dry-run плане.
// H-2 fix: обрабатывает расширенный набор форматов:
// - /P password, /P "password", /P 'password'
// - /P=password (без пробела)
// - -P password, -P=password (дефис вместо слэша)
// - password=value (generic connection string формат)
// H-4 fix: работает и когда /P или -P в начале строки.
// Формат: /S server\base /N user /P password → /S server\base /N user /P ***
func MaskPassword(connectString string) string {
	result := connectString
	// Для первых 4 regex: группа 1 = prefix (пробел или начало), группа 2 = /P или -P, группа 3 = пароль
	// Заменяем на $1$2*** (сохраняем prefix и флаг, маскируем пароль)
	for i, regex := range passwordRegexes {
		if i < 4 {
			// /P и -P форматы: 3 группы
			result = regex.ReplaceAllString(result, "$1$2***")
		} else {
			// password= формат: 2 группы
			result = regex.ReplaceAllString(result, "$1***")
		}
	}
	return result
}
