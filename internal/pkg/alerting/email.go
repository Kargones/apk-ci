package alerting

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"text/template"
	"time"

	"github.com/Kargones/apk-ci/internal/pkg/logging"
)

// emailBodyTemplate — шаблон тела письма.
const emailBodyTemplate = `apk-ci Alert

Error Code: {{.ErrorCode}}
Severity: {{.SeverityStr}}
Command: {{.Command}}
Infobase: {{.Infobase}}

Message:
{{.Message}}

Trace ID: {{.TraceID}}
Timestamp: {{.TimestampStr}}

---
This is an automated alert from apk-ci.
`

// SMTPDialer определяет интерфейс для создания SMTP соединений.
// Используется для тестирования (mock SMTP).
type SMTPDialer interface {
	DialContext(ctx context.Context, addr string) (SMTPClient, error)
}

// SMTPClient определяет интерфейс SMTP клиента.
// Используется для тестирования (mock SMTP).
type SMTPClient interface {
	StartTLS(config *tls.Config) error
	Auth(a smtp.Auth) error
	Mail(from string) error
	Rcpt(to string) error
	Data() (WriteCloser, error)
	Close() error
	Extension(ext string) (bool, string)
}

// WriteCloser определяет интерфейс для записи и закрытия.
type WriteCloser interface {
	Write(p []byte) (n int, err error)
	Close() error
}

// emailTemplateData содержит данные для шаблонов email.
type emailTemplateData struct {
	ErrorCode    string
	SeverityStr  string
	Command      string
	Infobase     string
	Message      string
	TraceID      string
	TimestampStr string
}

// EmailAlerter реализует Alerter для отправки email через SMTP.
// Поддерживает TLS (StartTLS и implicit SSL), rate limiting.
type EmailAlerter struct {
	config      EmailConfig
	rateLimiter *RateLimiter
	logger      logging.Logger
	dialer      SMTPDialer
	subjectTmpl *template.Template
	bodyTmpl    *template.Template
}

// NewEmailAlerter создаёт EmailAlerter с указанной конфигурацией.
// Параметры:
//   - config: конфигурация email канала
//   - rateLimiter: rate limiter для ограничения частоты алертов
//   - logger: логгер для записи ошибок и информационных сообщений
//
// Возвращает ошибку если шаблоны некорректны.
func NewEmailAlerter(config EmailConfig, rateLimiter *RateLimiter, logger logging.Logger) (*EmailAlerter, error) {
	// Парсим шаблон темы
	subjectTemplate := config.SubjectTemplate
	if subjectTemplate == "" {
		subjectTemplate = DefaultSubjectTemplate
	}
	subjectTmpl, err := template.New("subject").Parse(subjectTemplate)
	if err != nil {
		return nil, fmt.Errorf("alerting: invalid subject template: %w", err)
	}

	// Парсим шаблон тела
	bodyTmpl, err := template.New("body").Parse(emailBodyTemplate)
	if err != nil {
		return nil, fmt.Errorf("alerting: invalid body template: %w", err)
	}

	return &EmailAlerter{
		config:      config,
		rateLimiter: rateLimiter,
		logger:      logger,
		dialer: &defaultDialer{
			timeout:    config.Timeout,
			useTLS:     config.UseTLS,
			smtpPort:   config.SMTPPort,
			serverName: config.SMTPHost,
		},
		subjectTmpl: subjectTmpl,
		bodyTmpl:    bodyTmpl,
	}, nil
}

// SetDialer устанавливает кастомный SMTPDialer (для тестирования).
func (e *EmailAlerter) SetDialer(dialer SMTPDialer) {
	e.dialer = dialer
}

