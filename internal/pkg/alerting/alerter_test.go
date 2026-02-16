package alerting

import (
	"context"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/pkg/logging"
)

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		want     string
	}{
		{
			name:     "SeverityInfo",
			severity: SeverityInfo,
			want:     "INFO",
		},
		{
			name:     "SeverityWarning",
			severity: SeverityWarning,
			want:     "WARNING",
		},
		{
			name:     "SeverityCritical",
			severity: SeverityCritical,
			want:     "CRITICAL",
		},
		{
			name:     "Unknown severity",
			severity: Severity(999),
			want:     "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.severity.String()
			if got != tt.want {
				t.Errorf("Severity.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNopAlerter_DoesNothing(t *testing.T) {
	alerter := NewNopAlerter()

	alert := Alert{
		ErrorCode: "TEST_ERROR",
		Message:   "Test message",
		TraceID:   "test-trace-id",
		Timestamp: time.Now(),
		Command:   "test-command",
		Infobase:  "test-infobase",
		Severity:  SeverityCritical,
	}

	// NopAlerter должен возвращать nil без ошибок
	err := alerter.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("NopAlerter.Send() error = %v, want nil", err)
	}
}

func TestNewAlerter_DisabledByDefault(t *testing.T) {
	config := Config{
		Enabled: false,
	}

	logger := &nopTestLogger{}
	alerter, err := NewAlerter(config, RulesConfig{}, logger)

	if err != nil {
		t.Fatalf("NewAlerter() error = %v", err)
	}

	// При enabled=false должен возвращать NopAlerter
	_, ok := alerter.(*NopAlerter)
	if !ok {
		t.Errorf("NewAlerter() returned %T, want *NopAlerter", alerter)
	}
}

func TestNewAlerter_ReturnsEmailAlerter_WhenEnabled(t *testing.T) {
	config := Config{
		Enabled:         true,
		RateLimitWindow: 5 * time.Minute,
		Email: EmailConfig{
			Enabled:  true,
			SMTPHost: "smtp.example.com",
			SMTPPort: 587,
			From:     "alerts@example.com",
			To:       []string{"devops@example.com"},
		},
	}

	logger := &nopTestLogger{}
	alerter, err := NewAlerter(config, RulesConfig{}, logger)

	if err != nil {
		t.Fatalf("NewAlerter() error = %v", err)
	}

	// При enabled=true и email.enabled=true должен возвращать MultiChannelAlerter
	// (с rules engine для per-channel фильтрации)
	_, ok := alerter.(*MultiChannelAlerter)
	if !ok {
		t.Errorf("NewAlerter() returned %T, want *MultiChannelAlerter", alerter)
	}
}

func TestNewAlerter_ReturnsNopAlerter_WhenNoChannels(t *testing.T) {
	config := Config{
		Enabled: true,
		Email: EmailConfig{
			Enabled: false,
		},
	}

	logger := &nopTestLogger{}
	alerter, err := NewAlerter(config, RulesConfig{}, logger)

	if err != nil {
		t.Fatalf("NewAlerter() error = %v", err)
	}

	// При enabled=true, но нет настроенных каналов — возвращает NopAlerter
	_, ok := alerter.(*NopAlerter)
	if !ok {
		t.Errorf("NewAlerter() returned %T, want *NopAlerter", alerter)
	}
}

func TestNewAlerter_ValidationError_MissingSMTPHost(t *testing.T) {
	config := Config{
		Enabled: true,
		Email: EmailConfig{
			Enabled:  true,
			SMTPHost: "", // Missing
			From:     "alerts@example.com",
			To:       []string{"devops@example.com"},
		},
	}

	logger := &nopTestLogger{}
	_, err := NewAlerter(config, RulesConfig{}, logger)

	if err == nil {
		t.Error("NewAlerter() error = nil, want error for missing SMTP host")
	}
}

func TestNewAlerter_ValidationError_MissingFrom(t *testing.T) {
	config := Config{
		Enabled: true,
		Email: EmailConfig{
			Enabled:  true,
			SMTPHost: "smtp.example.com",
			From:     "", // Missing
			To:       []string{"devops@example.com"},
		},
	}

	logger := &nopTestLogger{}
	_, err := NewAlerter(config, RulesConfig{}, logger)

	if err == nil {
		t.Error("NewAlerter() error = nil, want error for missing From address")
	}
}

func TestNewAlerter_ValidationError_EmptyTo(t *testing.T) {
	config := Config{
		Enabled: true,
		Email: EmailConfig{
			Enabled:  true,
			SMTPHost: "smtp.example.com",
			From:     "alerts@example.com",
			To:       []string{}, // Empty
		},
	}

	logger := &nopTestLogger{}
	_, err := NewAlerter(config, RulesConfig{}, logger)

	if err == nil {
		t.Error("NewAlerter() error = nil, want error for empty To list")
	}
}

// nopTestLogger — тестовый логгер без вывода.
type nopTestLogger struct{}

func (n *nopTestLogger) Debug(_ string, _ ...any)                   {}
func (n *nopTestLogger) Info(_ string, _ ...any)                    {}
func (n *nopTestLogger) Warn(_ string, _ ...any)                    {}
func (n *nopTestLogger) Error(_ string, _ ...any)                   {}
func (n *nopTestLogger) With(_ ...any) logging.Logger { return n }
