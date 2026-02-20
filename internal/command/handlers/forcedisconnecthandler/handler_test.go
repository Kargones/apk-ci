package forcedisconnecthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/Kargones/apk-ci/internal/adapter/onec/rac"
	"github.com/Kargones/apk-ci/internal/adapter/onec/rac/ractest"
	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForceDisconnectHandler_Name(t *testing.T) {
	h := &ForceDisconnectHandler{}
	assert.Equal(t, "nr-force-disconnect-sessions", h.Name())
	assert.Equal(t, constants.ActNRForceDisconnectSessions, h.Name())
}

func TestForceDisconnectHandler_Description(t *testing.T) {
	h := &ForceDisconnectHandler{}
	desc := h.Description()
	assert.NotEmpty(t, desc)
	assert.Equal(t, "Принудительное завершение сессий информационной базы", desc)
}

func TestForceDisconnectHandler_Registration(t *testing.T) {
	h, ok := command.Get("nr-force-disconnect-sessions")
	require.True(t, ok, "handler nr-force-disconnect-sessions должен быть зарегистрирован в registry")
	assert.Equal(t, constants.ActNRForceDisconnectSessions, h.Name())
}

func TestForceDisconnectHandler_Execute_TextOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	sessions := ractest.SessionData()
	mockClient := ractest.NewMockRACClientWithSessions(sessions)

	h := &ForceDisconnectHandler{racClient: mockClient}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr)
	assert.Contains(t, out, "Принудительное завершение сессий: TestBase")
	assert.Contains(t, out, "Завершено сессий: 2")
	assert.Contains(t, out, "Иванов (1CV8C) — workstation-01")
	assert.Contains(t, out, "Петров (1CV8) — workstation-02")
}

func TestForceDisconnectHandler_Execute_JSONOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	sessions := ractest.SessionData()
	mockClient := ractest.NewMockRACClientWithSessions(sessions)

	h := &ForceDisconnectHandler{racClient: mockClient}
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
	assert.Equal(t, "nr-force-disconnect-sessions", result.Command)
	assert.NotNil(t, result.Data)

	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok, "Data должен быть map")
	assert.Equal(t, float64(2), dataMap["terminated_sessions_count"])
	assert.Equal(t, false, dataMap["no_active_sessions"])
	assert.Equal(t, true, dataMap["state_changed"])
	assert.Equal(t, false, dataMap["partial_failure"])
	assert.Equal(t, "TestBase", dataMap["infobase_name"])

	sessions2, ok := dataMap["sessions"].([]any)
	require.True(t, ok, "sessions должен быть массивом")
	assert.Len(t, sessions2, 2)

	// Проверяем что errors — пустой массив при отсутствии ошибок
	errorsVal, hasErrors := dataMap["errors"]
	require.True(t, hasErrors, "errors должен присутствовать в JSON")
	errorsArr, ok := errorsVal.([]any)
	require.True(t, ok, "errors должен быть массивом")
	assert.Empty(t, errorsArr, "errors должен быть пустым при отсутствии ошибок")

	// Проверяем metadata
	require.NotNil(t, result.Metadata)
	assert.NotEmpty(t, result.Metadata.TraceID)
	assert.GreaterOrEqual(t, result.Metadata.DurationMs, int64(0))
	assert.Equal(t, constants.APIVersion, result.Metadata.APIVersion)
}

func TestForceDisconnectHandler_Execute_NoSessions(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	mockClient := &ractest.MockRACClient{
		GetSessionsFunc: func(_ context.Context, _, _ string) ([]rac.SessionInfo, error) {
			return []rac.SessionInfo{}, nil
		},
	}

	h := &ForceDisconnectHandler{racClient: mockClient}
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
	assert.Equal(t, true, dataMap["no_active_sessions"])
	assert.Equal(t, false, dataMap["state_changed"])
	assert.Equal(t, float64(0), dataMap["terminated_sessions_count"])
	assert.Equal(t, false, dataMap["partial_failure"])

	// Проверяем что sessions — пустой массив, а не null
	sessionsVal, ok := dataMap["sessions"].([]any)
	require.True(t, ok, "sessions должен быть массивом, не null")
	assert.Empty(t, sessionsVal)

	// Проверяем что errors — пустой массив при отсутствии ошибок
	errorsVal2, hasErrors := dataMap["errors"]
	require.True(t, hasErrors, "errors должен присутствовать в JSON")
	errorsArr2, ok := errorsVal2.([]any)
	require.True(t, ok, "errors должен быть массивом")
	assert.Empty(t, errorsArr2, "errors должен быть пустым при отсутствии ошибок")
}

