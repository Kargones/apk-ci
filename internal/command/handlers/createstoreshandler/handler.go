// Package createstoreshandler реализует NR-команду nr-create-stores
// для инициализации хранилищ конфигурации 1C для проекта и расширений.
package createstoreshandler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
)

// Compile-time interface check (AC-7).
var _ command.Handler = (*CreateStoresHandler)(nil)

func init() {
	// AC-5: Deprecated alias через DeprecatedBridge
	command.RegisterWithAlias(&CreateStoresHandler{}, constants.ActCreateStores)
}

// CreateStoresData содержит данные ответа о создании хранилищ (AC-3, AC-11).
type CreateStoresData struct {
	// StateChanged — изменилось ли состояние системы (AC-6)
	StateChanged bool `json:"state_changed"`
	// StoreRoot — корневой путь для хранилищ (AC-11)
	StoreRoot string `json:"store_root"`
	// MainStorePath — путь к основному хранилищу (AC-3)
	MainStorePath string `json:"main_store_path"`
	// ExtensionStores — результаты создания хранилищ расширений (AC-3)
	ExtensionStores []ExtensionStoreResult `json:"extension_stores,omitempty"`
	// DurationMs — длительность операции в миллисекундах (AC-3)
	DurationMs int64 `json:"duration_ms"`
}

// ExtensionStoreResult содержит результат создания хранилища расширения (AC-3).
type ExtensionStoreResult struct {
	// Name — имя расширения
	Name string `json:"name"`
	// Path — путь к хранилищу расширения
	Path string `json:"path"`
	// Success — успешно ли создано
	Success bool `json:"success"`
	// Error — ошибка создания (если была)
	Error string `json:"error,omitempty"`
}

// writeText выводит результат в человекочитаемом формате (AC-4, AC-11).
func (d *CreateStoresData) writeText(w io.Writer) error {
	// Статус операции
	statusText := "успешно"
	if !d.StateChanged {
		statusText = "без изменений"
	}

	_, err := fmt.Fprintf(w, "Создание хранилищ: %s\n", statusText)
	if err != nil {
		return err
	}

	// Summary (AC-11)
	if _, err = fmt.Fprintf(w, "\nСводка:\n"); err != nil {
		return err
	}

	if _, err = fmt.Fprintf(w, "  Корневой путь: %s\n", d.StoreRoot); err != nil {
		return err
	}

	if _, err = fmt.Fprintf(w, "  Основное хранилище: %s\n", d.MainStorePath); err != nil {
		return err
	}

	// Вывод расширений
	if len(d.ExtensionStores) > 0 {
		if _, err = fmt.Fprintf(w, "  Хранилища расширений:\n"); err != nil {
			return err
		}
		for _, ext := range d.ExtensionStores {
			status := "✓"
			if !ext.Success {
				status = "✗"
			}
			if _, err = fmt.Fprintf(w, "    %s %s: %s\n", status, ext.Name, ext.Path); err != nil {
				return err
			}
			if ext.Error != "" {
				if _, err = fmt.Fprintf(w, "      Ошибка: %s\n", ext.Error); err != nil {
					return err
				}
			}
		}
	} else {
		if _, err = fmt.Fprintf(w, "  Хранилища расширений: нет\n"); err != nil {
			return err
		}
	}

	return nil
}

// StoreCreator — интерфейс для создания хранилищ (AC-7: тестируемость).
type StoreCreator interface {
	// CreateStores создаёт хранилища для основной конфигурации и расширений
	CreateStores(l *slog.Logger, cfg *config.Config, storeRoot string, dbConnectString string, arrayAdd []string) error
}

// TempDbCreator — интерфейс для создания временной БД (AC-7: тестируемость).
type TempDbCreator interface {
	// CreateTempDb создаёт временную базу данных и возвращает строку подключения
	CreateTempDb(ctx context.Context, l *slog.Logger, cfg *config.Config) (string, error)
}

// CreateStoresHandler обрабатывает команду nr-create-stores.
type CreateStoresHandler struct {
	// storeCreator — опциональный создатель хранилищ (nil в production, mock в тестах)
	storeCreator StoreCreator
	// tempDbCreator — опциональный создатель временной БД (nil в production, mock в тестах)
	tempDbCreator TempDbCreator
}

