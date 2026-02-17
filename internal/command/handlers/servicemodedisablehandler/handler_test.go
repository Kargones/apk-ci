package servicemodedisablehandler

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

func TestServiceModeDisableHandler_Name(t *testing.T) {
	h := &ServiceModeDisableHandler{}
	assert.Equal(t, "nr-service-mode-disable", h.Name())
	assert.Equal(t, constants.ActNRServiceModeDisable, h.Name())
}

func TestServiceModeDisableHandler_Description(t *testing.T) {
	h := &ServiceModeDisableHandler{}
	desc := h.Description()
	assert.NotEmpty(t, desc)
	assert.Equal(t, "Отключение сервисного режима информационной базы", desc)
}

func TestServiceModeDisableHandler_Registration(t *testing.T) {
	// RegisterCmd() вызван в TestMain — проверяем что handler зарегистрирован
	h, ok := command.Get("nr-service-mode-disable")
	require.True(t, ok, "handler nr-service-mode-disable должен быть зарегистрирован в registry")
	assert.Equal(t, constants.ActNRServiceModeDisable, h.Name())
}

func TestServiceModeDisableHandler_DeprecatedAlias(t *testing.T) {
	// Проверяем что deprecated alias "service-mode-disable" тоже работает
	h, ok := command.Get("service-mode-disable")
	require.True(t, ok, "deprecated alias service-mode-disable должен быть зарегистрирован в registry")

	// Проверяем что это DeprecatedBridge
	dep, isDep := h.(command.Deprecatable)
	require.True(t, isDep, "handler должен реализовывать Deprecatable")
	assert.True(t, dep.IsDeprecated())
	assert.Equal(t, "nr-service-mode-disable", dep.NewName())
}

func TestServiceModeDisableHandler_Execute_TextOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	mockClient := newDisableMock(true)

	h := &ServiceModeDisableHandler{racClient: mockClient}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)
	assert.Contains(t, out, "Сервисный режим: ОТКЛЮЧЁН")
	assert.Contains(t, out, "Информационная база: TestBase")
	assert.Contains(t, out, "Регламентные задания: разблокированы")
	assert.NotContains(t, out, "уже был отключён")
}

func TestServiceModeDisableHandler_Execute_JSONOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mockClient := newDisableMock(true)

	h := &ServiceModeDisableHandler{racClient: mockClient}
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
	assert.Equal(t, "nr-service-mode-disable", result.Command)
	assert.NotNil(t, result.Data)

	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok, "Data должен быть map")
	assert.Equal(t, true, dataMap["disabled"])
	assert.Equal(t, false, dataMap["already_disabled"])
	assert.Equal(t, true, dataMap["state_changed"])
	assert.Equal(t, true, dataMap["scheduled_jobs_unblocked"])
	assert.Equal(t, "TestBase", dataMap["infobase_name"])

	// Проверяем metadata
	require.NotNil(t, result.Metadata)
	assert.NotEmpty(t, result.Metadata.TraceID)
	assert.GreaterOrEqual(t, result.Metadata.DurationMs, int64(0))
	assert.Equal(t, constants.APIVersion, result.Metadata.APIVersion)
}

func TestServiceModeDisableHandler_Execute_AlreadyDisabled(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	// Mock: режим уже отключён (Enabled: false)
	mockClient := ractest.NewMockRACClientWithServiceMode(false, "", 0)
	h := &ServiceModeDisableHandler{racClient: mockClient}
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
	assert.Equal(t, true, dataMap["already_disabled"])
	assert.Equal(t, true, dataMap["disabled"])
	assert.Equal(t, false, dataMap["state_changed"])
}

func TestServiceModeDisableHandler_Execute_AlreadyDisabled_TextOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	mockClient := ractest.NewMockRACClientWithServiceMode(false, "", 0)
	h := &ServiceModeDisableHandler{racClient: mockClient}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)
	assert.Contains(t, out, "ОТКЛЮЧЁН (уже был отключён)")
	assert.Contains(t, out, "Информационная база: TestBase")
	// При already_disabled НЕ должно быть строки "Регламентные задания"
	assert.NotContains(t, out, "Регламентные задания")
}

func TestServiceModeDisableHandler_Execute_ScheduledJobsNotUnblocked(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	mockClient := newDisableMock(false) // scheduledJobsUnblocked = false

	h := &ServiceModeDisableHandler{racClient: mockClient}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)
	assert.Contains(t, out, "Сервисный режим: ОТКЛЮЧЁН")
	assert.Contains(t, out, "Регламентные задания: не разблокированы (отдельная блокировка)")
}

func TestServiceModeDisableHandler_Execute_NoInfobase(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &ServiceModeDisableHandler{}
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
	assert.Equal(t, "nr-service-mode-disable", result.Command)
	require.NotNil(t, result.Error)
	assert.Equal(t, "CONFIG.INFOBASE_MISSING", result.Error.Code)
	assert.Contains(t, result.Error.Message, "BR_INFOBASE_NAME")
}

