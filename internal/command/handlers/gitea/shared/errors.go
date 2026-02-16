// Package shared содержит общую логику для Gitea команд.
package shared

import (
	commonerrors "github.com/Kargones/apk-ci/internal/command/handlers/shared"
)

// Общие коды ошибок — импортируем из централизованного пакета (H-3 fix).
const (
	ErrConfigMissing    = commonerrors.ErrConfigMissing
	ErrMissingOwnerRepo = commonerrors.ErrMissingOwnerRepo
	ErrGiteaAPI         = commonerrors.ErrGiteaAPI
)

// Специфичные коды ошибок для Gitea команд.
const (
	// ErrBranchCreate — ошибка создания ветки.
	ErrBranchCreate = "GITEA.BRANCH_CREATE_FAILED"
	// ErrNoDatabases — нет баз данных в конфигурации.
	ErrNoDatabases = "CONFIG.NO_DATABASES"
	// ErrTemplateProcess — ошибка обработки шаблона.
	ErrTemplateProcess = "TEMPLATE.PROCESS_FAILED"
	// ErrSyncFailed — ошибка синхронизации файлов.
	ErrSyncFailed = "SYNC.FAILED"
)
