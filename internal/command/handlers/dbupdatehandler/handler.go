// Package dbupdatehandler реализует NR-команду nr-dbupdate
// для обновления структуры базы данных по конфигурации 1C.
package dbupdatehandler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/onec"
	"github.com/Kargones/apk-ci/internal/adapter/onec/rac"
	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/progress"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
	errhandler "github.com/Kargones/apk-ci/internal/command/handlers/shared"
)

// Коды ошибок для команды nr-dbupdate.
const (
	ErrDbUpdateValidation       = "DBUPDATE.VALIDATION_FAILED"
	ErrDbUpdateConfig           = "DBUPDATE.CONFIG_ERROR"
	ErrDbUpdateFailed           = "DBUPDATE.UPDATE_FAILED"
	ErrDbUpdateSecondPassFailed = "DBUPDATE.SECOND_PASS_FAILED"
	ErrDbUpdateTimeout          = "DBUPDATE.TIMEOUT"
	ErrDbUpdateAutoDeps         = "DBUPDATE.AUTO_DEPS_FAILED"

	// defaultTimeout — таймаут по умолчанию для обновления БД.
	defaultTimeout = 30 * time.Minute

	// maxMessages — максимальное количество сообщений в результате (C1 fix).
	maxMessages = 100

	// cleanupTimeout — таймаут для cleanup операций (M1 fix).
	// H4 note: Этот таймаут покрывает 3 RAC вызова в disableServiceModeIfNeeded:
	// GetClusterInfo + GetInfobaseInfo + DisableServiceMode.
	// При необходимости увеличить, учитывая сетевые задержки.
	cleanupTimeout = 30 * time.Second

	// racOperationTimeout — таймаут для операций RAC (H3 fix).
	racOperationTimeout = 60 * time.Second
)

func RegisterCmd() {
	command.RegisterWithAlias(&DbUpdateHandler{}, constants.ActDbupdate)
}

// DbUpdateData содержит данные ответа о результате обновления.
type DbUpdateData struct {
	// InfobaseName — имя информационной базы
	InfobaseName string `json:"infobase_name"`
	// Extension — имя расширения (если обновлялось)
	Extension string `json:"extension,omitempty"`
	// Success — успешно ли обновление
	Success bool `json:"success"`
	// Messages — сообщения от платформы
	Messages []string `json:"messages,omitempty"`
	// DurationMs — время выполнения в миллисекундах
	DurationMs int64 `json:"duration_ms"`
	// AutoDeps — был ли использован режим автоматического управления зависимостями
	AutoDeps bool `json:"auto_deps"`
}

