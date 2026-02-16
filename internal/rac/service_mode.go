package rac

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/constants"
)

// Константы перенесены в internal/constants/constants.go

// ServiceModeStatus представляет статус сервисного режима
type ServiceModeStatus struct {
	Enabled        bool
	Message        string
	ActiveSessions int
}

// SessionInfo представляет информацию о сессии
type SessionInfo struct {
	SessionID    string
	UserName     string
	AppID        string
	StartedAt    time.Time
	LastActiveAt time.Time
}

// EnableServiceMode включает сервисный режим для указанной информационной базы.
// Устанавливает блокировку сессий и регламентных заданий с заданным сообщением.
// Параметры:
//   - ctx: контекст выполнения операции
//   - clusterUUID: уникальный идентификатор кластера
//   - infobaseUUID: уникальный идентификатор информационной базы
//   - terminateSessions: флаг принудительного завершения активных сессий
//
// Возвращает:
//   - error: ошибка выполнения операции или nil при успехе
func (c *Client) EnableServiceMode(ctx context.Context, clusterUUID, infobaseUUID string, terminateSessions bool) error {
	c.Logger.Debug("Enabling service mode", "cluster", clusterUUID, "infobase", infobaseUUID, "message", constants.DefaultServiceModeMessage)

	// Используем константу по умолчанию для сообщения
	message := constants.DefaultServiceModeMessage

	// // Получаем текущее состояние scheduled-jobs-deny
	// currentScheduledJobsStatus, err := c.getScheduledJobsDenyStatus(ctx, clusterUUID, infobaseUUID)
	// if err != nil {
	// 	c.Logger.Warn("Failed to get current scheduled-jobs-deny status", "error", err)
	currentScheduledJobsStatus := "off" // По умолчанию считаем выключенным
	// }

	// Формируем denied-message в зависимости от текущего состояния scheduled-jobs-deny
	// Всегда включаем фоновые задания после отключения сервисного режима.
	deniedMessage := message
	if currentScheduledJobsStatus == "on" {
		deniedMessage = message + "."
	}

	// Включаем сервисный режим
	args := []string{
		"infobase", "update",
		"--cluster=" + clusterUUID,
		"--infobase=" + infobaseUUID,
		"--denied-from=" + time.Now().Format("2006-01-02T15:04:05"),
		"--denied-message=" + deniedMessage,
		"--permission-code=" + "ServiceMode",
		"--sessions-deny=on",
		"--scheduled-jobs-deny=on",
	}

	// Добавляем учетные данные кластера, если они указаны
	if c.User != "" {
		args = append(args, "--cluster-user="+c.User)
	}
	if c.Password != "" {
		args = append(args, "--cluster-pwd="+c.Password)
	}

	// Добавляем учетные данные информационной базы, если они указаны
	if c.DbUser != "" {
		args = append(args, "--infobase-user="+c.DbUser)
	}
	if c.DbPassword != "" {
		args = append(args, "--infobase-pwd="+c.DbPassword)
	}

	_, err := c.ExecuteCommand(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to enable service mode: %w", err)
	}

	c.Logger.Debug("Service mode enabled successfully")

	// Если нужно завершить сессии, делаем это после включения блокировки
	if terminateSessions {
		if err := c.TerminateAllSessions(ctx, clusterUUID, infobaseUUID); err != nil {
			c.Logger.Warn("Failed to terminate sessions before enabling service mode", "error", err)
		}
	}

	return nil
}

// DisableServiceMode отключает сервисный режим для указанной информационной базы.
// Снимает блокировку сессий и регламентных заданий.
// Параметры:
//   - ctx: контекст выполнения операции
//   - clusterUUID: уникальный идентификатор кластера
//   - infobaseUUID: уникальный идентификатор информационной базы
//
// Возвращает:
//   - error: ошибка выполнения операции или nil при успехе
func (c *Client) DisableServiceMode(ctx context.Context, clusterUUID, infobaseUUID string) error {
	c.Logger.Debug("Disabling service mode", "cluster", clusterUUID, "infobase", infobaseUUID)

	// Получаем текущее denied-message для проверки
	status, err := c.GetServiceModeStatus(ctx, clusterUUID, infobaseUUID)
	if err != nil {
		c.Logger.Warn("Failed to get current service mode status", "error", err)
	}

	args := []string{
		"infobase", "update",
		"--cluster=" + clusterUUID,
		"--infobase=" + infobaseUUID,
		"--denied-from=",
		"--denied-message=",
		"--permission-code=",
		"--sessions-deny=off",
	}

	// Добавляем --scheduled-jobs-deny=off только если текущее denied-message не равно DefaultServiceModeMessage + "."
	if status == nil || status.Message != fmt.Sprintf("\"%s.\"", constants.DefaultServiceModeMessage) {
		args = append(args, "--scheduled-jobs-deny=off")
		// fmt.Println(status.Message)
		// fmt.Printf("\"%s.\"\n", DefaultServiceModeMessage)
	}

	// Добавляем учетные данные кластера, если они указаны
	if c.User != "" {
		args = append(args, "--cluster-user="+c.User)
	}
	if c.Password != "" {
		args = append(args, "--cluster-pwd="+c.Password)
	}

	// Добавляем учетные данные информационной базы, если они указаны
	if c.DbUser != "" {
		args = append(args, "--infobase-user="+c.DbUser)
	}
	if c.DbPassword != "" {
		args = append(args, "--infobase-pwd="+c.DbPassword)
	}

	_, err = c.ExecuteCommand(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to disable service mode: %w", err)
	}

	c.Logger.Debug("Service mode disabled successfully")
	return nil
}

