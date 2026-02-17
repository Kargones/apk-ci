package servicemodeenablehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"

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

func TestServiceModeEnableHandler_Name(t *testing.T) {
	h := &ServiceModeEnableHandler{}
	assert.Equal(t, "nr-service-mode-enable", h.Name())
	assert.Equal(t, constants.ActNRServiceModeEnable, h.Name())
}

func TestServiceModeEnableHandler_Description(t *testing.T) {
	h := &ServiceModeEnableHandler{}
	desc := h.Description()
	assert.NotEmpty(t, desc)
	assert.Equal(t, "Включение сервисного режима информационной базы", desc)
}

func TestServiceModeEnableHandler_Registration(t *testing.T) {
	// RegisterCmd() вызван в TestMain — проверяем что handler зарегистрирован
	h, ok := command.Get("nr-service-mode-enable")
	require.True(t, ok, "handler nr-service-mode-enable должен быть зарегистрирован в registry")
	assert.Equal(t, constants.ActNRServiceModeEnable, h.Name())
}

func TestServiceModeEnableHandler_DeprecatedAlias(t *testing.T) {
	// Проверяем что deprecated alias "service-mode-enable" тоже работает
	h, ok := command.Get("service-mode-enable")
	require.True(t, ok, "deprecated alias service-mode-enable должен быть зарегистрирован в registry")

	// Проверяем что это DeprecatedBridge
	dep, isDep := h.(command.Deprecatable)
	require.True(t, isDep, "handler должен реализовывать Deprecatable")
	assert.True(t, dep.IsDeprecated())
	assert.Equal(t, "nr-service-mode-enable", dep.NewName())
}

func TestServiceModeEnableHandler_Execute_TextOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	mockClient := newEnableMock(false, 0)

	h := &ServiceModeEnableHandler{racClient: mockClient}
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
	assert.Contains(t, out, "Код разрешения: ServiceMode")
	assert.Contains(t, out, "Регламентные задания: заблокированы")
	// При terminateSessions=false строки "Завершено сессий" быть не должно
	assert.NotContains(t, out, "Завершено сессий")
}

func TestServiceModeEnableHandler_Execute_JSONOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mockClient := newEnableMock(false, 0)

	h := &ServiceModeEnableHandler{racClient: mockClient}
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
	assert.Equal(t, "nr-service-mode-enable", result.Command)
	assert.NotNil(t, result.Data)

	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok, "Data должен быть map")
	assert.Equal(t, true, dataMap["enabled"])
	assert.Equal(t, false, dataMap["already_enabled"])
	assert.Equal(t, true, dataMap["state_changed"])
	assert.Equal(t, "Система находится в режиме обслуживания", dataMap["message"])
	assert.Equal(t, "ServiceMode", dataMap["permission_code"])
	assert.Equal(t, true, dataMap["scheduled_jobs_blocked"])
	assert.Equal(t, float64(0), dataMap["terminated_sessions_count"])
	assert.Equal(t, "TestBase", dataMap["infobase_name"])

	// Проверяем metadata
	require.NotNil(t, result.Metadata)
	assert.NotEmpty(t, result.Metadata.TraceID)
	assert.GreaterOrEqual(t, result.Metadata.DurationMs, int64(0))
	assert.Equal(t, constants.APIVersion, result.Metadata.APIVersion)
}

func TestServiceModeEnableHandler_Execute_AlreadyEnabled(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mockClient := ractest.NewMockRACClientWithServiceMode(true, "Режим обслуживания", 0)
	h := &ServiceModeEnableHandler{racClient: mockClient}
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
	assert.Equal(t, "success", result.Status)

	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, dataMap["already_enabled"])
	assert.Equal(t, true, dataMap["enabled"])
	assert.Equal(t, false, dataMap["state_changed"])
	assert.Equal(t, "Режим обслуживания", dataMap["message"])
}

func TestServiceModeEnableHandler_Execute_AlreadyEnabled_TextOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	mockClient := ractest.NewMockRACClientWithServiceMode(true, "Режим обслуживания", 0)
	h := &ServiceModeEnableHandler{racClient: mockClient}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)
	assert.Contains(t, out, "ВКЛЮЧЁН (уже был включён)")
	assert.Contains(t, out, "Информационная база: TestBase")
	// При already_enabled НЕ должно быть строки "Регламентные задания"
	assert.NotContains(t, out, "Регламентные задания")
}

func TestServiceModeEnableHandler_Execute_WithTerminateSessions(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mockClient := newEnableMock(false, 3)

	h := &ServiceModeEnableHandler{racClient: mockClient}
	cfg := &config.Config{
		InfobaseName:      "TestBase",
		TerminateSessions: true,
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)

	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, false, dataMap["already_enabled"])
	assert.Equal(t, float64(3), dataMap["terminated_sessions_count"])
}

func TestServiceModeEnableHandler_Execute_WithTerminateSessions_TextOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	mockClient := newEnableMock(false, 5)

	h := &ServiceModeEnableHandler{racClient: mockClient}
	cfg := &config.Config{
		InfobaseName:      "TestBase",
		TerminateSessions: true,
	}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)
	assert.Contains(t, out, "Завершено сессий: 5")
}

