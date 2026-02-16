// Package onec содержит адаптеры для работы с 1C:Предприятие.
package onec

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/util/runner"
)

// CreatorIbcmd реализует DatabaseCreator через ibcmd.
// Использует команду ibcmd infobase create для создания информационной базы.
type CreatorIbcmd struct {
	binIbcmd string
}

// Compile-time проверка интерфейса.
var _ DatabaseCreator = (*CreatorIbcmd)(nil)

// NewCreatorIbcmd создаёт новый CreatorIbcmd с указанным путём к ibcmd.
func NewCreatorIbcmd(binIbcmd string) *CreatorIbcmd {
	return &CreatorIbcmd{
		binIbcmd: binIbcmd,
	}
}

// CreateDB создаёт информационную базу через ibcmd infobase create.
func (c *CreatorIbcmd) CreateDB(ctx context.Context, opts CreateDBOptions) (*CreateDBResult, error) {
	start := time.Now()
	log := slog.Default().With(slog.String("operation", "CreateDB"), slog.String("tool", "ibcmd"))

	if c.binIbcmd == "" {
		return nil, fmt.Errorf("путь к ibcmd не указан")
	}

	r := runner.Runner{}
	r.RunString = c.binIbcmd

	// Формируем параметры команды
	if opts.ServerBased {
		// Серверная база
		r.Params = []string{
			"infobase", "create",
			"--create-database",
			"--dbms=MSSQLServer",
			fmt.Sprintf("--database-server=%s", opts.Server),
			fmt.Sprintf("--database-name=%s", opts.DbName),
		}
	} else {
		// Файловая база
		r.Params = []string{
			"infobase", "create",
			"--create-database",
			fmt.Sprintf("--db-path=%s", opts.DbPath),
		}
	}

	// Устанавливаем таймаут
	ctxWithTimeout := ctx
	var cancel context.CancelFunc
	if opts.Timeout > 0 {
		ctxWithTimeout, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	log.Info("Запуск создания информационной базы через ibcmd",
		slog.String("db_path", opts.DbPath),
		slog.Bool("server_based", opts.ServerBased))

	// Выполняем команду
	_, err := r.RunCommand(ctxWithTimeout, log)

	// Формируем строку подключения
	connectString := c.buildConnectString(opts)

	result := &CreateDBResult{
		ConnectString: connectString,
		DbPath:        opts.DbPath,
		CreatedAt:     time.Now(),
		DurationMs:    time.Since(start).Milliseconds(),
	}

	// Проверка успеха по сообщениям
	output := string(r.ConsoleOut)
	if err == nil && (strings.Contains(output, constants.SearchMsgBaseCreateOk) ||
		strings.Contains(output, constants.SearchMsgBaseCreateOkEn)) {
		result.Success = true
		log.Info("Информационная база через ibcmd создана успешно",
			slog.String("db_path", opts.DbPath),
			slog.Int64("duration_ms", result.DurationMs))
	} else {
		result.Success = false
		if err == nil {
			err = fmt.Errorf("создание базы не завершено успешно: %s", trimOutput(output))
		}
		log.Error("Ошибка создания информационной базы через ibcmd",
			slog.String("output", trimOutput(output)),
			slog.Int64("duration_ms", result.DurationMs))
	}

	return result, err
}

// buildConnectString формирует строку подключения для созданной базы.
func (c *CreatorIbcmd) buildConnectString(opts CreateDBOptions) string {
	if opts.ServerBased {
		// Серверная база: /S server\dbname
		return fmt.Sprintf("/S %s\\%s", opts.Server, opts.DbName)
	}
	// Файловая база: /F path
	return "/F " + opts.DbPath
}
