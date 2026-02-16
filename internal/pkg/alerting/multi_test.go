package alerting

import (
	"context"
	"testing"
	"time"
)

// mockAlerter — mock для Alerter интерфейса в тестах multi-channel.
// Не использует atomic — MultiChannelAlerter.Send() вызывает каналы последовательно.
type mockAlerter struct {
	sendCount int
	lastAlert Alert
	sendError error
}

func (m *mockAlerter) Send(_ context.Context, alert Alert) error {
	m.sendCount++
	m.lastAlert = alert
	return m.sendError
}

func TestMultiChannelAlerter_BothChannels(t *testing.T) {
	emailMock := &mockAlerter{}
	telegramMock := &mockAlerter{}

	channels := map[string]Alerter{
		"email":    emailMock,
		"telegram": telegramMock,
	}
	multi := NewMultiChannelAlerter(channels, nil, nil, &testLogger{})

	alert := Alert{
		ErrorCode: "MULTI_TEST",
		Severity:  SeverityCritical,
		Command:   "test-command",
		Message:   "Test message",
		Timestamp: time.Now(),
	}

	err := multi.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	// Оба канала должны получить alert
	if emailMock.sendCount != 1 {
		t.Errorf("email alerter sendCount = %d, want 1", emailMock.sendCount)
	}
	if telegramMock.sendCount != 1 {
		t.Errorf("telegram alerter sendCount = %d, want 1", telegramMock.sendCount)
	}

	// Проверяем что alert передан корректно
	if emailMock.lastAlert.ErrorCode != alert.ErrorCode {
		t.Errorf("email alerter got wrong error_code: %s", emailMock.lastAlert.ErrorCode)
	}
	if telegramMock.lastAlert.ErrorCode != alert.ErrorCode {
		t.Errorf("telegram alerter got wrong error_code: %s", telegramMock.lastAlert.ErrorCode)
	}
}

func TestMultiChannelAlerter_SingleChannel(t *testing.T) {
	emailMock := &mockAlerter{}

	channels := map[string]Alerter{
		"email": emailMock,
	}
	multi := NewMultiChannelAlerter(channels, nil, nil, &testLogger{})

	alert := Alert{
		ErrorCode: "SINGLE_TEST",
		Timestamp: time.Now(),
	}

	err := multi.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	if emailMock.sendCount != 1 {
		t.Errorf("alerter sendCount = %d, want 1", emailMock.sendCount)
	}
}

func TestMultiChannelAlerter_NoChannels(t *testing.T) {
	channels := map[string]Alerter{}
	multi := NewMultiChannelAlerter(channels, nil, nil, &testLogger{})

	alert := Alert{
		ErrorCode: "NO_CHANNELS_TEST",
		Timestamp: time.Now(),
	}

	// Не должно быть паники при отсутствии каналов
	err := multi.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}
}

// H-2 fix: Тест что rate limiter на уровне MultiChannelAlerter
// подавляет все каналы одновременно (а не только первый по алфавиту).
func TestMultiChannelAlerter_RateLimiterAppliesToAllChannels(t *testing.T) {
	emailMock := &mockAlerter{}
	telegramMock := &mockAlerter{}
	webhookMock := &mockAlerter{}

	channels := map[string]Alerter{
		"email":    emailMock,
		"telegram": telegramMock,
		"webhook":  webhookMock,
	}

	rateLimiter := NewRateLimiter(5 * time.Minute)
	multi := NewMultiChannelAlerter(channels, nil, rateLimiter, &testLogger{})

	alert := Alert{
		ErrorCode: "RATE_LIMIT_MULTI_TEST",
		Severity:  SeverityCritical,
		Command:   "test-command",
		Message:   "Test message",
		Timestamp: time.Now(),
	}

	// Первый вызов — все 3 канала должны получить алерт
	err := multi.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("first Send() error = %v", err)
	}
	if emailMock.sendCount != 1 {
		t.Errorf("email sendCount after 1st send = %d, want 1", emailMock.sendCount)
	}
	if telegramMock.sendCount != 1 {
		t.Errorf("telegram sendCount after 1st send = %d, want 1", telegramMock.sendCount)
	}
	if webhookMock.sendCount != 1 {
		t.Errorf("webhook sendCount after 1st send = %d, want 1", webhookMock.sendCount)
	}

	// Второй вызов с тем же error_code — ВСЕ каналы должны быть подавлены
	err = multi.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("second Send() error = %v", err)
	}
	if emailMock.sendCount != 1 {
		t.Errorf("email sendCount after 2nd send = %d, want 1 (rate limited)", emailMock.sendCount)
	}
	if telegramMock.sendCount != 1 {
		t.Errorf("telegram sendCount after 2nd send = %d, want 1 (rate limited)", telegramMock.sendCount)
	}
	if webhookMock.sendCount != 1 {
		t.Errorf("webhook sendCount after 2nd send = %d, want 1 (rate limited)", webhookMock.sendCount)
	}
}

// M-3/Review #10: Тест что summary debug лог содержит channels_sent/channels_total.
func TestMultiChannelAlerter_SummaryLog(t *testing.T) {
	emailMock := &mockAlerter{}
	telegramMock := &mockAlerter{}

	channels := map[string]Alerter{
		"email":    emailMock,
		"telegram": telegramMock,
	}
	logger := &testLogger{}
	multi := NewMultiChannelAlerter(channels, nil, nil, logger)

	alert := Alert{
		ErrorCode: "SUMMARY_LOG_TEST",
		Severity:  SeverityCritical,
		Command:   "test-command",
		Timestamp: time.Now(),
	}

	_ = multi.Send(context.Background(), alert)

	// Должен быть debug лог с информацией о рассылке
	foundSummaryLog := false
	for _, msg := range logger.debugMsgs {
		if msg == "multi-channel рассылка завершена" {
			foundSummaryLog = true
			break
		}
	}
	if !foundSummaryLog {
		t.Error("expected debug summary log 'multi-channel рассылка завершена'")
	}
}

func TestMultiChannelAlerter_ContinuesOnError(t *testing.T) {
	// Первый канал возвращает ошибку, второй должен всё равно вызваться
	failingMock := &mockAlerter{sendError: ErrSMTPConnection}
	successMock := &mockAlerter{}

	channels := map[string]Alerter{
		"failing": failingMock,
		"success": successMock,
	}
	multi := NewMultiChannelAlerter(channels, nil, nil, &testLogger{})

	alert := Alert{
		ErrorCode: "ERROR_CONTINUE_TEST",
		Timestamp: time.Now(),
	}

	err := multi.Send(context.Background(), alert)
	// MultiChannelAlerter всегда возвращает nil
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	// Оба канала должны быть вызваны
	if failingMock.sendCount != 1 {
		t.Errorf("failing alerter sendCount = %d, want 1", failingMock.sendCount)
	}
	if successMock.sendCount != 1 {
		t.Errorf("success alerter sendCount = %d, want 1", successMock.sendCount)
	}
}
