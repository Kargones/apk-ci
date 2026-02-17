package createtempdbhandler

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
)

// buildPlan создаёт план операций для предпросмотра.
// Используется в dry-run, plan-only и verbose режимах.
// Story 7.3: извлечено из executeDryRun для переиспользования.
func (h *CreateTempDbHandler) buildPlan(
	cfg *config.Config,
	dbPath string,
	extensions []string,
	timeout time.Duration,
	ttlHours int,
) *output.DryRunPlan {
	extensionsStr := extensionsOrNone(extensions)

	steps := []output.PlanStep{
		{
			Order:     1,
			Operation: "Валидация конфигурации",
			Parameters: map[string]any{
				"ibcmd_path": cfg.AppConfig.Paths.BinIbcmd,
			},
			ExpectedChanges: []string{"Нет изменений — только валидация"},
		},
		{
			Order:     2,
			Operation: "Генерация пути к временной базе",
			Parameters: map[string]any{
				"db_path": dbPath,
			},
			ExpectedChanges: []string{
				fmt.Sprintf("Будет создана директория: %s", dbPath),
			},
		},
		{
			Order:     3,
			Operation: "Создание базы данных",
			Parameters: map[string]any{
				"db_path":    dbPath,
				"extensions": extensionsStr,
				"timeout":    timeout.String(),
			},
			ExpectedChanges: []string{
				"Будет создана пустая информационная база",
			},
		},
	}

	if len(extensions) > 0 {
		steps = append(steps, output.PlanStep{
			Order:     4,
			Operation: "Добавление расширений",
			Parameters: map[string]any{
				"extensions": extensionsStr,
			},
			ExpectedChanges: []string{
				fmt.Sprintf("Расширения %s будут добавлены в базу", extensionsStr),
			},
		})
	}

	if ttlHours > 0 {
		steps = append(steps, output.PlanStep{
			Order:     len(steps) + 1,
			Operation: "Создание TTL metadata",
			Parameters: map[string]any{
				"ttl_hours": ttlHours,
				"ttl_file":  dbPath + ".ttl",
			},
			ExpectedChanges: []string{
				fmt.Sprintf("База будет удалена через %d часов", ttlHours),
			},
		})
	}

	summary := fmt.Sprintf("Создание временной БД: %s", dbPath)
	if len(extensions) > 0 {
		summary += fmt.Sprintf(" с расширениями: %s", extensionsStr)
	}

	return dryrun.BuildPlanWithSummary(constants.ActNRCreateTempDb, steps, summary)
}

// executeDryRun выполняет dry-run режим для команды nr-create-temp-db.
// AC-1: Возвращает план действий БЕЗ выполнения.
// AC-2: План содержит операции, параметры, ожидаемые изменения.
// AC-8: НЕ вызывается client.CreateTempDB().
func (h *CreateTempDbHandler) executeDryRun(
	cfg *config.Config,
	dbPath string,
	extensions []string,
	timeout time.Duration,
	ttlHours int,
	format, traceID string,
	start time.Time,
) error {
	plan := h.buildPlan(cfg, dbPath, extensions, timeout, ttlHours)
	return output.WriteDryRunResult(os.Stdout, format, constants.ActNRCreateTempDb, traceID, constants.APIVersion, start, plan)
}

// extensionsOrNone возвращает строку расширений или "нет" если пусто.
func extensionsOrNone(extensions []string) string {
	if len(extensions) == 0 {
		return "нет"
	}
	return strings.Join(extensions, ", ")
}
