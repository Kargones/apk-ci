package rac_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/onec/rac"
	"github.com/Kargones/apk-ci/internal/adapter/onec/rac/ractest"
)

// TestCompileTimeInterfaceChecks проверяет, что MockRACClient реализует все интерфейсы.
// Compile-time проверки уже есть в ractest/mock.go, но тест делает это явным.
func TestCompileTimeInterfaceChecks(_ *testing.T) {
	var _ rac.Client = (*ractest.MockRACClient)(nil)
	var _ rac.ClusterProvider = (*ractest.MockRACClient)(nil)
	var _ rac.InfobaseProvider = (*ractest.MockRACClient)(nil)
	var _ rac.SessionProvider = (*ractest.MockRACClient)(nil)
	var _ rac.ServiceModeManager = (*ractest.MockRACClient)(nil)
}

// TestMockRACClient_DefaultBehavior проверяет дефолтное поведение MockRACClient
// (когда пользовательские функции не установлены).
func TestMockRACClient_DefaultBehavior(t *testing.T) {
	ctx := context.Background()
	mock := ractest.NewMockRACClient()

	t.Run("GetClusterInfo возвращает тестовые данные", func(t *testing.T) {
		info, err := mock.GetClusterInfo(ctx)
		if err != nil {
			t.Fatalf("Не ожидалась ошибка, получено: %v", err)
		}
		if info.UUID != "test-cluster-uuid" {
			t.Errorf("Ожидался UUID 'test-cluster-uuid', получено: %s", info.UUID)
		}
		if info.Name != "test-cluster" {
			t.Errorf("Ожидалось имя 'test-cluster', получено: %s", info.Name)
		}
		if info.Host != "localhost" {
			t.Errorf("Ожидался хост 'localhost', получено: %s", info.Host)
		}
		if info.Port != 1541 {
			t.Errorf("Ожидался порт 1541, получено: %d", info.Port)
		}
	})

	t.Run("GetInfobaseInfo возвращает тестовые данные", func(t *testing.T) {
		info, err := mock.GetInfobaseInfo(ctx, "cluster-uuid", "test-db")
		if err != nil {
			t.Fatalf("Не ожидалась ошибка, получено: %v", err)
		}
		if info.UUID != "test-infobase-uuid" {
			t.Errorf("Ожидался UUID 'test-infobase-uuid', получено: %s", info.UUID)
		}
		if info.Name != "test-db" {
			t.Errorf("Ожидалось имя 'test-db', получено: %s", info.Name)
		}
	})

	t.Run("GetSessions возвращает пустой срез", func(t *testing.T) {
		sessions, err := mock.GetSessions(ctx, "cluster-uuid", "infobase-uuid")
		if err != nil {
			t.Fatalf("Не ожидалась ошибка, получено: %v", err)
		}
		if len(sessions) != 0 {
			t.Errorf("Ожидался пустой срез, получено %d элементов", len(sessions))
		}
	})

	t.Run("TerminateSession возвращает nil", func(t *testing.T) {
		err := mock.TerminateSession(ctx, "cluster-uuid", "session-uuid")
		if err != nil {
			t.Errorf("Не ожидалась ошибка, получено: %v", err)
		}
	})

	t.Run("TerminateAllSessions возвращает nil", func(t *testing.T) {
		err := mock.TerminateAllSessions(ctx, "cluster-uuid", "infobase-uuid")
		if err != nil {
			t.Errorf("Не ожидалась ошибка, получено: %v", err)
		}
	})

	t.Run("EnableServiceMode возвращает nil", func(t *testing.T) {
		err := mock.EnableServiceMode(ctx, "cluster-uuid", "infobase-uuid", false)
		if err != nil {
			t.Errorf("Не ожидалась ошибка, получено: %v", err)
		}
	})

	t.Run("DisableServiceMode возвращает nil", func(t *testing.T) {
		err := mock.DisableServiceMode(ctx, "cluster-uuid", "infobase-uuid")
		if err != nil {
			t.Errorf("Не ожидалась ошибка, получено: %v", err)
		}
	})

	t.Run("GetServiceModeStatus возвращает выключенный режим", func(t *testing.T) {
		status, err := mock.GetServiceModeStatus(ctx, "cluster-uuid", "infobase-uuid")
		if err != nil {
			t.Fatalf("Не ожидалась ошибка, получено: %v", err)
		}
		if status.Enabled {
			t.Error("Ожидался выключенный режим")
		}
		if status.ActiveSessions != 0 {
			t.Errorf("Ожидалось 0 активных сессий, получено: %d", status.ActiveSessions)
		}
	})

	t.Run("VerifyServiceMode с expectedEnabled=false возвращает nil", func(t *testing.T) {
		err := mock.VerifyServiceMode(ctx, "cluster-uuid", "infobase-uuid", false)
		if err != nil {
			t.Errorf("Не ожидалась ошибка, получено: %v", err)
		}
	})

	t.Run("VerifyServiceMode с expectedEnabled=true возвращает ошибку", func(t *testing.T) {
		err := mock.VerifyServiceMode(ctx, "cluster-uuid", "infobase-uuid", true)
		if err == nil {
			t.Error("Ожидалась ошибка несоответствия статуса")
		}
	})
}

