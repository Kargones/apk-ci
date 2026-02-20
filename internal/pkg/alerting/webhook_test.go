package alerting

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newTestWebhookAlerter создаёт WebhookAlerter для тестирования.
func newTestWebhookAlerter(t *testing.T, config WebhookConfig) (*WebhookAlerter, *mockHTTPClient) {
	t.Helper()
	mockClient := &mockHTTPClient{}
	alerter, err := NewWebhookAlerter(config, nil, &testLogger{})
	if err != nil {
		t.Fatalf("failed to create WebhookAlerter: %v", err)
	}
	alerter.SetHTTPClient(mockClient)
	return alerter, mockClient
}

func TestWebhookAlerter_Send(t *testing.T) {
	config := WebhookConfig{
		Enabled:    true,
		URLs:       []string{"https://hooks.example.com/webhook"},
		Timeout:    10 * time.Second,
		MaxRetries: 3,
	}

	alerter, mockClient := newTestWebhookAlerter(t, config)

	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		// Проверяем URL
		if req.URL.String() != config.URLs[0] {
			t.Errorf("unexpected URL: got %s, want %s", req.URL.String(), config.URLs[0])
		}

		// Проверяем Content-Type
		if ct := req.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("unexpected Content-Type: got %s, want application/json", ct)
		}

		// Проверяем User-Agent
		if ua := req.Header.Get("User-Agent"); ua != "apk-ci/1.0" {
			t.Errorf("unexpected User-Agent: got %s, want apk-ci/1.0", ua)
		}

		// Проверяем body
		var body WebhookPayload
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}
		if body.ErrorCode != "TEST_ERROR" {
			t.Errorf("unexpected error_code: got %s, want TEST_ERROR", body.ErrorCode)
		}
		if body.Source != "apk-ci" {
			t.Errorf("unexpected source: got %s, want apk-ci", body.Source)
		}

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"ok": true}`)),
		}, nil
	}

	alert := Alert{
		ErrorCode: "TEST_ERROR",
		Severity:  SeverityCritical,
		Command:   "test-command",
		Message:   "Test message",
		Infobase:  "TestDB",
		TraceID:   "trace-123",
		Timestamp: time.Now(),
	}

	err := alerter.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	if len(mockClient.Requests) != 1 {
		t.Errorf("expected 1 request, got %d", len(mockClient.Requests))
	}
}

func TestWebhookAlerter_MultipleURLs(t *testing.T) {
	config := WebhookConfig{
		Enabled: true,
		URLs: []string{
			"https://hooks.slack.com/services/XXX",
			"https://api.pagerduty.com/v2/enqueue",
			"https://custom.webhook.io/alert",
		},
		Timeout:    10 * time.Second,
		MaxRetries: 3,
	}

	alerter, mockClient := newTestWebhookAlerter(t, config)

	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"ok": true}`)),
		}, nil
	}

	alert := Alert{
		ErrorCode: "MULTI_URL_TEST",
		Severity:  SeverityCritical,
		Command:   "test-command",
		Message:   "Test message",
		Timestamp: time.Now(),
	}

	err := alerter.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	// Должно быть 3 запроса (по одному на каждый URL)
	if len(mockClient.Requests) != 3 {
		t.Errorf("expected 3 requests, got %d", len(mockClient.Requests))
	}

	// Проверяем что все URL были вызваны
	urlsCalled := make(map[string]bool)
	for _, req := range mockClient.Requests {
		urlsCalled[req.URL.String()] = true
	}
	for _, url := range config.URLs {
		if !urlsCalled[url] {
			t.Errorf("URL %s was not called", url)
		}
	}
}

