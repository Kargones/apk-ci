package service

import (
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/gitea"
)

// GiteaFactory создает компоненты для работы с Gitea
type GiteaFactory struct{}

// NewGiteaFactory создает новую фабрику для компонентов Gitea.
// Инициализирует фабрику, которая используется для создания
// различных компонентов системы работы с Gitea.
// Возвращает:
//   - *GiteaFactory: новый экземпляр фабрики
func NewGiteaFactory() *GiteaFactory {
	return &GiteaFactory{}
}

// CreateGiteaConfig создает конфигурацию Gitea из общей конфигурации.
// Преобразует настройки приложения в специфичную для Gitea
// конфигурацию с необходимыми параметрами.
// Параметры:
//   - cfg: общая конфигурация приложения
//
// Возвращает:
//   - gitea.Config: конфигурация для работы с Gitea
func (f *GiteaFactory) CreateGiteaConfig(cfg *config.Config) gitea.Config {
	return gitea.Config{
		GiteaURL:    cfg.GiteaURL,
		Owner:       cfg.Owner,
		Repo:        cfg.Repo,
		AccessToken: cfg.AccessToken,
		BaseBranch:  cfg.BaseBranch,
		NewBranch:   cfg.NewBranch,
		Command:     cfg.Command,
	}
}

// CreateGiteaService создает полностью настроенный GiteaService.
// Инициализирует все необходимые компоненты (API, анализатор)
// и создает готовый к использованию сервис.
// Параметры:
//   - cfg: конфигурация приложения
//
// Возвращает:
//   - *GiteaService: настроенный сервис Gitea
//   - error: ошибка создания или nil при успехе
func (f *GiteaFactory) CreateGiteaService(cfg *config.Config) (*GiteaService, error) {
	// Создаем конфигурацию Gitea
	giteaConfig := gitea.Config{
		GiteaURL:    cfg.GiteaURL,
		Owner:       cfg.Owner,
		Repo:        cfg.Repo,
		AccessToken: cfg.AccessToken,
		BaseBranch:  cfg.BaseBranch,
		NewBranch:   cfg.NewBranch,
		Command:     cfg.Command,
	}
	api := gitea.NewGiteaAPI(giteaConfig)
	analyzer := NewConfigAnalyzer(cfg)

	return NewGiteaService(api, cfg, analyzer), nil
}
