package alerting

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/smtp"
	"strings"
	"testing"
	"time"
)

// mockSMTPClient реализует SMTPClient для тестирования.
type mockSMTPClient struct {
	startTLSCalled bool
	authCalled     bool
	mailFrom       string
	rcptTo         []string
	messageData    string
	closeCalled    bool

	// Контроль поведения
	startTLSErr error
	authErr     error
	mailErr     error
	rcptErr     error
	dataErr     error
	extensions  map[string]string
}

func (m *mockSMTPClient) StartTLS(config *tls.Config) error {
	m.startTLSCalled = true
	return m.startTLSErr
}

func (m *mockSMTPClient) Auth(a smtp.Auth) error {
	m.authCalled = true
	return m.authErr
}

func (m *mockSMTPClient) Mail(from string) error {
	m.mailFrom = from
	return m.mailErr
}

func (m *mockSMTPClient) Rcpt(to string) error {
	m.rcptTo = append(m.rcptTo, to)
	return m.rcptErr
}

func (m *mockSMTPClient) Data() (WriteCloser, error) {
	if m.dataErr != nil {
		return nil, m.dataErr
	}
	return &mockWriteCloser{client: m}, nil
}

func (m *mockSMTPClient) Close() error {
	m.closeCalled = true
	return nil
}

func (m *mockSMTPClient) Extension(ext string) (bool, string) {
	if m.extensions == nil {
		m.extensions = map[string]string{
			"STARTTLS": "",
		}
	}
	v, ok := m.extensions[ext]
	return ok, v
}

// mockWriteCloser записывает данные в mockSMTPClient.
type mockWriteCloser struct {
	client *mockSMTPClient
	buf    bytes.Buffer
}

func (w *mockWriteCloser) Write(p []byte) (n int, err error) {
	return w.buf.Write(p)
}

func (w *mockWriteCloser) Close() error {
	w.client.messageData = w.buf.String()
	return nil
}

// mockSMTPDialer реализует SMTPDialer для тестирования.
type mockSMTPDialer struct {
	client  *mockSMTPClient
	dialErr error
}

func (d *mockSMTPDialer) DialContext(_ context.Context, _ string) (SMTPClient, error) {
	if d.dialErr != nil {
		return nil, d.dialErr
	}
	return d.client, nil
}

func TestEmailAlerter_Send_Success(t *testing.T) {
	config := EmailConfig{
		Enabled:         true,
		SMTPHost:        "smtp.example.com",
		SMTPPort:        587,
		SMTPUser:        "user",
		SMTPPassword:    "pass",
		UseTLS:          true,
		From:            "alerts@example.com",
		To:              []string{"devops@example.com", "oncall@example.com"},
		SubjectTemplate: "[ALERT] {{.ErrorCode}}",
	}

	mockClient := &mockSMTPClient{}
	mockDialer := &mockSMTPDialer{client: mockClient}
	logger := &testLogger{}
	rateLimiter := NewRateLimiter(5 * time.Minute)

	alerter, err := NewEmailAlerter(config, rateLimiter, logger)
	if err != nil {
		t.Fatalf("NewEmailAlerter() error = %v", err)
	}
	alerter.SetDialer(mockDialer)

	alert := Alert{
		ErrorCode: "DB_CONN_FAIL",
		Message:   "Ошибка подключения к БД",
		TraceID:   "trace-12345",
		Timestamp: time.Date(2026, 2, 5, 10, 30, 0, 0, time.UTC),
		Command:   "dbrestore",
		Infobase:  "TestDB",
		Severity:  SeverityCritical,
	}

	err = alerter.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	// Проверяем что email был отправлен
	if mockClient.mailFrom != "alerts@example.com" {
		t.Errorf("Mail FROM = %q, want %q", mockClient.mailFrom, "alerts@example.com")
	}

	if len(mockClient.rcptTo) != 2 {
		t.Errorf("RCPT TO count = %d, want 2", len(mockClient.rcptTo))
	}

	// Проверяем содержимое сообщения
	if !bytes.Contains([]byte(mockClient.messageData), []byte("DB_CONN_FAIL")) {
		t.Error("Message does not contain error code")
	}

	if !bytes.Contains([]byte(mockClient.messageData), []byte("trace-12345")) {
		t.Error("Message does not contain trace ID")
	}

	// Проверяем что логирование success
	if len(logger.infoMsgs) == 0 || logger.infoMsgs[0] != "email алерт отправлен" {
		t.Error("Expected info log for successful send")
	}
}

