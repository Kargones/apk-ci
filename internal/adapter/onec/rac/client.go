package rac

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/apperrors"
)

// Коды ошибок RAC клиента.
const (
	ErrRACExec     = "RAC.EXEC_FAILED"     // Ошибка запуска RAC-процесса
	ErrRACTimeout  = "RAC.TIMEOUT"         // Timeout при выполнении команды
	ErrRACParse    = "RAC.PARSE_FAILED"    // Ошибка парсинга вывода RAC
	ErrRACNotFound = "RAC.NOT_FOUND"       // Объект не найден (cluster, infobase)
	ErrRACSession  = "RAC.SESSION_FAILED"  // Ошибка операции с сессиями
	ErrRACVerify   = "RAC.VERIFY_FAILED"  // Несоответствие ожидаемого и фактического состояния
)

// ClientOptions — параметры для создания RAC клиента.
type ClientOptions struct {
	// RACPath — путь к исполняемому файлу rac
	RACPath string
	// Server — адрес сервера 1C
	Server string
	// Port — порт RAC (по умолчанию "1545")
	Port string
	// Timeout — таймаут выполнения команд (по умолчанию 30s)
	Timeout time.Duration
	// ClusterUser — администратор кластера (опционально)
	ClusterUser string
	// ClusterPass — пароль администратора кластера (опционально)
	ClusterPass string
	// InfobaseUser — пользователь информационной базы (опционально)
	InfobaseUser string
	// InfobasePass — пароль пользователя информационной базы (опционально)
	InfobasePass string
	// Logger — логгер (если nil — slog.Default())
	Logger *slog.Logger
}

// racClient — реализация интерфейса Client для работы с RAC CLI.
type racClient struct {
	racPath      string
	server       string
	port         string
	timeout      time.Duration
	clusterUser  string
	clusterPass  string
	infobaseUser string
	infobasePass string
	logger       *slog.Logger
}

// Compile-time проверка интерфейса.
var _ Client = (*racClient)(nil)

// NewClient создаёт новый RAC клиент с валидацией параметров.
func NewClient(opts ClientOptions) (Client, error) {
	if opts.RACPath == "" {
		return nil, apperrors.NewAppError(ErrRACExec, "путь к rac не указан", nil)
	}
	// Проверка существования файла RAC для раннего обнаружения ошибок конфигурации
	if _, err := os.Stat(opts.RACPath); err != nil {
		if os.IsNotExist(err) {
			return nil, apperrors.NewAppError(ErrRACExec,
				fmt.Sprintf("исполняемый файл rac не найден: %s", opts.RACPath), err)
		}
		return nil, apperrors.NewAppError(ErrRACExec,
			fmt.Sprintf("ошибка доступа к rac: %s", opts.RACPath), err)
	}
	if opts.Server == "" {
		return nil, apperrors.NewAppError(ErrRACExec, "адрес сервера не указан", nil)
	}
	if opts.Port == "" {
		opts.Port = "1545"
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &racClient{
		racPath:      opts.RACPath,
		server:       opts.Server,
		port:         opts.Port,
		timeout:      opts.Timeout,
		clusterUser:  opts.ClusterUser,
		clusterPass:  opts.ClusterPass,
		infobaseUser: opts.InfobaseUser,
		infobasePass: opts.InfobasePass,
		logger:       opts.Logger,
	}, nil
}

// executeRAC запускает RAC как subprocess с таймаутом.
func (c *racClient) executeRAC(ctx context.Context, args []string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	serverAddr := c.server + ":" + c.port
	fullArgs := make([]string, 0, len(args)+1)
	fullArgs = append(fullArgs, serverAddr)
	fullArgs = append(fullArgs, args...)

	c.logger.Debug("Выполнение RAC команды",
		"racPath", c.racPath,
		"args", sanitizeArgs(fullArgs),
	)

	cmd := exec.CommandContext(ctx, c.racPath, fullArgs...) //nolint:gosec // аргументы формируются программно, не из пользовательского ввода
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", apperrors.NewAppError(ErrRACTimeout,
				fmt.Sprintf("таймаут выполнения команды (%s)", c.timeout), err)
		}
		if ctx.Err() == context.Canceled {
			return "", apperrors.NewAppError(ErrRACExec, "выполнение отменено", err)
		}
		// Извлекаем stderr из ExitError для диагностики (stdout и stderr разделены)
		// Применяем sanitizeString для предотвращения утечки credentials в сообщениях об ошибках
		errMsg := "ошибка выполнения RAC"
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := sanitizeString(strings.TrimSpace(string(exitErr.Stderr)))
			if stderr != "" {
				errMsg = fmt.Sprintf("ошибка выполнения RAC: %s", stderr)
			}
		}
		return "", apperrors.NewAppError(ErrRACExec, errMsg, err)
	}

	return string(output), nil
}