// TestMockRACClient_CustomFunctions проверяет работу пользовательских функций MockRACClient.
func TestMockRACClient_CustomFunctions(t *testing.T) {
	ctx := context.Background()

	t.Run("GetClusterInfo с пользовательской функцией", func(t *testing.T) {
		mock := &ractest.MockRACClient{
			GetClusterInfoFunc: func(_ context.Context) (*rac.ClusterInfo, error) {
				return &rac.ClusterInfo{UUID: "custom-uuid", Name: "production"}, nil
			},
		}
		info, err := mock.GetClusterInfo(ctx)
		if err != nil {
			t.Fatalf("Не ожидалась ошибка, получено: %v", err)
		}
		if info.UUID != "custom-uuid" {
			t.Errorf("Ожидался UUID 'custom-uuid', получено: %s", info.UUID)
		}
		if info.Name != "production" {
			t.Errorf("Ожидалось имя 'production', получено: %s", info.Name)
		}
	})

	t.Run("GetClusterInfo с ошибкой", func(t *testing.T) {
		mock := &ractest.MockRACClient{
			GetClusterInfoFunc: func(_ context.Context) (*rac.ClusterInfo, error) {
				return nil, errors.New("кластер не найден")
			},
		}
		_, err := mock.GetClusterInfo(ctx)
		if err == nil {
			t.Error("Ожидалась ошибка")
		}
		if err.Error() != "кластер не найден" {
			t.Errorf("Ожидалась ошибка 'кластер не найден', получено: %s", err.Error())
		}
	})

	t.Run("GetInfobaseInfo с пользовательской функцией", func(t *testing.T) {
		mock := &ractest.MockRACClient{
			GetInfobaseInfoFunc: func(_ context.Context, _, infobaseName string) (*rac.InfobaseInfo, error) {
				return &rac.InfobaseInfo{UUID: "ib-uuid", Name: infobaseName, Description: "Тест"}, nil
			},
		}
		info, err := mock.GetInfobaseInfo(ctx, "cluster", "mydb")
		if err != nil {
			t.Fatalf("Не ожидалась ошибка, получено: %v", err)
		}
		if info.Name != "mydb" {
			t.Errorf("Ожидалось имя 'mydb', получено: %s", info.Name)
		}
	})

	t.Run("GetSessions с пользовательскими данными", func(t *testing.T) {
		testSessions := ractest.SessionData()
		mock := ractest.NewMockRACClientWithSessions(testSessions)
		sessions, err := mock.GetSessions(ctx, "cluster", "infobase")
		if err != nil {
			t.Fatalf("Не ожидалась ошибка, получено: %v", err)
		}
		if len(sessions) != 2 {
			t.Fatalf("Ожидалось 2 сессии, получено: %d", len(sessions))
		}
		if sessions[0].UserName != "Иванов" {
			t.Errorf("Ожидался пользователь 'Иванов', получено: %s", sessions[0].UserName)
		}
		if sessions[1].UserName != "Петров" {
			t.Errorf("Ожидался пользователь 'Петров', получено: %s", sessions[1].UserName)
		}
	})

	t.Run("TerminateSession с ошибкой", func(t *testing.T) {
		mock := &ractest.MockRACClient{
			TerminateSessionFunc: func(_ context.Context, _, _ string) error {
				return errors.New("сессия не найдена")
			},
		}
		err := mock.TerminateSession(ctx, "cluster", "session")
		if err == nil {
			t.Error("Ожидалась ошибка")
		}
	})

	t.Run("TerminateAllSessions с ошибкой", func(t *testing.T) {
		mock := &ractest.MockRACClient{
			TerminateAllSessionsFunc: func(_ context.Context, _, _ string) error {
				return errors.New("ошибка завершения сессий")
			},
		}
		err := mock.TerminateAllSessions(ctx, "cluster", "infobase")
		if err == nil {
			t.Error("Ожидалась ошибка")
		}
	})

	t.Run("EnableServiceMode с ошибкой", func(t *testing.T) {
		mock := &ractest.MockRACClient{
			EnableServiceModeFunc: func(_ context.Context, _, _ string, _ bool) error {
				return errors.New("ошибка включения режима")
			},
		}
		err := mock.EnableServiceMode(ctx, "cluster", "infobase", true)
		if err == nil {
			t.Error("Ожидалась ошибка")
		}
	})

	t.Run("DisableServiceMode с ошибкой", func(t *testing.T) {
		mock := &ractest.MockRACClient{
			DisableServiceModeFunc: func(_ context.Context, _, _ string) error {
				return errors.New("ошибка отключения режима")
			},
		}
		err := mock.DisableServiceMode(ctx, "cluster", "infobase")
		if err == nil {
			t.Error("Ожидалась ошибка")
		}
	})

	t.Run("GetServiceModeStatus с пользовательской функцией", func(t *testing.T) {
		mock := ractest.NewMockRACClientWithServiceMode(true, "Обновление конфигурации", 3)
		status, err := mock.GetServiceModeStatus(ctx, "cluster", "infobase")
		if err != nil {
			t.Fatalf("Не ожидалась ошибка, получено: %v", err)
		}
		if !status.Enabled {
			t.Error("Ожидался включённый режим")
		}
		if status.Message != "Обновление конфигурации" {
			t.Errorf("Ожидалось сообщение 'Обновление конфигурации', получено: %s", status.Message)
		}
		if status.ActiveSessions != 3 {
			t.Errorf("Ожидалось 3 активных сессии, получено: %d", status.ActiveSessions)
		}
	})

	t.Run("VerifyServiceMode с пользовательской функцией — совпадение", func(t *testing.T) {
		mock := ractest.NewMockRACClientWithServiceMode(true, "", 0)
		err := mock.VerifyServiceMode(ctx, "cluster", "infobase", true)
		if err != nil {
			t.Errorf("Не ожидалась ошибка, получено: %v", err)
		}
	})

	t.Run("VerifyServiceMode с пользовательской функцией — несовпадение", func(t *testing.T) {
		mock := ractest.NewMockRACClientWithServiceMode(false, "", 0)
		err := mock.VerifyServiceMode(ctx, "cluster", "infobase", true)
		if err == nil {
			t.Error("Ожидалась ошибка несоответствия")
		}
	})
}