// Name возвращает имя команды.
func (h *CreateStoresHandler) Name() string {
	return constants.ActNRCreateStores
}

// Description возвращает описание команды для вывода в help.
func (h *CreateStoresHandler) Description() string {
	return "Инициализация хранилищ конфигурации для проекта"
}

// Execute выполняет команду nr-create-stores (AC-1, AC-2, AC-6, AC-9, AC-10).
func (h *CreateStoresHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Story 7.3 AC-8: plan-only для команд без поддержки плана
	// Review #36: !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRCreateStores)
	}

	log := slog.Default().With(
		slog.String("trace_id", traceID),
		slog.String("command", constants.ActNRCreateStores),
	)

	// Progress: validating (AC-10)
	log.Info("validating: проверка параметров")

	// Валидация конфигурации (AC-9)
	if cfg == nil {
		log.Error("Конфигурация не указана")
		return h.writeError(format, traceID, start, "CONFIG.MISSING", "Конфигурация не указана")
	}

	if cfg.AppConfig == nil || cfg.AppConfig.Paths.Bin1cv8 == "" {
		log.Error("Не указан путь к 1cv8")
		return h.writeError(format, traceID, start, "CONFIG.BIN1CV8_MISSING",
			"Не указан путь к 1cv8 (AppConfig.Paths.Bin1cv8)")
	}

	if cfg.TmpDir == "" {
		log.Error("Не указана временная директория")
		return h.writeError(format, traceID, start, "CONFIG.TMPDIR_MISSING",
			"Не указана временная директория (TmpDir)")
	}

	if cfg.Owner == "" {
		log.Error("Не указан владелец репозитория")
		return h.writeError(format, traceID, start, "CONFIG.OWNER_MISSING",
			"Не указан владелец репозитория (Owner)")
	}

	if cfg.Repo == "" {
		log.Error("Не указано имя репозитория")
		return h.writeError(format, traceID, start, "CONFIG.REPO_MISSING",
			"Не указано имя репозитория (Repo)")
	}

	// Получаем список расширений из cfg.AddArray (AC-2)
	extensions := cfg.AddArray
	log.Info("Параметры валидации пройдены",
		slog.String("owner", cfg.Owner),
		slog.String("repo", cfg.Repo),
		slog.Int("extensions_count", len(extensions)))

	// Progress: creating_temp_db (AC-10)
	log.Info("creating_temp_db: создание временной базы данных")

	// Создание временной БД
	ctxPtr := ctx
	dbConnectString, err := h.createTempDb(ctxPtr, log, cfg)
	if err != nil {
		log.Error("Ошибка создания временной БД", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start, "ERR_TEMP_DB", err.Error())
	}

	// M-3 fix: не логируем connect_string — может содержать sensitive data
	log.Info("Временная БД создана")

	// Генерируем storeRoot (AC-3: совместимость с legacy)
	// Примечание: storeRoot всегда уникален благодаря timestamp, поэтому idempotency
	// на уровне пути не требуется — каждый вызов создаёт новые хранилища.
	storeRoot := filepath.Join(cfg.TmpDir, "store_"+time.Now().Format("20060102_150405"), cfg.Owner, cfg.Repo)
	mainStorePath := filepath.Join(storeRoot, "main")

	log.Info("creating_main_store: создание основного хранилища",
		slog.String("store_root", storeRoot),
		slog.String("main_store_path", mainStorePath))

	// Progress для расширений ПЕРЕД вызовом createStores (AC-10)
	if len(extensions) > 0 {
		log.Info("creating_extension_stores: подготовка к созданию хранилищ расширений",
			slog.Int("count", len(extensions)),
			slog.Any("extensions", extensions))
	}

	// Создание хранилищ через интерфейс
	err = h.createStores(log, cfg, storeRoot, dbConnectString, extensions)
	if err != nil {
		log.Error("Ошибка создания хранилищ", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start, "ERR_STORE_CREATE", err.Error())
	}

	// H-2 fix: Проверяем создание основного хранилища
	if _, statErr := os.Stat(mainStorePath); statErr != nil {
		log.Error("Основное хранилище не найдено после создания",
			slog.String("main_store_path", mainStorePath),
			slog.String("error", statErr.Error()))
		return h.writeError(format, traceID, start, "ERR_STORE_CREATE",
			fmt.Sprintf("основное хранилище не создано: %s", mainStorePath))
	}

	// Подготовка результатов для расширений
	extensionResults := make([]ExtensionStoreResult, 0, len(extensions))
	var failedExtensions []string

	// Проверяем результаты создания каждого расширения
	for _, extName := range extensions {
		extPath := filepath.Join(storeRoot, "add", extName)
		extResult := ExtensionStoreResult{
			Name: extName,
			Path: extPath,
		}

		// Проверяем существование директории расширения
		if _, statErr := os.Stat(extPath); statErr == nil {
			extResult.Success = true
			log.Info("creating_extension_store: хранилище расширения создано",
				slog.String("extension", extName),
				slog.String("path", extPath))
		} else {
			extResult.Success = false
			extResult.Error = "директория хранилища не найдена"
			failedExtensions = append(failedExtensions, extName)
			log.Warn("creating_extension_store: хранилище расширения не найдено",
				slog.String("extension", extName),
				slog.String("path", extPath))
		}

		extensionResults = append(extensionResults, extResult)
	}

	// H-3 fix: Ошибка если хоть одно расширение не создано (AC-9)
	if len(failedExtensions) > 0 {
		errMsg := fmt.Sprintf("не созданы хранилища расширений: %v", failedExtensions)
		log.Error("Частичный сбой создания хранилищ",
			slog.Any("failed_extensions", failedExtensions))
		return h.writeError(format, traceID, start, "ERR_STORE_CREATE", errMsg)
	}

	// Формирование результата (AC-3, AC-6, AC-11)
	durationMs := time.Since(start).Milliseconds()
	data := &CreateStoresData{
		StateChanged:    true, // Всегда true при успешном создании (storeRoot уникален)
		StoreRoot:       storeRoot,
		MainStorePath:   mainStorePath,
		ExtensionStores: extensionResults,
		DurationMs:      durationMs,
	}

	log.Info("Хранилища успешно созданы",
		slog.String("store_root", data.StoreRoot),
		slog.Int("extensions_count", len(extensionResults)),
		slog.Int64("duration_ms", durationMs))

	return h.writeSuccess(format, traceID, data)
}

