package dbupdatehandler

import (
	"fmt"
	"os"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
)

// buildPlan создаёт план операций для предпросмотра.
// Используется в dry-run, plan-only и verbose режимах.
// Story 7.3: извлечено из executeDryRun для переиспользования.
func (h *DbUpdateHandler) buildPlan(
	cfg *config.Config,
	dbInfo *config.DatabaseInfo,
	connectString string,
	extension string,
	timeout time.Duration,
) *output.DryRunPlan {
	// Определяем режим auto-deps
	autoDeps := os.Getenv("BR_AUTO_DEPS") == "true"

	// Маскируем пароль в connect string (SECURITY!)
	maskedConnectString := dryrun.MaskPassword(connectString)

	steps := []output.PlanStep{
		{
			Order:     1,
			Operation: "Валидация конфигурации",
			Parameters: map[string]any{
				"infobase_name": cfg.InfobaseName,
				"extension":     valueOrNone(extension),
			},
			ExpectedChanges: []string{"Нет изменений — только валидация"},
		},
	}

	if autoDeps {
		steps = append(steps, output.PlanStep{
			Order:     2,
			Operation: "Включение сервисного режима (auto-deps)",
			Parameters: map[string]any{
				"server":   getServerFromDbInfo(dbInfo),
				"database": cfg.InfobaseName,
			},
			ExpectedChanges: []string{
				"Сервисный режим будет включён",
				"Пользователи будут отключены от базы",
			},
		})
	} else {
		steps = append(steps, output.PlanStep{
			Order:      2,
			Operation:  "Сервисный режим",
			Skipped:    true,
			SkipReason: "BR_AUTO_DEPS не включён",
		})
	}

	updateStep := output.PlanStep{
		Order:     3,
		Operation: "Обновление структуры базы данных",
		Parameters: map[string]any{
			"connect_string": maskedConnectString,
			"extension":      valueOrNone(extension),
			"timeout":        timeout.String(),
			"bin_1cv8":       cfg.AppConfig.Paths.Bin1cv8,
		},
		ExpectedChanges: []string{
			"Структура базы данных будет обновлена по конфигурации",
		},
	}
	if extension != "" {
		updateStep.ExpectedChanges = append(updateStep.ExpectedChanges,
			fmt.Sprintf("Расширение '%s' будет применено", extension),
			"Второй проход обновления для расширения",
		)
	}
	steps = append(steps, updateStep)

	if autoDeps {
		steps = append(steps, output.PlanStep{
			Order:     4,
			Operation: "Отключение сервисного режима (auto-deps)",
			Parameters: map[string]any{
				"server":   getServerFromDbInfo(dbInfo),
				"database": cfg.InfobaseName,
			},
			ExpectedChanges: []string{
				"Сервисный режим будет отключён",
				"Пользователи смогут подключиться к базе",
			},
		})
	}

	summary := fmt.Sprintf("Обновление %s", cfg.InfobaseName)
	if extension != "" {
		summary += fmt.Sprintf(" (расширение: %s)", extension)
	}

	return dryrun.BuildPlanWithSummary(constants.ActNRDbupdate, steps, summary)
}

// executeDryRun выполняет dry-run режим для команды nr-dbupdate.
// AC-1: Возвращает план действий БЕЗ выполнения.
// AC-2: План содержит операции, параметры, ожидаемые изменения.
// AC-8: НЕ вызываются 1cv8/ibcmd, RAC операции.
func (h *DbUpdateHandler) executeDryRun(
	cfg *config.Config,
	dbInfo *config.DatabaseInfo,
	connectString string,
	extension string,
	timeout time.Duration,
	format, traceID string,
	start time.Time,
) error {
	plan := h.buildPlan(cfg, dbInfo, connectString, extension, timeout)
	return output.WriteDryRunResult(os.Stdout, format, constants.ActNRDbupdate, traceID, constants.APIVersion, start, plan)
}
