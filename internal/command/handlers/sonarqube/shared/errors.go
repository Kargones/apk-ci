// Package shared содержит общую логику для SonarQube команд.
package shared

import (
	commonerrors "github.com/Kargones/apk-ci/internal/command/handlers/shared"
)

// Общие коды ошибок — импортируем из централизованного пакета (H-3 fix).
const (
	ErrConfigMissing    = commonerrors.ErrConfigMissing
	ErrMissingOwnerRepo = commonerrors.ErrMissingOwnerRepo
	ErrGiteaAPI         = commonerrors.ErrGiteaAPI
	ErrSonarQubeAPI     = commonerrors.ErrSonarQubeAPI
)

// Специфичные коды ошибок для SonarQube команд.
const (
	// ErrBranchMissing — не указана ветка.
	ErrBranchMissing = "BRANCH.MISSING"
	// ErrBranchInvalidFormat — ветка не соответствует критериям.
	ErrBranchInvalidFormat = "BRANCH.INVALID_FORMAT"
	// ErrBranchNoChanges — нет изменений в конфигурации.
	ErrBranchNoChanges = "BRANCH.NO_CHANGES"
	// ErrProjectNotFound — проект не найден в SonarQube.
	ErrProjectNotFound = "SONARQUBE.PROJECT_NOT_FOUND"
	// ErrPRMissing — не указан номер PR.
	ErrPRMissing = "PR.MISSING"
	// ErrPRInvalid — некорректный номер PR.
	ErrPRInvalid = "PR.INVALID"
	// ErrPRNotFound — PR не найден.
	ErrPRNotFound = "PR.NOT_FOUND"
	// ErrPRNotOpen — PR не в состоянии "open".
	ErrPRNotOpen = "PR.NOT_OPEN"
)
