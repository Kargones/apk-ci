// Package onec содержит адаптеры для работы с 1C:Предприятие.
package onec

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/util/runner"
)

// Creator1cv8 реализует DatabaseCreator через 1cv8.
// Использует режим CREATEINFOBASE для создания информационной базы.
type Creator1cv8 struct {
	bin1cv8 string
	workDir string
	tmpDir  string
}

// Compile-time проверка интерфейса.
var _ DatabaseCreator = (*Creator1cv8)(nil)

// NewCreator1cv8 создаёт новый Creator1cv8 с указанными путями.
func NewCreator1cv8(bin1cv8, workDir, tmpDir string) *Creator1cv8 {
	return &Creator1cv8{
		bin1cv8: bin1cv8,
		workDir: workDir,
		tmpDir:  tmpDir,
	}
}

// CreateDB создаёт информационную базу через 1cv8 CREATEINFOBASE.
func (c *Creator1cv8) CreateDB(ctx context.Context, opts CreateDBOptions) (*CreateDBResult, error) {
	start := time.Now()
	log := slog.Default().With(slog.String("operation", "CreateDB"), slog.String("tool", "1cv8"))

	if c.bin1cv8 == "" {
		return nil, fmt.Errorf("путь к 1cv8 не указан")
	}

	r := runner.Runner{}
	r.TmpDir = c.tmpDir
	r.WorkDir = c.workDir
	r.RunString = c.bin1cv8

	// Формируем строку подключения
	connectString := c.buildConnectString(opts)

	// Формируем параметры команды
	r.Params = []string{
		"@",              // Использовать параметр-файл
		"CREATEINFOBASE", // Режим создания базы
		connectString,
	}

	// Отключаем GUI диалоги
	addDisableParam(&r)

	// Перенаправляем вывод в файл
	r.Params = append(r.Params, "/Out")

	// Устанавливаем таймаут
	ctxWithTimeout := ctx
	var cancel context.CancelFunc
	if opts.Timeout > 0 {
		ctxWithTimeout, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	log.Info("Запуск создания информационной базы",
		slog.String("db_path", opts.DbPath),
		slog.Bool("server_based", opts.ServerBased))

	// Выполняем команду
	_, err := r.RunCommand(ctxWithTimeout, log)

	result := &CreateDBResult{
		ConnectString: connectString,
		DbPath:        opts.DbPath,
		CreatedAt:     time.Now(),
		DurationMs:    time.Since(start).Milliseconds(),
	}

	// Извлекаем сообщения из вывода
	output := string(r.FileOut)

	// Проверка успеха по отсутствию ошибок и сообщениям платформы
	if err == nil && (strings.Contains(output, "Информационная база создана") ||
		strings.Contains(output, "Infobase has been created") ||
		strings.Contains(output, "База данных создана") ||
		len(output) < 100) {
		result.Success = true
		log.Info("Информационная база создана успешно",
			slog.String("db_path", opts.DbPath),
			slog.Int64("duration_ms", result.DurationMs))
	} else {
		result.Success = false
		if err == nil {
			err = fmt.Errorf("создание базы не завершено успешно: %s", trimOutput(output))
		}
		log.Error("Ошибка создания информационной базы",
			slog.String("output", trimOutput(output)),
			slog.Int64("duration_ms", result.DurationMs))
	}

	return result, err
}

// buildConnectString формирует строку подключения для создания базы.
func (c *Creator1cv8) buildConnectString(opts CreateDBOptions) string {
	if opts.ServerBased {
		// Серверная база: Srvr="server";Ref="dbname"
		return fmt.Sprintf(`Srvr="%s";Ref="%s"`, opts.Server, opts.DbName)
	}
	// Файловая база: File="path"
	return fmt.Sprintf(`File="%s"`, opts.DbPath)
}
