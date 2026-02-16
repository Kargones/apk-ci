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

// Updater реализует интерфейс DatabaseUpdater для работы с 1cv8.
type Updater struct {
	bin1cv8 string
	workDir string
	tmpDir  string
}

// Compile-time проверка интерфейса.
var _ DatabaseUpdater = (*Updater)(nil)

// NewUpdater создаёт новый Updater с указанными путями.
func NewUpdater(bin1cv8, workDir, tmpDir string) *Updater {
	return &Updater{
		bin1cv8: bin1cv8,
		workDir: workDir,
		tmpDir:  tmpDir,
	}
}

// UpdateDBCfg выполняет команду 1cv8 DESIGNER /UpdateDBCfg.
func (u *Updater) UpdateDBCfg(ctx context.Context, opts UpdateOptions) (*UpdateResult, error) {
	start := time.Now()
	log := slog.Default().With(slog.String("operation", "UpdateDBCfg"))

	// Определяем путь к 1cv8
	bin1cv8 := opts.Bin1cv8
	if bin1cv8 == "" {
		bin1cv8 = u.bin1cv8
	}
	if bin1cv8 == "" {
		return nil, fmt.Errorf("путь к 1cv8 не указан")
	}

	// Создаём runner
	r := runner.Runner{}
	r.TmpDir = u.tmpDir
	r.WorkDir = u.workDir
	r.RunString = bin1cv8

	// Формируем параметры команды
	r.Params = append(r.Params, "@")           // Использовать параметр-файл
	r.Params = append(r.Params, "DESIGNER")    // Режим дизайнера
	r.Params = append(r.Params, opts.ConnectString)
	r.Params = append(r.Params, "/UpdateDBCfg") // Команда обновления

	// Добавляем расширение если указано
	if opts.Extension != "" {
		r.Params = append(r.Params, "-Extension")
		r.Params = append(r.Params, opts.Extension)
	}

	// Отключаем GUI параметры
	addDisableParam(&r)

	// Перенаправляем вывод в файл (runner автоматически создаст temp файл)
	r.Params = append(r.Params, "/Out")

	// Устанавливаем таймаут
	ctxWithTimeout := ctx
	var cancel context.CancelFunc
	if opts.Timeout > 0 {
		ctxWithTimeout, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	log.Info("Запуск обновления конфигурации",
		slog.String("extension", opts.Extension),
		slog.Duration("timeout", opts.Timeout))

	// Выполняем команду
	_, err := r.RunCommand(ctxWithTimeout, log)

	result := &UpdateResult{
		DurationMs: time.Since(start).Milliseconds(),
	}

	// Извлекаем сообщения из вывода
	output := string(r.FileOut)
	result.Messages = extractMessages(output)

	// Проверка успеха по сообщениям
	if strings.Contains(output, constants.SearchMsgBaseLoadOk) ||
		strings.Contains(output, constants.SearchMsgEmptyFile) {
		result.Success = true
		log.Info("Обновление конфигурации завершено успешно",
			slog.Int64("duration_ms", result.DurationMs))
	} else {
		result.Success = false
		if err == nil {
			err = fmt.Errorf("обновление не завершено успешно: %s", trimOutput(output))
		}
		log.Error("Ошибка обновления конфигурации",
			slog.String("output", trimOutput(output)),
			slog.Int64("duration_ms", result.DurationMs))
	}

	return result, err
}

// addDisableParam добавляет параметры для отключения GUI диалогов.
func addDisableParam(r *runner.Runner) {
	r.Params = append(r.Params, "/DisableStartupDialogs")
	r.Params = append(r.Params, "/DisableStartupMessages")
	r.Params = append(r.Params, "/DisableUnrecoverableErrorMessage")
	r.Params = append(r.Params, "/UC ServiceMode")
}

// maxExtractedMessages — лимит сообщений для предотвращения memory issues (M4 fix).
const maxExtractedMessages = 100

// extractMessages извлекает сообщения из вывода команды 1cv8.
//
// Входные данные:
//   - output: строка с выводом команды 1cv8, может содержать BOM (\ufeff),
//     различные окончания строк (LF, CRLF, CR), пустые строки
//
// Возвращаемые данные:
//   - срез строк с сообщениями, очищенными от BOM и пробелов
//   - максимум maxExtractedMessages (100) сообщений для предотвращения memory issues
//
// Особенности:
//   - M4 fix: добавлен лимит на количество сообщений
//   - M1-v2 fix: корректная обработка Windows (CRLF) и Unix (LF) окончаний строк
func extractMessages(output string) []string {
	var messages []string
	// M1-v2 fix: нормализуем окончания строк (CRLF -> LF, CR -> LF)
	output = strings.ReplaceAll(output, "\r\n", "\n")
	output = strings.ReplaceAll(output, "\r", "\n")
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && line != "\ufeff" { // Пропускаем BOM и пустые строки
			messages = append(messages, line)
			// M4 fix: не извлекаем больше maxExtractedMessages
			if len(messages) >= maxExtractedMessages {
				break
			}
		}
	}
	return messages
}

// trimOutput обрезает вывод до максимальной длины с учётом UTF-8.
func trimOutput(output string) string {
	const maxLength = 500
	runes := []rune(output)
	if len(runes) > maxLength {
		return string(runes[:maxLength]) + "..."
	}
	return output
}
