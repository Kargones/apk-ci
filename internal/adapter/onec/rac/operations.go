package rac

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/apperrors"
)

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

	args := []string{"infobase", "summary", "list", "--cluster=" + clusterUUID} //nolint:prealloc // dynamic append based on auth
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

	args := []string{"session", "list", "--cluster=" + clusterUUID, "--infobase=" + infobaseUUID} //nolint:prealloc // dynamic append based on auth
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

	args := []string{"session", "terminate", "--cluster=" + clusterUUID, "--session=" + sessionID} //nolint:prealloc // dynamic append based on auth
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

	args := []string{ //nolint:prealloc // dynamic append based on auth
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
	args := []string{"infobase", "info", "--cluster=" + clusterUUID, "--infobase=" + infobaseUUID} //nolint:prealloc // dynamic append based on auth
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