// Send отправляет алерт через email.
// Применяет rate limiting по ErrorCode.
// При ошибке SMTP — логирует ошибку и возвращает nil (приложение продолжает работу).
func (e *EmailAlerter) Send(ctx context.Context, alert Alert) error {
	// Проверяем rate limiting.
	// Примечание: при создании через factory rateLimiter=nil (rate limiting на уровне
	// MultiChannelAlerter). Guard оставлен для прямого использования EmailAlerter.
	if e.rateLimiter != nil && !e.rateLimiter.Allow(alert.ErrorCode) {
		e.logger.Debug("алерт подавлен rate limiter",
			"error_code", alert.ErrorCode,
		)
		return nil // Rate limited — не ошибка
	}

	// Формируем email
	subject, body, err := e.formatEmail(alert)
	if err != nil {
		e.logger.Error("ошибка форматирования email",
			"error", err.Error(),
			"error_code", alert.ErrorCode,
		)
		return nil // Ошибка форматирования — логируем, продолжаем
	}

	// Отправляем email
	if err := e.sendEmail(ctx, subject, body); err != nil {
		e.logger.Error("ошибка отправки email алерта",
			"error", err.Error(),
			"error_code", alert.ErrorCode,
			"smtp_host", e.config.SMTPHost,
		)
		return nil // SMTP ошибка — логируем, приложение продолжает работу (AC10)
	}

	e.logger.Info("email алерт отправлен",
		"error_code", alert.ErrorCode,
		"severity", alert.Severity.String(),
		"recipients", len(e.config.To),
	)

	return nil
}

// formatEmail форматирует subject и body email из Alert.
func (e *EmailAlerter) formatEmail(alert Alert) (subject, body string, err error) {
	data := emailTemplateData{
		ErrorCode:    alert.ErrorCode,
		SeverityStr:  alert.Severity.String(),
		Command:      alert.Command,
		Infobase:     alert.Infobase,
		Message:      alert.Message,
		TraceID:      alert.TraceID,
		TimestampStr: alert.Timestamp.Format(time.RFC3339),
	}

	// Форматируем subject
	var subjectBuf bytes.Buffer
	if err := e.subjectTmpl.Execute(&subjectBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to format subject: %w", err)
	}
	subject = subjectBuf.String()

	// Форматируем body
	var bodyBuf bytes.Buffer
	if err := e.bodyTmpl.Execute(&bodyBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to format body: %w", err)
	}
	body = bodyBuf.String()

	return subject, body, nil
}

