package app

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/Kargones/apk-ci/internal/config"
)

func TestExecuteEpf(t *testing.T) {
	tests := []struct {
		name        string
		startEpf    string
		expectError bool
		description string
	}{
		{
			name:        "empty_url",
			startEpf:    "",
			expectError: true,
			description: "Должен возвращать ошибку при пустом URL",
		},
		{
			name:        "invalid_url",
			startEpf:    "invalid-url",
			expectError: true,
			description: "Должен возвращать ошибку при некорректном URL",
		},
		{
			name:        "valid_url_format",
			startEpf:    "https://example.com/test.epf",
			expectError: true, // Ожидаем ошибку, так как файл не существует
			description: "Должен пытаться скачать файл по корректному URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем тестовую конфигурацию
			cfg := &config.Config{
				StartEpf: tt.startEpf,
				AppConfig: &config.AppConfig{
					WorkDir: "/tmp/test",
					TmpDir:  "/tmp/test/temp",
					Paths: struct {
						Bin1cv8  string `yaml:"bin1cv8"`
						BinIbcmd string `yaml:"binIbcmd"`
						EdtCli   string `yaml:"edtCli"`
						Rac      string `yaml:"rac"`
					}{
						Bin1cv8: "/opt/1cv8/test/1cv8",
					},
				},
			}

			// Создаем логгер для тестирования
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))

			// Создаем контекст
			ctx := context.Background()

			// Выполняем тест
			err := ExecuteEpf(&ctx, logger, cfg)

			// Проверяем результат
			if tt.expectError {
				if err == nil {
					t.Errorf("Ожидалась ошибка для теста %s, но ошибки не было", tt.name)
				}
				t.Logf("Тест %s: получена ожидаемая ошибка: %v", tt.name, err)
			} else {
				if err != nil {
					t.Errorf("Неожиданная ошибка для теста %s: %v", tt.name, err)
				}
				t.Logf("Тест %s: выполнен успешно", tt.name)
			}
		})
	}
}

func TestExecuteEpfValidation(t *testing.T) {
	// Тест валидации параметров
	t.Run("nil_config", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		ctx := context.Background()

		err := ExecuteEpf(&ctx, logger, nil)
		if err == nil {
			t.Error("Ожидалась ошибка при передаче nil конфигурации")
		}
	})

	t.Run("missing_onedb_config", func(t *testing.T) {
		cfg := &config.Config{
			StartEpf: "https://example.com/test.epf",
			AppConfig: &config.AppConfig{
				WorkDir: "/tmp/test",
				TmpDir:  "/tmp/test/temp",
				Paths: struct {
					Bin1cv8  string `yaml:"bin1cv8"`
					BinIbcmd string `yaml:"binIbcmd"`
					EdtCli   string `yaml:"edtCli"`
					Rac      string `yaml:"rac"`
				}{
					Bin1cv8: "/usr/bin/1cv8",
				},
			},
			// Convert не установлен
		}

		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		ctx := context.Background()

		err := ExecuteEpf(&ctx, logger, cfg)
		if err == nil {
			t.Error("Ожидалась ошибка при отсутствии конфигурации OneDB")
		}
	})
}

func TestExecuteEpfTempFileHandling(t *testing.T) {
	// Тест работы с временными файлами
	t.Run("temp_dir_creation", func(t *testing.T) {
		tempDir := filepath.Join(os.TempDir(), "test_execute_epf")
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("Failed to remove temp dir: %v", err)
			}
		}()

		cfg := &config.Config{
			StartEpf: "https://example.com/test.epf",
			AppConfig: &config.AppConfig{
				WorkDir: tempDir,
				TmpDir:  filepath.Join(tempDir, "temp"),
				Paths: struct {
					Bin1cv8  string `yaml:"bin1cv8"`
					BinIbcmd string `yaml:"binIbcmd"`
					EdtCli   string `yaml:"edtCli"`
					Rac      string `yaml:"rac"`
				}{
					Bin1cv8: "/opt/1cv8/test/1cv8",
				},
			},
		}

		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		ctx := context.Background()

		// Выполняем функцию (ожидаем ошибку скачивания, но не ошибку создания директории)
		err := ExecuteEpf(&ctx, logger, cfg)

		// Проверяем, что временная директория была создана
		if _, statErr := os.Stat(cfg.AppConfig.TmpDir); os.IsNotExist(statErr) {
			t.Error("Временная директория не была создана")
		}

		// Ошибка должна быть связана со скачиванием, а не с созданием директории
		if err != nil {
			t.Logf("Получена ожидаемая ошибка скачивания: %v", err)
		}
	})
}
