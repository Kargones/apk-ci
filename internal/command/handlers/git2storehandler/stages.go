// Package git2storehandler — stage execution functions for git2store workflow.
package git2storehandler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
)

// executeStageValidating выполняет этап валидации (AC-2).
func (h *Git2StoreHandler) executeStageValidating(cfg *config.Config, log *slog.Logger, data *Git2StoreData) error {
	stageStart := time.Now()
	stageName := StageValidating
	log.Info(stageName + ": проверка параметров")
	data.StageCurrent = stageName

	if cfg == nil {
		return h.recordStageError(data, stageName, stageStart, "CONFIG.MISSING", "Конфигурация не указана")
	}

	if cfg.AppConfig == nil || cfg.AppConfig.Paths.Bin1cv8 == "" {
		return h.recordStageError(data, stageName, stageStart, "CONFIG.BIN1CV8_MISSING",
			"Не указан путь к 1cv8 (AppConfig.Paths.Bin1cv8)")
	}

	if cfg.TmpDir == "" {
		return h.recordStageError(data, stageName, stageStart, "CONFIG.TMPDIR_MISSING",
			"Не указана временная директория (TmpDir)")
	}

	if cfg.Owner == "" {
		return h.recordStageError(data, stageName, stageStart, "CONFIG.OWNER_MISSING",
			"Не указан владелец репозитория (Owner)")
	}

	if cfg.Repo == "" {
		return h.recordStageError(data, stageName, stageStart, "CONFIG.REPO_MISSING",
			"Не указано имя репозитория (Repo)")
	}

	if cfg.WorkDir == "" {
		return h.recordStageError(data, stageName, stageStart, "CONFIG.WORKDIR_MISSING",
			"Не указана рабочая директория (WorkDir)")
	}

	if cfg.InfobaseName == "" {
		return h.recordStageError(data, stageName, stageStart, "CONFIG.INFOBASE_MISSING",
			"Не указано имя информационной базы (BR_INFOBASE_NAME)")
	}

	log.Info("Параметры валидации пройдены",
		slog.String("owner", cfg.Owner),
		slog.String("repo", cfg.Repo),
		slog.Int("extensions_count", len(cfg.AddArray)))

	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageCreatingBackup создаёт резервную копию хранилища (AC-8).
func (h *Git2StoreHandler) executeStageCreatingBackup(cfg *config.Config, log *slog.Logger, data *Git2StoreData, storeRoot string) (string, error) {
	stageStart := time.Now()
	stageName := StageCreatingBackup
	log.Info(stageName + ": создание резервной копии хранилища")
	data.StageCurrent = stageName

	backupPath, err := h.createBackup(cfg, storeRoot)
	if err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_BACKUP", err.Error())
		return "", fmt.Errorf("ERR_GIT2STORE_BACKUP: %s", err.Error())
	}

	log.Info(stageName+": резервная копия создана", slog.String("backup_path", backupPath))
	h.recordStageSuccess(data, stageName, stageStart)
	return backupPath, nil
}

// executeStageCloning клонирует репозиторий (AC-2).
func (h *Git2StoreHandler) executeStageCloning(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData) (GitOperator, error) {
	stageStart := time.Now()
	stageName := StageCloning
	log.Info(stageName + ": клонирование репозитория")
	data.StageCurrent = stageName

	repPath, err := os.MkdirTemp(cfg.WorkDir, "s")
	if err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_CLONE", err.Error())
		return nil, fmt.Errorf("ERR_GIT2STORE_CLONE: %s", err.Error())
	}
	cfg.RepPath = repPath

	gitOp, err := h.createGit(log, cfg)
	if err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_CLONE", err.Error())
		return nil, fmt.Errorf("ERR_GIT2STORE_CLONE: %s", err.Error())
	}

	if err := gitOp.Clone(ctx, log); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_CLONE", err.Error())
		return nil, fmt.Errorf("ERR_GIT2STORE_CLONE: %s", err.Error())
	}

	log.Info(stageName+": репозиторий клонирован", slog.String("rep_path", repPath))
	h.recordStageSuccess(data, stageName, stageStart)
	return gitOp, nil
}