// TestMockImplementsAllLegacyOperations проверяет, что mock реализует методы,
// соответствующие всем операциям из legacy internal/rac/ пакета.
// Проверка выполняется косвенно: если mock компилируется и реализует Client,
// значит все методы интерфейса определены. Тест вызывает каждый метод mock,
// подтверждая работоспособность дефолтных реализаций.
func TestMockImplementsAllLegacyOperations(t *testing.T) {
	// Маппинг legacy → новый интерфейс:
	// - GetClusterUUID → ClusterProvider.GetClusterInfo (возвращает ClusterInfo с UUID)
	// - GetInfobaseUUID → InfobaseProvider.GetInfobaseInfo (возвращает InfobaseInfo с UUID)
	// - GetSessions → SessionProvider.GetSessions
	// - TerminateSession → SessionProvider.TerminateSession
	// - TerminateAllSessions → SessionProvider.TerminateAllSessions
	// - EnableServiceMode → ServiceModeManager.EnableServiceMode
	// - DisableServiceMode → ServiceModeManager.DisableServiceMode
	// - GetServiceModeStatus → ServiceModeManager.GetServiceModeStatus
	// - VerifyServiceMode → ServiceModeManager.VerifyServiceMode

	mock := ractest.NewMockRACClient()
	ctx := context.Background()

	// ClusterProvider — аналог GetClusterUUID
	clusterInfo, err := mock.GetClusterInfo(ctx)
	if err != nil {
		t.Fatalf("GetClusterInfo не должен возвращать ошибку: %v", err)
	}
	if clusterInfo.UUID == "" {
		t.Error("ClusterInfo.UUID не должен быть пустым")
	}

	// InfobaseProvider — аналог GetInfobaseUUID
	ibInfo, err := mock.GetInfobaseInfo(ctx, clusterInfo.UUID, "test")
	if err != nil {
		t.Fatalf("GetInfobaseInfo не должен возвращать ошибку: %v", err)
	}
	if ibInfo.UUID == "" {
		t.Error("InfobaseInfo.UUID не должен быть пустым")
	}

	// SessionProvider — аналог GetSessions, TerminateSession, TerminateAllSessions
	_, err = mock.GetSessions(ctx, clusterInfo.UUID, ibInfo.UUID)
	if err != nil {
		t.Fatalf("GetSessions не должен возвращать ошибку: %v", err)
	}
	err = mock.TerminateSession(ctx, clusterInfo.UUID, "session-id")
	if err != nil {
		t.Fatalf("TerminateSession не должен возвращать ошибку: %v", err)
	}
	err = mock.TerminateAllSessions(ctx, clusterInfo.UUID, ibInfo.UUID)
	if err != nil {
		t.Fatalf("TerminateAllSessions не должен возвращать ошибку: %v", err)
	}

	// ServiceModeManager — аналог Enable/Disable/GetStatus/Verify
	err = mock.EnableServiceMode(ctx, clusterInfo.UUID, ibInfo.UUID, false)
	if err != nil {
		t.Fatalf("EnableServiceMode не должен возвращать ошибку: %v", err)
	}
	err = mock.DisableServiceMode(ctx, clusterInfo.UUID, ibInfo.UUID)
	if err != nil {
		t.Fatalf("DisableServiceMode не должен возвращать ошибку: %v", err)
	}
	_, err = mock.GetServiceModeStatus(ctx, clusterInfo.UUID, ibInfo.UUID)
	if err != nil {
		t.Fatalf("GetServiceModeStatus не должен возвращать ошибку: %v", err)
	}
	err = mock.VerifyServiceMode(ctx, clusterInfo.UUID, ibInfo.UUID, false)
	if err != nil {
		t.Fatalf("VerifyServiceMode не должен возвращать ошибку: %v", err)
	}
}

