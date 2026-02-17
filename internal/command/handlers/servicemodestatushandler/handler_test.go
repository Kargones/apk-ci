package servicemodestatushandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/onec/rac"
	"github.com/Kargones/apk-ci/internal/adapter/onec/rac/ractest"
	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/command/handlers/racutil"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceModeStatusHandler_Name(t *testing.T) {
	h := &ServiceModeStatusHandler{}
	assert.Equal(t, "nr-service-mode-status", h.Name())
	assert.Equal(t, constants.ActNRServiceModeStatus, h.Name())
}

func TestServiceModeStatusHandler_Description(t *testing.T) {
	h := &ServiceModeStatusHandler{}
	desc := h.Description()
	assert.NotEmpty(t, desc)
	assert.Equal(t, "Проверка статуса сервисного режима информационной базы", desc)
}

func TestServiceModeStatusHandler_Registration(t *testing.T) {
	// RegisterCmd() вызван в TestMain — проверяем что handler зарегистрирован
	h, ok := command.Get("nr-service-mode-status")
	require.True(t, ok, "handler nr-service-mode-status должен быть зарегистрирован в registry")
	assert.Equal(t, constants.ActNRServiceModeStatus, h.Name())
}

func TestServiceModeStatusHandler_DeprecatedAlias(t *testing.T) {
	// Проверяем что deprecated alias "service-mode-status" тоже работает
	h, ok := command.Get("service-mode-status")
	require.True(t, ok, "deprecated alias service-mode-status должен быть зарегистрирован в registry")

	// Проверяем что это DeprecatedBridge
	dep, isDep := h.(command.Deprecatable)
	require.True(t, isDep, "handler должен реализовывать Deprecatable")
	assert.True(t, dep.IsDeprecated())
	assert.Equal(t, "nr-service-mode-status", dep.NewName())
}

func TestServiceModeStatusHandler_Execute_TextOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	mockClient := ractest.NewMockRACClientWithServiceMode(true, "Система находится в режиме обслуживания", 5)
	// Устанавливаем ScheduledJobsBlocked для полного покрытия текстового вывода
	mockClient.GetServiceModeStatusFunc = func(_ context.Context, _, _ string) (*rac.ServiceModeStatus, error) {
		return &rac.ServiceModeStatus{
			Enabled:              true,
			Message:              "Система находится в режиме обслуживания",
			ScheduledJobsBlocked: true,
			ActiveSessions:       5,
		}, nil
	}

	h := &ServiceModeStatusHandler{racClient: mockClient}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)
	assert.Contains(t, out, "Сервисный режим: ВКЛЮЧЁН")
	assert.Contains(t, out, "Информационная база: TestBase")
	assert.Contains(t, out, "Система находится в режиме обслуживания")
	assert.Contains(t, out, "Регламентные задания: заблокированы")
	assert.Contains(t, out, "Активные сессии: 5")
}

func TestServiceModeStatusHandler_Execute_JSONOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mockClient := ractest.NewMockRACClientWithServiceMode(true, "Система находится в режиме обслуживания", 3)
	// Настраиваем mock для получения ScheduledJobsBlocked
	mockClient.GetServiceModeStatusFunc = func(_ context.Context, _, _ string) (*rac.ServiceModeStatus, error) {
		return &rac.ServiceModeStatus{
			Enabled:              true,
			Message:              "Система находится в режиме обслуживания",
			ScheduledJobsBlocked: true,
			ActiveSessions:       3,
		}, nil
	}

	h := &ServiceModeStatusHandler{racClient: mockClient}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err, "stdout должен содержать валидный JSON")

	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "nr-service-mode-status", result.Command)
	assert.NotNil(t, result.Data)

	// Проверяем поля data
	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok, "Data должен быть map")
	assert.Equal(t, true, dataMap["enabled"])
	assert.Equal(t, "Система находится в режиме обслуживания", dataMap["message"])
	assert.Equal(t, true, dataMap["scheduled_jobs_blocked"])
	assert.Equal(t, float64(3), dataMap["active_sessions"])
	assert.Equal(t, "TestBase", dataMap["infobase_name"])

	// Проверяем metadata
	require.NotNil(t, result.Metadata)
	assert.NotEmpty(t, result.Metadata.TraceID)
	assert.GreaterOrEqual(t, result.Metadata.DurationMs, int64(0))
	assert.Equal(t, constants.APIVersion, result.Metadata.APIVersion)
}

