// Package converthandler содержит production реализации для конвертации EDT/XML.
package converthandler

import (
	"context"
	"log/slog"
	"os"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/one/edt"
)

// convertProduction — production реализация конвертации через edt.Cli.
func convertProduction(ctx context.Context, l *slog.Logger, cfg *config.Config, direction, pathIn, pathOut string) error {
	cli := &edt.Cli{}
	cli.Init(cfg)

	// Настройка параметров конвертации
	cli.Direction = direction
	cli.PathIn = pathIn
	cli.PathOut = pathOut

	// H-1 fix: Флаг для отслеживания созданной временной директории
	var createdWorkspace string

	// Создаём временную рабочую область EDT если не указана
	if cli.WorkSpace == "" {
		tmpDir := cfg.TmpDir
		if tmpDir == "" {
			tmpDir = os.TempDir()
		}
		ws, err := os.MkdirTemp(tmpDir, "ws")
		if err != nil {
			l.Error("Не удалось создать временный каталог для workspace",
				slog.String("tmp_dir", tmpDir),
				slog.String("error", err.Error()))
			return err
		}
		cli.WorkSpace = ws
		createdWorkspace = ws
	}

	// H-1 fix: Очищаем временную директорию после завершения
	if createdWorkspace != "" {
		defer func() {
			if err := os.RemoveAll(createdWorkspace); err != nil {
				l.Warn("Не удалось удалить временную директорию workspace",
					slog.String("workspace", createdWorkspace),
					slog.String("error", err.Error()))
			} else {
				l.Debug("Временная директория workspace удалена",
					slog.String("workspace", createdWorkspace))
			}
		}()
	}

	// Выполняем конвертацию
	cli.Convert(&ctx, l, cfg)

	// Проверяем результат
	if cli.LastErr != nil {
		return cli.LastErr
	}

	return nil
}