// GetServiceModeStatus получает текущий статус сервисного режима для указанной информационной базы.
// Возвращает информацию о состоянии блокировки, сообщении и количестве активных сессий.
// Параметры:
//   - ctx: контекст выполнения операции
//   - clusterUUID: уникальный идентификатор кластера
//   - infobaseUUID: уникальный идентификатор информационной базы
//
// Возвращает:
//   - *ServiceModeStatus: структура с информацией о статусе сервисного режима
//   - error: ошибка выполнения операции или nil при успехе
func (c *Client) GetServiceModeStatus(ctx context.Context, clusterUUID, infobaseUUID string) (*ServiceModeStatus, error) {
	c.Logger.Debug("Getting service mode status", "cluster", clusterUUID, "infobase", infobaseUUID)

	// Формируем аргументы команды
	args := []string{"infobase", "info", "--cluster=" + clusterUUID, "--infobase=" + infobaseUUID}

	// Добавляем учетные данные кластера, если они указаны
	if c.User != "" {
		args = append(args, "--cluster-user="+c.User)
	}
	if c.Password != "" {
		args = append(args, "--cluster-pwd="+c.Password)
	}

	// Добавляем учетные данные информационной базы, если они указаны
	if c.DbUser != "" {
		args = append(args, "--infobase-user="+c.DbUser)
	}
	if c.DbPassword != "" {
		args = append(args, "--infobase-pwd="+c.DbPassword)
	}

	// Получаем информацию об информационной базе
	output, err := c.ExecuteCommand(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get infobase info: %w", err)
	}

	status := &ServiceModeStatus{}

	// Парсим вывод для определения статуса сервисного режима
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "denied-from") {
			// Если есть значение denied-from, значит сервисный режим включен
			parts := strings.Split(line, ":")
			if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
				status.Enabled = true
			}
		} else if strings.Contains(line, "denied-message") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				status.Message = strings.TrimSpace(parts[1])
			}
		}
	}

	// Получаем количество активных сессий
	sessions, err := c.GetSessions(ctx, clusterUUID, infobaseUUID)
	if err != nil {
		c.Logger.Warn("Failed to get active sessions count", "error", err)
	} else {
		status.ActiveSessions = len(sessions)
	}

	return status, nil
}

// GetSessions получает список всех активных сессий для указанной информационной базы.
// Возвращает детальную информацию о каждой сессии включая ID, пользователя и время активности.
// Параметры:
//   - ctx: контекст выполнения операции
//   - clusterUUID: уникальный идентификатор кластера
//   - infobaseUUID: уникальный идентификатор информационной базы
//
// Возвращает:
//   - []SessionInfo: срез структур с информацией об активных сессиях
//   - error: ошибка выполнения операции или nil при успехе
func (c *Client) GetSessions(ctx context.Context, clusterUUID, infobaseUUID string) ([]SessionInfo, error) {
	c.Logger.Debug("Getting sessions list", "cluster", clusterUUID, "infobase", infobaseUUID)

	// Формируем аргументы команды
	args := []string{"session", "list", "--cluster=" + clusterUUID, "--infobase=" + infobaseUUID}

	// Добавляем учетные данные кластера, если они указаны
	if c.User != "" {
		args = append(args, "--cluster-user="+c.User)
	}
	if c.Password != "" {
		args = append(args, "--cluster-pwd="+c.Password)
	}

	output, err := c.ExecuteCommand(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions list: %w", err)
	}

	var sessions []SessionInfo
	lines := strings.Split(output, "\n")
	var currentSession SessionInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch {
		case strings.Contains(line, "session"):
			// Начало новой сессии
			if currentSession.SessionID != "" {
				sessions = append(sessions, currentSession)
			}
			currentSession = SessionInfo{}

			// Извлекаем ID сессии
			uuidRegex := regexp.MustCompile(`session\s*:\s*([a-f0-9-]{36})`)
			matches := uuidRegex.FindStringSubmatch(line)
			if len(matches) >= 2 {
				currentSession.SessionID = matches[1]
			}
		case strings.Contains(line, "user-name"):
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				currentSession.UserName = strings.TrimSpace(parts[1])
			}
		case strings.Contains(line, "app-id"):
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				currentSession.AppID = strings.TrimSpace(parts[1])
			}
		case strings.Contains(line, "started-at"):
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				if t, err := time.Parse("2006-01-02T15:04:05", strings.TrimSpace(parts[1])); err == nil {
					currentSession.StartedAt = t
				}
			}
		case strings.Contains(line, "last-active-at"):
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				if t, err := time.Parse("2006-01-02T15:04:05", strings.TrimSpace(parts[1])); err == nil {
					currentSession.LastActiveAt = t
				}
			}
		}
	}

	// Добавляем последнюю сессию
	if currentSession.SessionID != "" {
		sessions = append(sessions, currentSession)
	}

	return sessions, nil
}