func TestServiceModeStatusHandler_Execute_NoInfobase(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &ServiceModeStatusHandler{}
	cfg := &config.Config{InfobaseName: ""}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.INFOBASE_MISSING")

	// Проверяем структурированный вывод ошибки
	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err, "stdout должен содержать валидный JSON ошибки")

	assert.Equal(t, "error", result.Status)
	assert.Equal(t, "nr-service-mode-status", result.Command)
	require.NotNil(t, result.Error)
	assert.Equal(t, "CONFIG.INFOBASE_MISSING", result.Error.Code)
	assert.Contains(t, result.Error.Message, "BR_INFOBASE_NAME")
}

func TestServiceModeStatusHandler_Execute_NilConfig(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &ServiceModeStatusHandler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.INFOBASE_MISSING")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
}

func TestServiceModeStatusHandler_Execute_RACError(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mockClient := &ractest.MockRACClient{
		GetClusterInfoFunc: func(_ context.Context) (*rac.ClusterInfo, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	h := &ServiceModeStatusHandler{racClient: mockClient}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "RAC.CLUSTER_FAILED")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "RAC.CLUSTER_FAILED", result.Error.Code)
}

func TestServiceModeStatusHandler_Execute_RACStepErrors(t *testing.T) {
	tests := []struct {
		name     string
		mock     *ractest.MockRACClient
		errCode  string
	}{
		{
			name: "InfobaseError",
			mock: &ractest.MockRACClient{
				GetInfobaseInfoFunc: func(_ context.Context, _, _ string) (*rac.InfobaseInfo, error) {
					return nil, fmt.Errorf("infobase not found")
				},
			},
			errCode: "RAC.INFOBASE_FAILED",
		},
		{
			name: "StatusError",
			mock: &ractest.MockRACClient{
				GetServiceModeStatusFunc: func(_ context.Context, _, _ string) (*rac.ServiceModeStatus, error) {
					return nil, fmt.Errorf("permission denied")
				},
			},
			errCode: "RAC.STATUS_FAILED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BR_OUTPUT_FORMAT", "json")

			h := &ServiceModeStatusHandler{racClient: tt.mock}
			cfg := &config.Config{InfobaseName: "TestBase"}
			ctx := context.Background()

			var execErr error
			out := testutil.CaptureStdout(t, func() {
				execErr = h.Execute(ctx, cfg)
			})

			require.Error(t, execErr)
			assert.Contains(t, execErr.Error(), tt.errCode)

			var result output.Result
			err := json.Unmarshal([]byte(out), &result)
			require.NoError(t, err)
			assert.Equal(t, "error", result.Status)
			require.NotNil(t, result.Error)
			assert.Equal(t, tt.errCode, result.Error.Code)
		})
	}
}

func TestServiceModeStatusHandler_Execute_TextErrorOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	h := &ServiceModeStatusHandler{}
	cfg := &config.Config{InfobaseName: ""}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.INFOBASE_MISSING")

	// Проверяем человекочитаемый формат ошибки (не JSON)
	assert.Contains(t, out, "Ошибка:")
	assert.Contains(t, out, "BR_INFOBASE_NAME")
	assert.Contains(t, out, "Код: CONFIG.INFOBASE_MISSING")
	// Текстовый формат НЕ должен содержать JSON-структуру
	assert.NotContains(t, out, `"status"`)
}

