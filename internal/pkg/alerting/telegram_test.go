package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// mockHTTPClient — mock для HTTPClient интерфейса.
type mockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
	// Requests хранит все полученные запросы для проверки.
	Requests []*http.Request
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.Requests = append(m.Requests, req)
	return m.DoFunc(req)
}

// mockHTTPResponse создаёт mock HTTP response.
func mockHTTPResponse(statusCode int, body interface{}) *http.Response {
	jsonBody, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader(jsonBody)),
	}
}

// newTestTelegramAlerter создаёт TelegramAlerter для тестирования.
func newTestTelegramAlerter(t *testing.T, config TelegramConfig) (*TelegramAlerter, *mockHTTPClient) {
	t.Helper()
	mockClient := &mockHTTPClient{}
	alerter, err := NewTelegramAlerter(config, nil, &testLogger{})
	if err != nil {
		t.Fatalf("failed to create TelegramAlerter: %v", err)
	}
	alerter.SetHTTPClient(mockClient)
	return alerter, mockClient
}

func TestTelegramAlerter_Send(t *testing.T) {
	config := TelegramConfig{
		Enabled:  true,
		BotToken: "123456:ABC-DEF-TEST-TOKEN",
		ChatIDs:  []string{"-1001234567890"},
		Timeout:  10 * time.Second,
	}

	alerter, mockClient := newTestTelegramAlerter(t, config)

	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		// Проверяем URL
		expectedURL := TelegramAPIBaseURL + config.BotToken + "/sendMessage"
		if req.URL.String() != expectedURL {
			t.Errorf("unexpected URL: got %s, want %s", req.URL.String(), expectedURL)
		}

		// Проверяем Content-Type
		if ct := req.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("unexpected Content-Type: got %s, want application/json", ct)
		}

		// Проверяем body
		var body telegramRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}
		if body.ChatID != config.ChatIDs[0] {
			t.Errorf("unexpected chat_id: got %s, want %s", body.ChatID, config.ChatIDs[0])
		}
		if body.ParseMode != "Markdown" {
			t.Errorf("unexpected parse_mode: got %s, want Markdown", body.ParseMode)
		}

		return mockHTTPResponse(200, telegramResponse{OK: true}), nil
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

func TestTelegramAlerter_MultipleChatIDs(t *testing.T) {
	config := TelegramConfig{
		Enabled:  true,
		BotToken: "123456:ABC-DEF-TEST-TOKEN",
		ChatIDs:  []string{"-1001234567890", "987654321", "@public_channel"},
		Timeout:  10 * time.Second,
	}

	alerter, mockClient := newTestTelegramAlerter(t, config)

	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		return mockHTTPResponse(200, telegramResponse{OK: true}), nil
	}

	alert := Alert{
		ErrorCode: "MULTI_CHAT_ERROR",
		Severity:  SeverityCritical,
		Command:   "test-command",
		Message:   "Test message",
		Timestamp: time.Now(),
	}

	err := alerter.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	// Должно быть 3 запроса (по одному на каждый chat_id)
	if len(mockClient.Requests) != 3 {
		t.Errorf("expected 3 requests, got %d", len(mockClient.Requests))
	}
}

// H2 fix: Переименован тест — проверяет отсутствие запросов при пустых ChatIDs.
// Enabled проверяется в factory (NewAlerter), не в TelegramAlerter.Send().
func TestTelegramAlerter_NoChatIDs_NoRequests(t *testing.T) {
	// Создаём alerter с пустыми ChatIDs
	alerter := &TelegramAlerter{
		config: TelegramConfig{
			Enabled:  true,
			BotToken: "test-token",
			ChatIDs:  []string{}, // Пустой slice
		},
		logger: &testLogger{},
	}

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			t.Error("HTTP client should not be called when no chat_ids")
			return nil, errors.New("should not be called")
		},
	}
	alerter.httpClient = mockClient

	alert := Alert{
		ErrorCode: "TEST_ERROR",
		Timestamp: time.Now(),
	}

	// Без chat_ids не будет запросов
	_ = alerter.Send(context.Background(), alert)

	if len(mockClient.Requests) != 0 {
		t.Errorf("expected 0 requests when no chat_ids, got %d", len(mockClient.Requests))
	}
}