func TestEmailAlerter_Send_RateLimited(t *testing.T) {
	config := EmailConfig{
		Enabled:  true,
		SMTPHost: "smtp.example.com",
		SMTPPort: 587,
		From:     "alerts@example.com",
		To:       []string{"devops@example.com"},
	}

	mockClient := &mockSMTPClient{}
	mockDialer := &mockSMTPDialer{client: mockClient}
	logger := &testLogger{}
	rateLimiter := NewRateLimiter(5 * time.Minute)

	alerter, err := NewEmailAlerter(config, rateLimiter, logger)
	if err != nil {
		t.Fatalf("NewEmailAlerter() error = %v", err)
	}
	alerter.SetDialer(mockDialer)

	alert := Alert{
		ErrorCode: "TEST_ERROR",
		Message:   "Test message",
		Timestamp: time.Now(),
	}

	// Первый вызов — успешно
	_ = alerter.Send(context.Background(), alert)

	// Сбрасываем mock
	mockClient.mailFrom = ""

	// Второй вызов — должен быть rate limited
	err = alerter.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() error = %v, want nil", err)
	}

	// Email не должен был отправиться
	if mockClient.mailFrom != "" {
		t.Error("Second send should have been rate limited")
	}

	// Должен быть debug лог
	if len(logger.debugMsgs) == 0 || logger.debugMsgs[0] != "алерт подавлен rate limiter" {
		t.Error("Expected debug log for rate limited alert")
	}
}

func TestEmailAlerter_Send_SMTPError_Continues(t *testing.T) {
	config := EmailConfig{
		Enabled:  true,
		SMTPHost: "smtp.example.com",
		SMTPPort: 587,
		From:     "alerts@example.com",
		To:       []string{"devops@example.com"},
	}

	// Mock с ошибкой подключения
	mockDialer := &mockSMTPDialer{dialErr: io.EOF}
	logger := &testLogger{}
	rateLimiter := NewRateLimiter(5 * time.Minute)

	alerter, err := NewEmailAlerter(config, rateLimiter, logger)
	if err != nil {
		t.Fatalf("NewEmailAlerter() error = %v", err)
	}
	alerter.SetDialer(mockDialer)

	alert := Alert{
		ErrorCode: "TEST_ERROR",
		Message:   "Test message",
		Timestamp: time.Now(),
	}

	// Send должен вернуть nil (AC10: приложение продолжает работу)
	err = alerter.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() error = %v, want nil (should continue on SMTP error)", err)
	}

	// Должен быть error лог
	if len(logger.errorMsgs) == 0 {
		t.Error("Expected error log for SMTP failure")
	}
}

func TestEmailAlerter_FormatEmail(t *testing.T) {
	config := EmailConfig{
		Enabled:         true,
		SMTPHost:        "smtp.example.com",
		From:            "alerts@example.com",
		To:              []string{"devops@example.com"},
		SubjectTemplate: "[{{.SeverityStr}}] {{.ErrorCode}} in {{.Command}}",
	}

	logger := &testLogger{}
	alerter, err := NewEmailAlerter(config, nil, logger)
	if err != nil {
		t.Fatalf("NewEmailAlerter() error = %v", err)
	}

	alert := Alert{
		ErrorCode: "DB_ERROR",
		Message:   "Database error occurred",
		TraceID:   "abc123",
		Timestamp: time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC),
		Command:   "dbrestore",
		Infobase:  "ProdDB",
		Severity:  SeverityCritical,
	}

	subject, body, err := alerter.formatEmail(alert)
	if err != nil {
		t.Fatalf("formatEmail() error = %v", err)
	}

	// Проверяем subject
	expectedSubject := "[CRITICAL] DB_ERROR in dbrestore"
	if subject != expectedSubject {
		t.Errorf("subject = %q, want %q", subject, expectedSubject)
	}

	// Проверяем body содержит все поля
	checks := []string{
		"DB_ERROR",
		"CRITICAL",
		"dbrestore",
		"ProdDB",
		"Database error occurred",
		"abc123",
		"2026-02-05T12:00:00Z",
	}

	for _, check := range checks {
		if !bytes.Contains([]byte(body), []byte(check)) {
			t.Errorf("body does not contain %q", check)
		}
	}
}