func TestServiceModeStatusHandler_CreateRACClient_Errors(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr string
	}{
		{
			name: "NilAppConfig",
			cfg: &config.Config{
				InfobaseName: "TestBase",
			},
			wantErr: "конфигурация приложения не загружена",
		},
		{
			name: "EmptyServer",
			cfg: &config.Config{
				InfobaseName: "TestBase",
				AppConfig: &config.AppConfig{},
			},
			wantErr: "не удалось определить сервер",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := racutil.NewClient(tt.cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestServiceModeStatusHandler_Execute_DisabledMode(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	mockClient := ractest.NewMockRACClientWithServiceMode(false, "", 0)

	h := &ServiceModeStatusHandler{racClient: mockClient}
	cfg := &config.Config{InfobaseName: "ProdBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)
	assert.Contains(t, out, "Сервисный режим: ВЫКЛЮЧЕН")
	assert.Contains(t, out, "Информационная база: ProdBase")
	assert.Contains(t, out, "Регламентные задания: разблокированы")
	assert.Contains(t, out, "Активные сессии: 0")
}

// === Story 2.4: Session Info тесты ===

func TestServiceModeStatusHandler_Execute_JSONOutput_WithSessions(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mockClient := ractest.NewMockRACClientWithServiceMode(true, "Обслуживание", 2)
	mockClient.GetSessionsFunc = func(_ context.Context, _, _ string) ([]rac.SessionInfo, error) {
		return ractest.SessionData(), nil
	}

	h := &ServiceModeStatusHandler{racClient: mockClient}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err, "stdout должен содержать валидный JSON")

	assert.Equal(t, "success", result.Status)

	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok, "Data должен быть map")

	sessionsRaw, ok := dataMap["sessions"]
	require.True(t, ok, "Data должен содержать sessions")

	sessions, ok := sessionsRaw.([]any)
	require.True(t, ok, "sessions должен быть массивом")
	assert.Len(t, sessions, 2)

	// Проверяем поля первой сессии
	s0, ok := sessions[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Иванов", s0["user_name"])
	assert.Equal(t, "workstation-01", s0["host"])
	assert.Equal(t, "1CV8C", s0["app_id"])
	assert.NotEmpty(t, s0["started_at"])
	assert.NotEmpty(t, s0["last_active_at"])
}

func TestServiceModeStatusHandler_Execute_TextOutput_WithSessions(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	mockClient := ractest.NewMockRACClientWithServiceMode(true, "Обслуживание", 2)
	mockClient.GetSessionsFunc = func(_ context.Context, _, _ string) ([]rac.SessionInfo, error) {
		return ractest.SessionData(), nil
	}

	h := &ServiceModeStatusHandler{racClient: mockClient}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)
	assert.Contains(t, out, "Детали сессий:")
	assert.Contains(t, out, "Иванов")
	assert.Contains(t, out, "1CV8C")
	assert.Contains(t, out, "workstation-01")
	assert.Contains(t, out, "Петров")
	assert.Contains(t, out, "workstation-02")
	assert.NotContains(t, out, "Нет активных сессий")
	assert.NotContains(t, out, "... и ещё")
}

func TestServiceModeStatusHandler_Execute_TextOutput_TopFiveSessions(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	// Создаём 7 сессий для проверки top-5 truncation
	sessions := make([]rac.SessionInfo, 7)
	for i := range 7 {
		sessions[i] = rac.SessionInfo{
			SessionID:    fmt.Sprintf("session-%d", i),
			UserName:     fmt.Sprintf("User%d", i+1),
			AppID:        "1CV8C",
			Host:         fmt.Sprintf("host-%d", i),
			StartedAt:    time.Date(2026, 1, 27, 9, 0, 0, 0, time.UTC),
			LastActiveAt: time.Date(2026, 1, 27, 10, 0, 0, 0, time.UTC),
		}
	}

	mockClient := ractest.NewMockRACClientWithServiceMode(true, "Обслуживание", 7)
	mockClient.GetSessionsFunc = func(_ context.Context, _, _ string) ([]rac.SessionInfo, error) {
		return sessions, nil
	}

	h := &ServiceModeStatusHandler{racClient: mockClient}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)
	assert.Contains(t, out, "Детали сессий:")
	// Первые 5 сессий отображаются
	assert.Contains(t, out, "User1")
	assert.Contains(t, out, "User5")
	// 6-я и 7-я не отображаются
	assert.NotContains(t, out, "User6")
	assert.NotContains(t, out, "User7")
	// Показываем оставшиеся
	assert.Contains(t, out, "... и ещё 2 сессий")
}

func TestServiceModeStatusHandler_Execute_NoSessions(t *testing.T) {
	t.Run("TextFormat", func(t *testing.T) {
		t.Setenv("BR_OUTPUT_FORMAT", "text")

		mockClient := ractest.NewMockRACClientWithServiceMode(false, "", 0)
		// GetSessions возвращает пустой список (дефолт mock)

		h := &ServiceModeStatusHandler{racClient: mockClient}
		cfg := &config.Config{InfobaseName: "TestBase"}
		ctx := context.Background()

		var execErr error
		out := testutil.CaptureStdout(t, func() {
			execErr = h.Execute(ctx, cfg)
		})

		require.NoError(t, execErr)
		assert.Contains(t, out, "Детали сессий:")
		assert.Contains(t, out, "Нет активных сессий")
	})

	t.Run("JSONFormat", func(t *testing.T) {
		t.Setenv("BR_OUTPUT_FORMAT", "json")

		mockClient := ractest.NewMockRACClientWithServiceMode(false, "", 0)

		h := &ServiceModeStatusHandler{racClient: mockClient}
		cfg := &config.Config{InfobaseName: "TestBase"}
		ctx := context.Background()

		var execErr error
		out := testutil.CaptureStdout(t, func() {
			execErr = h.Execute(ctx, cfg)
		})

		require.NoError(t, execErr)

		var result output.Result
		err := json.Unmarshal([]byte(out), &result)
		require.NoError(t, err)

		dataMap, ok := result.Data.(map[string]any)
		require.True(t, ok)

		sessionsRaw, ok := dataMap["sessions"]
		require.True(t, ok, "sessions должен присутствовать в JSON")

		sessions, ok := sessionsRaw.([]any)
		require.True(t, ok, "sessions должен быть массивом")
		assert.Empty(t, sessions, "sessions должен быть пустым массивом")
	})
}

func TestServiceModeStatusHandler_Execute_SessionsFetchError(t *testing.T) {
	t.Run("JSONFormat", func(t *testing.T) {
		t.Setenv("BR_OUTPUT_FORMAT", "json")

		mockClient := ractest.NewMockRACClientWithServiceMode(true, "Обслуживание", 3)
		mockClient.GetSessionsFunc = func(_ context.Context, _, _ string) ([]rac.SessionInfo, error) {
			return nil, fmt.Errorf("connection failed")
		}

		h := &ServiceModeStatusHandler{racClient: mockClient}
		cfg := &config.Config{InfobaseName: "TestBase"}
		ctx := context.Background()

		var execErr error
		out := testutil.CaptureStdout(t, func() {
			execErr = h.Execute(ctx, cfg)
		})

		// Команда не падает — graceful degradation
		require.NoError(t, execErr)

		var result output.Result
		err := json.Unmarshal([]byte(out), &result)
		require.NoError(t, err)

		assert.Equal(t, "success", result.Status)

		dataMap, ok := result.Data.(map[string]any)
		require.True(t, ok)

		// Сессии пустые при ошибке
		sessionsRaw, ok := dataMap["sessions"]
		require.True(t, ok)

		sessions, ok := sessionsRaw.([]any)
		require.True(t, ok)
		assert.Empty(t, sessions, "При ошибке GetSessions sessions должен быть пустым массивом")

		// Основные данные статуса присутствуют
		assert.Equal(t, true, dataMap["enabled"])
		assert.Equal(t, float64(3), dataMap["active_sessions"])
	})

	t.Run("TextFormat", func(t *testing.T) {
		t.Setenv("BR_OUTPUT_FORMAT", "text")

		mockClient := ractest.NewMockRACClientWithServiceMode(true, "Обслуживание", 3)
		mockClient.GetSessionsFunc = func(_ context.Context, _, _ string) ([]rac.SessionInfo, error) {
			return nil, fmt.Errorf("connection failed")
		}

		h := &ServiceModeStatusHandler{racClient: mockClient}
		cfg := &config.Config{InfobaseName: "TestBase"}
		ctx := context.Background()

		var execErr error
		out := testutil.CaptureStdout(t, func() {
			execErr = h.Execute(ctx, cfg)
		})

		require.NoError(t, execErr)
		assert.Contains(t, out, "Активные сессии: 3")
		assert.Contains(t, out, "Детали сессий:")
		// При ошибке GetSessions и ActiveSessions > 0 — не "Нет активных сессий", а предупреждение
		assert.Contains(t, out, "Не удалось получить детали сессий")
		assert.NotContains(t, out, "Нет активных сессий")
	})
}

// ==== PLAN-ONLY TESTS (Story 7.3) ====

// TestServiceModeStatusHandler_PlanOnly проверяет что при BR_PLAN_ONLY=true
// выводится сообщение о неподдерживаемом плане и операция не выполняется (AC-8).
func TestServiceModeStatusHandler_PlanOnly(t *testing.T) {
	t.Setenv("BR_PLAN_ONLY", "true")

	// Используем mock который падает при вызове — доказываем что RAC не вызывается
	racMock := ractest.NewMockRACClient()
	racMock.GetServiceModeStatusFunc = func(ctx context.Context, clusterUUID, infobaseUUID string) (*rac.ServiceModeStatus, error) {
		t.Fatal("GetServiceModeStatus() не должен вызываться в plan-only режиме")
		return nil, nil
	}

	h := &ServiceModeStatusHandler{racClient: racMock}
	cfg := &config.Config{InfobaseName: "TestBase"}

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(context.Background(), cfg)
	})

	require.NoError(t, execErr)
	assert.Contains(t, out, "не поддерживает отображение плана операций")
	assert.Contains(t, out, constants.ActNRServiceModeStatus)
}