// sendEmail отправляет email через SMTP.
func (e *EmailAlerter) sendEmail(ctx context.Context, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", e.config.SMTPHost, e.config.SMTPPort)

	// Создаём соединение с поддержкой context для отмены
	client, err := e.dialer.DialContext(ctx, addr)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSMTPConnection, err)
	}
	defer client.Close()

	// StartTLS если включён TLS и порт не 465 (implicit SSL уже установлен в Dial)
	if e.config.UseTLS && e.config.SMTPPort != SMTPPortImplicitTLS {
		// Проверяем поддержку STARTTLS
		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsConfig := &tls.Config{
				ServerName: e.config.SMTPHost,
				MinVersion: tls.VersionTLS12,
			}
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("STARTTLS failed: %w", err)
			}
		}
	}

	// Авторизация если указаны credentials
	if e.config.SMTPUser != "" && e.config.SMTPPassword != "" {
		auth := smtp.PlainAuth("", e.config.SMTPUser, e.config.SMTPPassword, e.config.SMTPHost)
		if err := client.Auth(auth); err != nil {
			// M1 fix: Не включаем оригинальную ошибку — она может содержать credentials
			return ErrSMTPAuth
		}
	} else if e.config.SMTPUser != "" || e.config.SMTPPassword != "" {
		// H-3/Review #10: предупреждение при неполных credentials — вероятно ошибка конфигурации.
		e.logger.Warn("неполные SMTP credentials: указан SMTPUser или SMTPPassword, но не оба — авторизация пропущена",
			"smtp_host", e.config.SMTPHost,
			"has_user", e.config.SMTPUser != "",
			"has_password", e.config.SMTPPassword != "",
		)
	}

	// Отправка
	if err := client.Mail(e.config.From); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}

	for _, to := range e.config.To {
		// Проверяем контекст перед каждым получателем
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("RCPT TO failed for %s: %w", to, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA failed: %w", err)
	}

	// Формируем полное сообщение с заголовками
	msg := e.buildMessage(subject, body)
	if _, err := w.Write([]byte(msg)); err != nil {
		w.Close()
		return fmt.Errorf("write message failed: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("%w: %v", ErrSMTPSend, err)
	}

	return nil
}

// buildMessage формирует полное email сообщение с заголовками.
// H3 fix: Proper MIME encoding для кириллицы и других non-ASCII символов.
func (e *EmailAlerter) buildMessage(subject, body string) string {
	var buf bytes.Buffer

	buf.WriteString("From: ")
	buf.WriteString(e.config.From)
	buf.WriteString("\r\n")

	buf.WriteString("To: ")
	for i, to := range e.config.To {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(to)
	}
	buf.WriteString("\r\n")

	// RFC 2047: Encoded-Word для non-ASCII в Subject
	// Формат: =?charset?encoding?encoded_text?=
	buf.WriteString("Subject: ")
	buf.WriteString(encodeRFC2047(subject))
	buf.WriteString("\r\n")

	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	buf.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	buf.WriteString("\r\n")

	buf.WriteString(body)

	return buf.String()
}

// encodeRFC2047 кодирует строку в формате RFC 2047 для использования в email headers.
// Использует Base64 encoding для поддержки любых Unicode символов.
//
// TODO (M-3/Review #9): RFC 2047 ограничивает длину encoded-word до 75 символов.
// При длинных subject (>~40 символов non-ASCII) необходимо разбивать на несколько
// encoded-word блоков. На практике apk-ci использует короткие subject
// (error_code + command), поэтому лимит не превышается.
//
// TODO (L-3/Review #9): ASCII строки содержащие последовательность "=?" могут быть
// ошибочно интерпретированы как начало encoded-word. При необходимости — кодировать
// такие строки принудительно.
func encodeRFC2047(s string) string {
	// Проверяем нужно ли кодировать (есть non-ASCII символы)
	needsEncoding := false
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			needsEncoding = true
			break
		}
	}

	if !needsEncoding {
		return s
	}

	// RFC 2047 Base64 encoding: =?UTF-8?B?base64_encoded_text?=
	encoded := base64.StdEncoding.EncodeToString([]byte(s))
	return "=?UTF-8?B?" + encoded + "?="
}

// SMTP порты.
const (
	// SMTPPortStartTLS — порт для SMTP с StartTLS (рекомендуемый).
	SMTPPortStartTLS = 587
	// SMTPPortImplicitTLS — порт для SMTP с implicit TLS (SSL).
	SMTPPortImplicitTLS = 465
	// SMTPPortPlain — порт для SMTP без шифрования (не рекомендуется).
	SMTPPortPlain = 25
)

// defaultDialer реализует SMTPDialer для реального SMTP.
type defaultDialer struct {
	timeout    time.Duration
	useTLS     bool
	smtpPort   int
	serverName string
}

func (d *defaultDialer) DialContext(ctx context.Context, addr string) (SMTPClient, error) {
	timeout := d.timeout
	if timeout == 0 {
		timeout = DefaultSMTPTimeout
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP address %q: %w", addr, err)
	}
	if d.serverName != "" {
		host = d.serverName
	}

	dialer := &net.Dialer{Timeout: timeout}
	var conn net.Conn

	// H2 fix: Implicit TLS для порта 465.
	// Используем dialer.DialContext + tls.Client вместо tls.DialWithDialer,
	// чтобы context cancellation работала и для implicit TLS (H-2/Review #8).
	if d.useTLS && d.smtpPort == SMTPPortImplicitTLS {
		tlsConfig := &tls.Config{
			ServerName: host,
			MinVersion: tls.VersionTLS12,
		}
		rawConn, dialErr := dialer.DialContext(ctx, "tcp", addr)
		if dialErr != nil {
			return nil, dialErr
		}
		conn = tls.Client(rawConn, tlsConfig)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}

	if err != nil {
		return nil, err
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &smtpClientWrapper{client}, nil
}

// smtpClientWrapper оборачивает smtp.Client для реализации SMTPClient интерфейса.
type smtpClientWrapper struct {
	*smtp.Client
}

func (w *smtpClientWrapper) Data() (WriteCloser, error) {
	return w.Client.Data()
}