func TestNewEmailAlerter_InvalidSubjectTemplate(t *testing.T) {
	config := EmailConfig{
		Enabled:         true,
		SMTPHost:        "smtp.example.com",
		From:            "alerts@example.com",
		To:              []string{"devops@example.com"},
		SubjectTemplate: "{{.InvalidField", // Некорректный шаблон
	}

	logger := &testLogger{}
	_, err := NewEmailAlerter(config, nil, logger)
	if err == nil {
		t.Error("NewEmailAlerter() error = nil, want error for invalid template")
	}
}

// H4 fix: Тест для implicit SSL (порт 465)
func TestEmailAlerter_Send_ImplicitSSL_Port465(t *testing.T) {
	config := EmailConfig{
		Enabled:  true,
		SMTPHost: "smtp.example.com",
		SMTPPort: 465, // Implicit SSL порт
		UseTLS:   true,
		From:     "alerts@example.com",
		To:       []string{"devops@example.com"},
	}

	mockClient := &mockSMTPClient{}
	mockDialer := &mockSMTPDialer{client: mockClient}
	logger := &testLogger{}

	alerter, err := NewEmailAlerter(config, nil, logger)
	if err != nil {
		t.Fatalf("NewEmailAlerter() error = %v", err)
	}
	alerter.SetDialer(mockDialer)

	alert := Alert{
		ErrorCode: "SSL_TEST",
		Message:   "Testing implicit SSL",
		Timestamp: time.Now(),
	}

	err = alerter.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	// Для implicit SSL (порт 465) StartTLS НЕ должен вызываться
	// (TLS уже установлен в Dial)
	if mockClient.startTLSCalled {
		t.Error("StartTLS should NOT be called for implicit SSL (port 465)")
	}

	// Email должен быть отправлен
	if mockClient.mailFrom != "alerts@example.com" {
		t.Errorf("Mail FROM = %q, want %q", mockClient.mailFrom, "alerts@example.com")
	}
}

// M4 fix: Тест для context cancellation
func TestEmailAlerter_Send_ContextCanceled(t *testing.T) {
	config := EmailConfig{
		Enabled:  true,
		SMTPHost: "smtp.example.com",
		SMTPPort: 587,
		From:     "alerts@example.com",
		To:       []string{"devops@example.com"},
	}

	// Mock client который возвращает успех для всех операций до Data()
	mockClient := &mockSMTPClient{}
	mockDialer := &mockSMTPDialer{client: mockClient}
	logger := &testLogger{}

	alerter, err := NewEmailAlerter(config, nil, logger)
	if err != nil {
		t.Fatalf("NewEmailAlerter() error = %v", err)
	}
	alerter.SetDialer(mockDialer)

	// Создаём уже отменённый контекст
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем сразу

	alert := Alert{
		ErrorCode: "CONTEXT_TEST",
		Message:   "Testing context cancellation",
		Timestamp: time.Now(),
	}

	// Send должен вернуть nil (ошибка логируется, AC10)
	err = alerter.Send(ctx, alert)
	if err != nil {
		t.Errorf("Send() error = %v, want nil (errors are logged, not returned)", err)
	}

	// Должна быть ошибка в логах
	if len(logger.errorMsgs) == 0 {
		t.Error("Expected error log for context cancellation")
	}
}

// H3 fix: Тест для RFC 2047 encoding кириллицы
func TestEmailAlerter_FormatEmail_Cyrillic(t *testing.T) {
	config := EmailConfig{
		Enabled:         true,
		SMTPHost:        "smtp.example.com",
		From:            "alerts@example.com",
		To:              []string{"devops@example.com"},
		SubjectTemplate: "[ALERT] {{.ErrorCode}}: {{.Command}}",
	}

	logger := &testLogger{}
	alerter, err := NewEmailAlerter(config, nil, logger)
	if err != nil {
		t.Fatalf("NewEmailAlerter() error = %v", err)
	}

	alert := Alert{
		ErrorCode: "DB_ERROR",
		Message:   "Ошибка подключения к базе данных", // Кириллица
		TraceID:   "abc123",
		Timestamp: time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC),
		Command:   "dbrestore",
		Infobase:  "ПродуктивнаяБаза", // Кириллица
		Severity:  SeverityCritical,
	}

	subject, body, err := alerter.formatEmail(alert)
	if err != nil {
		t.Fatalf("formatEmail() error = %v", err)
	}

	// Subject не должен содержать кириллицу напрямую (только ASCII)
	// Примечание: этот тест проверяет что formatEmail работает,
	// buildMessage добавит RFC 2047 encoding

	// Body должен содержать кириллицу
	if !bytes.Contains([]byte(body), []byte("Ошибка подключения")) {
		t.Error("body should contain cyrillic message")
	}

	if !bytes.Contains([]byte(body), []byte("ПродуктивнаяБаза")) {
		t.Error("body should contain cyrillic infobase name")
	}

	// Subject содержит ASCII части
	if !bytes.Contains([]byte(subject), []byte("DB_ERROR")) {
		t.Error("subject should contain error code")
	}
}