// createTempDb создаёт временную БД через интерфейс или production реализацию.
func (h *CreateStoresHandler) createTempDb(ctx context.Context, l *slog.Logger, cfg *config.Config) (string, error) {
	if h.tempDbCreator != nil {
		return h.tempDbCreator.CreateTempDb(ctx, l, cfg)
	}
	// Production: используем app.CreateTempDbWrapper
	// Импортируем через lazy loading чтобы избежать cyclic dependency
	return createTempDbProduction(ctx, l, cfg)
}

// createStores создаёт хранилища через интерфейс или production реализацию.
func (h *CreateStoresHandler) createStores(l *slog.Logger, cfg *config.Config, storeRoot string, dbConnectString string, arrayAdd []string) error {
	if h.storeCreator != nil {
		return h.storeCreator.CreateStores(l, cfg, storeRoot, dbConnectString, arrayAdd)
	}
	// Production: используем store.CreateStores
	return createStoresProduction(l, cfg, storeRoot, dbConnectString, arrayAdd)
}

// writeSuccess выводит успешный результат (AC-3, AC-4).
// M-2 fix: используем durationMs из data вместо пересчёта.
func (h *CreateStoresHandler) writeSuccess(format, traceID string, data *CreateStoresData) error {
	// Текстовый формат (AC-4)
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат (AC-3)
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRCreateStores,
		Data:    data,
		Metadata: &output.Metadata{
			DurationMs: data.DurationMs, // M-2 fix: используем значение из data
			TraceID:    traceID,
			APIVersion: constants.APIVersion,
		},
	}

	writer := output.NewWriter(format)
	return writer.Write(os.Stdout, result)
}

// writeError выводит структурированную ошибку и возвращает error (AC-9).
// Для text формата НЕ выводим в stdout — main.go уже логирует ошибку.
func (h *CreateStoresHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — только возвращаем error, main.go выведет через logger
	if format != output.FormatJSON {
		return fmt.Errorf("%s: %s", code, message)
	}

	// JSON формат — структурированный вывод (AC-9)
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRCreateStores,
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