// TestDataTypes проверяет структуры данных.
func TestDataTypes(t *testing.T) {
	t.Run("ClusterInfo", func(t *testing.T) {
		info := rac.ClusterInfo{
			UUID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			Name: "production",
			Host: "server01",
			Port: 1541,
		}
		if info.UUID == "" || info.Name == "" || info.Host == "" || info.Port == 0 {
			t.Error("Все поля ClusterInfo должны быть заполнены")
		}
	})

	t.Run("InfobaseInfo", func(t *testing.T) {
		info := rac.InfobaseInfo{
			UUID:        "b2c3d4e5-f6a7-8901-bcde-f12345678901",
			Name:        "test-db",
			Description: "Тестовая база данных",
		}
		if info.UUID == "" || info.Name == "" || info.Description == "" {
			t.Error("Все поля InfobaseInfo должны быть заполнены")
		}
	})

	t.Run("ServiceModeStatus", func(t *testing.T) {
		status := rac.ServiceModeStatus{
			Enabled:              true,
			Message:              "Обновление конфигурации",
			ScheduledJobsBlocked: true,
			ActiveSessions:       5,
		}
		if !status.Enabled {
			t.Error("Ожидался включённый режим")
		}
		if status.Message != "Обновление конфигурации" {
			t.Errorf("Неверное сообщение: %s", status.Message)
		}
		if !status.ScheduledJobsBlocked {
			t.Error("Ожидалась блокировка регламентных заданий")
		}
		if status.ActiveSessions != 5 {
			t.Errorf("Ожидалось 5 активных сессий, получено: %d", status.ActiveSessions)
		}
	})

	t.Run("SessionInfo", func(t *testing.T) {
		now := time.Now()
		session := rac.SessionInfo{
			SessionID:    "c3d4e5f6-a7b8-9012-cdef-123456789012",
			UserName:     "Иванов",
			AppID:        "1CV8C",
			Host:         "workstation-01",
			StartedAt:    now.Add(-time.Hour),
			LastActiveAt: now,
		}
		if session.SessionID == "" || session.UserName == "" || session.AppID == "" {
			t.Error("Обязательные поля SessionInfo должны быть заполнены")
		}
		if session.Host == "" {
			t.Error("Поле Host должно быть заполнено")
		}
		if session.StartedAt.IsZero() || session.LastActiveAt.IsZero() {
			t.Error("Временные поля SessionInfo должны быть заполнены")
		}
		if !session.LastActiveAt.After(session.StartedAt) {
			t.Error("LastActiveAt должен быть после StartedAt")
		}
	})
}