func TestForceDisconnectHandler_Execute_WithDelay(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	// Используем delay=0 для скорости теста, проверяя только корректный парсинг
	t.Setenv("BR_DISCONNECT_DELAY_SEC", "0")

	sessions := ractest.SessionData()
	mockClient := ractest.NewMockRACClientWithSessions(sessions)

	h := &ForceDisconnectHandler{racClient: mockClient}
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
	assert.Equal(t, float64(0), dataMap["delay_sec"])
	assert.Equal(t, float64(2), dataMap["terminated_sessions_count"])
}

func TestForceDisconnectHandler_Execute_DelayParsing(t *testing.T) {
	tests := []struct {
		name     string
		delayVal string
		expected float64
	}{
		{name: "корректное значение", delayVal: "5", expected: 5},
		{name: "максимальное допустимое", delayVal: "300", expected: 300},
		{name: "превышение лимита", delayVal: "301", expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BR_OUTPUT_FORMAT", "json")
			t.Setenv("BR_DISCONNECT_DELAY_SEC", tt.delayVal)

			// Используем пустые сессии чтобы избежать реального delay
			mockClient := &ractest.MockRACClient{
				GetSessionsFunc: func(_ context.Context, _, _ string) ([]rac.SessionInfo, error) {
					return []rac.SessionInfo{}, nil
				},
			}

			h := &ForceDisconnectHandler{racClient: mockClient}
			cfg := &config.Config{InfobaseName: "TestBase"}

			var execErr error
			out := testutil.CaptureStdout(t, func() {
				execErr = h.Execute(context.Background(), cfg)
			})

			require.NoError(t, execErr)

			var result output.Result
			err := json.Unmarshal([]byte(out), &result)
			require.NoError(t, err)

			dataMap, ok := result.Data.(map[string]any)
			require.True(t, ok)
			assert.Equal(t, tt.expected, dataMap["delay_sec"], "delay_sec должен быть %v для ввода %s", tt.expected, tt.delayVal)
		})
	}
}

func TestForceDisconnectHandler_Execute_PartialFailure(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	sessions := ractest.SessionData()
	mockClient := ractest.NewMockRACClientWithSessions(sessions)
	// Первая сессия завершается, вторая — ошибка
	var callCount atomic.Int32
	mockClient.TerminateSessionFunc = func(_ context.Context, _, _ string) error {
		callCount.Add(1)
		if callCount.Load() == 2 {
			return fmt.Errorf("connection timeout")
		}
		return nil
	}

	h := &ForceDisconnectHandler{racClient: mockClient}
	cfg := &config.Config{InfobaseName: "TestBase"}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.NoError(t, execErr) // Команда не фейлится при частичных ошибках

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)

	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, dataMap["partial_failure"])
	assert.Equal(t, float64(1), dataMap["terminated_sessions_count"])

	errorsVal, ok := dataMap["errors"].([]any)
	require.True(t, ok)
	assert.Len(t, errorsVal, 1)
	assert.Contains(t, errorsVal[0].(string), "connection timeout")
}

func TestForceDisconnectHandler_Execute_AllTerminateFailed(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	sessions := ractest.SessionData()
	mockClient := ractest.NewMockRACClientWithSessions(sessions)
	// Все terminate вызовы возвращают ошибку
	mockClient.TerminateSessionFunc = func(_ context.Context, _, _ string) error {
		return fmt.Errorf("connection refused")
	}

	h := &ForceDisconnectHandler{racClient: mockClient}
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
	// Все terminate провалились — partial_failure=false (не partial, а total failure)
	assert.Equal(t, false, dataMap["partial_failure"])
	assert.Equal(t, float64(0), dataMap["terminated_sessions_count"])
	// Ни одна сессия не завершена — state_changed должен быть false
	assert.Equal(t, false, dataMap["state_changed"])

	errorsVal, ok := dataMap["errors"].([]any)
	require.True(t, ok)
	assert.Len(t, errorsVal, 2, "должно быть 2 ошибки — по одной на каждую сессию")
}