// sanitizeArgs маскирует пароли в аргументах для логирования.
// Имена пользователей НЕ маскируются — они не являются конфиденциальными данными
// и полезны для отладки проблем с аутентификацией.
func sanitizeArgs(args []string) []string {
	sanitized := make([]string, len(args))
	for i, arg := range args {
		if strings.HasPrefix(arg, "--cluster-pwd=") ||
			strings.HasPrefix(arg, "--infobase-pwd=") {
			eqIdx := strings.Index(arg, "=")
			sanitized[i] = arg[:eqIdx+1] + "***"
		} else {
			sanitized[i] = arg
		}
	}
	return sanitized
}

// sanitizeString маскирует пароли в произвольной строке (например, stderr от RAC).
// Используется для предотвращения утечки credentials в сообщениях об ошибках.
func sanitizeString(s string) string {
	// Маскируем значения после --cluster-pwd= и --infobase-pwd=
	// Паттерн: --xxx-pwd=VALUE где VALUE продолжается до пробела, кавычки или конца строки
	result := s
	for _, prefix := range []string{"--cluster-pwd=", "--infobase-pwd="} {
		offset := 0
		for {
			idx := strings.Index(result[offset:], prefix)
			if idx == -1 {
				break
			}
			// Корректируем индекс относительно всей строки
			idx += offset
			// Находим конец значения пароля
			start := idx + len(prefix)
			end := start
			for end < len(result) {
				ch := result[end]
				// Пароль заканчивается на пробеле, кавычке, или конце строки
				if ch == ' ' || ch == '\t' || ch == '\n' || ch == '"' || ch == '\'' {
					break
				}
				end++
			}
			// Заменяем пароль на ***
			result = result[:start] + "***" + result[end:]
			// Продвигаем offset за обработанную область
			offset = start + 3 // len("***") = 3
		}
	}
	return result
}

// clusterAuthArgs возвращает аргументы аутентификации кластера.
func (c *racClient) clusterAuthArgs() []string {
	var args []string
	if c.clusterUser != "" {
		args = append(args, "--cluster-user="+c.clusterUser)
	}
	if c.clusterPass != "" {
		args = append(args, "--cluster-pwd="+c.clusterPass)
	}
	return args
}

// infobaseAuthArgs возвращает аргументы аутентификации информационной базы.
func (c *racClient) infobaseAuthArgs() []string {
	var args []string
	if c.infobaseUser != "" {
		args = append(args, "--infobase-user="+c.infobaseUser)
	}
	if c.infobasePass != "" {
		args = append(args, "--infobase-pwd="+c.infobasePass)
	}
	return args
}

// --- Парсинг вывода RAC ---

// parseBlocks разбивает вывод RAC на блоки key-value, разделённые пустыми строками.
func parseBlocks(output string) []map[string]string {
	var blocks []map[string]string
	current := make(map[string]string)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			if len(current) > 0 {
				blocks = append(blocks, current)
				current = make(map[string]string)
			}
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		current[key] = value
	}
	if len(current) > 0 {
		blocks = append(blocks, current)
	}
	return blocks
}

// trimQuotes удаляет обрамляющие кавычки из строки.
func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// parseClusterInfo парсит блок key-value в ClusterInfo.
func parseClusterInfo(block map[string]string) (*ClusterInfo, error) {
	uuid, ok := block["cluster"]
	if !ok || uuid == "" {
		return nil, apperrors.NewAppError(ErrRACParse, "UUID кластера отсутствует в выводе", nil)
	}
	portStr := block["port"]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, apperrors.NewAppError(ErrRACParse,
			fmt.Sprintf("невалидный порт кластера: %s", portStr), err)
	}
	return &ClusterInfo{
		UUID: uuid,
		Host: block["host"],
		Port: port,
		Name: trimQuotes(block["name"]),
	}, nil
}

