package output

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusConstants(t *testing.T) {
	assert.Equal(t, "success", StatusSuccess)
	assert.Equal(t, "error", StatusError)
}

func TestResult_JSON_Serialization(t *testing.T) {
	tests := []struct {
		name   string
		result *Result
		want   map[string]any
	}{
		{
			name: "успешный результат с данными",
			result: &Result{
				Status:  StatusSuccess,
				Command: "test-command",
				Data:    map[string]string{"version": "1.0.0"},
				Metadata: &Metadata{
					DurationMs: 150,
					APIVersion: "v1",
				},
			},
			want: map[string]any{
				"status":  "success",
				"command": "test-command",
				"data":    map[string]any{"version": "1.0.0"},
				"metadata": map[string]any{
					"duration_ms": float64(150),
					"api_version": "v1",
				},
			},
		},
		{
			name: "результат с ошибкой",
			result: &Result{
				Status:  StatusError,
				Command: "test-command",
				Error: &ErrorInfo{
					Code:    "CONFIG.LOAD_FAILED",
					Message: "не удалось загрузить конфигурацию",
				},
				Metadata: &Metadata{
					DurationMs: 50,
					APIVersion: "v1",
				},
			},
			want: map[string]any{
				"status":  "error",
				"command": "test-command",
				"error": map[string]any{
					"code":    "CONFIG.LOAD_FAILED",
					"message": "не удалось загрузить конфигурацию",
				},
				"metadata": map[string]any{
					"duration_ms": float64(50),
					"api_version": "v1",
				},
			},
		},
		{
			name: "минимальный результат",
			result: &Result{
				Status:  StatusSuccess,
				Command: "test-command",
			},
			want: map[string]any{
				"status":  "success",
				"command": "test-command",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.result)
			require.NoError(t, err)

			var got map[string]any
			err = json.Unmarshal(data, &got)
			require.NoError(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResult_OmitsEmptyFields(t *testing.T) {
	result := &Result{
		Status:  StatusSuccess,
		Command: "test-command",
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	// Проверяем что пустые поля не включены в JSON
	jsonStr := string(data)
	assert.NotContains(t, jsonStr, "data")
	assert.NotContains(t, jsonStr, "error")
	assert.NotContains(t, jsonStr, "metadata")
}

func TestMetadata_OmitsEmptyTraceID(t *testing.T) {
	metadata := &Metadata{
		DurationMs: 100,
		APIVersion: "v1",
	}

	data, err := json.Marshal(metadata)
	require.NoError(t, err)

	// Проверяем что trace_id не включён когда пустой
	jsonStr := string(data)
	assert.NotContains(t, jsonStr, "trace_id")
}

func TestMetadata_IncludesTraceID(t *testing.T) {
	metadata := &Metadata{
		DurationMs: 100,
		TraceID:    "abc123",
		APIVersion: "v1",
	}

	data, err := json.Marshal(metadata)
	require.NoError(t, err)

	var got map[string]any
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	assert.Equal(t, "abc123", got["trace_id"])
}
