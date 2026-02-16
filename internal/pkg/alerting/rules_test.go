package alerting

import (
	"context"
	"testing"
	"time"
)

// Subtask 5.1: пустые правила пропускают всё
func TestRulesEngine_DefaultAllowAll(t *testing.T) {
	engine := NewRulesEngine(RulesConfig{})

	tests := []struct {
		name    string
		alert   Alert
		channel string
	}{
		{
			name:    "info алерт проходит",
			alert:   Alert{Severity: SeverityInfo, ErrorCode: "TEST", Command: "cmd"},
			channel: "email",
		},
		{
			name:    "warning алерт проходит",
			alert:   Alert{Severity: SeverityWarning, ErrorCode: "TEST", Command: "cmd"},
			channel: "telegram",
		},
		{
			name:    "critical алерт проходит",
			alert:   Alert{Severity: SeverityCritical, ErrorCode: "TEST", Command: "cmd"},
			channel: "webhook",
		},
		{
			name:    "пустой алерт проходит",
			alert:   Alert{},
			channel: "email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := engine.Evaluate(tt.alert, tt.channel); !got {
				t.Errorf("Evaluate() = false, want true для пустых правил")
			}
		})
	}
}

// Subtask 5.2: фильтрация по минимальному severity
func TestRulesEngine_SeverityFilter(t *testing.T) {
	tests := []struct {
		name        string
		minSeverity string
		alert       Alert
		want        bool
	}{
		{
			name:        "info проходит при minSeverity=INFO",
			minSeverity: "INFO",
			alert:       Alert{Severity: SeverityInfo},
			want:        true,
		},
		{
			name:        "warning проходит при minSeverity=INFO",
			minSeverity: "INFO",
			alert:       Alert{Severity: SeverityWarning},
			want:        true,
		},
		{
			name:        "critical проходит при minSeverity=WARNING",
			minSeverity: "WARNING",
			alert:       Alert{Severity: SeverityCritical},
			want:        true,
		},
		{
			name:        "warning проходит при minSeverity=WARNING",
			minSeverity: "WARNING",
			alert:       Alert{Severity: SeverityWarning},
			want:        true,
		},
		{
			name:        "info НЕ проходит при minSeverity=WARNING",
			minSeverity: "WARNING",
			alert:       Alert{Severity: SeverityInfo},
			want:        false,
		},
		{
			name:        "info НЕ проходит при minSeverity=CRITICAL",
			minSeverity: "CRITICAL",
			alert:       Alert{Severity: SeverityInfo},
			want:        false,
		},
		{
			name:        "warning НЕ проходит при minSeverity=CRITICAL",
			minSeverity: "CRITICAL",
			alert:       Alert{Severity: SeverityWarning},
			want:        false,
		},
		{
			name:        "critical проходит при minSeverity=CRITICAL",
			minSeverity: "CRITICAL",
			alert:       Alert{Severity: SeverityCritical},
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewRulesEngine(RulesConfig{
				MinSeverity: tt.minSeverity,
			})
			if got := engine.Evaluate(tt.alert, "email"); got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Subtask 5.3: whitelist кодов ошибок
func TestRulesEngine_ErrorCodeInclude(t *testing.T) {
	engine := NewRulesEngine(RulesConfig{
		IncludeErrorCodes: []string{"ERR_DB_CONN", "ERR_SMTP"},
	})

	tests := []struct {
		name  string
		alert Alert
		want  bool
	}{
		{
			name:  "код из whitelist проходит",
			alert: Alert{ErrorCode: "ERR_DB_CONN"},
			want:  true,
		},
		{
			name:  "второй код из whitelist проходит",
			alert: Alert{ErrorCode: "ERR_SMTP"},
			want:  true,
		},
		{
			name:  "код НЕ из whitelist отклоняется",
			alert: Alert{ErrorCode: "ERR_TIMEOUT"},
			want:  false,
		},
		{
			name:  "пустой код отклоняется",
			alert: Alert{ErrorCode: ""},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := engine.Evaluate(tt.alert, "email"); got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Subtask 5.4: blacklist кодов ошибок
func TestRulesEngine_ErrorCodeExclude(t *testing.T) {
	engine := NewRulesEngine(RulesConfig{
		ExcludeErrorCodes: []string{"ERR_SONARQUBE_API", "ERR_LINT"},
	})

	tests := []struct {
		name  string
		alert Alert
		want  bool
	}{
		{
			name:  "исключённый код отклоняется",
			alert: Alert{ErrorCode: "ERR_SONARQUBE_API"},
			want:  false,
		},
		{
			name:  "второй исключённый код отклоняется",
			alert: Alert{ErrorCode: "ERR_LINT"},
			want:  false,
		},
		{
			name:  "не исключённый код проходит",
			alert: Alert{ErrorCode: "ERR_DB_CONN"},
			want:  true,
		},
		{
			name:  "пустой код проходит",
			alert: Alert{ErrorCode: ""},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := engine.Evaluate(tt.alert, "email"); got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Subtask 5.5: whitelist команд
func TestRulesEngine_CommandInclude(t *testing.T) {
	engine := NewRulesEngine(RulesConfig{
		IncludeCommands: []string{"nr-dbrestore", "nr-service-mode-enable"},
	})

	tests := []struct {
		name  string
		alert Alert
		want  bool
	}{
		{
			name:  "команда из whitelist проходит",
			alert: Alert{Command: "nr-dbrestore"},
			want:  true,
		},
		{
			name:  "команда НЕ из whitelist отклоняется",
			alert: Alert{Command: "nr-version"},
			want:  false,
		},
		{
			name:  "пустая команда отклоняется",
			alert: Alert{Command: ""},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := engine.Evaluate(tt.alert, "email"); got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Subtask 5.6: blacklist команд
func TestRulesEngine_CommandExclude(t *testing.T) {
	engine := NewRulesEngine(RulesConfig{
		ExcludeCommands: []string{"nr-version", "nr-action-menu-build"},
	})

	tests := []struct {
		name  string
		alert Alert
		want  bool
	}{
		{
			name:  "исключённая команда отклоняется",
			alert: Alert{Command: "nr-version"},
			want:  false,
		},
		{
			name:  "не исключённая команда проходит",
			alert: Alert{Command: "nr-dbrestore"},
			want:  true,
		},
		{
			name:  "пустая команда проходит",
			alert: Alert{Command: ""},
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := engine.Evaluate(tt.alert, "email"); got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Subtask 5.7: разные правила для каналов
func TestRulesEngine_PerChannelOverride(t *testing.T) {
	engine := NewRulesEngine(RulesConfig{
		MinSeverity: "WARNING", // глобально: только WARNING+
		Channels: map[string]ChannelRulesConfig{
			"email": {
				MinSeverity: "CRITICAL", // email: только CRITICAL
			},
			"telegram": {
				ExcludeErrorCodes: []string{"ERR_SONARQUBE_API"},
			},
			"webhook": {
				IncludeCommands: []string{"nr-dbrestore", "nr-service-mode-enable"},
			},
		},
	})

	tests := []struct {
		name    string
		alert   Alert
		channel string
		want    bool
	}{
		{
			name:    "warning → email: deny (нужен CRITICAL)",
			alert:   Alert{Severity: SeverityWarning},
			channel: "email",
			want:    false,
		},
		{
			name:    "critical → email: allow",
			alert:   Alert{Severity: SeverityCritical},
			channel: "email",
			want:    true,
		},
		{
			name:    "warning → telegram: allow (severity OK, error code не исключён)",
			alert:   Alert{Severity: SeverityWarning, ErrorCode: "ERR_DB_CONN"},
			channel: "telegram",
			want:    true,
		},
		{
			name:    "ERR_SONARQUBE_API → telegram: deny (исключён)",
			alert:   Alert{Severity: SeverityWarning, ErrorCode: "ERR_SONARQUBE_API"},
			channel: "telegram",
			want:    false,
		},
		{
			name:    "nr-dbrestore → webhook: allow (в whitelist)",
			alert:   Alert{Severity: SeverityWarning, Command: "nr-dbrestore"},
			channel: "webhook",
			want:    true,
		},
		{
			name:    "nr-version → webhook: deny (не в whitelist)",
			alert:   Alert{Severity: SeverityWarning, Command: "nr-version"},
			channel: "webhook",
			want:    false,
		},
		{
			name:    "warning → unknown channel: allow (global rule)",
			alert:   Alert{Severity: SeverityWarning},
			channel: "slack",
			want:    true,
		},
		{
			name:    "info → unknown channel: deny (global minSeverity=WARNING)",
			alert:   Alert{Severity: SeverityInfo},
			channel: "slack",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := engine.Evaluate(tt.alert, tt.channel); got != tt.want {
				t.Errorf("Evaluate(%q, %q) = %v, want %v", tt.alert.ErrorCode, tt.channel, got, tt.want)
			}
		})
	}
}

// Subtask 5.8: комбинация severity + error_code + command
func TestRulesEngine_CombinedRules(t *testing.T) {
	engine := NewRulesEngine(RulesConfig{
		MinSeverity:       "WARNING",
		ExcludeErrorCodes: []string{"ERR_LINT"},
		ExcludeCommands:   []string{"nr-version"},
	})

	tests := []struct {
		name  string
		alert Alert
		want  bool
	}{
		{
			name:  "все проверки OK",
			alert: Alert{Severity: SeverityWarning, ErrorCode: "ERR_DB_CONN", Command: "nr-dbrestore"},
			want:  true,
		},
		{
			name:  "severity fail",
			alert: Alert{Severity: SeverityInfo, ErrorCode: "ERR_DB_CONN", Command: "nr-dbrestore"},
			want:  false,
		},
		{
			name:  "error code exclude",
			alert: Alert{Severity: SeverityWarning, ErrorCode: "ERR_LINT", Command: "nr-dbrestore"},
			want:  false,
		},
		{
			name:  "command exclude",
			alert: Alert{Severity: SeverityWarning, ErrorCode: "ERR_DB_CONN", Command: "nr-version"},
			want:  false,
		},
		{
			name:  "все три fail одновременно",
			alert: Alert{Severity: SeverityInfo, ErrorCode: "ERR_LINT", Command: "nr-version"},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := engine.Evaluate(tt.alert, "email"); got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Subtask 5.8 дополнение: include имеет приоритет над exclude
func TestRulesEngine_IncludePriorityOverExclude(t *testing.T) {
	engine := NewRulesEngine(RulesConfig{
		IncludeErrorCodes: []string{"ERR_DB_CONN"},
		ExcludeErrorCodes: []string{"ERR_DB_CONN"}, // оба заданы — include побеждает
		IncludeCommands:   []string{"nr-dbrestore"},
		ExcludeCommands:   []string{"nr-dbrestore"},
	})

	alert := Alert{
		Severity:  SeverityInfo,
		ErrorCode: "ERR_DB_CONN",
		Command:   "nr-dbrestore",
	}

	if got := engine.Evaluate(alert, "email"); !got {
		t.Errorf("Evaluate() = false, want true (include приоритетнее exclude)")
	}
}

// Subtask 5.9: интеграционный тест rules + multi-channel
func TestMultiChannelAlerter_WithRules(t *testing.T) {
	emailMock := &mockAlerter{}
	telegramMock := &mockAlerter{}
	webhookMock := &mockAlerter{}

	channels := map[string]Alerter{
		"email":    emailMock,
		"telegram": telegramMock,
		"webhook":  webhookMock,
	}

	// email: только CRITICAL, telegram: все, webhook: только nr-dbrestore
	rules := NewRulesEngine(RulesConfig{
		Channels: map[string]ChannelRulesConfig{
			"email": {
				MinSeverity: "CRITICAL",
			},
			"webhook": {
				IncludeCommands: []string{"nr-dbrestore"},
			},
		},
	})

	multi := NewMultiChannelAlerter(channels, rules, nil, &testLogger{})

	// WARNING алерт для nr-dbrestore
	alert := Alert{
		ErrorCode: "ERR_TEST",
		Severity:  SeverityWarning,
		Command:   "nr-dbrestore",
		Message:   "Test",
		Timestamp: time.Now(),
	}

	err := multi.Send(context.Background(), alert)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	// email: deny (WARNING < CRITICAL)
	if emailMock.sendCount != 0 {
		t.Errorf("email sendCount = %d, want 0 (severity filter)", emailMock.sendCount)
	}

	// telegram: allow (нет channel override → global default = allow all)
	if telegramMock.sendCount != 1 {
		t.Errorf("telegram sendCount = %d, want 1", telegramMock.sendCount)
	}

	// webhook: allow (nr-dbrestore в whitelist)
	if webhookMock.sendCount != 1 {
		t.Errorf("webhook sendCount = %d, want 1", webhookMock.sendCount)
	}
}

// Дополнительный интеграционный тест: rules = nil → все алерты проходят
func TestMultiChannelAlerter_NilRules(t *testing.T) {
	emailMock := &mockAlerter{}

	channels := map[string]Alerter{
		"email": emailMock,
	}

	// rules = nil — backward compatibility
	multi := NewMultiChannelAlerter(channels, nil, nil, &testLogger{})

	alert := Alert{
		ErrorCode: "TEST",
		Severity:  SeverityInfo,
		Timestamp: time.Now(),
	}

	err := multi.Send(context.Background(), alert)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if emailMock.sendCount != 1 {
		t.Errorf("email sendCount = %d, want 1 (nil rules = allow all)", emailMock.sendCount)
	}
}

// Тест Evaluate с пустым channel name — должен использовать global rules
func TestRulesEngine_EmptyChannelName(t *testing.T) {
	engine := NewRulesEngine(RulesConfig{
		MinSeverity: "WARNING",
		Channels: map[string]ChannelRulesConfig{
			"email": {
				MinSeverity: "CRITICAL",
			},
		},
	})

	// Пустой channel name → global rules (minSeverity=WARNING)
	if got := engine.Evaluate(Alert{Severity: SeverityWarning}, ""); !got {
		t.Error("Evaluate(WARNING, \"\") = false, want true (global rule)")
	}
	if got := engine.Evaluate(Alert{Severity: SeverityInfo}, ""); got {
		t.Error("Evaluate(INFO, \"\") = true, want false (global minSeverity=WARNING)")
	}
}

// Тест parseSeverity
func TestParseSeverity(t *testing.T) {
	tests := []struct {
		input string
		want  Severity
	}{
		{"INFO", SeverityInfo},
		{"info", SeverityInfo},
		{"WARNING", SeverityWarning},
		{"warning", SeverityWarning},
		{"CRITICAL", SeverityCritical},
		{"critical", SeverityCritical},
		{"", SeverityInfo},        // пустая строка → INFO (default)
		{"UNKNOWN", SeverityInfo}, // неизвестное → INFO (default)
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parseSeverity(tt.input); got != tt.want {
				t.Errorf("parseSeverity(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