// parseInfobaseInfo парсит блок key-value в InfobaseInfo.
func parseInfobaseInfo(block map[string]string) (*InfobaseInfo, error) {
	uuid, ok := block["infobase"]
	if !ok || uuid == "" {
		return nil, apperrors.NewAppError(ErrRACParse, "UUID информационной базы отсутствует в выводе", nil)
	}
	return &InfobaseInfo{
		UUID:        uuid,
		Name:        block["name"],
		Description: trimQuotes(block["descr"]),
	}, nil
}

// parseSessionInfo парсит блок key-value в SessionInfo.
func parseSessionInfo(block map[string]string) (*SessionInfo, error) {
	sessionID, ok := block["session"]
	if !ok || sessionID == "" {
		return nil, apperrors.NewAppError(ErrRACParse, "ID сессии отсутствует в выводе", nil)
	}
	info := &SessionInfo{
		SessionID: sessionID,
		UserName:  block["user-name"],
		AppID:     block["app-id"],
		Host:      block["host"],
	}
	// Парсинг времени — ошибки игнорируются, т.к. время сессии не критично для функционала.
	// RAC возвращает время без timezone: "2006-01-02T15:04:05" (локальное время сервера).
	// При ошибке парсинга поля остаются нулевыми (time.Time{}).
	if v, ok := block["started-at"]; ok && v != "" {
		if t, err := time.Parse("2006-01-02T15:04:05", v); err == nil {
			info.StartedAt = t
		}
	}
	if v, ok := block["last-active-at"]; ok && v != "" {
		if t, err := time.Parse("2006-01-02T15:04:05", v); err == nil {
			info.LastActiveAt = t
		}
	}
	return info, nil
}

// parseServiceModeStatus парсит блок key-value в ServiceModeStatus.
func parseServiceModeStatus(block map[string]string) *ServiceModeStatus {
	status := &ServiceModeStatus{}
	if v, ok := block["sessions-deny"]; ok {
		status.Enabled = v == "on"
	}
	if v, ok := block["scheduled-jobs-deny"]; ok {
		status.ScheduledJobsBlocked = v == "on"
	}
	if v, ok := block["denied-message"]; ok {
		status.Message = trimQuotes(v)
	}
	return status
}

// --- Реализация интерфейса Client ---

// GetClusterInfo возвращает информацию о первом кластере 1C.
func (c *racClient) GetClusterInfo(ctx context.Context) (*ClusterInfo, error) {
	c.logger.Debug("Получение информации о кластере")

	output, err := c.executeRAC(ctx, []string{"cluster", "list"})
	if err != nil {
		return nil, err
	}

	blocks := parseBlocks(output)
	if len(blocks) == 0 {
		return nil, apperrors.NewAppError(ErrRACNotFound, "кластер не найден в выводе RAC", nil)
	}

	return parseClusterInfo(blocks[0])
}

// GetInfobaseInfo возвращает информацию об информационной базе по имени.
func (c *racClient) GetInfobaseInfo(ctx context.Context, clusterUUID, infobaseName string) (*InfobaseInfo, error) {
	c.logger.Debug("Получение информации об информационной базе",
		"cluster", clusterUUID, "infobase", infobaseName)

	args := []string{"infobase", "summary", "list", "--cluster=" + clusterUUID}
	args = append(args, c.clusterAuthArgs()...)

	output, err := c.executeRAC(ctx, args)
	if err != nil {
		return nil, err
	}

	blocks := parseBlocks(output)
	for _, block := range blocks {
		if block["name"] == infobaseName {
			return parseInfobaseInfo(block)
		}
	}

	return nil, apperrors.NewAppError(ErrRACNotFound,
		fmt.Sprintf("информационная база '%s' не найдена", infobaseName), nil)
}

// GetSessions возвращает список активных сессий для информационной базы.
func (c *racClient) GetSessions(ctx context.Context, clusterUUID, infobaseUUID string) ([]SessionInfo, error) {
	c.logger.Debug("Получение списка сессий",
		"cluster", clusterUUID, "infobase", infobaseUUID)

	args := []string{"session", "list", "--cluster=" + clusterUUID, "--infobase=" + infobaseUUID}
	args = append(args, c.clusterAuthArgs()...)

	output, err := c.executeRAC(ctx, args)
	if err != nil {
		return nil, err
	}

	blocks := parseBlocks(output)
	sessions := make([]SessionInfo, 0, len(blocks))
	for _, block := range blocks {
		s, err := parseSessionInfo(block)
		if err != nil {
			c.logger.Warn("Пропуск невалидной сессии при парсинге", "error", err)
			continue
		}
		sessions = append(sessions, *s)
	}

	return sessions, nil
}

