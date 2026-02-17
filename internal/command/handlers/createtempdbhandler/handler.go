// Package createtempdbhandler реализует NR-команду nr-create-temp-db
// для создания временной локальной базы данных 1C с расширениями.
package createtempdbhandler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/onec"
	"github.com/Kargones/apk-ci/internal/command"
	errhandler "github.com/Kargones/apk-ci/internal/command/handlers/shared"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
)

// Коды ошибок для команды nr-create-temp-db.
const (
	ErrCreateTempDbValidation = "CREATETEMPDB.VALIDATION_FAILED"
	ErrCreateTempDbFailed     = "CREATETEMPDB.CREATE_FAILED"
	ErrExtensionAddFailed     = "CREATETEMPDB.EXTENSION_FAILED"
	ErrContextCancelled       = "CREATETEMPDB.CONTEXT_CANCELLED"

	// defaultTimeout — таймаут по умолчанию для создания БД.
	defaultTimeout = 30 * time.Minute

	// maxPathLength — максимальная длина пути к БД (255 символов для большинства FS).
	maxPathLength = 255

	// maxExtensions — максимальное количество расширений для предотвращения DoS.
	// M-5 fix: ограничиваем количество для защиты от злоупотреблений.
	maxExtensions = 50
)

func RegisterCmd() {
	command.RegisterWithAlias(&CreateTempDbHandler{}, constants.ActCreateTempDb)
}

// CreateTempDbData содержит результат создания временной БД для JSON вывода.
type CreateTempDbData struct {
	// ConnectString — строка подключения "/F <path>"
	ConnectString string `json:"connect_string"`
	// DbPath — полный путь к созданной БД
	DbPath string `json:"db_path"`
	// Extensions — список добавленных расширений
	Extensions []string `json:"extensions,omitempty"`
	// TTLHours — TTL в часах (0 = без TTL)
	TTLHours int `json:"ttl_hours,omitempty"`
	// CreatedAt — время создания в формате ISO 8601
	CreatedAt string `json:"created_at"`
	// DurationMs — время выполнения в миллисекундах
	DurationMs int64 `json:"duration_ms"`
}

// writeText выводит результат создания в человекочитаемом формате.
func (d *CreateTempDbData) writeText(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "Временная база данных создана\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Путь: %s\n", d.DbPath); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Строка подключения: %s\n", d.ConnectString); err != nil {
		return err
	}

	if len(d.Extensions) > 0 {
		if _, err := fmt.Fprintf(w, "Расширения: %s\n", strings.Join(d.Extensions, ", ")); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(w, "Расширения: нет\n"); err != nil {
			return err
		}
	}

	if d.TTLHours > 0 {
		if _, err := fmt.Fprintf(w, "TTL: %d часов\n", d.TTLHours); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "Время создания: %s\n", d.CreatedAt); err != nil {
		return err
	}

	// Корректный вывод для очень быстрых операций
	var durationStr string
	if d.DurationMs == 0 {
		durationStr = "< 1ms"
	} else {
		duration := time.Duration(d.DurationMs) * time.Millisecond
		durationStr = duration.Round(time.Millisecond).String()
	}
	if _, err := fmt.Fprintf(w, "Время выполнения: %s\n", durationStr); err != nil {
		return err
	}

	return nil
}

// TTLMetadata структура для файла .ttl.
type TTLMetadata struct {
	CreatedAt time.Time `json:"created_at"`
	TTLHours  int       `json:"ttl_hours"`
	ExpiresAt time.Time `json:"expires_at"`
}

// CreateTempDbHandler обрабатывает команду nr-create-temp-db.
type CreateTempDbHandler struct {
	// dbCreator — клиент для создания БД; если nil — создаётся реальный клиент
	dbCreator onec.TempDatabaseCreator
	// verbosePlan — план операций для verbose режима (Story 7.3), добавляется в JSON результат
	verbosePlan *output.DryRunPlan
}

// Name возвращает имя команды.
func (h *CreateTempDbHandler) Name() string {
	return constants.ActNRCreateTempDb
}

// Description возвращает описание команды для вывода в help.
// AC-10: включает описание BR_DRY_RUN для документации.
func (h *CreateTempDbHandler) Description() string {
	return "Создать временную локальную базу данных с расширениями. " +
		"Переменная BR_DRY_RUN=true выводит план операций без выполнения"
}

