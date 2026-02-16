// Package shared содержит общие компоненты для всех command handlers.
// Коды ошибок и утилиты используются в sonarqube и gitea хендлерах.
package shared

// Общие коды ошибок для всех команд.
// Централизованы для соблюдения DRY principle.
// Fix H-3: устранено дублирование между sonarqube/shared и gitea/shared.
const (
	// ErrConfigMissing — отсутствует необходимая конфигурация.
	ErrConfigMissing = "CONFIG.MISSING"
	// ErrMissingOwnerRepo — не указан owner или repo.
	ErrMissingOwnerRepo = "CONFIG.MISSING_OWNER_REPO"
	// ErrGiteaAPI — ошибка Gitea API.
	ErrGiteaAPI = "GITEA.API_FAILED"
	// ErrSonarQubeAPI — ошибка SonarQube API.
	ErrSonarQubeAPI = "SONARQUBE.API_FAILED"
)