// TerminateSession завершает конкретную сессию.
func (c *racClient) TerminateSession(ctx context.Context, clusterUUID, sessionID string) error {
	c.logger.Info("Завершение сессии", "cluster", clusterUUID, "session", sessionID)

	args := []string{"session", "terminate", "--cluster=" + clusterUUID, "--session=" + sessionID}
	args = append(args, c.clusterAuthArgs()...)

	_, err := c.executeRAC(ctx, args)
	if err != nil {
		return apperrors.NewAppError(ErrRACSession,
			fmt.Sprintf("ошибка завершения сессии %s", sessionID), err)
	}

	c.logger.Info("Сессия завершена", "session", sessionID)
	return nil
}

// TerminateAllSessions завершает все сессии для информационной базы.
func (c *racClient) TerminateAllSessions(ctx context.Context, clusterUUID, infobaseUUID string) error {
	c.logger.Info("Завершение всех сессий", "cluster", clusterUUID, "infobase", infobaseUUID)

	sessions, err := c.GetSessions(ctx, clusterUUID, infobaseUUID)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		c.logger.Info("Нет активных сессий для завершения")
		return nil
	}

	var errs []string
	for _, s := range sessions {
		if err := c.TerminateSession(ctx, clusterUUID, s.SessionID); err != nil {
			errs = append(errs, fmt.Sprintf("сессия %s: %v", s.SessionID, err))
		}
	}

	if len(errs) > 0 {
		return apperrors.NewAppError(ErrRACSession,
			fmt.Sprintf("ошибки при завершении сессий: %s", strings.Join(errs, "; ")), nil)
	}

	c.logger.Info("Все сессии завершены", "count", len(sessions))
	return nil
}

// EnableServiceMode включает сервисный режим для информационной базы.
func (c *racClient) EnableServiceMode(ctx context.Context, clusterUUID, infobaseUUID string, terminateSessions bool) error {
	c.logger.Debug("Включение сервисного режима",
		"cluster", clusterUUID, "infobase", infobaseUUID)

	// Проверяем, были ли регламентные задания заблокированы отдельно до сервисного режима.
	// Если да — добавляем маркер "." в denied-message, чтобы DisableServiceMode
	// не разблокировал их автоматически (legacy-паттерн).
	deniedMessage := constants.DefaultServiceModeMessage
	currentStatus, statusErr := c.getInfobaseRawStatus(ctx, clusterUUID, infobaseUUID)
	if statusErr != nil {
		c.logger.Warn("Не удалось проверить статус регламентных заданий, маркер не будет добавлен",
			"error", statusErr)
	} else if currentStatus.ScheduledJobsBlocked {
		deniedMessage += "."
		c.logger.Debug("Регламентные задания уже заблокированы, добавлен маркер в denied-message")
	}

	args := []string{
		"infobase", "update",
		"--cluster=" + clusterUUID,
		"--infobase=" + infobaseUUID,
		"--sessions-deny=on",
		"--scheduled-jobs-deny=on",
		"--denied-from=" + time.Now().Format("2006-01-02T15:04:05"),
		"--denied-message=" + deniedMessage,
		"--permission-code=ServiceMode",
	}
	args = append(args, c.clusterAuthArgs()...)
	args = append(args, c.infobaseAuthArgs()...)

	_, err := c.executeRAC(ctx, args)
	if err != nil {
		return err
	}

	c.logger.Debug("Сервисный режим включён")

	if terminateSessions {
		// TODO(#42): Рассмотреть возврат ошибки или partial success статуса вместо swallow.
		// Текущее поведение: сервисный режим уже включён, ошибка завершения сессий
		// не должна приводить к откату - это legacy паттерн для обратной совместимости.
		if err := c.TerminateAllSessions(ctx, clusterUUID, infobaseUUID); err != nil {
			c.logger.Warn("Ошибка завершения сессий после включения сервисного режима", "error", err)
		}
	}

	return nil
}

