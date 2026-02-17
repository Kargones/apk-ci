// Package createtempdbhandler реализует NR-команду nr-create-temp-db
// для создания временной локальной базы данных 1C с расширениями.
package createtempdbhandler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/onec"
	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/progress"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
	errhandler "github.com/Kargones/apk-ci/internal/command/handlers/shared"
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

// generateDbPath генерирует уникальный путь для временной БД.
// Валидирует что путь находится в разрешённой директории.
// Использует наносекундную точность и случайный суффикс для предотвращения коллизий.
func (h *CreateTempDbHandler) generateDbPath(cfg *config.Config) (string, error) {
	// Используем TmpDir из конфигурации или дефолтную директорию
	baseDir := constants.TempDir
	if cfg.TmpDir != "" {
		baseDir = cfg.TmpDir
	}

	// H2 fix: валидация что baseDir — безопасный путь (должен быть в /tmp или TempDir)
	cleanPath := filepath.Clean(baseDir)

	// Разрешаем symlinks для предотвращения path traversal через symlinks
	resolvedPath, err := filepath.EvalSymlinks(cleanPath)
	if err == nil {
		// Путь существует и symlinks разрешены — используем разрешённый путь для проверки
		cleanPath = resolvedPath
	}
	// Если ошибка (путь не существует) — проверяем оригинальный cleanPath

	allowedPrefixes := []string{"/tmp", constants.TempDir, os.TempDir()}
	isAllowed := false
	for _, prefix := range allowedPrefixes {
		resolvedPrefix := filepath.Clean(prefix)
		// Также пытаемся разрешить symlinks в prefix
		if rp, err := filepath.EvalSymlinks(resolvedPrefix); err == nil {
			resolvedPrefix = rp
		}
		if strings.HasPrefix(cleanPath, resolvedPrefix) {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		return "", fmt.Errorf("путь %s не находится в разрешённой директории (допустимы: /tmp, %s)", baseDir, constants.TempDir)
	}

	// Наносекундная точность + случайный суффикс для гарантии уникальности
	// Формат: temp_db_YYYYMMDD_HHMMSS_NNNNNNNNN_RRRRRRRR
	now := time.Now()
	randomSuffix := make([]byte, 4)
	if _, err := rand.Read(randomSuffix); err != nil {
		// Fallback на наносекунды если crypto/rand недоступен
		randomSuffix = []byte(fmt.Sprintf("%04d", now.Nanosecond()%10000))
	}
	timestamp := now.Format("20060102_150405") + fmt.Sprintf("_%09d_%s", now.Nanosecond(), hex.EncodeToString(randomSuffix))
	dbPath := filepath.Join(baseDir, fmt.Sprintf("temp_db_%s", timestamp))

	// H-3 fix: проверка максимальной длины пути
	if len(dbPath) > maxPathLength {
		return "", fmt.Errorf("путь слишком длинный (%d символов, максимум %d): %s", len(dbPath), maxPathLength, dbPath)
	}

	// M-6 fix: проверка что путь ещё не существует (защита от коллизий)
	if _, err := os.Stat(dbPath); err == nil {
		return "", fmt.Errorf("путь уже существует (коллизия): %s", dbPath)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("ошибка проверки пути: %w", err)
	}

	return dbPath, nil
}

// parseExtensions парсит список расширений из переменной окружения или конфигурации.
// Приоритет: BR_EXTENSIONS > cfg.AddArray.
// M-5 fix: ограничивает количество расширений до maxExtensions для защиты от DoS.
func (h *CreateTempDbHandler) parseExtensions(cfg *config.Config) []string {
	log := slog.Default()

	// Приоритет: BR_EXTENSIONS > cfg.AddArray
	extEnv := os.Getenv("BR_EXTENSIONS")
	if extEnv != "" {
		// Парсим через запятую и очищаем пробелы
		parts := strings.Split(extEnv, ",")
		var extensions []string
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				extensions = append(extensions, trimmed)
			}
		}
		// M-5 fix: лимит на количество расширений
		if len(extensions) > maxExtensions {
			log.Warn("Количество расширений превышает лимит, обрезано",
				slog.Int("requested", len(extensions)),
				slog.Int("max", maxExtensions))
			extensions = extensions[:maxExtensions]
		}
		return extensions
	}

	// Fallback на cfg.AddArray
	if len(cfg.AddArray) > 0 {
		log.Debug("BR_EXTENSIONS не задан, используется cfg.AddArray",
			slog.Any("extensions", cfg.AddArray))
		extensions := cfg.AddArray
		// M-5 fix: лимит на количество расширений
		if len(extensions) > maxExtensions {
			log.Warn("Количество расширений из cfg.AddArray превышает лимит, обрезано",
				slog.Int("requested", len(extensions)),
				slog.Int("max", maxExtensions))
			extensions = extensions[:maxExtensions]
		}
		return extensions
	}

	return nil
}

// getTimeout возвращает таймаут для операции создания БД.
func (h *CreateTempDbHandler) getTimeout() time.Duration {
	// Проверяем явный таймаут через BR_TIMEOUT_MIN
	if timeoutMinStr := os.Getenv("BR_TIMEOUT_MIN"); timeoutMinStr != "" {
		if timeoutMin, err := strconv.Atoi(timeoutMinStr); err == nil && timeoutMin > 0 {
			return time.Duration(timeoutMin) * time.Minute
		}
	}
	return defaultTimeout
}

