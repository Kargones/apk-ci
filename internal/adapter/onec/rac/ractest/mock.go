// Package ractest предоставляет тестовые утилиты для пакета rac:
// мок-реализации интерфейсов и вспомогательные конструкторы.
package ractest

import (
	"context"
	"fmt"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/onec/rac"
)

// Compile-time проверки реализации интерфейсов
var (
	_ rac.Client             = (*MockRACClient)(nil)
	_ rac.ClusterProvider    = (*MockRACClient)(nil)
	_ rac.InfobaseProvider   = (*MockRACClient)(nil)
	_ rac.SessionProvider    = (*MockRACClient)(nil)
	_ rac.ServiceModeManager = (*MockRACClient)(nil)
)

// MockRACClient — мок-реализация rac.Client для тестирования.
// Использует функциональные поля для гибкой настройки поведения в тестах.
type MockRACClient struct {
	// GetClusterInfoFunc — пользовательская реализация GetClusterInfo
	GetClusterInfoFunc func(ctx context.Context) (*rac.ClusterInfo, error)
	// GetInfobaseInfoFunc — пользовательская реализация GetInfobaseInfo
	GetInfobaseInfoFunc func(ctx context.Context, clusterUUID, infobaseName string) (*rac.InfobaseInfo, error)
	// GetSessionsFunc — пользовательская реализация GetSessions
	GetSessionsFunc func(ctx context.Context, clusterUUID, infobaseUUID string) ([]rac.SessionInfo, error)
	// TerminateSessionFunc — пользовательская реализация TerminateSession
	TerminateSessionFunc func(ctx context.Context, clusterUUID, sessionID string) error
	// TerminateAllSessionsFunc — пользовательская реализация TerminateAllSessions
	TerminateAllSessionsFunc func(ctx context.Context, clusterUUID, infobaseUUID string) error
	// EnableServiceModeFunc — пользовательская реализация EnableServiceMode
	EnableServiceModeFunc func(ctx context.Context, clusterUUID, infobaseUUID string, terminateSessions bool) error
	// DisableServiceModeFunc — пользовательская реализация DisableServiceMode
	DisableServiceModeFunc func(ctx context.Context, clusterUUID, infobaseUUID string) error
	// GetServiceModeStatusFunc — пользовательская реализация GetServiceModeStatus
	GetServiceModeStatusFunc func(ctx context.Context, clusterUUID, infobaseUUID string) (*rac.ServiceModeStatus, error)
	// VerifyServiceModeFunc — пользовательская реализация VerifyServiceMode
	VerifyServiceModeFunc func(ctx context.Context, clusterUUID, infobaseUUID string, expectedEnabled bool) error
}

// GetClusterInfo возвращает информацию о кластере.
// При отсутствии пользовательской функции возвращает тестовые данные.
func (m *MockRACClient) GetClusterInfo(ctx context.Context) (*rac.ClusterInfo, error) {
	if m.GetClusterInfoFunc != nil {
		return m.GetClusterInfoFunc(ctx)
	}
	return &rac.ClusterInfo{
		UUID: "test-cluster-uuid",
		Name: "test-cluster",
		Host: "localhost",
		Port: 1541,
	}, nil
}

// GetInfobaseInfo возвращает информацию об информационной базе.
// При отсутствии пользовательской функции возвращает тестовые данные.
func (m *MockRACClient) GetInfobaseInfo(ctx context.Context, clusterUUID, infobaseName string) (*rac.InfobaseInfo, error) {
	if m.GetInfobaseInfoFunc != nil {
		return m.GetInfobaseInfoFunc(ctx, clusterUUID, infobaseName)
	}
	return &rac.InfobaseInfo{
		UUID:        "test-infobase-uuid",
		Name:        infobaseName,
		Description: "Тестовая информационная база",
	}, nil
}

// GetSessions возвращает список сессий.
// При отсутствии пользовательской функции возвращает пустой срез.
func (m *MockRACClient) GetSessions(ctx context.Context, clusterUUID, infobaseUUID string) ([]rac.SessionInfo, error) {
	if m.GetSessionsFunc != nil {
		return m.GetSessionsFunc(ctx, clusterUUID, infobaseUUID)
	}
	return []rac.SessionInfo{}, nil
}

// TerminateSession завершает сессию.
// При отсутствии пользовательской функции возвращает nil.
func (m *MockRACClient) TerminateSession(ctx context.Context, clusterUUID, sessionID string) error {
	if m.TerminateSessionFunc != nil {
		return m.TerminateSessionFunc(ctx, clusterUUID, sessionID)
	}
	return nil
}