// TerminateSession принудительно завершает указанную сессию пользователя.
// Используется для завершения конкретной сессии по её идентификатору.
// Параметры:
//   - ctx: контекст выполнения операции
//   - clusterUUID: уникальный идентификатор кластера
//   - sessionID: уникальный идентификатор сессии для завершения
//
// Возвращает:
//   - error: ошибка выполнения операции или nil при успехе
func (c *Client) TerminateSession(ctx context.Context, clusterUUID, sessionID string) error {
	c.Logger.Info("Terminating session", "cluster", clusterUUID, "session", sessionID)

	// Формируем аргументы команды
	args := []string{"session", "terminate", "--cluster=" + clusterUUID, "--session=" + sessionID}

	// Добавляем учетные данные кластера, если они указаны
	if c.User != "" {
		args = append(args, "--cluster-user="+c.User)
	}
	if c.Password != "" {
		args = append(args, "--cluster-pwd="+c.Password)
	}

	_, err := c.ExecuteCommand(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to terminate session %s: %w", sessionID, err)
	}

	c.Logger.Info("Session terminated successfully", "session", sessionID)
	return nil
}

// TerminateAllSessions принудительно завершает все активные сессии для указанной информационной базы.
// Получает список всех активных сессий и завершает каждую из них.
// Параметры:
//   - ctx: контекст выполнения операции
//   - clusterUUID: уникальный идентификатор кластера
//   - infobaseUUID: уникальный идентификатор информационной базы
//
// Возвращает:
//   - error: ошибка выполнения операции или nil при успехе
func (c *Client) TerminateAllSessions(ctx context.Context, clusterUUID, infobaseUUID string) error {
	c.Logger.Info("Terminating all sessions", "cluster", clusterUUID, "infobase", infobaseUUID)

	sessions, err := c.GetSessions(ctx, clusterUUID, infobaseUUID)
	if err != nil {
		return fmt.Errorf("failed to get sessions for termination: %w", err)
	}

	if len(sessions) == 0 {
		c.Logger.Info("No active sessions to terminate")
		return nil
	}

	var errors []string
	for _, session := range sessions {
		if err := c.TerminateSession(ctx, clusterUUID, session.SessionID); err != nil {
			errors = append(errors, fmt.Sprintf("session %s: %v", session.SessionID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to terminate some sessions: %s", strings.Join(errors, "; "))
	}

	c.Logger.Info("All sessions terminated successfully", "count", len(sessions))
	return nil
}

// VerifyServiceMode проверяет соответствие текущего статуса сервисного режима ожидаемому состоянию.
// Используется для валидации успешности операций включения/отключения сервисного режима.
// Параметры:
//   - ctx: контекст выполнения операции
//   - clusterUUID: уникальный идентификатор кластера
//   - infobaseUUID: уникальный идентификатор информационной базы
//   - expectedEnabled: ожидаемое состояние сервисного режима (true - включен, false - отключен)
//
// Возвращает:
//   - error: ошибка несоответствия статуса или выполнения операции, nil при успехе
func (c *Client) VerifyServiceMode(ctx context.Context, clusterUUID, infobaseUUID string, expectedEnabled bool) error {
	status, err := c.GetServiceModeStatus(ctx, clusterUUID, infobaseUUID)
	if err != nil {
		return fmt.Errorf("failed to verify service mode: %w", err)
	}

	if status.Enabled != expectedEnabled {
		return fmt.Errorf("service mode status mismatch: expected %v, got %v", expectedEnabled, status.Enabled)
	}

	c.Logger.Info("Service mode verification successful",
		"enabled", status.Enabled,
		"message", status.Message,
		"active_sessions", status.ActiveSessions)

	return nil
}

// getScheduledJobsDenyStatus получает текущее состояние scheduled-jobs-deny
func (c *Client) getScheduledJobsDenyStatus(ctx context.Context, clusterUUID, infobaseUUID string) (string, error) {
	// Формируем аргументы команды
	args := []string{"infobase", "info", "--cluster=" + clusterUUID, "--infobase=" + infobaseUUID}

	// Добавляем учетные данные кластера, если они указаны
	if c.User != "" {
		args = append(args, "--cluster-user="+c.User)
	}
	if c.Password != "" {
		args = append(args, "--cluster-pwd="+c.Password)
	}

	// Добавляем учетные данные информационной базы, если они указаны
	if c.DbUser != "" {
		args = append(args, "--infobase-user="+c.DbUser)
	}
	if c.DbPassword != "" {
		args = append(args, "--infobase-pwd="+c.DbPassword)
	}

	output, err := c.ExecuteCommand(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("failed to get infobase info: %w", err)
	}

	// Парсим вывод для поиска scheduled-jobs-deny
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "scheduled-jobs-deny") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "off", nil // По умолчанию считаем, что scheduled-jobs-deny выключен
}