func TestWebhookAlerter_Disabled(t *testing.T) {
	// Проверяем что disabled alerter не делает запросов
	// (disabled проверяется в factory, но мы тестируем с пустыми URLs)
	alerter := &WebhookAlerter{
		config: WebhookConfig{
			Enabled: true,
			URLs:    []string{}, // Пустой slice
		},
		logger: &testLogger{},
	}

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			t.Error("HTTP client should not be called when no URLs")
			return nil, errors.New("should not be called")
		},
	}
	alerter.httpClient = mockClient

	alert := Alert{
		ErrorCode: "DISABLED_TEST",
		Timestamp: time.Now(),
	}

	// Без URLs не будет запросов
	_ = alerter.Send(context.Background(), alert)

	if len(mockClient.Requests) != 0 {
		t.Errorf("expected 0 requests when no URLs, got %d", len(mockClient.Requests))
	}
}

// H-2/Review #9: Тест warning log при полном отказе доставки.
func TestWebhookAlerter_TotalFailure_WarningLog(t *testing.T) {
	config := WebhookConfig{
		Enabled:    true,
		URLs:       []string{"https://url1.com/hook", "https://url2.com/hook"},
		Timeout:    10 * time.Second,
		MaxRetries: 0, // Без retry для быстроты теста
	}

	alerter, mockClient := newTestWebhookAlerter(t, config)
	logger := &testLogger{}
	alerter.logger = logger

	// Все URL возвращают ошибку
	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	}

	alert := Alert{
		ErrorCode: "TOTAL_FAIL_TEST",
		Message:   "Test total delivery failure",
		Timestamp: time.Now(),
	}

	err := alerter.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() error = %v, want nil", err)
	}

	// Должен быть warn лог о полном отказе
	foundWarnLog := false
	for _, msg := range logger.warnMsgs {
		if strings.Contains(msg, "не доставлен") {
			foundWarnLog = true
			break
		}
	}
	if !foundWarnLog {
		t.Error("expected warning log when all URLs failed delivery")
	}
}

func TestWebhookAlerter_HTTPError(t *testing.T) {
	config := WebhookConfig{
		Enabled:    true,
		URLs:       []string{"https://hooks.example.com/webhook"},
		Timeout:    10 * time.Second,
		MaxRetries: 0, // Без retry
	}

	alerter, mockClient := newTestWebhookAlerter(t, config)

	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(strings.NewReader(`{"error": "Internal Server Error"}`)),
		}, nil
	}

	alert := Alert{
		ErrorCode: "HTTP_ERROR_TEST",
		Severity:  SeverityCritical,
		Command:   "test-command",
		Message:   "Test message",
		Timestamp: time.Now(),
	}

	// Send() должен вернуть nil (ошибки логируются, но не возвращаются)
	err := alerter.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() should return nil on HTTP error, got %v", err)
	}
}