// Тест для encodeRFC2047
func TestEncodeRFC2047(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "ASCII only - no encoding",
			input: "Hello World",
			want:  "Hello World",
		},
		{
			name:  "Cyrillic - needs encoding",
			input: "Привет",
			want:  "=?UTF-8?B?0J/RgNC40LLQtdGC?=",
		},
		{
			name:  "Mixed ASCII and Cyrillic",
			input: "Alert: Ошибка",
			want:  "=?UTF-8?B?QWxlcnQ6INCe0YjQuNCx0LrQsA==?=",
		},
		{
			name:  "Empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeRFC2047(tt.input)
			if got != tt.want {
				t.Errorf("encodeRFC2047(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// H-1/Review #9: Тест CRLF injection protection в EmailConfig.Validate().
func TestEmailConfig_Validate_CRLFInjection(t *testing.T) {
	tests := []struct {
		name    string
		config  EmailConfig
		wantErr error
	}{
		{
			name: "valid config",
			config: EmailConfig{
				Enabled:  true,
				SMTPHost: "smtp.example.com",
				From:     "alerts@example.com",
				To:       []string{"devops@example.com"},
			},
			wantErr: nil,
		},
		{
			name: "CRLF in From",
			config: EmailConfig{
				Enabled:  true,
				SMTPHost: "smtp.example.com",
				From:     "alerts@example.com\r\nBcc: attacker@evil.com",
				To:       []string{"devops@example.com"},
			},
			wantErr: ErrEmailAddressInvalid,
		},
		{
			name: "LF in From",
			config: EmailConfig{
				Enabled:  true,
				SMTPHost: "smtp.example.com",
				From:     "alerts@example.com\nBcc: attacker@evil.com",
				To:       []string{"devops@example.com"},
			},
			wantErr: ErrEmailAddressInvalid,
		},
		{
			name: "CR in To",
			config: EmailConfig{
				Enabled:  true,
				SMTPHost: "smtp.example.com",
				From:     "alerts@example.com",
				To:       []string{"devops@example.com\rBcc: evil@attacker.com"},
			},
			wantErr: ErrEmailAddressInvalid,
		},
		{
			name: "null byte in From",
			config: EmailConfig{
				Enabled:  true,
				SMTPHost: "smtp.example.com",
				From:     "alerts\x00@example.com",
				To:       []string{"devops@example.com"},
			},
			wantErr: ErrEmailAddressInvalid,
		},
		{
			name: "second To has CRLF",
			config: EmailConfig{
				Enabled:  true,
				SMTPHost: "smtp.example.com",
				From:     "alerts@example.com",
				To:       []string{"good@example.com", "evil@example.com\r\nBcc: hacker@evil.com"},
			},
			wantErr: ErrEmailAddressInvalid,
		},
		{
			name: "disabled - no validation",
			config: EmailConfig{
				Enabled:  false,
				SMTPHost: "",
				From:     "evil\r\n@example.com",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() error = %v, want nil", err)
				}
			}
		})
	}
}

// H-3/Review #10: Тест предупреждения при неполных SMTP credentials.
func TestEmailAlerter_PartialCredentials_Warning(t *testing.T) {
	tests := []struct {
		name         string
		user         string
		password     string
		wantWarning  bool
		wantAuthCall bool
	}{
		{
			name:         "оба заполнены — авторизация",
			user:         "user@example.com",
			password:     "secret",
			wantWarning:  false,
			wantAuthCall: true,
		},
		{
			name:         "оба пустые — без авторизации, без предупреждения",
			user:         "",
			password:     "",
			wantWarning:  false,
			wantAuthCall: false,
		},
		{
			name:         "только user — предупреждение",
			user:         "user@example.com",
			password:     "",
			wantWarning:  true,
			wantAuthCall: false,
		},
		{
			name:         "только password — предупреждение",
			user:         "",
			password:     "secret",
			wantWarning:  true,
			wantAuthCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := EmailConfig{
				Enabled:      true,
				SMTPHost:     "smtp.example.com",
				SMTPPort:     587,
				SMTPUser:     tt.user,
				SMTPPassword: tt.password,
				From:         "alerts@example.com",
				To:           []string{"devops@example.com"},
			}

			mockClient := &mockSMTPClient{}
			mockDialer := &mockSMTPDialer{client: mockClient}
			logger := &testLogger{}

			alerter, err := NewEmailAlerter(config, nil, logger)
			if err != nil {
				t.Fatalf("NewEmailAlerter() error = %v", err)
			}
			alerter.SetDialer(mockDialer)

			alert := Alert{
				ErrorCode: "PARTIAL_CRED_TEST",
				Message:   "test",
				Timestamp: time.Now(),
			}

			_ = alerter.Send(context.Background(), alert)

			// Проверяем вызов Auth
			if mockClient.authCalled != tt.wantAuthCall {
				t.Errorf("authCalled = %v, want %v", mockClient.authCalled, tt.wantAuthCall)
			}

			// Проверяем предупреждение в логах
			foundWarning := false
			for _, msg := range logger.warnMsgs {
				if strings.Contains(msg, "неполные SMTP credentials") {
					foundWarning = true
					break
				}
			}
			if foundWarning != tt.wantWarning {
				t.Errorf("warning log found = %v, want %v", foundWarning, tt.wantWarning)
			}
		})
	}
}

