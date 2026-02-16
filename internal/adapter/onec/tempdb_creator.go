// Package onec содержит адаптеры для работы с 1C:Предприятие.
package onec

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/util/runner"
)

// TempDbCreator реализует интерфейс TempDatabaseCreator через ibcmd.
type TempDbCreator struct{}

// NewTempDbCreator создаёт новый TempDbCreator.
func NewTempDbCreator() *TempDbCreator {
	return &TempDbCreator{}
}

// Compile-time проверка интерфейса.
var _ TempDatabaseCreator = (*TempDbCreator)(nil)

// CreateTempDB создаёт локальную файловую БД через ibcmd.
// H4 fix (Review #3): при ошибке добавления расширения выполняется cleanup созданной БД.
func (c *TempDbCreator) CreateTempDB(ctx context.Context, opts CreateTempDBOptions) (*TempDBResult, error) {
	start := time.Now()
	log := slog.Default()

	// 1. Создание информационной базы
	// MEDIUM-1 fix (Review #4): используем типизированную ошибку ErrInfobaseCreate
	if err := c.createInfobase(ctx, opts, log); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInfobaseCreate, err)
	}

	// H4 fix (Review #3): флаг для отслеживания необходимости cleanup
	dbCreated := true

	// HIGH-2 fix (Review #4): проверка отмены контекста между созданием базы и расширениями
	// Предотвращает "сиротские" базы при отмене операции
	if ctx.Err() != nil {
		if cleanupErr := c.cleanupDb(opts.DbPath, log); cleanupErr != nil {
			log.Warn("Не удалось удалить БД после отмены контекста",
				slog.String("path", opts.DbPath),
				slog.String("cleanup_error", cleanupErr.Error()))
		}
		// MEDIUM-1 fix (Review #4): используем типизированную ошибку ErrContextCancelled
		return nil, fmt.Errorf("%w после создания базы: %v", ErrContextCancelled, ctx.Err())
	}

	// 2. Добавление расширений с cleanup при ошибке
	for _, ext := range opts.Extensions {
		if ext == "" {
			continue // Пропускаем пустые имена
		}
		if err := c.addExtension(ctx, opts, ext, log); err != nil {
			// H4 fix (Review #3): cleanup — удаляем созданную БД при ошибке расширения
			if dbCreated {
				if cleanupErr := c.cleanupDb(opts.DbPath, log); cleanupErr != nil {
					log.Warn("Не удалось удалить частично созданную БД при ошибке расширения",
						slog.String("path", opts.DbPath),
						slog.String("cleanup_error", cleanupErr.Error()))
				} else {
					log.Debug("Частично созданная БД удалена после ошибки расширения",
						slog.String("path", opts.DbPath))
				}
			}
			// MEDIUM-1 fix (Review #4): используем типизированную ошибку ErrExtensionAdd
			return nil, fmt.Errorf("%w '%s': %v", ErrExtensionAdd, ext, err)
		}
	}

	return &TempDBResult{
		ConnectString: "/F " + opts.DbPath,
		DbPath:        opts.DbPath,
		Extensions:    opts.Extensions,
		CreatedAt:     time.Now(),
		DurationMs:    time.Since(start).Milliseconds(),
	}, nil
}

// cleanupDb удаляет директорию БД при ошибке (H4 fix Review #3).
func (c *TempDbCreator) cleanupDb(dbPath string, log *slog.Logger) error {
	log.Debug("Cleanup: удаление директории БД", slog.String("path", dbPath))
	return os.RemoveAll(dbPath)
}

// createInfobase создаёт информационную базу через ibcmd.
func (c *TempDbCreator) createInfobase(ctx context.Context, opts CreateTempDBOptions, log *slog.Logger) error {
	r := runner.Runner{}
	r.RunString = opts.BinIbcmd
	// H2 fix: явная инициализация slice для избежания потенциальных проблем
	r.Params = []string{
		"infobase", "create",
		"--create-database",
		fmt.Sprintf("--db-path=%s", opts.DbPath),
	}

	log.Debug("Создание информационной базы",
		slog.String("command", r.RunString),
		slog.Any("params", r.Params))

	ctxWithTimeout, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	_, err := r.RunCommand(ctxWithTimeout, log)
	if err != nil {
		return err
	}

	// Проверка успеха по сообщениям (поддерживаем русский и английский варианты)
	output := string(r.ConsoleOut)
	if !strings.Contains(output, constants.SearchMsgBaseCreateOk) &&
		!strings.Contains(output, constants.SearchMsgBaseCreateOkEn) {
		return fmt.Errorf("неожиданный результат: %s", output)
	}

	log.Debug("Информационная база успешно создана", slog.String("path", opts.DbPath))
	return nil
}

// addExtension добавляет расширение в информационную базу через ibcmd.
func (c *TempDbCreator) addExtension(ctx context.Context, opts CreateTempDBOptions, extName string, log *slog.Logger) error {
	r := runner.Runner{}
	r.RunString = opts.BinIbcmd
	// H2 fix: явная инициализация slice для избежания потенциальных проблем
	r.Params = []string{
		"extension", "create",
		fmt.Sprintf("--db-path=%s", opts.DbPath),
		fmt.Sprintf("--name=%s", extName),
		"--name-prefix=p",
	}

	log.Debug("Добавление расширения",
		slog.String("extension", extName),
		slog.String("command", r.RunString),
		slog.Any("params", r.Params))

	ctxWithTimeout, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	_, err := r.RunCommand(ctxWithTimeout, log)
	if err != nil {
		return err
	}

	// Проверка успеха
	output := string(r.ConsoleOut)
	if !strings.Contains(output, constants.SearchMsgBaseAddOk) {
		return fmt.Errorf("неожиданный результат: %s", output)
	}

	log.Debug("Расширение успешно добавлено",
		slog.String("extension", extName),
		slog.String("path", opts.DbPath))
	return nil
}
