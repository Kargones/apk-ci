package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Kargones/apk-ci/internal/pkg/logging"
	"github.com/Kargones/apk-ci/internal/pkg/urlutil"
)

// WebhookAlerter реализует Alerter для отправки через HTTP webhook.
type WebhookAlerter struct {
	config      WebhookConfig
	rateLimiter *RateLimiter
	logger      logging.Logger
	httpClient  HTTPClient
	hostname    string // H-3/Review #9: кэшированный hostname
}

// WebhookPayload представляет JSON payload для webhook.
type WebhookPayload struct {
	ErrorCode string    `json:"error_code"`
	Message   string    `json:"message"`
	TraceID   string    `json:"trace_id"`
	Timestamp time.Time `json:"timestamp"`
	Command   string    `json:"command"`
	Infobase  string    `json:"infobase,omitempty"`
	Severity  string    `json:"severity"`
	Source    string    `json:"source"`
	Hostname  string    `json:"hostname,omitempty"` // H-3/Review #8: идентификация инстанса
}

// httpError представляет HTTP ошибку (не network).
type httpError struct {
	StatusCode int
	Body       string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Body)
}

// NewWebhookAlerter создаёт WebhookAlerter с указанной конфигурацией.
// Параметры:
//   - config: конфигурация webhook канала
//   - rateLimiter: rate limiter для ограничения частоты алертов
//   - logger: логгер для записи ошибок и информационных сообщений
func NewWebhookAlerter(config WebhookConfig, rateLimiter *RateLimiter, logger logging.Logger) (*WebhookAlerter, error) {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = DefaultWebhookTimeout
	}

	// H-3/Review #9: кэшируем hostname при создании, а не при каждом Send().
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	return &WebhookAlerter{
		config:      config,
		rateLimiter: rateLimiter,
		logger:      logger,
		httpClient:  &http.Client{Timeout: timeout},
		hostname:    hostname,
	}, nil
}

// SetHTTPClient устанавливает кастомный HTTPClient (для тестирования).
func (w *WebhookAlerter) SetHTTPClient(client HTTPClient) {
	w.httpClient = client
}

// Send отправляет алерт через webhook.
// Применяет rate limiting по ErrorCode.
// При ошибке HTTP — логирует ошибку и возвращает nil (приложение продолжает работу).
func (w *WebhookAlerter) Send(ctx context.Context, alert Alert) error {
	// Проверяем rate limiting.
	// Примечание: при создании через factory rateLimiter=nil (rate limiting на уровне
	// MultiChannelAlerter). Guard оставлен для прямого использования WebhookAlerter.
	if w.rateLimiter != nil && !w.rateLimiter.Allow(alert.ErrorCode) {
		w.logger.Debug("алерт подавлен rate limiter",
			"error_code", alert.ErrorCode,
			"channel", ChannelWebhook,
		)
		return nil // Rate limited — не ошибка
	}

	// Создаём payload
	payload := w.createPayload(alert)

	// Отправляем на все URL
	successCount := 0
	for i, url := range w.config.URLs {
		// Проверяем контекст перед каждой отправкой
		select {
		case <-ctx.Done():
			w.logger.Debug("отправка webhook алерта отменена",
				"error_code", alert.ErrorCode,
				"remaining_urls", len(w.config.URLs)-i,
			)
			return nil // Отмена — не ошибка
		default:
		}

		if err := w.sendWithRetry(ctx, url, payload); err != nil {
			w.logger.Error("ошибка отправки webhook алерта",
				"error", err.Error(),
				"url", urlutil.MaskURL(url),
				"error_code", alert.ErrorCode,
			)
			// Продолжаем отправку на другие URL
		} else {
			successCount++
		}
	}

	if successCount > 0 {
		w.logger.Info("webhook алерт отправлен",
			"error_code", alert.ErrorCode,
			"severity", alert.Severity.String(),
			"urls_success", successCount,
			"urls_total", len(w.config.URLs),
		)
	} else if len(w.config.URLs) > 0 {
		// H-2/Review #9: предупреждение при полном отказе доставки на все URL.
		w.logger.Warn("webhook алерт не доставлен ни на один URL",
			"error_code", alert.ErrorCode,
			"urls_total", len(w.config.URLs),
		)
	}

	return nil
}

// createPayload создаёт WebhookPayload из Alert.
func (w *WebhookAlerter) createPayload(alert Alert) WebhookPayload {
	return WebhookPayload{
		ErrorCode: alert.ErrorCode,
		Message:   alert.Message,
		TraceID:   alert.TraceID,
		Timestamp: alert.Timestamp,
		Command:   alert.Command,
		Infobase:  alert.Infobase,
		Severity:  alert.Severity.String(),
		Source:    "apk-ci",
		Hostname:  w.hostname,
	}
}

// sendWithRetry отправляет запрос с retry логикой.
// Retry происходит для network ошибок и retryable HTTP ошибок (5xx).
// HTTP 4xx (клиентские ошибки) не ретраятся — они указывают на проблему конфигурации.
func (w *WebhookAlerter) sendWithRetry(ctx context.Context, url string, payload WebhookPayload) error {
	maxRetries := w.config.MaxRetries

	var lastErr error
	backoff := 1 * time.Second
	// L-6/Review #15: maxBackoff=4s достаточен для CLI (короткоживущий процесс).
	// Для long-running daemon рекомендуется 30-60s.
	const maxBackoff = 4 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s (capped)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}

			w.logger.Debug("webhook retry",
				"attempt", attempt,
				"max_retries", maxRetries,
				"error", lastErr.Error(),
				"url", urlutil.MaskURL(url),
			)
		}

		lastErr = w.sendRequest(ctx, url, payload)
		if lastErr == nil {
			return nil // Success
		}

		// HTTP 4xx — клиентская ошибка, retry бесполезен
		if isClientHTTPError(lastErr) {
			return lastErr
		}
		// Network errors и 5xx (502, 503, 504) — retry
	}

	return fmt.Errorf("all %d attempts failed: %w", maxRetries+1, lastErr)
}

// sendRequest отправляет HTTP POST запрос с payload.
func (w *WebhookAlerter) sendRequest(ctx context.Context, url string, payload WebhookPayload) error {
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "apk-ci/1.0")

	// Добавляем custom headers
	for key, value := range w.config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err // Network error — retry
	}
	defer resp.Body.Close()

	// 2xx — успех. Дренируем body для корректного переиспользования HTTP keep-alive соединений.
	// H-2/Review #10: ограничиваем размер drain для защиты от OOM при аномально большом ответе.
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBodySize)) //nolint:errcheck // best-effort drain
		return nil
	}

	// Читаем body для диагностики (ограничиваем размер для безопасности)
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	return &httpError{
		StatusCode: resp.StatusCode,
		Body:       string(body),
	}
}

// isClientHTTPError проверяет, является ли ошибка клиентской HTTP ошибкой (4xx).
// 5xx ошибки считаются retryable и не являются "клиентскими".
func isClientHTTPError(err error) bool {
	var httpErr *httpError
	if !errors.As(err, &httpErr) {
		return false
	}
	return httpErr.StatusCode >= 400 && httpErr.StatusCode < 500
}

// maxResponseBodySize — максимальный размер тела HTTP ответа для диагностики (1 KB).
const maxResponseBodySize = 1024