// DisableServiceMode отключает сервисный режим для информационной базы.
// Условно снимает блокировку регламентных заданий: если denied-message
// оканчивается на ".", это означает, что задания были заблокированы отдельно
// и не должны быть разблокированы при отключении сервисного режима.
func (c *racClient) DisableServiceMode(ctx context.Context, clusterUUID, infobaseUUID string) error {
	c.logger.Debug("Отключение сервисного режима",
		"cluster", clusterUUID, "infobase", infobaseUUID)

	// Получаем текущий статус для условного снятия блокировки регламентных заданий
	status, err := c.getInfobaseRawStatus(ctx, clusterUUID, infobaseUUID)
	if err != nil {
		c.logger.Warn("Не удалось получить текущий статус сервисного режима, "+
			"блокировка регламентных заданий будет снята (fail-open)", "error", err)
	}

	args := []string{
		"infobase", "update",
		"--cluster=" + clusterUUID,
		"--infobase=" + infobaseUUID,
		"--sessions-deny=off",
		"--denied-from=",
		"--denied-message=",
		"--permission-code=",
	}

	// Снимаем блокировку регламентных заданий только если сообщение
	// не оканчивается на "." (legacy-паттерн: точка в конце означает
	// что задания были заблокированы отдельно от сервисного режима)
	if status == nil || !strings.HasSuffix(status.Message, ".") {
		args = append(args, "--scheduled-jobs-deny=off")
	} else {
		c.logger.Debug("Блокировка регламентных заданий не снята (обнаружен маркер отдельной блокировки)")
	}

	args = append(args, c.clusterAuthArgs()...)
	args = append(args, c.infobaseAuthArgs()...)

	_, err = c.executeRAC(ctx, args)
	if err != nil {
		return err
	}

	c.logger.Debug("Сервисный режим отключён")
	return nil
}

// getInfobaseRawStatus получает статус информационной базы без запроса сессий.
// Используется для внутренних проверок перед изменением состояния.
func (c *racClient) getInfobaseRawStatus(ctx context.Context, clusterUUID, infobaseUUID string) (*ServiceModeStatus, error) {
	args := []string{"infobase", "info", "--cluster=" + clusterUUID, "--infobase=" + infobaseUUID}
	args = append(args, c.clusterAuthArgs()...)
	args = append(args, c.infobaseAuthArgs()...)

	output, err := c.executeRAC(ctx, args)
	if err != nil {
		return nil, err
	}

	blocks := parseBlocks(output)
	if len(blocks) == 0 {
		return nil, apperrors.NewAppError(ErrRACNotFound,
			"информация о базе не найдена в выводе RAC", nil)
	}

	return parseServiceModeStatus(blocks[0]), nil
}

// GetServiceModeStatus возвращает текущий статус сервисного режима.
//
// ВАЖНО: Метод выполняет два последовательных RAC-запроса (getInfobaseRawStatus + GetSessions),
// поэтому значение ActiveSessions может быть неактуальным к моменту использования на
// высоконагруженных системах. Для критичных операций, где важна точность количества сессий,
// рекомендуется вызывать GetSessions отдельно непосредственно перед операцией.
func (c *racClient) GetServiceModeStatus(ctx context.Context, clusterUUID, infobaseUUID string) (*ServiceModeStatus, error) {
	c.logger.Debug("Получение статуса сервисного режима",
		"cluster", clusterUUID, "infobase", infobaseUUID)

	status, err := c.getInfobaseRawStatus(ctx, clusterUUID, infobaseUUID)
	if err != nil {
		return nil, err
	}

	// Подсчёт активных сессий
	sessions, err := c.GetSessions(ctx, clusterUUID, infobaseUUID)
	if err != nil {
		c.logger.Warn("Не удалось получить количество активных сессий", "error", err)
	} else {
		status.ActiveSessions = len(sessions)
	}

	return status, nil
}

// VerifyServiceMode проверяет, соответствует ли текущее состояние ожидаемому.
func (c *racClient) VerifyServiceMode(ctx context.Context, clusterUUID, infobaseUUID string, expectedEnabled bool) error {
	status, err := c.GetServiceModeStatus(ctx, clusterUUID, infobaseUUID)
	if err != nil {
		return err
	}

	if status.Enabled != expectedEnabled {
		return apperrors.NewAppError(ErrRACVerify,
			fmt.Sprintf("несоответствие статуса сервисного режима: ожидалось %v, получено %v",
				expectedEnabled, status.Enabled), nil)
	}

	c.logger.Info("Проверка сервисного режима успешна",
		"enabled", status.Enabled,
		"message", status.Message,
		"activeSessions", status.ActiveSessions)

	return nil
}
