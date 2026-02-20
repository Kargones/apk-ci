// Package config содержит конфигурацию приложения.
package config

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kargones/apk-ci/internal/entity/gitea"
)

// CreateGiteaAPI создает экземпляр Gitea API с конфигурацией из Config
func CreateGiteaAPI(cfg *Config) gitea.APIInterface {
	if cfg == nil {
		return nil
	}

	giteaConfig := gitea.Config{
		GiteaURL:    cfg.GiteaURL,
		Owner:       cfg.Owner,
		Repo:        cfg.Repo,
		AccessToken: cfg.AccessToken,
		BaseBranch:  cfg.BaseBranch,
		NewBranch:   cfg.NewBranch,
		Command:     cfg.Command,
	}
	return gitea.NewGiteaAPI(giteaConfig)
}

// GetSonarQubeConfig возвращает конфигурацию SonarQube.
// Если конфигурация не загружена, возвращает nil.
// Возвращает:
//   - *SonarQubeConfig: указатель на конфигурацию SonarQube
func (cfg *Config) GetSonarQubeConfig() *SonarQubeConfig {
	return cfg.SonarQubeConfig
}

// GetScannerConfig возвращает конфигурацию сканера.
// Если конфигурация не загружена, возвращает nil.
// Возвращает:
//   - *ScannerConfig: указатель на конфигурацию сканера
func (cfg *Config) GetScannerConfig() *ScannerConfig {
	return cfg.ScannerConfig
}

// SetSonarQubeConfig устанавливает конфигурацию SonarQube.
// Параметры:
//   - config: указатель на конфигурацию SonarQube
func (cfg *Config) SetSonarQubeConfig(config *SonarQubeConfig) {
	cfg.SonarQubeConfig = config
}

// SetScannerConfig устанавливает конфигурацию сканера.
// Параметры:
//   - config: указатель на конфигурацию сканера
func (cfg *Config) SetScannerConfig(config *ScannerConfig) {
	cfg.ScannerConfig = config
}

// LoadSonarQubeConfigFromEnv загружает конфигурацию SonarQube из переменных окружения.
// Возвращает:
//   - *SonarQubeConfig: указатель на загруженную конфигурацию SonarQube
//   - error: ошибка загрузки конфигурации или nil при успехе
func (cfg *Config) LoadSonarQubeConfigFromEnv() (*SonarQubeConfig, error) {
	sonarQubeConfig, err := GetSonarQubeConfig(cfg.Logger, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load SonarQube config: %w", err)
	}

	cfg.SetSonarQubeConfig(sonarQubeConfig)
	return sonarQubeConfig, nil
}

// LoadScannerConfigFromEnv загружает конфигурацию сканера из переменных окружения.
// Возвращает:
//   - *ScannerConfig: указатель на загруженную конфигурацию сканера
//   - error: ошибка загрузки конфигурации или nil при успехе
func (cfg *Config) LoadScannerConfigFromEnv() (*ScannerConfig, error) {
	scannerConfig, err := GetScannerConfig(cfg.Logger, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load Scanner config: %w", err)
	}

	cfg.SetScannerConfig(scannerConfig)
	return scannerConfig, nil
}

// FileInfo теперь используется из модуля gitea

// AnalyzeProject анализирует проект и заполняет поля ProjectName и AddArray в конфигурации.
// Выполняет анализ структуры проекта и его конфигурации, проверяет целостность настроек проекта,
// валидирует конфигурацию баз данных и анализирует зависимости между компонентами системы.
// Параметры:
//   - l: логгер для записи информации о процессе анализа
//   - branch: имя ветки для анализа проекта
//
// Возвращает:
//   - error: ошибка анализа проекта или nil при успехе
func (cfg *Config) AnalyzeProject(ctx context.Context, l *slog.Logger, branch string) error {
	// Инициализируем Gitea API
	g := CreateGiteaAPI(cfg)

	// Анализируем проект
	analysis, err := g.AnalyzeProject(ctx, branch)
	if err != nil {
		l.Error("Ошибка анализа проекта",
			slog.String("error", err.Error()),
		)
		return err
	}

	// Заполняем поля конфигурации
	if len(analysis) == 0 {
		l.Info("Проект не найден или не соответствует критериям")
		cfg.ProjectName = ""
		cfg.AddArray = []string{}
	} else {
		cfg.ProjectName = analysis[0]
		cfg.AddArray = analysis[1:]
		l.Info("Результат анализа проекта",
			slog.String("project_name", cfg.ProjectName),
			slog.Any("extensions", cfg.AddArray),
		)
	}

	return nil
}