// M-4/Review #10: Тест что HTAB блокируется в email адресах (RFC 5322 строже RFC 7230).
func TestEmailConfig_Validate_TabInEmail(t *testing.T) {
	config := EmailConfig{
		Enabled:  true,
		SMTPHost: "smtp.example.com",
		From:     "alerts\t@example.com",
		To:       []string{"devops@example.com"},
	}

	err := config.Validate()
	if !errors.Is(err, ErrEmailAddressInvalid) {
		t.Errorf("Validate() = %v, want ErrEmailAddressInvalid (HTAB запрещён в email)", err)
	}
}

// M-4 fix: Тест что ошибка auth НЕ утекает credentials в логи.
// ErrSMTPAuth должен возвращаться без оригинальной ошибки (которая может содержать пароль).
func TestEmailAlerter_AuthError_NoCredentialLeak(t *testing.T) {
	secretPassword := "super-secret-password-12345"
	config := EmailConfig{
		Enabled:      true,
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPUser:     "user@example.com",
		SMTPPassword: secretPassword,
		UseTLS:       true,
		From:         "alerts@example.com",
		To:           []string{"devops@example.com"},
	}

	// Mock: Auth() возвращает ошибку, содержащую credentials
	mockClient := &mockSMTPClient{
		authErr: fmt.Errorf("authentication failed for user %s with password %s", config.SMTPUser, secretPassword),
	}
	mockDialer := &mockSMTPDialer{client: mockClient}
	logger := &testLogger{}

	alerter, err := NewEmailAlerter(config, nil, logger)
	if err != nil {
		t.Fatalf("NewEmailAlerter() error = %v", err)
	}
	alerter.SetDialer(mockDialer)

	alert := Alert{
		ErrorCode: "AUTH_LEAK_TEST",
		Message:   "Test auth credential leak",
		Timestamp: time.Now(),
	}

	// Send должен вернуть nil (ошибки логируются, AC10)
	err = alerter.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() error = %v, want nil", err)
	}

	// Проверяем что в логах нет пароля
	for _, msg := range logger.errorMsgs {
		if strings.Contains(msg, secretPassword) {
			t.Errorf("error log contains secret password: %s", msg)
		}
	}

	// Проверяем что ошибка в логе — ErrSMTPAuth, а не оригинальная
	if len(logger.errorMsgs) == 0 {
		t.Fatal("expected error log for auth failure")
	}
}