func TestServiceModeDisableHandler_Execute_RACError(t *testing.T) {
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
			mock: func() *ractest.MockRACClient {
				m := ractest.NewMockRACClient()
				m.GetInfobaseInfoFunc = func(_ context.Context, _, _ string) (*rac.InfobaseInfo, error) {
					return nil, fmt.Errorf("infobase not found")
				}
				return m
			}(),
			errCode: "RAC.INFOBASE_FAILED",
		},
		{
			name: "DisableError",
			mock: func() *ractest.MockRACClient {
				m := ractest.NewMockRACClient()
				m.GetServiceModeStatusFunc = func(_ context.Context, _, _ string) (*rac.ServiceModeStatus, error) {
					return &rac.ServiceModeStatus{Enabled: true, Message: "Обслуживание"}, nil
				}
				m.DisableServiceModeFunc = func(_ context.Context, _, _ string) error {
					return fmt.Errorf("disable failed")
				}
				return m
			}(),
			errCode: "RAC.DISABLE_FAILED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ServiceModeDisableHandler{racClient: tt.mock}
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

func TestServiceModeDisableHandler_Execute_VerifyFailed(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mockClient := ractest.NewMockRACClient()
	mockClient.GetServiceModeStatusFunc = func(_ context.Context, _, _ string) (*rac.ServiceModeStatus, error) {
		return &rac.ServiceModeStatus{Enabled: true, Message: "Обслуживание"}, nil
	}
	mockClient.DisableServiceModeFunc = func(_ context.Context, _, _ string) error {
		return nil
	}
	mockClient.VerifyServiceModeFunc = func(_ context.Context, _, _ string, _ bool) error {
		return fmt.Errorf("service mode verification failed: expected false, got true")
	}

	h := &ServiceModeDisableHandler{racClient: mockClient}
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

func TestServiceModeDisableHandler_Execute_StatusCheckFailOpen(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	// Mock: GetServiceModeStatus возвращает ошибку — handler должен продолжить (fail-open)
	mockClient := ractest.NewMockRACClient()
	mockClient.GetServiceModeStatusFunc = func(_ context.Context, _, _ string) (*rac.ServiceModeStatus, error) {
		return nil, fmt.Errorf("status check unavailable")
	}
	mockClient.DisableServiceModeFunc = func(_ context.Context, _, _ string) error {
		return nil
	}
	mockClient.VerifyServiceModeFunc = func(_ context.Context, _, _ string, _ bool) error {
		return nil
	}

	h := &ServiceModeDisableHandler{racClient: mockClient}
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
	assert.Equal(t, true, dataMap["disabled"])
	assert.Equal(t, false, dataMap["already_disabled"])
}

func TestServiceModeDisableHandler_Execute_PostStatusNilNil(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	// Mock: pre-check возвращает enabled, post-check возвращает (nil, nil) — не должно быть паники
	var callCount atomic.Int32
	mockClient := ractest.NewMockRACClient()
	mockClient.GetServiceModeStatusFunc = func(_ context.Context, _, _ string) (*rac.ServiceModeStatus, error) {
		n := callCount.Add(1)
		if n == 1 {
			return &rac.ServiceModeStatus{Enabled: true, Message: "Обслуживание"}, nil
		}
		// Второй вызов — (nil, nil)
		return nil, nil
	}
	mockClient.DisableServiceModeFunc = func(_ context.Context, _, _ string) error {
		return nil
	}
	mockClient.VerifyServiceModeFunc = func(_ context.Context, _, _ string, _ bool) error {
		return nil
	}

	h := &ServiceModeDisableHandler{racClient: mockClient}
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
	assert.Equal(t, true, dataMap["disabled"])
	// При (nil, nil) от post-check — scheduledJobsUnblocked должен быть true (default)
	assert.Equal(t, true, dataMap["scheduled_jobs_unblocked"])
}

func TestServiceModeDisableHandler_Execute_NilConfig(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &ServiceModeDisableHandler{}
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

func TestServiceModeDisableHandler_Execute_TextErrorOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	h := &ServiceModeDisableHandler{}
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

func TestServiceModeDisableHandler_CreateRACClient_Errors(t *testing.T) {
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

// newDisableMock создаёт MockRACClient для тестирования disable flow.
// scheduledJobsUnblocked определяет результат проверки заданий после отключения.
func newDisableMock(scheduledJobsUnblocked bool) *ractest.MockRACClient {
	var callCount atomic.Int32
	mock := ractest.NewMockRACClient()
	mock.GetServiceModeStatusFunc = func(_ context.Context, _, _ string) (*rac.ServiceModeStatus, error) {
		n := callCount.Add(1)
		if n == 1 {
			// Первый вызов — проверка идемпотентности: режим включён
			return &rac.ServiceModeStatus{
				Enabled:              true,
				Message:              "Обслуживание",
				ScheduledJobsBlocked: true,
			}, nil
		}
		// Второй вызов — после отключения: проверка заданий
		return &rac.ServiceModeStatus{
			Enabled:              false,
			ScheduledJobsBlocked: !scheduledJobsUnblocked,
		}, nil
	}
	mock.DisableServiceModeFunc = func(_ context.Context, _, _ string) error {
		return nil
	}
	mock.VerifyServiceModeFunc = func(_ context.Context, _, _ string, _ bool) error {
		return nil
	}
	return mock
}