// executeStageCheckoutEdt переключается на EDT ветку (AC-2).
func (h *Git2StoreHandler) executeStageCheckoutEdt(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, gitOp GitOperator) error {
	stageStart := time.Now()
	stageName := StageCheckoutEdt
	log.Info(stageName+": переключение на EDT ветку", slog.String("branch", constants.EdtBranch))
	data.StageCurrent = stageName

	gitOp.SetBranch(constants.EdtBranch)
	if err := gitOp.Switch(ctx, log); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_CHECKOUT", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_CHECKOUT: %s", err.Error())
	}

	log.Info(stageName + ": переключено на EDT ветку")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageCreatingTempDb создаёт временную БД (AC-2).
// Возвращает путь к созданной БД для последующей очистки (H-1).
func (h *Git2StoreHandler) executeStageCreatingTempDb(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) (string, error) {
	stageStart := time.Now()
	stageName := StageCreatingTempDb
	log.Info(stageName + ": создание временной базы данных")
	data.StageCurrent = stageName

	dbConnectString, err := h.createTempDb(ctx, log, cfg)
	if err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_TEMP_DB", err.Error())
		return "", fmt.Errorf("ERR_GIT2STORE_TEMP_DB: %s", err.Error())
	}

	tempDbPath := dbConnectString
	if strings.HasPrefix(dbConnectString, "/F ") {
		tempDbPath = strings.TrimPrefix(dbConnectString, "/F ")
	}

	user := constants.DefaultUser
	pass := constants.DefaultPass
	if cfg.AppConfig != nil && cfg.AppConfig.Users.Db != "" {
		user = cfg.AppConfig.Users.Db
	}
	if cfg.SecretConfig != nil && cfg.SecretConfig.Passwords.Db != "" {
		pass = cfg.SecretConfig.Passwords.Db
	}
	ccOp.SetOneDB(dbConnectString, user, pass)

	log.Info(stageName+": временная БД создана", slog.String("path", tempDbPath))
	h.recordStageSuccess(data, stageName, stageStart)
	return tempDbPath, nil
}

// executeStageLoadingConfig загружает конфигурацию конвертации (AC-2).
func (h *Git2StoreHandler) executeStageLoadingConfig(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) error {
	stageStart := time.Now()
	stageName := StageLoadingConfig
	log.Info(stageName + ": загрузка конфигурации конвертации")
	data.StageCurrent = stageName

	if err := ccOp.Load(ctx, log, cfg, cfg.InfobaseName); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_LOAD", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_LOAD: %s", err.Error())
	}

	log.Info(stageName + ": конфигурация загружена")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageCheckoutXml переключается на XML ветку (AC-2).
func (h *Git2StoreHandler) executeStageCheckoutXml(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, gitOp GitOperator) error {
	stageStart := time.Now()
	stageName := StageCheckoutXml
	log.Info(stageName+": переключение на XML ветку", slog.String("branch", constants.OneCBranch))
	data.StageCurrent = stageName

	gitOp.SetBranch(constants.OneCBranch)
	if err := gitOp.Switch(ctx, log); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_CHECKOUT", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_CHECKOUT: %s", err.Error())
	}

	log.Info(stageName + ": переключено на XML ветку")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageInitDb инициализирует базу данных (AC-2).
func (h *Git2StoreHandler) executeStageInitDb(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) error {
	stageStart := time.Now()
	stageName := StageInitDb
	log.Info(stageName + ": инициализация базы данных")
	data.StageCurrent = stageName

	if err := ccOp.InitDb(ctx, log, cfg); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_INIT_DB", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_INIT_DB: %s", err.Error())
	}

	log.Info(stageName + ": база данных инициализирована")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageUnbinding отключает от хранилища (AC-2).
func (h *Git2StoreHandler) executeStageUnbinding(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) error {
	stageStart := time.Now()
	stageName := StageUnbinding
	log.Info(stageName + ": отключение от хранилища")
	data.StageCurrent = stageName

	if err := ccOp.StoreUnBind(ctx, log, cfg); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_UNBIND", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_UNBIND: %s", err.Error())
	}

	log.Info(stageName + ": отключено от хранилища")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageLoadingDb загружает конфигурацию в БД (AC-2).
