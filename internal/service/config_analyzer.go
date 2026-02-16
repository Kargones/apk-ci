// Package service содержит бизнес-логику приложения
package service

import (
	"log/slog"

	"github.com/Kargones/apk-ci/internal/config"
)

// ConfigAnalyzer адаптер для config.Config, реализующий ProjectAnalyzer
type ConfigAnalyzer struct {
	config *config.Config
}

// NewConfigAnalyzer создает новый адаптер
// NewConfigAnalyzer создает новый адаптер для анализа конфигурации.
// Инициализирует анализатор, который адаптирует config.Config
// для реализации интерфейса ProjectAnalyzer.
// Параметры:
//   - cfg: конфигурация приложения
//
// Возвращает:
//   - *ConfigAnalyzer: новый экземпляр анализатора
func NewConfigAnalyzer(cfg *config.Config) *ConfigAnalyzer {
	return &ConfigAnalyzer{config: cfg}
}

// AnalyzeProject реализует интерфейс ProjectAnalyzer
// AnalyzeProject реализует интерфейс ProjectAnalyzer.
// Выполняет анализ проекта, используя методы конфигурации
// для изучения структуры проекта в указанной ветке.
// Параметры:
//   - l: логгер для записи сообщений
//   - branch: имя ветки для анализа
//
// Возвращает:
//   - error: ошибка анализа или nil при успехе
func (ca *ConfigAnalyzer) AnalyzeProject(l *slog.Logger, branch string) error {
	return ca.config.AnalyzeProject(l, branch)
}
