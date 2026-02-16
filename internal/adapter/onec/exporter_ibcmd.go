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

// ExporterIbcmd реализует ConfigExporter через ibcmd.
// Использует команду ibcmd infobase config export для выгрузки конфигурации.
type ExporterIbcmd struct {
	binIbcmd string
}

// Compile-time проверка интерфейса.
var _ ConfigExporter = (*ExporterIbcmd)(nil)

// NewExporterIbcmd создаёт новый ExporterIbcmd с указанным путём к ibcmd.
func NewExporterIbcmd(binIbcmd string) *ExporterIbcmd {
	return &ExporterIbcmd{
		binIbcmd: binIbcmd,
	}
}

// Export выгружает конфигурацию через ibcmd infobase config export.
//
// ОГРАНИЧЕНИЕ: ibcmd config export поддерживает только файловые базы (/F path).
// Для серверных баз (/S server\dbname) используйте 1cv8 реализацию:
// config.implementations.config_export = "1cv8"
func (e *ExporterIbcmd) Export(ctx context.Context, opts ExportOptions) (*ExportResult, error) {
	start := time.Now()
	log := slog.Default().With(slog.String("operation", "Export"), slog.String("tool", "ibcmd"))

	if e.binIbcmd == "" {
		return nil, fmt.Errorf("путь к ibcmd не указан")
	}

	// H-1/H-3 fix: ibcmd config export не поддерживает серверные базы
	if isServerConnectString(opts.ConnectString) {
		return nil, fmt.Errorf("ibcmd config export не поддерживает серверные базы (/S ...): используйте config.implementations.config_export=\"1cv8\"")
	}

	r := runner.Runner{}
	r.RunString = e.binIbcmd
	r.Params = []string{
		"infobase", "config", "export",
		fmt.Sprintf("--db-path=%s", extractDbPath(opts.ConnectString)),
		fmt.Sprintf("--path=%s", opts.OutputPath),
	}

	// Добавляем расширение если указано
	if opts.Extension != "" {
		r.Params = append(r.Params, fmt.Sprintf("--extension=%s", opts.Extension))
	}

	// Устанавливаем таймаут
	ctxWithTimeout := ctx
	var cancel context.CancelFunc
	if opts.Timeout > 0 {
		ctxWithTimeout, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	log.Info("Запуск выгрузки конфигурации через ibcmd",
		slog.String("output_path", opts.OutputPath),
		slog.String("extension", opts.Extension))

	// Выполняем команду
	_, err := r.RunCommand(ctxWithTimeout, log)

	result := &ExportResult{
		OutputPath: opts.OutputPath,
		DurationMs: time.Since(start).Milliseconds(),
	}

	// Извлекаем сообщения из вывода
	output := string(r.ConsoleOut)
	result.Messages = extractMessages(output)

	// ibcmd успешен если нет ошибки
	if err == nil {
		result.Success = true
		log.Info("Выгрузка конфигурации через ibcmd завершена успешно",
			slog.Int64("duration_ms", result.DurationMs))
	} else {
		result.Success = false
		log.Error("Ошибка выгрузки конфигурации через ibcmd",
			slog.String("output", trimOutput(output)),
			slog.Int64("duration_ms", result.DurationMs))
	}

	return result, err
}

// extractDbPath извлекает путь к БД из connect string для файловых баз.
// Поддерживает форматы: "/F path" и "path".
//
// ВАЖНО: Эта функция предназначена только для файловых баз.
// Для серверных баз используйте isServerConnectString() для проверки.
func extractDbPath(connectString string) string {
	connectString = strings.TrimSpace(connectString)
	if strings.HasPrefix(connectString, "/F ") {
		return strings.TrimPrefix(connectString, "/F ")
	}
	return connectString
}

// isServerConnectString проверяет, является ли строка подключения серверной.
// Серверная строка начинается с "/S " или содержит "Srvr=".
//
// Примеры серверных строк:
//   - "/S server\dbname"
//   - "/S server\dbname /N user /P pass"
//   - `Srvr="server";Ref="dbname"`
func isServerConnectString(connectString string) bool {
	s := strings.TrimSpace(connectString)
	return strings.HasPrefix(s, "/S ") || strings.Contains(s, "Srvr=")
}