// writeText выводит результат обновления в человекочитаемом формате.
func (d *DbUpdateData) writeText(w io.Writer) error {
	status := "✅ Обновление завершено успешно"
	if !d.Success {
		status = "❌ Обновление завершено с ошибками"
	}

	_, err := fmt.Fprintf(w, "%s\n", status)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "База данных: %s\n", d.InfobaseName)
	if err != nil {
		return err
	}

	if d.Extension != "" {
		_, err = fmt.Fprintf(w, "Расширение: %s\n", d.Extension)
		if err != nil {
			return err
		}
	}

	duration := time.Duration(d.DurationMs) * time.Millisecond
	_, err = fmt.Fprintf(w, "Время выполнения: %v\n", duration.Round(time.Millisecond))
	if err != nil {
		return err
	}

	if d.AutoDeps {
		_, err = fmt.Fprintf(w, "Auto-deps: включён\n")
		if err != nil {
			return err
		}
	}

	if len(d.Messages) > 0 {
		_, err = fmt.Fprintf(w, "\nСообщения:\n")
		if err != nil {
			return err
		}
		for _, msg := range d.Messages {
			_, err = fmt.Fprintf(w, "  - %s\n", msg)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// DbUpdateHandler обрабатывает команду nr-dbupdate.
type DbUpdateHandler struct {
	// oneCClient — клиент 1C для тестирования; если nil — создаётся реальный клиент
	oneCClient onec.DatabaseUpdater
	// racClient — клиент RAC для тестирования; если nil — создаётся реальный клиент
	racClient rac.Client
	// verbosePlan — план операций для verbose режима (Story 7.3), добавляется в JSON результат
	verbosePlan *output.DryRunPlan
}

// Name возвращает имя команды.
func (h *DbUpdateHandler) Name() string {
	return constants.ActNRDbupdate
}

// Description возвращает описание команды для вывода в help.
// AC-10: включает описание BR_DRY_RUN для документации.
func (h *DbUpdateHandler) Description() string {
	return "Обновить структуру базы данных по конфигурации. " +
		"Переменная BR_DRY_RUN=true выводит план операций без выполнения"
}

// Execute выполняет команду nr-dbupdate.
func (h *DbUpdateHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")
	log := slog.Default().With(
		slog.String("trace_id", traceID),
		slog.String("command", constants.ActNRDbupdate),
	)

	// 1. Проверка отмены контекста перед началом работы
	if err := ctx.Err(); err != nil {
		log.Warn("Context отменён до начала выполнения", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start, ErrDbUpdateFailed,
			"операция отменена: "+err.Error())
	}

	// 2. Валидация
	if cfg == nil || cfg.InfobaseName == "" {
		log.Error("Не указано имя информационной базы")
		return h.writeError(format, traceID, start, ErrDbUpdateValidation,
			"BR_INFOBASE_NAME обязателен")
	}

	log = log.With(slog.String("infobase", cfg.InfobaseName))
	log.Info("Запуск обновления структуры базы данных")

	// 3. Получение информации о БД
	dbInfo := cfg.GetDatabaseInfo(cfg.InfobaseName)
	if dbInfo == nil {
		log.Error("Информационная база не найдена в конфигурации", slog.String("infobase", cfg.InfobaseName))
		return h.writeError(format, traceID, start, ErrDbUpdateConfig,
			fmt.Sprintf("информационная база '%s' не найдена в конфигурации", cfg.InfobaseName))
	}

	// 4. Проверка пути к 1cv8 до начала операций
	if cfg.AppConfig == nil || cfg.AppConfig.Paths.Bin1cv8 == "" {
		log.Error("Путь к 1cv8 не указан в конфигурации")
		return h.writeError(format, traceID, start, ErrDbUpdateConfig,
			"путь к 1cv8 не указан в конфигурации (app.yaml:paths.bin1cv8)")
	}

	// 5. Валидация WorkDir и TmpDir
	if cfg.WorkDir == "" {
		log.Warn("WorkDir не указан, используется системная временная директория")
	}
	if cfg.TmpDir == "" {
		log.Warn("TmpDir не указан, используется системная временная директория")
	}

	// 6. Построение строки подключения
	connectString := h.buildConnectString(dbInfo, cfg)

	// 7. Получение расширения из переменной окружения
	extension := os.Getenv("BR_EXTENSION")

	// 8. Определение таймаута
	timeout := h.getTimeout()

	// === РЕЖИМЫ ПРЕДПРОСМОТРА (порядок приоритетов!) ===

	// 1. Dry-run: план без выполнения (высший приоритет)
	if dryrun.IsDryRun() {
		log.Info("Dry-run режим: построение плана")
		return h.executeDryRun(cfg, dbInfo, connectString, extension, timeout, format, traceID, start)
	}

	// 2. Plan-only: показать план, не выполнять (Story 7.3 AC-1)
	if dryrun.IsPlanOnly() {
		log.Info("Plan-only режим: отображение плана операций")
		plan := h.buildPlan(cfg, dbInfo, connectString, extension, timeout)
		return output.WritePlanOnlyResult(os.Stdout, format, constants.ActNRDbupdate, traceID, constants.APIVersion, start, plan)
	}

	// 3. Verbose: показать план, ПОТОМ выполнить (Story 7.3 AC-4)
	if dryrun.IsVerbose() {
		log.Info("Verbose режим: отображение плана перед выполнением")
		plan := h.buildPlan(cfg, dbInfo, connectString, extension, timeout)
		if format != output.FormatJSON {
			if err := plan.WritePlanText(os.Stdout); err != nil {
				log.Warn("Не удалось вывести план операций", slog.String("error", err.Error()))
			}
			fmt.Fprintln(os.Stdout)
		}
		// Сохраняем план для добавления в JSON результат
		h.verbosePlan = plan
	}
	// Verbose fall-through by design: план отображён, продолжаем реальное выполнение

	// 9. Создание клиента (или использование mock)
	client := h.getOrCreateOneCClient(cfg)

	// 10. Проверка режима auto-deps
	autoDeps := os.Getenv("BR_AUTO_DEPS") == "true"
	// weEnabledServiceMode = true означает, что МЫ включили сервисный режим (он был выключен)
	// weEnabledServiceMode = false означает, что режим УЖЕ был включён до нас
	var weEnabledServiceMode bool
	var racClient rac.Client

	if autoDeps {
		log.Info("Auto-deps режим включён")
		racClient = h.getOrCreateRacClient(cfg, dbInfo, log)
		if racClient != nil {
			// H3 fix: отдельный таймаут для RAC операций
			racCtx, racCancel := context.WithTimeout(ctx, racOperationTimeout)
			var smErr error
			var alreadyEnabled bool
			alreadyEnabled, smErr = h.enableServiceModeIfNeeded(racCtx, cfg, racClient, log)
			weEnabledServiceMode = !alreadyEnabled // мы включили, если НЕ был уже включён
			racCancel()
			if smErr != nil {
				log.Warn("Не удалось включить сервисный режим", slog.String("error", smErr.Error()))
				// Продолжаем без auto-deps
				autoDeps = false
			}
		} else {
			log.Warn("RAC клиент недоступен, продолжаем без auto-deps")
			autoDeps = false
		}
	}

	// 11. Defer для гарантированного восстановления service mode
	// Отключаем только если МЫ включили режим
	defer func() {
		if autoDeps && weEnabledServiceMode && racClient != nil {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), cleanupTimeout)
			defer cleanupCancel()
			h.disableServiceModeIfNeeded(cleanupCtx, cfg, racClient, log)
		}
	}()

	// 12. Progress bar
	prog := h.createProgress()
	prog.Start("Обновление структуры базы данных...")
	defer prog.Finish()

	// 13. Выполнение обновления
	opts := onec.UpdateOptions{
		ConnectString: connectString,
		Extension:     extension,
		Timeout:       timeout,
		Bin1cv8:       cfg.AppConfig.Paths.Bin1cv8,
	}

	result, err := client.UpdateDBCfg(ctx, opts)
	if err != nil {
		log.Error("Ошибка обновления структуры БД", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start, ErrDbUpdateFailed, err.Error())
	}

	// M-4 fix: применяем лимит сообщений после первого прохода тоже
	if len(result.Messages) > maxMessages {
		result.Messages = result.Messages[:maxMessages]
		log.Warn("Количество сообщений превысило лимит после первого прохода, обрезано", slog.Int("max", maxMessages))
	}

	// 14. Для расширений — второй проход (особенность платформы 1C)
	if extension != "" {
		log.Info("Первый проход обновления расширения завершён",
			slog.String("extension", extension),
			slog.Int64("first_pass_duration_ms", result.DurationMs),
			slog.Bool("first_pass_success", result.Success))

		if !result.Success {
			log.Warn("Первый проход расширения не успешен, но продолжаем второй проход",
				slog.String("extension", extension))
		}

		// Проверяем отмену контекста перед вторым проходом
		if ctx.Err() != nil {
			log.Error("Контекст отменён перед вторым проходом", slog.String("error", ctx.Err().Error()))
			return h.writeError(format, traceID, start, ErrDbUpdateFailed,
				fmt.Sprintf("операция отменена перед вторым проходом: %s", ctx.Err().Error()))
		}

		prog.Update(0, "Второй проход обновления расширения...")
		log.Info("Второй проход обновления для расширения", slog.String("extension", extension))
		result2, err := client.UpdateDBCfg(ctx, opts)
		if err != nil {
			log.Error("Ошибка второго прохода обновления", slog.String("error", err.Error()))
			return h.writeError(format, traceID, start, ErrDbUpdateSecondPassFailed,
				fmt.Sprintf("ошибка второго прохода обновления расширения: %s", err.Error()))
		}
		// Объединяем результаты с лимитом на количество сообщений
		result.Messages = append(result.Messages, result2.Messages...)
		if len(result.Messages) > maxMessages {
			result.Messages = result.Messages[:maxMessages]
			log.Warn("Количество сообщений превысило лимит, обрезано", slog.Int("max", maxMessages))
		}
		result.DurationMs += result2.DurationMs
	}

	duration := time.Since(start)
	if result.Success {
		log.Info("Обновление завершено успешно",
			slog.Duration("duration", duration))
	} else {
		log.Warn("Обновление завершено с предупреждениями",
			slog.Duration("duration", duration),
			slog.Int("messages_count", len(result.Messages)))
	}

	// 15. Формирование данных ответа
	data := &DbUpdateData{
		InfobaseName: cfg.InfobaseName,
		Extension:    extension,
		Success:      result.Success,
		Messages:     result.Messages,
		DurationMs:   duration.Milliseconds(),
		AutoDeps:     autoDeps,
	}

	// Текстовый формат
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат
	resultOutput := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRDbupdate,
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
func (h *DbUpdateHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — человекочитаемый вывод ошибки
	if format != output.FormatJSON {
		return errhandler.HandleError(message, code)
	}

	// JSON формат — структурированный вывод
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRDbupdate,
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

// buildConnectString строит строку подключения к информационной базе.
//
// БЕЗОПАСНОСТЬ: Пароль включается в строку подключения, но runner.Runner
// автоматически использует файл параметров (@) и маскирует пароли в логах.
// Эта строка НЕ должна логироваться напрямую.
func (h *DbUpdateHandler) buildConnectString(dbInfo *config.DatabaseInfo, cfg *config.Config) string {
	// Формат: /S server\base /N user /P pass
	server := dbInfo.GetServer()

	connectString := fmt.Sprintf("/S %s\\%s", server, cfg.InfobaseName)

	// Добавляем пользователя (из AppConfig.Users.Db)
	user := ""
	if cfg.AppConfig != nil && cfg.AppConfig.Users.Db != "" {
		user = cfg.AppConfig.Users.Db
	}
	if user != "" {
		connectString += fmt.Sprintf(" /N %s", user)
	}

	// Добавляем пароль (из SecretConfig.Passwords.Db)
	pass := ""
	if cfg.SecretConfig != nil && cfg.SecretConfig.Passwords.Db != "" {
		pass = cfg.SecretConfig.Passwords.Db
	}
	if pass != "" {
		connectString += fmt.Sprintf(" /P %s", pass)
	}

	return connectString
}

// getTimeout возвращает таймаут для операции обновления.
func (h *DbUpdateHandler) getTimeout() time.Duration {
	// Проверяем явный таймаут через BR_TIMEOUT_MIN
	if timeoutMinStr := os.Getenv("BR_TIMEOUT_MIN"); timeoutMinStr != "" {
		if timeoutMin, err := strconv.Atoi(timeoutMinStr); err == nil && timeoutMin > 0 {
			return time.Duration(timeoutMin) * time.Minute
		}
	}
	return defaultTimeout
}

// getOrCreateOneCClient возвращает существующий или создаёт новый 1C клиент.
func (h *DbUpdateHandler) getOrCreateOneCClient(cfg *config.Config) onec.DatabaseUpdater {
	if h.oneCClient != nil {
		return h.oneCClient
	}
	return onec.NewUpdater(cfg.AppConfig.Paths.Bin1cv8, cfg.WorkDir, cfg.TmpDir)
}

// getOrCreateRacClient возвращает существующий или создаёт новый RAC клиент.
// M3 fix: dbInfo передаётся как параметр вместо повторного вызова GetDatabaseInfo.
// M3-v2 fix: унифицирована логика fallback сервера с buildConnectString.
// H2 fix: корректная обработка nil log и nil конфигураций.
func (h *DbUpdateHandler) getOrCreateRacClient(cfg *config.Config, dbInfo *config.DatabaseInfo, log *slog.Logger) rac.Client {
	if h.racClient != nil {
		return h.racClient
	}

	// H2 fix: используем default logger если не передан
	if log == nil {
		log = slog.Default()
	}

	server := dbInfo.GetServer()
	if server == "" {
		log.Warn("Сервер 1C не указан в конфигурации БД (ни OneServer, ни DbServer)")
		return nil
	}

	// H2 fix: проверяем nil AppConfig перед доступом к Paths
	if cfg.AppConfig == nil {
		log.Warn("AppConfig не указан в конфигурации")
		return nil
	}

	// Получаем путь к rac
	racPath := cfg.AppConfig.Paths.Rac
	if racPath == "" {
		log.Warn("Путь к RAC не указан в конфигурации")
		return nil
	}

	// Получаем учётные данные
	// H2 fix: AppConfig уже проверен на nil выше
	clusterUser := cfg.AppConfig.Users.Rac
	infobaseUser := cfg.AppConfig.Users.Db

	clusterPass := ""
	infobasePass := ""
	if cfg.SecretConfig != nil {
		clusterPass = cfg.SecretConfig.Passwords.Rac
		infobasePass = cfg.SecretConfig.Passwords.Db
	}

	client, err := rac.NewClient(rac.ClientOptions{
		RACPath:      racPath,
		Server:       server,
		ClusterUser:  clusterUser,
		ClusterPass:  clusterPass,
		InfobaseUser: infobaseUser,
		InfobasePass: infobasePass,
		Logger:       log,
	})
	if err != nil {
		log.Warn("Не удалось создать RAC клиент", slog.String("error", err.Error()))
		return nil
	}

	return client
}

// enableServiceModeIfNeeded проверяет и включает сервисный режим если нужно.
// Возвращает true если режим был уже включён.
func (h *DbUpdateHandler) enableServiceModeIfNeeded(ctx context.Context, cfg *config.Config, racClient rac.Client, log *slog.Logger) (bool, error) {
	// Получаем информацию о кластере
	clusterInfo, err := racClient.GetClusterInfo(ctx)
	if err != nil {
		return false, fmt.Errorf("не удалось получить информацию о кластере: %w", err)
	}

	// Получаем информацию о базе
	infobaseInfo, err := racClient.GetInfobaseInfo(ctx, clusterInfo.UUID, cfg.InfobaseName)
	if err != nil {
		return false, fmt.Errorf("не удалось получить информацию о базе: %w", err)
	}

	// Проверяем текущий статус
	status, err := racClient.GetServiceModeStatus(ctx, clusterInfo.UUID, infobaseInfo.UUID)
	if err != nil {
		return false, fmt.Errorf("не удалось проверить статус сервисного режима: %w", err)
	}

	if status.Enabled {
		log.Info("Сервисный режим уже включён")
		return true, nil
	}

	// Включаем сервисный режим
	log.Info("Включаем сервисный режим (auto-deps)")
	if err := racClient.EnableServiceMode(ctx, clusterInfo.UUID, infobaseInfo.UUID, false); err != nil {
		return false, fmt.Errorf("не удалось включить сервисный режим: %w", err)
	}

	return false, nil
}

// disableServiceModeIfNeeded отключает сервисный режим.
func (h *DbUpdateHandler) disableServiceModeIfNeeded(ctx context.Context, cfg *config.Config, racClient rac.Client, log *slog.Logger) {
	// Получаем информацию о кластере
	clusterInfo, err := racClient.GetClusterInfo(ctx)
	if err != nil {
		log.Error("Не удалось получить информацию о кластере для отключения service mode", slog.String("error", err.Error()))
		return
	}

	// Получаем информацию о базе
	infobaseInfo, err := racClient.GetInfobaseInfo(ctx, clusterInfo.UUID, cfg.InfobaseName)
	if err != nil {
		log.Error("Не удалось получить информацию о базе для отключения service mode", slog.String("error", err.Error()))
		return
	}

	// Отключаем сервисный режим
	log.Info("Отключаем сервисный режим (auto-deps)")
	if err := racClient.DisableServiceMode(ctx, clusterInfo.UUID, infobaseInfo.UUID); err != nil {
		log.Error("Не удалось отключить сервисный режим", slog.String("error", err.Error()))
	}
}

// createProgress создаёт progress bar для отображения прогресса обновления.
// M-4 fix: используем общий helper progress.NewIndeterminate().
func (h *DbUpdateHandler) createProgress() progress.Progress {
	return progress.NewIndeterminate()
}

// buildPlan создаёт план операций для предпросмотра.
// Используется в dry-run, plan-only и verbose режимах.
// Story 7.3: извлечено из executeDryRun для переиспользования.
func (h *DbUpdateHandler) buildPlan(
	cfg *config.Config,
	dbInfo *config.DatabaseInfo,
	connectString string,
	extension string,
	timeout time.Duration,
) *output.DryRunPlan {
	// Определяем режим auto-deps
	autoDeps := os.Getenv("BR_AUTO_DEPS") == "true"

	// Маскируем пароль в connect string (SECURITY!)
	maskedConnectString := dryrun.MaskPassword(connectString)

	steps := []output.PlanStep{
		{
			Order:     1,
			Operation: "Валидация конфигурации",
			Parameters: map[string]any{
				"infobase_name": cfg.InfobaseName,
				"extension":     valueOrNone(extension),
			},
			ExpectedChanges: []string{"Нет изменений — только валидация"},
		},
	}

	if autoDeps {
		steps = append(steps, output.PlanStep{
			Order:     2,
			Operation: "Включение сервисного режима (auto-deps)",
			Parameters: map[string]any{
				"server":   getServerFromDbInfo(dbInfo),
				"database": cfg.InfobaseName,
			},
			ExpectedChanges: []string{
				"Сервисный режим будет включён",
				"Пользователи будут отключены от базы",
			},
		})
	} else {
		steps = append(steps, output.PlanStep{
			Order:      2,
			Operation:  "Сервисный режим",
			Skipped:    true,
			SkipReason: "BR_AUTO_DEPS не включён",
		})
	}

	updateStep := output.PlanStep{
		Order:     3,
		Operation: "Обновление структуры базы данных",
		Parameters: map[string]any{
			"connect_string": maskedConnectString,
			"extension":      valueOrNone(extension),
			"timeout":        timeout.String(),
			"bin_1cv8":       cfg.AppConfig.Paths.Bin1cv8,
		},
		ExpectedChanges: []string{
			"Структура базы данных будет обновлена по конфигурации",
		},
	}
	if extension != "" {
		updateStep.ExpectedChanges = append(updateStep.ExpectedChanges,
			fmt.Sprintf("Расширение '%s' будет применено", extension),
			"Второй проход обновления для расширения",
		)
	}
	steps = append(steps, updateStep)

	if autoDeps {
		steps = append(steps, output.PlanStep{
			Order:     4,
			Operation: "Отключение сервисного режима (auto-deps)",
			Parameters: map[string]any{
				"server":   getServerFromDbInfo(dbInfo),
				"database": cfg.InfobaseName,
			},
			ExpectedChanges: []string{
				"Сервисный режим будет отключён",
				"Пользователи смогут подключиться к базе",
			},
		})
	}

	summary := fmt.Sprintf("Обновление %s", cfg.InfobaseName)
	if extension != "" {
		summary += fmt.Sprintf(" (расширение: %s)", extension)
	}

	return dryrun.BuildPlanWithSummary(constants.ActNRDbupdate, steps, summary)
}

// executeDryRun выполняет dry-run режим для команды nr-dbupdate.
// AC-1: Возвращает план действий БЕЗ выполнения.
// AC-2: План содержит операции, параметры, ожидаемые изменения.
// AC-8: НЕ вызываются 1cv8/ibcmd, RAC операции.
func (h *DbUpdateHandler) executeDryRun(
	cfg *config.Config,
	dbInfo *config.DatabaseInfo,
	connectString string,
	extension string,
	timeout time.Duration,
	format, traceID string,
	start time.Time,
) error {
	plan := h.buildPlan(cfg, dbInfo, connectString, extension, timeout)
	return output.WriteDryRunResult(os.Stdout, format, constants.ActNRDbupdate, traceID, constants.APIVersion, start, plan)
}

// valueOrNone возвращает значение или "нет" если пустое.
func valueOrNone(value string) string {
	if value == "" {
		return "нет"
	}
	return value
}

// getServerFromDbInfo возвращает сервер из DatabaseInfo.
func getServerFromDbInfo(dbInfo *config.DatabaseInfo) string {
	return dbInfo.GetServer()
}

// writePlanOnlyResult и writeDryRunResult перенесены в output.WritePlanOnlyResult/WriteDryRunResult (CR-7.3 #2, #3).