// getTTLHours возвращает TTL в часах из переменной окружения.
func (h *CreateTempDbHandler) getTTLHours() int {
	ttlStr := os.Getenv("BR_TTL_HOURS")
	if ttlStr == "" {
		return 0
	}
	ttl, err := strconv.Atoi(ttlStr)
	if err != nil || ttl < 0 {
		return 0
	}
	return ttl
}

// getOrCreateClient возвращает существующий или создаёт новый клиент.
func (h *CreateTempDbHandler) getOrCreateClient(cfg *config.Config) onec.TempDatabaseCreator {
	if h.dbCreator != nil {
		return h.dbCreator
	}
	return onec.NewTempDbCreator()
}

// writeTTLMetadata записывает метаданные TTL в файл рядом с БД.
func (h *CreateTempDbHandler) writeTTLMetadata(dbPath string, ttlHours int, createdAt time.Time) error {
	metadata := TTLMetadata{
		CreatedAt: createdAt,
		TTLHours:  ttlHours,
		ExpiresAt: createdAt.Add(time.Duration(ttlHours) * time.Hour),
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("сериализация TTL metadata: %w", err)
	}

	ttlPath := dbPath + ".ttl"

	// Проверяем существование родительской директории
	parentDir := filepath.Dir(ttlPath)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		if mkdirErr := os.MkdirAll(parentDir, 0755); mkdirErr != nil {
			return fmt.Errorf("создание директории для TTL: %w", mkdirErr)
		}
	}

	// Используем 0600 для метаданных (только владелец может читать/писать)
	if err := os.WriteFile(ttlPath, data, 0600); err != nil {
		return fmt.Errorf("запись TTL metadata: %w", err)
	}

	return nil
}

// createProgress создаёт progress bar для отображения прогресса создания БД.
// M-4 fix: используем общий helper progress.NewIndeterminate().
func (h *CreateTempDbHandler) createProgress() progress.Progress {
	return progress.NewIndeterminate()
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

// buildPlan создаёт план операций для предпросмотра.
// Используется в dry-run, plan-only и verbose режимах.
// Story 7.3: извлечено из executeDryRun для переиспользования.
func (h *CreateTempDbHandler) buildPlan(
	cfg *config.Config,
	dbPath string,
	extensions []string,
	timeout time.Duration,
	ttlHours int,
) *output.DryRunPlan {
	extensionsStr := extensionsOrNone(extensions)

	steps := []output.PlanStep{
		{
			Order:     1,
			Operation: "Валидация конфигурации",
			Parameters: map[string]any{
				"ibcmd_path": cfg.AppConfig.Paths.BinIbcmd,
			},
			ExpectedChanges: []string{"Нет изменений — только валидация"},
		},
		{
			Order:     2,
			Operation: "Генерация пути к временной базе",
			Parameters: map[string]any{
				"db_path": dbPath,
			},
			ExpectedChanges: []string{
				fmt.Sprintf("Будет создана директория: %s", dbPath),
			},
		},
		{
			Order:     3,
			Operation: "Создание базы данных",
			Parameters: map[string]any{
				"db_path":    dbPath,
				"extensions": extensionsStr,
				"timeout":    timeout.String(),
			},
			ExpectedChanges: []string{
				"Будет создана пустая информационная база",
			},
		},
	}

	if len(extensions) > 0 {
		steps = append(steps, output.PlanStep{
			Order:     4,
			Operation: "Добавление расширений",
			Parameters: map[string]any{
				"extensions": extensionsStr,
			},
			ExpectedChanges: []string{
				fmt.Sprintf("Расширения %s будут добавлены в базу", extensionsStr),
			},
		})
	}

	if ttlHours > 0 {
		steps = append(steps, output.PlanStep{
			Order:     len(steps) + 1,
			Operation: "Создание TTL metadata",
			Parameters: map[string]any{
				"ttl_hours": ttlHours,
				"ttl_file":  dbPath + ".ttl",
			},
			ExpectedChanges: []string{
				fmt.Sprintf("База будет удалена через %d часов", ttlHours),
			},
		})
	}

	summary := fmt.Sprintf("Создание временной БД: %s", dbPath)
	if len(extensions) > 0 {
		summary += fmt.Sprintf(" с расширениями: %s", extensionsStr)
	}

	return dryrun.BuildPlanWithSummary(constants.ActNRCreateTempDb, steps, summary)
}

// executeDryRun выполняет dry-run режим для команды nr-create-temp-db.
// AC-1: Возвращает план действий БЕЗ выполнения.
// AC-2: План содержит операции, параметры, ожидаемые изменения.
// AC-8: НЕ вызывается client.CreateTempDB().
func (h *CreateTempDbHandler) executeDryRun(
	cfg *config.Config,
	dbPath string,
	extensions []string,
	timeout time.Duration,
	ttlHours int,
	format, traceID string,
	start time.Time,
) error {
	plan := h.buildPlan(cfg, dbPath, extensions, timeout, ttlHours)
	return output.WriteDryRunResult(os.Stdout, format, constants.ActNRCreateTempDb, traceID, constants.APIVersion, start, plan)
}

// extensionsOrNone возвращает строку расширений или "нет" если пусто.
func extensionsOrNone(extensions []string) string {
	if len(extensions) == 0 {
		return "нет"
	}
	return strings.Join(extensions, ", ")
}

// writePlanOnlyResult и writeDryRunResult перенесены в output.WritePlanOnlyResult/WriteDryRunResult (CR-7.3 #2, #3).
