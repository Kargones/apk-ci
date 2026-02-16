package tracing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate_Disabled(t *testing.T) {
	cfg := Config{Enabled: false}
	assert.NoError(t, cfg.Validate())
}

func TestConfig_Validate_EnabledValid(t *testing.T) {
	cfg := Config{
		Enabled:      true,
		Endpoint:     "http://jaeger:4318",
		ServiceName:  "test-service",
		Timeout:      5 * time.Second,
		SamplingRate: 1.0,
	}
	assert.NoError(t, cfg.Validate())
}

func TestConfig_Validate_Table(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name:    "disabled — всегда валиден",
			config:  Config{Enabled: false},
			wantErr: "",
		},
		{
			name: "enabled без endpoint",
			config: Config{
				Enabled:      true,
				Endpoint:     "",
				ServiceName:  "test",
				Timeout:      5 * time.Second,
				SamplingRate: 1.0,
			},
			wantErr: "tracing: endpoint обязателен",
		},
		{
			name: "enabled без service name",
			config: Config{
				Enabled:      true,
				Endpoint:     "http://jaeger:4318",
				ServiceName:  "",
				Timeout:      5 * time.Second,
				SamplingRate: 1.0,
			},
			wantErr: "tracing: service name обязателен",
		},
		{
			name: "enabled с невалидным endpoint (без scheme)",
			config: Config{
				Enabled:      true,
				Endpoint:     "jaeger:4318",
				ServiceName:  "test",
				Timeout:      5 * time.Second,
				SamplingRate: 1.0,
			},
			wantErr: "endpoint должен быть валидным URL",
		},
		{
			name: "enabled с нулевым timeout",
			config: Config{
				Enabled:      true,
				Endpoint:     "http://jaeger:4318",
				ServiceName:  "test",
				Timeout:      0,
				SamplingRate: 1.0,
			},
			wantErr: "tracing: timeout должен быть положительным",
		},
		{
			name: "enabled с отрицательным timeout",
			config: Config{
				Enabled:      true,
				Endpoint:     "http://jaeger:4318",
				ServiceName:  "test",
				Timeout:      -1 * time.Second,
				SamplingRate: 1.0,
			},
			wantErr: "tracing: timeout должен быть положительным",
		},
		{
			name: "enabled со всеми полями",
			config: Config{
				Enabled:      true,
				Endpoint:     "http://jaeger:4318",
				ServiceName:  "apk-ci",
				Version:      "1.0.0",
				Environment:  "production",
				Insecure:     true,
				Timeout:      5 * time.Second,
				SamplingRate: 1.0,
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestConfigValidate_SamplingRate(t *testing.T) {
	tests := []struct {
		name    string
		rate    float64
		wantErr bool
	}{
		{"отрицательный rate", -0.1, true},
		{"нулевой rate (валидно)", 0.0, false},
		{"половинный rate", 0.5, false},
		{"полный rate", 1.0, false},
		{"больше единицы", 1.1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Enabled:      true,
				Endpoint:     "http://localhost:4318",
				ServiceName:  "test",
				Timeout:      5 * time.Second,
				SamplingRate: tt.rate,
			}
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrTracingSamplingRateInvalid)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.False(t, cfg.Enabled, "трейсинг должен быть выключен по умолчанию")
	assert.Empty(t, cfg.Endpoint, "endpoint должен быть пустым")
	assert.Equal(t, "apk-ci", cfg.ServiceName)
	assert.Equal(t, "production", cfg.Environment)
	assert.False(t, cfg.Insecure, "insecure должен быть false по умолчанию (Review #36: secure by default)")
	assert.Equal(t, 5*time.Second, cfg.Timeout)
	assert.Equal(t, 1.0, cfg.SamplingRate, "sampling rate должен быть 1.0 по умолчанию")
}

func TestDefaultConfig_IsValid(t *testing.T) {
	cfg := DefaultConfig()
	assert.NoError(t, cfg.Validate(), "default config должен быть валидным (disabled)")
}
