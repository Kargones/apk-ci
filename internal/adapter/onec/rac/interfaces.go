// Package rac определяет интерфейсы и типы данных для работы с RAC (Remote Administration Console) 1C.
// Пакет предоставляет абстракцию над RAC клиентом, разделённую по принципу ISP
// (Interface Segregation Principle) на сфокусированные интерфейсы:
// ClusterProvider, InfobaseProvider, SessionProvider, ServiceModeManager.
// Композитный интерфейс Client объединяет все вышеперечисленные.
package rac

import (
	"context"
	"time"
)

// ClusterInfo содержит информацию о кластере 1C.
type ClusterInfo struct {
	// UUID — уникальный идентификатор кластера
	UUID string
	// Name — имя кластера
	Name string
	// Host — хост кластера
	Host string
	// Port — порт кластера
	Port int
}

// InfobaseInfo содержит информацию об информационной базе 1C.
type InfobaseInfo struct {
	// UUID — уникальный идентификатор информационной базы
	UUID string
	// Name — имя информационной базы
	Name string
	// Description — описание информационной базы
	Description string
}

// ServiceModeStatus содержит статус сервисного режима информационной базы.
type ServiceModeStatus struct {
	// Enabled — включён ли сервисный режим
	Enabled bool
	// Message — сообщение блокировки
	Message string
	// ScheduledJobsBlocked — заблокированы ли регламентные задания
	ScheduledJobsBlocked bool
	// ActiveSessions — количество активных сессий
	ActiveSessions int
}

// SessionInfo содержит информацию о сессии пользователя.
type SessionInfo struct {
	// SessionID — уникальный идентификатор сессии
	SessionID string
	// UserName — имя пользователя
	UserName string
	// AppID — идентификатор приложения
	AppID string
	// Host — хост, с которого установлена сессия
	Host string
	// StartedAt — время начала сессии
	StartedAt time.Time
	// LastActiveAt — время последней активности
	LastActiveAt time.Time
}

// ClusterProvider предоставляет операции для получения информации о кластере.
type ClusterProvider interface {
	// GetClusterInfo возвращает информацию о кластере 1C.
	GetClusterInfo(ctx context.Context) (*ClusterInfo, error)
}

// InfobaseProvider предоставляет операции для получения информации об информационной базе.
type InfobaseProvider interface {
	// GetInfobaseInfo возвращает информацию об информационной базе по имени.
	GetInfobaseInfo(ctx context.Context, clusterUUID, infobaseName string) (*InfobaseInfo, error)
}

// SessionProvider предоставляет операции для управления сессиями пользователей.
type SessionProvider interface {
	// GetSessions возвращает список активных сессий для информационной базы.
	GetSessions(ctx context.Context, clusterUUID, infobaseUUID string) ([]SessionInfo, error)
	// TerminateSession завершает конкретную сессию.
	TerminateSession(ctx context.Context, clusterUUID, sessionID string) error
	// TerminateAllSessions завершает все сессии для информационной базы.
	TerminateAllSessions(ctx context.Context, clusterUUID, infobaseUUID string) error
}

// ServiceModeManager предоставляет операции для управления сервисным режимом.
//
// Примечание: реализация EnableServiceMode может требовать завершения сессий
// (параметр terminateSessions), а GetServiceModeStatus может подсчитывать
// активные сессии. Поэтому конкретная реализация, вероятно, будет также
// зависеть от SessionProvider для выполнения этих операций.
type ServiceModeManager interface {
	// EnableServiceMode включает сервисный режим для информационной базы.
	EnableServiceMode(ctx context.Context, clusterUUID, infobaseUUID string, terminateSessions bool) error
	// DisableServiceMode отключает сервисный режим для информационной базы.
	DisableServiceMode(ctx context.Context, clusterUUID, infobaseUUID string) error
	// GetServiceModeStatus возвращает текущий статус сервисного режима.
	GetServiceModeStatus(ctx context.Context, clusterUUID, infobaseUUID string) (*ServiceModeStatus, error)
	// VerifyServiceMode проверяет, соответствует ли текущее состояние ожидаемому.
	VerifyServiceMode(ctx context.Context, clusterUUID, infobaseUUID string, expectedEnabled bool) error
}

// Client — композитный интерфейс, объединяющий все операции RAC.
type Client interface {
	ClusterProvider
	InfobaseProvider
	SessionProvider
	ServiceModeManager
}
