// Package shared предоставляет общие утилиты для обработчиков команд.
package shared

import (
	"fmt"
	"log/slog"

	adapter_gitea "github.com/Kargones/apk-ci/internal/adapter/gitea"
	adapter_sq "github.com/Kargones/apk-ci/internal/adapter/sonarqube"
	"github.com/Kargones/apk-ci/internal/config"
	entity_gitea "github.com/Kargones/apk-ci/internal/entity/gitea"
	entity_sq "github.com/Kargones/apk-ci/internal/entity/sonarqube"
)

// CreateGiteaClient создаёт реальный Gitea API клиент из конфигурации.
// Возвращает adapter_gitea.Client, готовый к использованию в обработчиках.
func CreateGiteaClient(cfg *config.Config) (adapter_gitea.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("конфигурация не может быть nil")
	}
	if cfg.GiteaURL == "" {
		return nil, fmt.Errorf("не указан Gitea URL (BR_GITEA_URL)")
	}
	if cfg.AccessToken == "" {
		return nil, fmt.Errorf("не указан Access Token (BR_ACCESS_TOKEN)")
	}

	entityCfg := entity_gitea.Config{
		GiteaURL:    cfg.GiteaURL,
		Owner:       cfg.Owner,
		Repo:        cfg.Repo,
		AccessToken: cfg.AccessToken,
		BaseBranch:  cfg.BaseBranch,
	}

	api := entity_gitea.NewGiteaAPI(entityCfg)
	return adapter_gitea.NewAPIClient(api), nil
}

// CreateSonarQubeClient создаёт реальный SonarQube API клиент из конфигурации.
// Возвращает adapter_sq.Client, готовый к использованию в обработчиках.
func CreateSonarQubeClient(cfg *config.Config) (adapter_sq.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("конфигурация не может быть nil")
	}

	logger := slog.Default()

	sqCfg, err := config.GetSonarQubeConfig(logger, cfg)
	if err != nil {
		return nil, fmt.Errorf("не удалось загрузить конфигурацию SonarQube: %w", err)
	}

	entity := entity_sq.NewEntity(sqCfg, logger)
	return adapter_sq.NewAPIClient(entity), nil
}
