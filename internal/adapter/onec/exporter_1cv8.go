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

// Exporter1cv8 реализует ConfigExporter через 1cv8 DESIGNER.
// Использует команду /DumpCfg для выгрузки конфигурации в XML формат.
type Exporter1cv8 struct {
	bin1cv8 string
	workDir string
	tmpDir  string
}

// Compile-time проверка интерфейса.
var _ ConfigExporter = (*Exporter1cv8)(nil)

// NewExporter1cv8 создаёт новый Exporter1cv8 с указанными путями.
func NewExporter1cv8(bin1cv8, workDir, tmpDir string) *Exporter1cv8 {
	return &Exporter1cv8{
		bin1cv8: bin1cv8,
		workDir: workDir,
		tmpDir:  tmpDir,
	}
}

// Export выгружает конфигурацию через 1cv8 DESIGNER /DumpCfg.
func (e *Exporter1cv8) Export(ctx context.Context, opts ExportOptions) (*ExportResult, error) {
	start := time.Now()
	log := slog.Default().With(slog.String("operation", "Export"), slog.String("tool", "1cv8"))

	if e.bin1cv8 == "" {
		return nil, fmt.Errorf("путь к 1cv8 не указан")
	}

	r := runner.Runner{}
	r.TmpDir = e.tmpDir
	r.WorkDir = e.workDir
	r.RunString = e.bin1cv8

	// Формируем параметры команды
	r.Params = []string{
		"@",        // Использовать параметр-файл
		"DESIGNER", // Режим дизайнера
	}
	r.Params = append(r.Params, opts.ConnectString)
	r.Params = append(r.Params, "/DumpCfg")
	r.Params = append(r.Params, opts.OutputPath)

	// Добавляем расширение если указано
	if opts.Extension != "" {
		r.Params = append(r.Params, "-Extension")
		r.Params = append(r.Params, opts.Extension)
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

	log.Info("Запуск выгрузки конфигурации",
		slog.String("output_path", opts.OutputPath),
		slog.String("extension", opts.Extension))

	// Выполняем команду
	_, err := r.RunCommand(ctxWithTimeout, log)

	result := &ExportResult{
		OutputPath: opts.OutputPath,
		DurationMs: time.Since(start).Milliseconds(),
	}

	// Извлекаем сообщения из вывода
	output := string(r.FileOut)
	result.Messages = extractMessages(output)

	// Проверка успеха по сообщениям (русский и английский варианты)
	if err == nil && (strings.Contains(output, "Выгрузка конфигурации завершена") ||
		strings.Contains(output, "Configuration dump completed") ||
		strings.Contains(output, "Выгрузка завершена") ||
		len(output) < 100) { // Успешный вывод обычно минимален
		result.Success = true
		log.Info("Выгрузка конфигурации завершена успешно",
			slog.Int64("duration_ms", result.DurationMs))
	} else {
		result.Success = false
		if err == nil {
			err = fmt.Errorf("выгрузка не завершена успешно: %s", trimOutput(output))
		}
		log.Error("Ошибка выгрузки конфигурации",
			slog.String("output", trimOutput(output)),
			slog.Int64("duration_ms", result.DurationMs))
	}

	return result, err
}
