package dbrestorehandler

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
)

// buildPlan создаёт план операций для предпросмотра.
// Используется в dry-run, plan-only и verbose режимах.
// Story 7.3: извлечено из executeDryRun для переиспользования.
func (h *DbRestoreHandler) buildPlan(cfg *config.Config, srcDB, srcServer, dstServer string) *output.DryRunPlan {
	timeout := h.getDryRunTimeout()

	steps := []output.PlanStep{
		{
			Order:     1,
			Operation: "Проверка production флага",
			Parameters: map[string]any{
				"database":      cfg.InfobaseName,
				"is_production": false,
			},
			ExpectedChanges: []string{"Нет изменений — только валидация"},
		},
		{
			Order:     2,
			Operation: "Подключение к MSSQL серверу",
			Parameters: map[string]any{
				"server":   dstServer,
				"database": "master",
			},
			ExpectedChanges: []string{"Установка соединения с сервером"},
		},
		{
			Order:     3,
			Operation: "Восстановление базы данных",
			Parameters: map[string]any{
				"src_server":   srcServer,
				"src_db":       srcDB,
				"dst_server":   dstServer,
				"dst_db":       cfg.InfobaseName,
				"timeout":      timeout.String(),
				"auto_timeout": h.getDryRunAutoTimeoutInfo(cfg),
			},
			ExpectedChanges: []string{
				fmt.Sprintf("База %s будет восстановлена из %s/%s", cfg.InfobaseName, srcServer, srcDB),
				"Все данные в целевой базе будут перезаписаны",
			},
		},
	}

	return dryrun.BuildPlanWithSummary(
		constants.ActNRDbrestore,
		steps,
		fmt.Sprintf("Восстановление %s/%s → %s/%s", srcServer, srcDB, dstServer, cfg.InfobaseName),
	)
}

// executeDryRun выполняет dry-run режим для команды nr-dbrestore.
// AC-1: Возвращает план действий БЕЗ выполнения.
// AC-2: План содержит операции, параметры, ожидаемые изменения.
// AC-8: НЕ выполняются SQL запросы.
func (h *DbRestoreHandler) executeDryRun(
	cfg *config.Config,
	srcDB, srcServer, dstServer string,
	format, traceID string,
	start time.Time,
	log *slog.Logger,
) error {
	log.Info("Построение плана восстановления (dry-run)")
	plan := h.buildPlan(cfg, srcDB, srcServer, dstServer)
	return output.WriteDryRunResult(os.Stdout, format, constants.ActNRDbrestore, traceID, constants.APIVersion, start, plan)
}

// getDryRunTimeout возвращает таймаут для dry-run плана.
// В dry-run режиме не подключаемся к БД, поэтому используем:
// 1. BR_TIMEOUT_MIN если задан
// 2. minTimeout (5 минут) как fallback
func (h *DbRestoreHandler) getDryRunTimeout() time.Duration {
	if timeoutMinStr := os.Getenv("BR_TIMEOUT_MIN"); timeoutMinStr != "" {
		if timeoutMin, err := strconv.Atoi(timeoutMinStr); err == nil && timeoutMin > 0 {
			return time.Duration(timeoutMin) * time.Minute
		}
	}
	return minTimeout
}

// getDryRunAutoTimeoutInfo возвращает информацию об auto-timeout для dry-run плана.
func (h *DbRestoreHandler) getDryRunAutoTimeoutInfo(cfg *config.Config) string {
	if timeoutMinStr := os.Getenv("BR_TIMEOUT_MIN"); timeoutMinStr != "" {
		return "отключён (BR_TIMEOUT_MIN задан явно)"
	}
	if h.isAutoTimeoutEnabled(cfg) {
		return "включён (будет рассчитан по статистике)"
	}
	return "отключён (BR_AUTO_TIMEOUT=false)"
}

// isAutoTimeoutEnabled проверяет включён ли auto-timeout.
// Логика: env переменная имеет приоритет над AppConfig.
// Допустимые значения для включения: "true", "1".
// По умолчанию (если не задано) — включён.
func (h *DbRestoreHandler) isAutoTimeoutEnabled(cfg *config.Config) bool {
	autoTimeoutStr := os.Getenv("BR_AUTO_TIMEOUT")
	if autoTimeoutStr != "" {
		return autoTimeoutStr == "true" || autoTimeoutStr == "1"
	}
	// Env переменная не задана — проверяем AppConfig, иначе true по умолчанию
	if cfg != nil && cfg.AppConfig != nil {
		return cfg.AppConfig.Dbrestore.Autotimeout
	}
	return true // по умолчанию включён
}