func (h *Git2StoreHandler) executeStageLoadingDb(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) error {
	stageStart := time.Now()
	stageName := StageLoadingDb
	log.Info(stageName + ": загрузка конфигурации в базу данных")
	data.StageCurrent = stageName

	if err := ccOp.LoadDb(ctx, log, cfg); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_LOAD_DB", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_LOAD_DB: %s", err.Error())
	}

	log.Info(stageName + ": конфигурация загружена в БД")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageUpdatingDb обновляет БД (AC-2).
func (h *Git2StoreHandler) executeStageUpdatingDb(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator, stageName string) error {
	stageStart := time.Now()
	log.Info(stageName + ": обновление базы данных")
	data.StageCurrent = stageName

	if err := ccOp.DbUpdate(ctx, log, cfg); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_UPDATE", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_UPDATE: %s", err.Error())
	}

	log.Info(stageName + ": база данных обновлена")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageDumpingDb выгружает БД (AC-2).
func (h *Git2StoreHandler) executeStageDumpingDb(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) error {
	stageStart := time.Now()
	stageName := StageDumpingDb
	log.Info(stageName + ": выгрузка базы данных")
	data.StageCurrent = stageName

	if err := ccOp.DumpDb(ctx, log, cfg); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_DUMP", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_DUMP: %s", err.Error())
	}

	log.Info(stageName + ": база данных выгружена")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageBinding привязывает к хранилищу (AC-2).
func (h *Git2StoreHandler) executeStageBinding(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) error {
	stageStart := time.Now()
	stageName := StageBinding
	log.Info(stageName + ": привязка к хранилищу")
	data.StageCurrent = stageName

	if err := ccOp.StoreBind(ctx, log, cfg); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_BIND", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_BIND: %s", err.Error())
	}

	log.Info(stageName + ": привязано к хранилищу")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageLocking блокирует объекты в хранилище (AC-2).
func (h *Git2StoreHandler) executeStageLocking(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) error {
	stageStart := time.Now()
	stageName := StageLocking
	log.Info(stageName + ": блокировка объектов в хранилище")
	data.StageCurrent = stageName

	if err := ccOp.StoreLock(ctx, log, cfg); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_LOCK", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_LOCK: %s", err.Error())
	}

	log.Info(stageName + ": объекты заблокированы")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageMerging выполняет слияние (AC-2).
func (h *Git2StoreHandler) executeStageMerging(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) error {
	stageStart := time.Now()
	stageName := StageMerging
	log.Info(stageName + ": слияние конфигураций")
	data.StageCurrent = stageName

	if err := ccOp.Merge(ctx, log, cfg); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_MERGE", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_MERGE: %s", err.Error())
	}

	log.Info(stageName + ": слияние выполнено")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// executeStageCommitting фиксирует изменения в хранилище (AC-2).
func (h *Git2StoreHandler) executeStageCommitting(ctx context.Context, cfg *config.Config, log *slog.Logger, data *Git2StoreData, ccOp ConvertConfigOperator) error {
	stageStart := time.Now()
	stageName := StageCommitting
	log.Info(stageName + ": фиксация изменений в хранилище")
	data.StageCurrent = stageName

	if err := ccOp.StoreCommit(ctx, log, cfg); err != nil {
		h.recordStageError(data, stageName, stageStart, "ERR_GIT2STORE_COMMIT", err.Error())
		return fmt.Errorf("ERR_GIT2STORE_COMMIT: %s", err.Error())
	}

	log.Info(stageName + ": изменения зафиксированы")
	h.recordStageSuccess(data, stageName, stageStart)
	return nil
}

// recordStageSuccess записывает успешный результат этапа.
func (h *Git2StoreHandler) recordStageSuccess(data *Git2StoreData, stageName string, stageStart time.Time) {
	data.StagesCompleted = append(data.StagesCompleted, StageResult{
		Name:       stageName,
		Success:    true,
		DurationMs: time.Since(stageStart).Milliseconds(),
	})
}

// recordStageError записывает ошибку этапа и возвращает форматированную ошибку.
func (h *Git2StoreHandler) recordStageError(data *Git2StoreData, stageName string, stageStart time.Time, code, message string) error {
	data.StagesCompleted = append(data.StagesCompleted, StageResult{
		Name:       stageName,
		Success:    false,
		DurationMs: time.Since(stageStart).Milliseconds(),
		Error:      message,
	})
	return fmt.Errorf("%s: %s", code, message)
}