func TestForceDisconnectHandler_Execute_NoInfobase(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &ForceDisconnectHandler{}
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
	assert.Equal(t, "nr-force-disconnect-sessions", result.Command)
	require.NotNil(t, result.Error)
	assert.Equal(t, "CONFIG.INFOBASE_MISSING", result.Error.Code)
	assert.Contains(t, result.Error.Message, "BR_INFOBASE_NAME")
}

func TestForceDisconnectHandler_Execute_RACErrors(t *testing.T) {
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
			name: "SessionsError",
			mock: func() *ractest.MockRACClient {
				m := ractest.NewMockRACClient()
				m.GetSessionsFunc = func(_ context.Context, _, _ string) ([]rac.SessionInfo, error) {
					return nil, fmt.Errorf("sessions list failed")
				}
				return m
			}(),
			errCode: "RAC.SESSIONS_FAILED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BR_OUTPUT_FORMAT", "json")

			h := &ForceDisconnectHandler{racClient: tt.mock}
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

func TestForceDisconnectHandler_Execute_NilConfig(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &ForceDisconnectHandler{}
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

func TestForceDisconnectHandler_Execute_InvalidDelay(t *testing.T) {
	tests := []struct {
		name     string
		delayVal string
	}{
		{name: "нечисловое значение", delayVal: "abc"},
		{name: "отрицательное значение", delayVal: "-5"},
		{name: "дробное значение", delayVal: "1.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BR_OUTPUT_FORMAT", "json")
			t.Setenv("BR_DISCONNECT_DELAY_SEC", tt.delayVal)

			sessions := ractest.SessionData()
			mockClient := ractest.NewMockRACClientWithSessions(sessions)

			h := &ForceDisconnectHandler{racClient: mockClient}
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
			assert.Equal(t, float64(0), dataMap["delay_sec"], "невалидный delay должен fallback на 0")
		})
	}
}

func TestForceDisconnectHandler_Execute_TextPartialFailure(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	sessions := ractest.SessionData()
	mockClient := ractest.NewMockRACClientWithSessions(sessions)
	var callCount atomic.Int32
	mockClient.TerminateSessionFunc = func(_ context.Context, _, _ string) error {
		callCount.Add(1)
		if callCount.Load() == 2 {
			return fmt.Errorf("connection timeout")
		}
		return nil
	}

	h := &ForceDisconnectHandler{racClient: mockClient}
	cfg := &config.Config{InfobaseName: "TestBase"}

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(context.Background(), cfg)
	})

	require.NoError(t, execErr)
	assert.Contains(t, out, "Принудительное завершение сессий: TestBase")
	assert.Contains(t, out, "Завершено сессий: 1 из 2")
	assert.Contains(t, out, "Ошибки:")
	assert.Contains(t, out, "connection timeout")
}

func TestForceDisconnectHandler_Execute_TextAllTerminateFailed(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	sessions := ractest.SessionData()
	mockClient := ractest.NewMockRACClientWithSessions(sessions)
	mockClient.TerminateSessionFunc = func(_ context.Context, _, _ string) error {
		return fmt.Errorf("connection refused")
	}

	h := &ForceDisconnectHandler{racClient: mockClient}
	cfg := &config.Config{InfobaseName: "TestBase"}

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(context.Background(), cfg)
	})

	require.NoError(t, execErr)
	assert.Contains(t, out, "Принудительное завершение сессий: TestBase")
	assert.Contains(t, out, "Состояние не изменено: не удалось завершить ни одну сессию")
	assert.Contains(t, out, "Ошибки:")
	assert.Contains(t, out, "connection refused")
}

func TestForceDisconnectHandler_Execute_ContextCancellation(t *testing.T) {
	ctx := context.Background()
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	t.Setenv("BR_DISCONNECT_DELAY_SEC", "10")

	sessions := ractest.SessionData()
	mockClient := ractest.NewMockRACClientWithSessions(sessions)

	h := &ForceDisconnectHandler{racClient: mockClient}
	cfg := &config.Config{InfobaseName: "TestBase"}

	ctx, cancel := context.WithCancel(context.Background())
	// Отменяем context немедленно — delay не должен блокировать
	cancel()

	var execErr error
	testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, cfg)
	})

	require.Error(t, execErr)
	assert.ErrorIs(t, execErr, context.Canceled)
}

func TestForceDisconnectHandler_Execute_TextError(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	h := &ForceDisconnectHandler{}
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