// TestMockHelpers проверяет вспомогательные конструкторы mock.
func TestMockHelpers(t *testing.T) {
	t.Run("NewMockRACClient", func(t *testing.T) {
		mock := ractest.NewMockRACClient()
		if mock == nil {
			t.Fatal("NewMockRACClient не должен возвращать nil")
		}
	})

	t.Run("NewMockRACClientWithSessions", func(t *testing.T) {
		sessions := ractest.SessionData()
		mock := ractest.NewMockRACClientWithSessions(sessions)
		if mock == nil {
			t.Fatal("NewMockRACClientWithSessions не должен возвращать nil")
		}
		result, err := mock.GetSessions(context.Background(), "c", "i")
		if err != nil {
			t.Fatalf("Не ожидалась ошибка: %v", err)
		}
		if len(result) != len(sessions) {
			t.Errorf("Ожидалось %d сессий, получено: %d", len(sessions), len(result))
		}
	})

	t.Run("NewMockRACClientWithServiceMode", func(t *testing.T) {
		mock := ractest.NewMockRACClientWithServiceMode(true, "test", 2)
		if mock == nil {
			t.Fatal("NewMockRACClientWithServiceMode не должен возвращать nil")
		}
		status, err := mock.GetServiceModeStatus(context.Background(), "c", "i")
		if err != nil {
			t.Fatalf("Не ожидалась ошибка: %v", err)
		}
		if !status.Enabled {
			t.Error("Ожидался включённый режим")
		}
	})

	t.Run("SessionData возвращает непустые данные", func(t *testing.T) {
		data := ractest.SessionData()
		if len(data) == 0 {
			t.Error("SessionData не должен возвращать пустой срез")
		}
		for i, s := range data {
			if s.SessionID == "" {
				t.Errorf("Сессия %d: SessionID не должен быть пустым", i)
			}
			if s.UserName == "" {
				t.Errorf("Сессия %d: UserName не должен быть пустым", i)
			}
		}
	})
}