// M-5 fix: Тест partial failure — один чат доступен, другой нет.
func TestTelegramAlerter_PartialFailure(t *testing.T) {
	config := TelegramConfig{
		Enabled:  true,
		BotToken: "123456:ABC-DEF-TEST-TOKEN",
		ChatIDs:  []string{"-1001111111111", "-1002222222222", "-1003333333333"},
		Timeout:  10 * time.Second,
	}

	alerter, mockClient := newTestTelegramAlerter(t, config)

	callCount := 0
	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		callCount++
		// Второй чат возвращает ошибку, остальные — OK
		if callCount == 2 {
			return mockHTTPResponse(400, telegramResponse{
				OK:          false,
				ErrorCode:   400,
				Description: "Bad Request: chat not found",
			}), nil
		}
		return mockHTTPResponse(200, telegramResponse{
			OK: true,
		}), nil
	}

	alert := Alert{
		ErrorCode: "PARTIAL_FAIL_TEST",
		Message:   "partial failure test",
		Timestamp: time.Now(),
	}

	err := alerter.Send(context.Background(), alert)
	if err != nil {
		t.Fatalf("Send() должен возвращать nil (AC10), got %v", err)
	}

	// Все 3 чата должны быть вызваны несмотря на ошибку во втором
	if callCount != 3 {
		t.Errorf("должны быть вызваны все 3 чата даже при ошибке во втором, got %d", callCount)
	}
}

// H-1 fix: Тест что BotToken не утекает в лог при HTTP ошибке.
// Go stdlib включает URL (с BotToken) в текст ошибки при ошибке HTTP клиента.
func TestTelegramAlerter_BotTokenNotLeakedInError(t *testing.T) {
	secretToken := "123456:SECRET-BOT-TOKEN-DO-NOT-LEAK"
	config := TelegramConfig{
		Enabled:  true,
		BotToken: secretToken,
		ChatIDs:  []string{"-1001234567890"},
		Timeout:  10 * time.Second,
	}

	alerter, mockClient := newTestTelegramAlerter(t, config)
	logger := &testLogger{}
	alerter.logger = logger

	// Имитируем ошибку HTTP клиента — Go stdlib включает URL в текст ошибки
	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		// Эмулируем реальную ошибку Go — URL с токеном в тексте
		return nil, errors.New("Post \"https://api.telegram.org/bot" + secretToken + "/sendMessage\": dial tcp: connection refused")
	}

	alert := Alert{
		ErrorCode: "TOKEN_LEAK_TEST",
		Message:   "Test bot token leak",
		Timestamp: time.Now(),
	}

	err := alerter.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() error = %v, want nil", err)
	}

	// Проверяем что в логах нет секретного токена
	for _, msg := range logger.errorMsgs {
		if strings.Contains(msg, secretToken) {
			t.Errorf("error log contains secret bot token: %s", msg)
		}
	}

	// Должен быть лог об ошибке, но с [REDACTED] вместо токена
	if len(logger.errorMsgs) == 0 {
		t.Fatal("expected error log for HTTP failure")
	}
}

// H-2/Review #9: Тест warning log при полном отказе доставки.
func TestTelegramAlerter_TotalFailure_WarningLog(t *testing.T) {
	config := TelegramConfig{
		Enabled:  true,
		BotToken: "123456:ABC-DEF-TEST-TOKEN",
		ChatIDs:  []string{"-1001111111111", "-1002222222222"},
		Timeout:  10 * time.Second,
	}

	alerter, mockClient := newTestTelegramAlerter(t, config)
	logger := &testLogger{}
	alerter.logger = logger

	// Все чаты возвращают ошибку
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
		t.Error("expected warning log when all chats failed delivery")
	}
}