func TestWebhookAlerter_RetryOnError(t *testing.T) {
	ctx := context.Background()
	config := WebhookConfig{
		Enabled:    true,
		URLs:       []string{"https://hooks.example.com/webhook"},
		Timeout:    100 * time.Millisecond,
		MaxRetries: 3,
	}

	alerter, err := NewWebhookAlerter(config, nil, &testLogger{})
	if err != nil {
		t.Fatalf("failed to create WebhookAlerter: %v", err)
	}

	var requestCount int32
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			count := atomic.AddInt32(&requestCount, 1)
			if count <= 2 {
				// Первые 2 попытки — network error (retriable)
				return nil, errors.New("connection refused")
			}
			// Третья попытка — успех
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"ok": true}`)),
			}, nil
		},
	}
	alerter.SetHTTPClient(mockClient)

	alert := Alert{
		ErrorCode: "RETRY_TEST",
		Timestamp: time.Now(),
	}

	// Используем контекст с таймаутом для ускорения теста
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = alerter.Send(ctx, alert)
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	// Должно быть 3 попытки: 2 failed + 1 success
	if requestCount != 3 {
		t.Errorf("expected 3 requests (2 retries + 1 success), got %d", requestCount)
	}
}

func TestWebhookAlerter_NoRetryOnHTTPError(t *testing.T) {
	config := WebhookConfig{
		Enabled:    true,
		URLs:       []string{"https://hooks.example.com/webhook"},
		Timeout:    100 * time.Millisecond,
		MaxRetries: 3,
	}

	alerter, mockClient := newTestWebhookAlerter(t, config)

	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		// HTTP 400 error — не должен retry
		return &http.Response{
			StatusCode: 400,
			Body:       io.NopCloser(strings.NewReader(`{"error": "bad request"}`)),
		}, nil
	}

	alert := Alert{
		ErrorCode: "NO_RETRY_HTTP_ERROR",
		Timestamp: time.Now(),
	}

	_ = alerter.Send(context.Background(), alert)

	// Должен быть только 1 запрос (без retry для HTTP ошибок)
	if len(mockClient.Requests) != 1 {
		t.Errorf("expected 1 request (no retry for HTTP errors), got %d", len(mockClient.Requests))
	}
}

func TestWebhookAlerter_RateLimited(t *testing.T) {
	config := WebhookConfig{
		Enabled:    true,
		URLs:       []string{"https://hooks.example.com/webhook"},
		Timeout:    10 * time.Second,
		MaxRetries: 3,
	}

	rateLimiter := NewRateLimiter(5 * time.Minute)

	alerter, err := NewWebhookAlerter(config, rateLimiter, &testLogger{})
	if err != nil {
		t.Fatalf("failed to create WebhookAlerter: %v", err)
	}

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"ok": true}`)),
			}, nil
		},
	}
	alerter.SetHTTPClient(mockClient)

	alert := Alert{
		ErrorCode: "RATE_LIMIT_TEST",
		Severity:  SeverityCritical,
		Command:   "test-command",
		Message:   "Test message",
		Timestamp: time.Now(),
	}

	// Первый вызов — должен пройти
	err = alerter.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("first Send() error = %v", err)
	}
	if len(mockClient.Requests) != 1 {
		t.Errorf("expected 1 request after first send, got %d", len(mockClient.Requests))
	}

	// Второй вызов с тем же error_code — должен быть rate limited
	err = alerter.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("second Send() error = %v", err)
	}
	// Количество запросов не должно измениться
	if len(mockClient.Requests) != 1 {
		t.Errorf("expected still 1 request after rate limited send, got %d", len(mockClient.Requests))
	}
}

func TestWebhookAlerter_CustomHeaders(t *testing.T) {
	config := WebhookConfig{
		Enabled: true,
		URLs:    []string{"https://hooks.example.com/webhook"},
		Headers: map[string]string{
			"Authorization": "Bearer secret-token-123",
			"X-Api-Key":     "api-key-456",
			"X-Custom":      "custom-value",
		},
		Timeout:    10 * time.Second,
		MaxRetries: 3,
	}

	alerter, mockClient := newTestWebhookAlerter(t, config)

	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		// Проверяем все custom headers
		for key, value := range config.Headers {
			if got := req.Header.Get(key); got != value {
				t.Errorf("header %s = %s, want %s", key, got, value)
			}
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"ok": true}`)),
		}, nil
	}

	alert := Alert{
		ErrorCode: "CUSTOM_HEADERS_TEST",
		Timestamp: time.Now(),
	}

	err := alerter.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	if len(mockClient.Requests) != 1 {
		t.Errorf("expected 1 request, got %d", len(mockClient.Requests))
	}
}

// H-3/Review #9: Тест что hostname кэшируется в конструкторе.
func TestWebhookAlerter_HostnameCached(t *testing.T) {
	config := WebhookConfig{
		Enabled:    true,
		URLs:       []string{"https://hooks.example.com/webhook"},
		Timeout:    10 * time.Second,
		MaxRetries: 0,
	}

	alerter, err := NewWebhookAlerter(config, nil, &testLogger{})
	if err != nil {
		t.Fatalf("NewWebhookAlerter() error = %v", err)
	}

	// hostname должен быть заполнен при создании (не пустой)
	if alerter.hostname == "" {
		t.Error("hostname should be cached in constructor, got empty string")
	}

	// Проверяем что hostname попадает в payload
	var capturedPayload WebhookPayload
	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(req.Body)
			_ = json.Unmarshal(body, &capturedPayload)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"ok": true}`)),
			}, nil
		},
	}
	alerter.SetHTTPClient(mockClient)

	alert := Alert{
		ErrorCode: "HOSTNAME_TEST",
		Timestamp: time.Now(),
	}
	_ = alerter.Send(context.Background(), alert)

	if capturedPayload.Hostname != alerter.hostname {
		t.Errorf("payload.hostname = %q, want cached %q", capturedPayload.Hostname, alerter.hostname)
	}
}