// Execute выполняет команду nr-create-temp-db.
func (h *CreateTempDbHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")
	log := slog.Default().With(
		slog.String("trace_id", traceID),
		slog.String("command", constants.ActNRCreateTempDb),
	)

	// H3 fix: проверка отмены context перед началом работы
	if err := ctx.Err(); err != nil {
		log.Warn("Context отменён до начала выполнения", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start, ErrContextCancelled,
			"операция отменена: "+err.Error())
	}

	// 1. Валидация конфигурации
	if cfg == nil {
		log.Error("Конфигурация не указана")
		return h.writeError(format, traceID, start, ErrCreateTempDbValidation,
			"конфигурация приложения не указана")
	}

	// Проверка путей к ibcmd
	if cfg.AppConfig == nil || cfg.AppConfig.Paths.BinIbcmd == "" {
		log.Error("Путь к ibcmd не указан в конфигурации")
		return h.writeError(format, traceID, start, ErrCreateTempDbValidation,
			"путь к ibcmd не указан в конфигурации (app.yaml:paths.binIbcmd)")
	}

	// Проверка существования и прав на выполнение ibcmd
	ibcmdInfo, err := os.Stat(cfg.AppConfig.Paths.BinIbcmd)
	if os.IsNotExist(err) {
		log.Error("Файл ibcmd не найден", slog.String("path", cfg.AppConfig.Paths.BinIbcmd))
		return h.writeError(format, traceID, start, ErrCreateTempDbValidation,
			fmt.Sprintf("файл ibcmd не найден: %s", cfg.AppConfig.Paths.BinIbcmd))
	}
	if err != nil {
		log.Error("Ошибка проверки файла ibcmd", slog.String("path", cfg.AppConfig.Paths.BinIbcmd), slog.String("error", err.Error()))
		return h.writeError(format, traceID, start, ErrCreateTempDbValidation,
			fmt.Sprintf("ошибка проверки файла ibcmd: %s", err.Error()))
	}
	// Проверка что файл исполняемый (хотя бы один execute bit)
	if ibcmdInfo.Mode()&0111 == 0 {
		log.Error("Файл ibcmd не является исполняемым", slog.String("path", cfg.AppConfig.Paths.BinIbcmd), slog.String("mode", ibcmdInfo.Mode().String()))
		return h.writeError(format, traceID, start, ErrCreateTempDbValidation,
			fmt.Sprintf("файл ibcmd не является исполняемым: %s (mode: %s)", cfg.AppConfig.Paths.BinIbcmd, ibcmdInfo.Mode().String()))
	}

	// 2. Генерация пути к БД (с H2 валидацией)
	dbPath, err := h.generateDbPath(cfg)
	if err != nil {
		log.Error("Небезопасный путь для временной БД", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start, ErrCreateTempDbValidation, err.Error())
	}
	log.Info("Генерация пути к временной БД", slog.String("path", dbPath))

	// M3 fix: создание родительской директории если не существует
	parentDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		log.Error("Не удалось создать директорию для БД", slog.String("path", parentDir), slog.String("error", err.Error()))
		return h.writeError(format, traceID, start, ErrCreateTempDbValidation,
			fmt.Sprintf("не удалось создать директорию %s: %s", parentDir, err.Error()))
	}

	// 3. Парсинг расширений
	extensions := h.parseExtensions(cfg)
	log.Info("Расширения для добавления", slog.Any("extensions", extensions))

	// 4. Получение таймаута
	timeout := h.getTimeout()

	// 5. Получение TTL
	ttlHours := h.getTTLHours()

	// === РЕЖИМЫ ПРЕДПРОСМОТРА (порядок приоритетов!) ===

	// 1. Dry-run: план без выполнения (высший приоритет)
	if dryrun.IsDryRun() {
		log.Info("Dry-run режим: построение плана")
		return h.executeDryRun(cfg, dbPath, extensions, timeout, ttlHours, format, traceID, start)
	}

	// 2. Plan-only: показать план, не выполнять (Story 7.3 AC-1)
	if dryrun.IsPlanOnly() {
		log.Info("Plan-only режим: отображение плана операций")
		plan := h.buildPlan(cfg, dbPath, extensions, timeout, ttlHours)
		return output.WritePlanOnlyResult(os.Stdout, format, constants.ActNRCreateTempDb, traceID, constants.APIVersion, start, plan)
	}

	// 3. Verbose: показать план, ПОТОМ выполнить (Story 7.3 AC-4)
	if dryrun.IsVerbose() {
		log.Info("Verbose режим: отображение плана перед выполнением")
		plan := h.buildPlan(cfg, dbPath, extensions, timeout, ttlHours)
		if format != output.FormatJSON {
			if err := plan.WritePlanText(os.Stdout); err != nil {
				log.Warn("Не удалось вывести план операций", slog.String("error", err.Error()))
			}
			fmt.Fprintln(os.Stdout)
		}
		h.verbosePlan = plan
	}
	// Verbose fall-through by design: план отображён, продолжаем реальное выполнение

	// 6. Создание клиента (или использование mock)
	client := h.getOrCreateClient(cfg)

	// 7. Progress bar для долгих операций (M4 fix)
	prog := h.createProgress()
	prog.Start("Создание временной базы данных...")
	defer prog.Finish()

	// 8. Проверка отмены context перед длительной операцией (H3 fix)
	if err := ctx.Err(); err != nil {
		log.Warn("Context отменён перед созданием БД", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start, ErrContextCancelled,
			"операция отменена: "+err.Error())
	}

	// 9. Выполнение создания БД
	opts := onec.CreateTempDBOptions{
		DbPath:     dbPath,
		Extensions: extensions,
		Timeout:    timeout,
		BinIbcmd:   cfg.AppConfig.Paths.BinIbcmd,
	}

	result, err := client.CreateTempDB(ctx, opts)
	if err != nil {
		log.Error("Ошибка создания временной БД", slog.String("error", err.Error()))
		// Используем errors.Is с fallback на строковую проверку для обратной совместимости
		errCode := ErrCreateTempDbFailed
		switch {
		case errors.Is(err, onec.ErrExtensionAdd):
			errCode = ErrExtensionAddFailed
		case errors.Is(err, onec.ErrContextCancelled):
			errCode = ErrContextCancelled
		case errors.Is(err, onec.ErrInfobaseCreate):
			errCode = ErrCreateTempDbFailed
		// Fallback на строковую проверку для обратной совместимости
		case strings.Contains(err.Error(), "расширения") || strings.Contains(err.Error(), "extension"):
			errCode = ErrExtensionAddFailed
		}
		return h.writeError(format, traceID, start, errCode, err.Error())
	}

	// 10. Создание TTL metadata (если указан)
	if ttlHours > 0 {
		if err := h.writeTTLMetadata(dbPath, ttlHours, result.CreatedAt); err != nil {
			log.Warn("Не удалось записать TTL metadata", slog.String("error", err.Error()))
			// Не прерываем выполнение — БД создана, TTL необязателен
		}
	}

	duration := time.Since(start)
	log.Info("Временная база данных создана",
		slog.String("path", result.DbPath),
		slog.Duration("duration", duration))

	// 11. Формирование данных ответа
	data := &CreateTempDbData{
		ConnectString: result.ConnectString,
		DbPath:        result.DbPath,
		Extensions:    result.Extensions,
		TTLHours:      ttlHours,
		CreatedAt:     result.CreatedAt.Format(time.RFC3339),
		DurationMs:    duration.Milliseconds(),
	}

	// Текстовый формат
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат
	resultOutput := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRCreateTempDb,
		Data:    data,
		Plan:    h.verbosePlan, // Story 7.3 AC-7: verbose JSON включает план
		Metadata: &output.Metadata{
			DurationMs: duration.Milliseconds(),
			TraceID:    traceID,
			APIVersion: constants.APIVersion,
		},
	}

	writer := output.NewWriter(format)
	return writer.Write(os.Stdout, resultOutput)
}

// writeError выводит структурированную ошибку и возвращает error.
func (h *CreateTempDbHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — человекочитаемый вывод ошибки
	if format != output.FormatJSON {
		return errhandler.HandleError(message, code)
	}

	// JSON формат — структурированный вывод
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRCreateTempDb,
		Error: &output.ErrorInfo{
			Code:    code,
			Message: message,
		},
		Metadata: &output.Metadata{
			DurationMs: time.Since(start).Milliseconds(),
			TraceID:    traceID,
			APIVersion: constants.APIVersion,
		},
	}

	writer := output.NewWriter(format)
	if writeErr := writer.Write(os.Stdout, result); writeErr != nil {
		slog.Default().Error("Не удалось записать JSON-ответ об ошибке",
			slog.String("trace_id", traceID),
			slog.String("error", writeErr.Error()))
	}

	return fmt.Errorf("%s: %s", code, message)
}