func TestTelegramAlerter_APIError(t *testing.T) {
	config := TelegramConfig{
		Enabled:  true,
		BotToken: "123456:ABC-DEF-TEST-TOKEN",
		ChatIDs:  []string{"-1001234567890"},
		Timeout:  10 * time.Second,
	}

	alerter, mockClient := newTestTelegramAlerter(t, config)

	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		return mockHTTPResponse(400, telegramResponse{
			OK:          false,
			ErrorCode:   400,
			Description: "Bad Request: chat not found",
		}), nil
	}

	alert := Alert{
		ErrorCode: "API_ERROR_TEST",
		Severity:  SeverityCritical,
		Command:   "test-command",
		Message:   "Test message",
		Timestamp: time.Now(),
	}

	// Send() должен вернуть nil (ошибки логируются, но не возвращаются)
	err := alerter.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() should return nil on API error, got %v", err)
	}
}

func TestTelegramAlerter_HTTPError(t *testing.T) {
	config := TelegramConfig{
		Enabled:  true,
		BotToken: "123456:ABC-DEF-TEST-TOKEN",
		ChatIDs:  []string{"-1001234567890"},
		Timeout:  10 * time.Second,
	}

	alerter, mockClient := newTestTelegramAlerter(t, config)

	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("network error")
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

func TestTelegramAlerter_RateLimited(t *testing.T) {
	config := TelegramConfig{
		Enabled:  true,
		BotToken: "123456:ABC-DEF-TEST-TOKEN",
		ChatIDs:  []string{"-1001234567890"},
		Timeout:  10 * time.Second,
	}

	rateLimiter := NewRateLimiter(5 * time.Minute)

	alerter, err := NewTelegramAlerter(config, rateLimiter, &testLogger{})
	if err != nil {
		t.Fatalf("failed to create TelegramAlerter: %v", err)
	}

	mockClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return mockHTTPResponse(200, telegramResponse{OK: true}), nil
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

func TestTelegramAlerter_MessageFormatting(t *testing.T) {
	config := TelegramConfig{
		Enabled:  true,
		BotToken: "123456:ABC-DEF-TEST-TOKEN",
		ChatIDs:  []string{"-1001234567890"},
		Timeout:  10 * time.Second,
	}

	alerter, mockClient := newTestTelegramAlerter(t, config)

	var capturedBody telegramRequest
	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		_ = json.NewDecoder(req.Body).Decode(&capturedBody)
		return mockHTTPResponse(200, telegramResponse{OK: true}), nil
	}

	alert := Alert{
		ErrorCode: "FORMAT_TEST",
		Severity:  SeverityCritical,
		Command:   "db-restore",
		Message:   "Database restore failed with *special* _chars_",
		Infobase:  "Test_DB",
		TraceID:   "trace-abc-123",
		Timestamp: time.Date(2026, 2, 5, 10, 30, 0, 0, time.UTC),
	}

	err := alerter.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	// Проверяем что сообщение содержит ключевые элементы
	text := capturedBody.Text
	if !strings.Contains(text, "🚨 *apk-ci Alert*") {
		t.Error("message should contain alert header")
	}
	if !strings.Contains(text, "`FORMAT\\_TEST`") {
		t.Errorf("message should contain escaped error code, got: %s", text)
	}
	if !strings.Contains(text, "db-restore") {
		t.Error("message should contain command")
	}
	if !strings.Contains(text, "Test\\_DB") {
		t.Errorf("message should contain escaped infobase, got: %s", text)
	}
	if !strings.Contains(text, "`trace-abc-123`") {
		t.Error("message should contain trace_id")
	}
	// Проверяем экранирование специальных символов
	if !strings.Contains(text, "\\*special\\*") {
		t.Errorf("message should have escaped asterisks, got: %s", text)
	}
	if !strings.Contains(text, "\\_chars\\_") {
		t.Errorf("message should have escaped underscores, got: %s", text)
	}
}

func TestTelegramConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  TelegramConfig
		wantErr error
	}{
		{
			name: "disabled - no validation",
			config: TelegramConfig{
				Enabled: false,
			},
			wantErr: nil,
		},
		{
			name: "enabled - missing bot_token",
			config: TelegramConfig{
				Enabled:  true,
				BotToken: "",
				ChatIDs:  []string{"123"},
			},
			wantErr: ErrTelegramBotTokenRequired,
		},
		{
			name: "enabled - empty chat_ids", // L3 fix: точное название
			config: TelegramConfig{
				Enabled:  true,
				BotToken: "123:ABC",
				ChatIDs:  []string{},
			},
			wantErr: ErrTelegramChatIDRequired,
		},
		{
			name: "enabled - valid config",
			config: TelegramConfig{
				Enabled:  true,
				BotToken: "123:ABC",
				ChatIDs:  []string{"123456"},
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

func TestEscapeMarkdown(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"*bold*", "\\*bold\\*"},
		{"_italic_", "\\_italic\\_"},
		{"`code`", "\\`code\\`"},
		{"[link]", "\\[link\\]"},
		{"*_`[]combined", "\\*\\_\\`\\[\\]combined"},
		{"no special chars", "no special chars"},
		{"[click](http://evil)", "\\[click\\]\\(http://evil\\)"},    // Полное экранирование ссылки
		{"func(arg)", "func\\(arg\\)"},                              // Скобки экранируются
		{"mixed *bold* and (parens)", "mixed \\*bold\\* and \\(parens\\)"},
		{`path\to\file`, `path\\to\\file`},                          // M-4/Review #8: backslash экранируется
		{`\_already_escaped`, `\\\_already\_escaped`},               // Backslash + underscore
		{"> quoted text", "\\> quoted text"},                         // M-1/Review #10: > экранируется
		{"line1\n> line2", "line1\n\\> line2"},                       // > внутри multiline
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeMarkdown(tt.input)
			if got != tt.expected {
				t.Errorf("escapeMarkdown(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// H1 fix: Тест проверяет что custom timeout применяется к HTTP клиенту.
func TestTelegramAlerter_CustomTimeout(t *testing.T) {
	customTimeout := 5 * time.Second

	config := TelegramConfig{
		Enabled:  true,
		BotToken: "123456:ABC-DEF-TEST-TOKEN",
		ChatIDs:  []string{"-1001234567890"},
		Timeout:  customTimeout,
	}

	alerter, err := NewTelegramAlerter(config, nil, &testLogger{})
	if err != nil {
		t.Fatalf("failed to create TelegramAlerter: %v", err)
	}

	// Проверяем что timeout был применён к internal HTTP client
	// Используем reflection для доступа к httpClient timeout
	httpClient, ok := alerter.httpClient.(*http.Client)
	if !ok {
		t.Skip("httpClient is mocked, skipping timeout check")
	}

	if httpClient.Timeout != customTimeout {
		t.Errorf("HTTP client timeout = %v, want %v", httpClient.Timeout, customTimeout)
	}
}

// H1 fix: Тест проверяет что default timeout (10s) применяется когда Timeout=0.
func TestTelegramAlerter_DefaultTimeout(t *testing.T) {
	config := TelegramConfig{
		Enabled:  true,
		BotToken: "123456:ABC-DEF-TEST-TOKEN",
		ChatIDs:  []string{"-1001234567890"},
		Timeout:  0, // zero value — должен использоваться default
	}

	alerter, err := NewTelegramAlerter(config, nil, &testLogger{})
	if err != nil {
		t.Fatalf("failed to create TelegramAlerter: %v", err)
	}

	httpClient, ok := alerter.httpClient.(*http.Client)
	if !ok {
		t.Skip("httpClient is mocked, skipping timeout check")
	}

	if httpClient.Timeout != DefaultTelegramTimeout {
		t.Errorf("HTTP client timeout = %v, want default %v", httpClient.Timeout, DefaultTelegramTimeout)
	}
}

// M1 fix: Тест проверяет что отменённый контекст прерывает отправку.
func TestTelegramAlerter_ContextCanceled(t *testing.T) {
	ctx := context.Background()
	config := TelegramConfig{
		Enabled:  true,
		BotToken: "123456:ABC-DEF-TEST-TOKEN",
		ChatIDs:  []string{"-1001234567890", "987654321", "111222333"}, // 3 чата
		Timeout:  10 * time.Second,
	}

	alerter, mockClient := newTestTelegramAlerter(t, config)
	logger := &testLogger{}
	alerter.logger = logger

	requestCount := 0
	mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
		requestCount++
		return mockHTTPResponse(200, telegramResponse{OK: true}), nil
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