func TestWebhookAlerter_CustomTimeout(t *testing.T) {
	customTimeout := 5 * time.Second

	config := WebhookConfig{
		Enabled:    true,
		URLs:       []string{"https://hooks.example.com/webhook"},
		Timeout:    customTimeout,
		MaxRetries: 3,
	}

	alerter, err := NewWebhookAlerter(config, nil, &testLogger{})
	if err != nil {
		t.Fatalf("failed to create WebhookAlerter: %v", err)
	}

	// Проверяем что timeout был применён к internal HTTP client
	httpClient, ok := alerter.httpClient.(*http.Client)
	if !ok {
		t.Skip("httpClient is mocked, skipping timeout check")
	}

	if httpClient.Timeout != customTimeout {
		t.Errorf("HTTP client timeout = %v, want %v", httpClient.Timeout, customTimeout)
	}
}

func TestWebhookAlerter_DefaultTimeout(t *testing.T) {
	config := WebhookConfig{
		Enabled:    true,
		URLs:       []string{"https://hooks.example.com/webhook"},
		Timeout:    0, // zero value — должен использоваться default
		MaxRetries: 3,
	}

	alerter, err := NewWebhookAlerter(config, nil, &testLogger{})
	if err != nil {
		t.Fatalf("failed to create WebhookAlerter: %v", err)
	}

	httpClient, ok := alerter.httpClient.(*http.Client)
	if !ok {
		t.Skip("httpClient is mocked, skipping timeout check")
	}

	if httpClient.Timeout != DefaultWebhookTimeout {
		t.Errorf("HTTP client timeout = %v, want default %v", httpClient.Timeout, DefaultWebhookTimeout)
	}
}