func TestServiceModeEnableHandler_Execute_NoInfobase(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &ServiceModeEnableHandler{}
	cfg := &config.Config{InfobaseName: ""}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.INFOBASE_MISSING")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err, "stdout должен содержать валидный JSON ошибки")

	assert.Equal(t, "error", result.Status)
	assert.Equal(t, "nr-service-mode-enable", result.Command)
	require.NotNil(t, result.Error)
	assert.Equal(t, "CONFIG.INFOBASE_MISSING", result.Error.Code)
	assert.Contains(t, result.Error.Message, "BR_INFOBASE_NAME")
}

func TestServiceModeEnableHandler_Execute_NilConfig(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &ServiceModeEnableHandler{}
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

func TestServiceModeEnableHandler_Execute_TextErrorOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	h := &ServiceModeEnableHandler{}
	cfg := &config.Config{InfobaseName: ""}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "CONFIG.INFOBASE_MISSING")

	assert.Contains(t, out, "Ошибка:")
	assert.Contains(t, out, "BR_INFOBASE_NAME")
	assert.Contains(t, out, "Код: CONFIG.INFOBASE_MISSING")
	assert.NotContains(t, out, `"status"`)
}

func TestServiceModeEnableHandler_Execute_RACError(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	tests := []struct {
		name    string
		mock    *ractest.MockRACClient
		errCode string
	}{
		{
			name: "ClusterError",
			mock: &ractest.MockRACClient{
				GetClusterInfoFunc: func(_ context.Context) (*rac.ClusterInfo, error) {
					return nil, fmt.Errorf("connection refused")
				},
			},
			errCode: "RAC.CLUSTER_FAILED",
		},
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
			name: "EnableError",
			mock: func() *ractest.MockRACClient {
				m := ractest.NewMockRACClient()
				m.GetServiceModeStatusFunc = func(_ context.Context, _, _ string) (*rac.ServiceModeStatus, error) {
					return &rac.ServiceModeStatus{Enabled: false}, nil
				}
				m.EnableServiceModeFunc = func(_ context.Context, _, _ string, _ bool) error {
					return fmt.Errorf("enable failed")
				}
				return m
			}(),
			errCode: "RAC.ENABLE_FAILED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BR_OUTPUT_FORMAT", "json")

			h := &ServiceModeEnableHandler{racClient: tt.mock}
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

func TestServiceModeEnableHandler_Execute_VerifyFailed(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mockClient := ractest.NewMockRACClient()
	mockClient.GetServiceModeStatusFunc = func(_ context.Context, _, _ string) (*rac.ServiceModeStatus, error) {
		return &rac.ServiceModeStatus{Enabled: false}, nil
	}
	mockClient.EnableServiceModeFunc = func(_ context.Context, _, _ string, _ bool) error {
		return nil
	}
	mockClient.VerifyServiceModeFunc = func(_ context.Context, _, _ string, _ bool) error {
		return fmt.Errorf("service mode verification failed: expected true, got false")
	}

	h := &ServiceModeEnableHandler{racClient: mockClient}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "RAC.VERIFY_FAILED")

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "error", result.Status)
	require.NotNil(t, result.Error)
	assert.Equal(t, "RAC.VERIFY_FAILED", result.Error.Code)
}

func TestServiceModeEnableHandler_CreateRACClient_Errors(t *testing.T) {
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
				AppConfig:    &config.AppConfig{},
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

func TestServiceModeEnableHandler_Execute_CustomEnvParams(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	t.Setenv("BR_SERVICE_MODE_MESSAGE", "Плановое обслуживание до 18:00")
	t.Setenv("BR_SERVICE_MODE_PERMISSION_CODE", "CustomCode")

	mockClient := newEnableMock(false, 0)

	h := &ServiceModeEnableHandler{racClient: mockClient}
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
	assert.Equal(t, "Плановое обслуживание до 18:00", dataMap["message"])
	assert.Equal(t, "CustomCode", dataMap["permission_code"])
}

// newEnableMock создаёт MockRACClient для тестирования enable flow.
// Если serviceEnabled=true, GetServiceModeStatus вернёт включённый режим (идемпотентность).
// activeSessions — количество активных сессий для подсчёта завершённых.
func newEnableMock(serviceEnabled bool, activeSessions int) *ractest.MockRACClient {
	mock := ractest.NewMockRACClient()
	// Используем счётчик для симуляции изменения состояния после EnableServiceMode:
	// - Первый вызов (pre-check): возвращает исходное состояние
	// - Второй вызов (post-check): после enable режим включён, задания заблокированы
	var callCount atomic.Int32
	mock.GetServiceModeStatusFunc = func(_ context.Context, _, _ string) (*rac.ServiceModeStatus, error) {
		callCount.Add(1)
		if serviceEnabled || callCount.Load() > 1 {
			// Режим включён (изначально или после enable)
			return &rac.ServiceModeStatus{
				Enabled:              true,
				Message:              "Система находится в режиме обслуживания",
				ScheduledJobsBlocked: true,
				ActiveSessions:       activeSessions,
			}, nil
		}
		// Режим выключен (pre-check)
		return &rac.ServiceModeStatus{
			Enabled:              false,
			Message:              "",
			ScheduledJobsBlocked: false,
			ActiveSessions:       activeSessions,
		}, nil
	}
	mock.EnableServiceModeFunc = func(_ context.Context, _, _ string, _ bool) error {
		return nil
	}
	mock.VerifyServiceModeFunc = func(_ context.Context, _, _ string, _ bool) error {
		return nil
	}
	return mock
}