// TerminateAllSessions завершает все сессии.
// При отсутствии пользовательской функции возвращает nil.
func (m *MockRACClient) TerminateAllSessions(ctx context.Context, clusterUUID, infobaseUUID string) error {
	if m.TerminateAllSessionsFunc != nil {
		return m.TerminateAllSessionsFunc(ctx, clusterUUID, infobaseUUID)
	}
	return nil
}

// EnableServiceMode включает сервисный режим.
// При отсутствии пользовательской функции возвращает nil.
func (m *MockRACClient) EnableServiceMode(ctx context.Context, clusterUUID, infobaseUUID string, terminateSessions bool) error {
	if m.EnableServiceModeFunc != nil {
		return m.EnableServiceModeFunc(ctx, clusterUUID, infobaseUUID, terminateSessions)
	}
	return nil
}

// DisableServiceMode отключает сервисный режим.
// При отсутствии пользовательской функции возвращает nil.
func (m *MockRACClient) DisableServiceMode(ctx context.Context, clusterUUID, infobaseUUID string) error {
	if m.DisableServiceModeFunc != nil {
		return m.DisableServiceModeFunc(ctx, clusterUUID, infobaseUUID)
	}
	return nil
}

// GetServiceModeStatus возвращает статус сервисного режима.
// При отсутствии пользовательской функции возвращает статус с выключенным режимом.
func (m *MockRACClient) GetServiceModeStatus(ctx context.Context, clusterUUID, infobaseUUID string) (*rac.ServiceModeStatus, error) {
	if m.GetServiceModeStatusFunc != nil {
		return m.GetServiceModeStatusFunc(ctx, clusterUUID, infobaseUUID)
	}
	return &rac.ServiceModeStatus{
		Enabled:        false,
		Message:        "",
		ActiveSessions: 0,
	}, nil
}

// VerifyServiceMode проверяет состояние сервисного режима.
// При отсутствии пользовательской функции сравнивает ожидаемое состояние с дефолтным (false).
func (m *MockRACClient) VerifyServiceMode(ctx context.Context, clusterUUID, infobaseUUID string, expectedEnabled bool) error {
	if m.VerifyServiceModeFunc != nil {
		return m.VerifyServiceModeFunc(ctx, clusterUUID, infobaseUUID, expectedEnabled)
	}
	// По умолчанию — режим выключен
	if expectedEnabled {
		return fmt.Errorf("service mode status mismatch: expected %v, got false", expectedEnabled)
	}
	return nil
}

// NewMockRACClient создаёт MockRACClient с дефолтными тестовыми данными.
func NewMockRACClient() *MockRACClient {
	return &MockRACClient{}
}

// NewMockRACClientWithSessions создаёт MockRACClient с предзаданными сессиями.
func NewMockRACClientWithSessions(sessions []rac.SessionInfo) *MockRACClient {
	return &MockRACClient{
		GetSessionsFunc: func(_ context.Context, _, _ string) ([]rac.SessionInfo, error) {
			return sessions, nil
		},
	}
}

// NewMockRACClientWithServiceMode создаёт MockRACClient с предзаданным статусом сервисного режима.
func NewMockRACClientWithServiceMode(enabled bool, message string, activeSessions int) *MockRACClient {
	return &MockRACClient{
		GetServiceModeStatusFunc: func(_ context.Context, _, _ string) (*rac.ServiceModeStatus, error) {
			return &rac.ServiceModeStatus{
				Enabled:        enabled,
				Message:        message,
				ActiveSessions: activeSessions,
			}, nil
		},
		VerifyServiceModeFunc: func(_ context.Context, _, _ string, expectedEnabled bool) error {
			if enabled != expectedEnabled {
				return fmt.Errorf("service mode status mismatch: expected %v, got %v", expectedEnabled, enabled)
			}
			return nil
		},
	}
}

// SessionData возвращает тестовые данные сессий для использования в тестах.
func SessionData() []rac.SessionInfo {
	return []rac.SessionInfo{
		{
			SessionID:    "session-uuid-1",
			UserName:     "Иванов",
			AppID:        "1CV8C",
			Host:         "workstation-01",
			StartedAt:    time.Date(2026, 1, 27, 9, 0, 0, 0, time.UTC),
			LastActiveAt: time.Date(2026, 1, 27, 10, 30, 0, 0, time.UTC),
		},
		{
			SessionID:    "session-uuid-2",
			UserName:     "Петров",
			AppID:        "1CV8",
			Host:         "workstation-02",
			StartedAt:    time.Date(2026, 1, 27, 8, 0, 0, 0, time.UTC),
			LastActiveAt: time.Date(2026, 1, 27, 10, 15, 0, 0, time.UTC),
		},
	}
}