func TestWebhookAlerter_PayloadFormat(t *testing.T) {
	config := WebhookConfig{
		Enabled:    true,
		URLs:       []string{"https://hooks.example.com/webhook"},
		Timeout:    10 * time.Second,
		MaxRetries: 3,
	}

	alerter, mockClient := newTestWebhookAlerter(t, config)

	var capturedPayload WebhookPayload
	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		if err := json.Unmarshal(body, &capturedPayload); err != nil {
			t.Errorf("failed to decode payload: %v", err)
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"ok": true}`)),
		}, nil
	}

	testTime := time.Date(2026, 2, 5, 10, 30, 0, 0, time.UTC)
	alert := Alert{
		ErrorCode: "PAYLOAD_TEST",
		Severity:  SeverityCritical,
		Command:   "db-restore",
		Message:   "Database restore failed",
		Infobase:  "TestDB",
		TraceID:   "trace-abc-123",
		Timestamp: testTime,
	}

	err := alerter.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	// Проверяем все поля payload
	if capturedPayload.ErrorCode != alert.ErrorCode {
		t.Errorf("payload.error_code = %s, want %s", capturedPayload.ErrorCode, alert.ErrorCode)
	}
	if capturedPayload.Message != alert.Message {
		t.Errorf("payload.message = %s, want %s", capturedPayload.Message, alert.Message)
	}
	if capturedPayload.TraceID != alert.TraceID {
		t.Errorf("payload.trace_id = %s, want %s", capturedPayload.TraceID, alert.TraceID)
	}
	if capturedPayload.Command != alert.Command {
		t.Errorf("payload.command = %s, want %s", capturedPayload.Command, alert.Command)
	}
	if capturedPayload.Infobase != alert.Infobase {
		t.Errorf("payload.infobase = %s, want %s", capturedPayload.Infobase, alert.Infobase)
	}
	if capturedPayload.Severity != "CRITICAL" {
		t.Errorf("payload.severity = %s, want CRITICAL", capturedPayload.Severity)
	}
	if capturedPayload.Source != "apk-ci" {
		t.Errorf("payload.source = %s, want apk-ci", capturedPayload.Source)
	}
	if !capturedPayload.Timestamp.Equal(testTime) {
		t.Errorf("payload.timestamp = %v, want %v", capturedPayload.Timestamp, testTime)
	}
}

func TestWebhookAlerter_ContextCanceled(t *testing.T) {
	ctx := context.Background()
	config := WebhookConfig{
		Enabled:    true,
		URLs:       []string{"https://url1.com", "https://url2.com", "https://url3.com"},
		Timeout:    10 * time.Second,
		MaxRetries: 3,
	}

	alerter, mockClient := newTestWebhookAlerter(t, config)
	logger := &testLogger{}
	alerter.logger = logger

	requestCount := 0
	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		requestCount++
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"ok": true}`)),
		}, nil
	}

	// Создаём уже отменённый контекст
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем сразу

	alert := Alert{
		ErrorCode: "CONTEXT_TEST",
		Message:   "Testing context cancellation",
		Timestamp: time.Now(),
	}

	// Send должен вернуть nil (ошибки логируются, не возвращаются)
	err := alerter.Send(ctx, alert)
	if err != nil {
		t.Errorf("Send() error = %v, want nil", err)
	}

	// При отменённом контексте не должно быть ни одного запроса
	if requestCount != 0 {
		t.Errorf("expected 0 requests with canceled context, got %d", requestCount)
	}

	// Должен быть debug лог об отмене
	foundCancelLog := false
	for _, msg := range logger.debugMsgs {
		if strings.Contains(msg, "отменена") {
			foundCancelLog = true
			break
		}
	}
	if !foundCancelLog {
		t.Error("expected debug log about context cancellation")
	}
}

func TestWebhookConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  WebhookConfig
		wantErr error
	}{
		{
			name: "disabled - no validation",
			config: WebhookConfig{
				Enabled: false,
			},
			wantErr: nil,
		},
		{
			name: "enabled - missing URLs",
			config: WebhookConfig{
				Enabled: true,
				URLs:    []string{},
			},
			wantErr: ErrWebhookURLRequired,
		},
		{
			name: "enabled - valid config",
			config: WebhookConfig{
				Enabled: true,
				URLs:    []string{"https://example.com/webhook"},
			},
			wantErr: nil,
		},
		{
			name: "enabled - multiple URLs valid",
			config: WebhookConfig{
				Enabled: true,
				URLs:    []string{"https://url1.com", "https://url2.com"},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMultiChannelAlerter_AllChannels(t *testing.T) {
	emailMock := &mockAlerter{}
	telegramMock := &mockAlerter{}
	webhookMock := &mockAlerter{}

	channels := map[string]Alerter{
		"email":    emailMock,
		"telegram": telegramMock,
		"webhook":  webhookMock,
	}
	multi := NewMultiChannelAlerter(channels, nil, nil, &testLogger{})

	alert := Alert{
		ErrorCode: "ALL_CHANNELS_TEST",
		Severity:  SeverityCritical,
		Command:   "test-command",
		Message:   "Test message for all channels",
		Timestamp: time.Now(),
	}

	err := multi.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	// Все 3 канала должны получить alert
	if emailMock.sendCount != 1 {
		t.Errorf("email alerter sendCount = %d, want 1", emailMock.sendCount)
	}
	if telegramMock.sendCount != 1 {
		t.Errorf("telegram alerter sendCount = %d, want 1", telegramMock.sendCount)
	}
	if webhookMock.sendCount != 1 {
		t.Errorf("webhook alerter sendCount = %d, want 1", webhookMock.sendCount)
	}

	// Проверяем что alert передан корректно
	if emailMock.lastAlert.ErrorCode != alert.ErrorCode {
		t.Errorf("email alerter got wrong error_code: %s", emailMock.lastAlert.ErrorCode)
	}
	if telegramMock.lastAlert.ErrorCode != alert.ErrorCode {
		t.Errorf("telegram alerter got wrong error_code: %s", telegramMock.lastAlert.ErrorCode)
	}
	if webhookMock.lastAlert.ErrorCode != alert.ErrorCode {
		t.Errorf("webhook alerter got wrong error_code: %s", webhookMock.lastAlert.ErrorCode)
	}
}

func TestIsClientHTTPError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "4xx — клиентская ошибка",
			err:      &httpError{StatusCode: 400, Body: "bad request"},
			expected: true,
		},
		{
			name:     "403 — клиентская ошибка",
			err:      &httpError{StatusCode: 403, Body: "forbidden"},
			expected: true,
		},
		{
			name:     "499 — клиентская ошибка",
			err:      &httpError{StatusCode: 499, Body: "client error"},
			expected: true,
		},
		{
			name:     "5xx — серверная ошибка (не клиентская)",
			err:      &httpError{StatusCode: 500, Body: "server error"},
			expected: false,
		},
		{
			name:     "503 — серверная ошибка (не клиентская)",
			err:      &httpError{StatusCode: 503, Body: "service unavailable"},
			expected: false,
		},
		{
			name:     "network error — не HTTP ошибка",
			err:      errors.New("connection refused"),
			expected: false,
		},
		{
			name:     "nil error — не HTTP ошибка",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isClientHTTPError(tt.err); got != tt.expected {
				t.Errorf("isClientHTTPError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHttpError_Error(t *testing.T) {
	err := &httpError{StatusCode: 404, Body: "Not Found"}
	expected := "HTTP 404: Not Found"
	if err.Error() != expected {
		t.Errorf("httpError.Error() = %s, want %s", err.Error(), expected)
	}
}

func TestWebhookConfig_Validate_InvalidURL(t *testing.T) {
	tests := []struct {
		name    string
		urls    []string
		wantErr error
	}{
		{
			name:    "valid https url",
			urls:    []string{"https://hooks.example.com/webhook"},
			wantErr: nil,
		},
		{
			name:    "valid http url",
			urls:    []string{"http://internal.webhook.local/alert"},
			wantErr: nil,
		},
		{
			name:    "missing scheme",
			urls:    []string{"hooks.example.com/webhook"},
			wantErr: ErrWebhookURLInvalid,
		},
		{
			name:    "missing host",
			urls:    []string{"/webhook"},
			wantErr: ErrWebhookURLInvalid,
		},
		{
			name:    "one valid one invalid",
			urls:    []string{"https://valid.com", "not-a-url"},
			wantErr: ErrWebhookURLInvalid,
		},
		{
			name:    "file scheme - SSRF",
			urls:    []string{"file:///etc/passwd"},
			wantErr: ErrWebhookURLInvalid,
		},
		{
			name:    "ftp scheme - SSRF",
			urls:    []string{"ftp://ftp.example.com/file"},
			wantErr: ErrWebhookURLInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := WebhookConfig{
				Enabled: true,
				URLs:    tt.urls,
			}
			err := config.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Тест: retry ПРОИСХОДИТ на 5xx HTTP ошибках (503, 502, 504).
// 5xx — серверные ошибки, часто временные (Slack, PagerDuty могут вернуть 503).
func TestWebhookAlerter_RetryOn5xx(t *testing.T) {
	ctx := context.Background()
	config := WebhookConfig{
		Enabled:    true,
		URLs:       []string{"https://hooks.example.com/webhook"},
		Timeout:    100 * time.Millisecond,
		MaxRetries: 2,
	}

	alerter, mockClient := newTestWebhookAlerter(t, config)

	var requestCount int32
	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		atomic.AddInt32(&requestCount, 1)
		// 503 Service Unavailable — серверная ошибка, должна retry'ться
		return &http.Response{
			StatusCode: 503,
			Body:       io.NopCloser(strings.NewReader(`{"error": "service unavailable"}`)),
		}, nil
	}

	alert := Alert{
		ErrorCode: "RETRY_5XX_TEST",
		Timestamp: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = alerter.Send(ctx, alert)

	// 5xx → retry: 1 (initial) + 2 (retries) = 3 запроса
	count := atomic.LoadInt32(&requestCount)
	if count != 3 {
		t.Errorf("expected 3 requests (initial + 2 retries for 5xx), got %d", count)
	}
}

// Тест: retry НЕ происходит на 4xx HTTP ошибках.
// 4xx — клиентские ошибки (неправильный URL, авторизация), retry бесполезен.
func TestWebhookAlerter_NoRetryOn4xx(t *testing.T) {
	config := WebhookConfig{
		Enabled:    true,
		URLs:       []string{"https://hooks.example.com/webhook"},
		Timeout:    100 * time.Millisecond,
		MaxRetries: 3,
	}

	alerter, mockClient := newTestWebhookAlerter(t, config)

	requestCount := 0
	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		requestCount++
		return &http.Response{
			StatusCode: 403,
			Body:       io.NopCloser(strings.NewReader(`{"error": "forbidden"}`)),
		}, nil
	}

	alert := Alert{
		ErrorCode: "NO_RETRY_4XX_TEST",
		Timestamp: time.Now(),
	}

	_ = alerter.Send(context.Background(), alert)

	// 4xx — retry НЕ происходит
	if requestCount != 1 {
		t.Errorf("expected 1 request (no retry for 4xx HTTP errors), got %d", requestCount)
	}
}

// H-2 fix: Тест валидации HTTP заголовков — защита от Header Injection.
func TestWebhookConfig_Validate_HeaderInjection(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		wantErr error
	}{
		{
			name:    "валидные headers — ok",
			headers: map[string]string{"Authorization": "Bearer token123", "X-Api-Key": "key"},
			wantErr: nil,
		},
		{
			name:    "\\r в значении — ошибка",
			headers: map[string]string{"X-Bad": "value\rinjection"},
			wantErr: ErrWebhookHeaderInvalid,
		},
		{
			name:    "\\n в значении — ошибка",
			headers: map[string]string{"X-Bad": "value\ninjection"},
			wantErr: ErrWebhookHeaderInvalid,
		},
		{
			name:    "\\r\\n в ключе — ошибка",
			headers: map[string]string{"X-Bad\r\n": "value"},
			wantErr: ErrWebhookHeaderInvalid,
		},
		{
			name:    "null byte в значении — ошибка",
			headers: map[string]string{"X-Bad": "value\x00injection"},
			wantErr: ErrWebhookHeaderInvalid,
		},
		{
			name:    "tab в значении — разрешён (HTAB допустим по RFC 7230)",
			headers: map[string]string{"X-Custom": "value\twith-tab"},
			wantErr: nil,
		},
		{
			name:    "DEL в ключе — ошибка (0x7f)",
			headers: map[string]string{"X-Bad\x7f": "value"},
			wantErr: ErrWebhookHeaderInvalid,
		},
		{
			name:    "пустые headers — ok",
			headers: map[string]string{},
			wantErr: nil,
		},
		{
			name:    "nil headers — ok",
			headers: nil,
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := WebhookConfig{
				Enabled: true,
				URLs:    []string{"https://example.com/webhook"},
				Headers: tt.headers,
			}
			err := cfg.Validate()
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

