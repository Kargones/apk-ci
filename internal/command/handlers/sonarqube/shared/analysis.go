// Package shared содержит общую логику для SonarQube команд.
// Вынесено из scanbranch и scanpr для соблюдения DRY principle.
package shared

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/gitea"
	"github.com/Kargones/apk-ci/internal/adapter/sonarqube"
)

// HasRelevantChangesInCommit проверяет наличие изменений в каталогах конфигурации.
// Используется в scanbranch и scanpr для фильтрации коммитов без изменений 1C кода.
func HasRelevantChangesInCommit(ctx context.Context, giteaClient gitea.Client, branch, commitSHA string) (bool, error) {
	// Получение структуры проекта
	projectStructure, err := giteaClient.AnalyzeProjectStructure(ctx, branch)
	if err != nil {
		return false, fmt.Errorf("ошибка анализа структуры проекта: %w", err)
	}

	if len(projectStructure) == 0 {
		// Структура проекта не определена — считаем что любые изменения релевантны
		return true, nil
	}

	// Формирование configDirs
	mainConfig := projectStructure[0]
	configDirs := []string{mainConfig}
	for i := 1; i < len(projectStructure); i++ {
		configDirs = append(configDirs, mainConfig+"."+projectStructure[i])
	}

	// Получение изменённых файлов
	changedFiles, err := giteaClient.GetCommitFiles(ctx, commitSHA)
	if err != nil {
		return false, fmt.Errorf("ошибка получения файлов коммита: %w", err)
	}

	// Проверка prefixes файлов в configDirs
	for _, file := range changedFiles {
		for _, configDir := range configDirs {
			if strings.HasPrefix(file.Filename, configDir+"/") {
				return true, nil
			}
		}
	}

	return false, nil
}

// WaitForAnalysisCompletion ожидает завершения анализа SonarQube.
// Используется в scanbranch и scanpr после запуска RunAnalysis.
func WaitForAnalysisCompletion(ctx context.Context, sqClient sonarqube.Client, taskID string, log *slog.Logger) (*sonarqube.AnalysisStatus, error) {
	const maxAttempts = 60
	const pollInterval = 5 * time.Second

	for i := range maxAttempts {
		// Проверяем context перед каждой итерацией
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		status, err := sqClient.GetAnalysisStatus(ctx, taskID)
		if err != nil {
			return nil, fmt.Errorf("ошибка получения статуса анализа: %w", err)
		}

		switch status.Status {
		case "SUCCESS":
			return status, nil
		case "FAILED", "CANCELED":
			return status, nil
		case "PENDING", "IN_PROGRESS":
			log.Debug("Ожидание завершения анализа",
				slog.String("task_id", taskID),
				slog.Int("attempt", i+1),
				slog.String("status", status.Status))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(pollInterval):
				// Продолжаем polling
			}
		default:
			return nil, fmt.Errorf("неизвестный статус анализа: %s", status.Status)
		}
	}

	return nil, fmt.Errorf("превышено время ожидания завершения анализа (task_id: %s)", taskID)
}
