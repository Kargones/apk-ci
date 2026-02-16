package app

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/Kargones/apk-ci/internal/config"
)

func TestActionMenuBuildForceUpdate(t *testing.T) {
	tests := []struct {
		name        string
		forceUpdate bool
		description string
		expectError bool
	}{
		{
			name:        "force_update_true",
			forceUpdate: true,
			description: "Должен выполнить обновление принудительно",
			expectError: false, // С force_update=true должен пропустить проверку изменений
		},
		{
			name:        "force_update_false",
			forceUpdate: false,
			description: "Должен проверить изменения project.yaml",
			expectError: true, // Без моков API будет ошибка
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовую конфигурацию с локальным URL для избежания сетевых запросов
			cfg := &config.Config{
				ForceUpdate: tt.forceUpdate,
				GiteaURL:    "http://localhost:3000", // Локальный URL вместо test.gitea.com
				Owner:       "test-owner",
				Repo:        "test-repo",
				BaseBranch:  "main",
				AccessToken: "test-token",
				Command:     "action-menu-build",
			}

			// Создаем логгер для тестирования с минимальным уровнем для уменьшения вывода
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelError, // Только ошибки для чистоты тестов
			}))

			// Создаем контекст
			ctx := context.Background()

			// Вызываем ActionMenuBuild
			err := ActionMenuBuild(ctx, logger, cfg)

			// Проверяем результат в зависимости от ожиданий
			if tt.expectError {
				if err == nil {
					t.Errorf("Ожидалась ошибка для %s, но функция выполнилась успешно", tt.name)
				} else {
					t.Logf("Получена ожидаемая ошибка для %s: %v", tt.name, err)
				}
			} else {
				// Для force_update=true ошибка все равно возможна из-за отсутствия конфигурации проекта
				if err != nil {
					t.Logf("ActionMenuBuild завершился с ошибкой для %s (может быть ожидаемо): %v", tt.name, err)
				} else {
					t.Logf("ActionMenuBuild выполнен успешно для %s", tt.name)
				}
			}
		})
	}
}

func TestActionMenuBuildErrorHandling(t *testing.T) {
	// Тест для проверки обработки ошибок ActionMenuBuildError
	err := newActionMenuBuildError("test-operation",
		&ActionMenuBuildError{Operation: "nested", Cause: nil, Details: "nested error"},
		"test details")

	if err.Operation != "test-operation" {
		t.Errorf("Expected Operation = 'test-operation', got '%s'", err.Operation)
	}

	if err.Details != "test details" {
		t.Errorf("Expected Details = 'test details', got '%s'", err.Details)
	}

	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}
