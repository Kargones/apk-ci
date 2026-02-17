package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/pkg/logging"
)

// TelegramAPIBaseURL — базовый URL Telegram Bot API.
const TelegramAPIBaseURL = "https://api.telegram.org/bot"

// TelegramParseMode — режим парсинга сообщений (Markdown или MarkdownV2).
// TODO: мигрировать на "MarkdownV2" для расширенного форматирования.
// Markdown v1 deprecated в Telegram API, но v2 требует другого escaping.
const TelegramParseMode = "Markdown"

// maxTelegramResponseSize — максимальный размер тела ответа Telegram API (1 KB).
// L-5/Review #15: 1024 байта достаточно для типичного Telegram API response (JSON ~200-500 байт).
// Ограничение защищает от OOM при аномально большом ответе.
const maxTelegramResponseSize = 1024

// HTTPClient определяет интерфейс HTTP клиента для тестирования.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// TelegramAlerter реализует Alerter для отправки в Telegram.
type TelegramAlerter struct {
	config      TelegramConfig
	rateLimiter *RateLimiter
	logger      logging.Logger
	httpClient  HTTPClient
}

// NewTelegramAlerter создаёт TelegramAlerter с указанной конфигурацией.
// Параметры:
//   - config: конфигурация telegram канала
//   - rateLimiter: rate limiter для ограничения частоты алертов
//   - logger: логгер для записи ошибок и информационных сообщений
func NewTelegramAlerter(config TelegramConfig, rateLimiter *RateLimiter, logger logging.Logger) (*TelegramAlerter, error) {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = DefaultTelegramTimeout
	}

	return &TelegramAlerter{
		config:      config,
		rateLimiter: rateLimiter,
		logger:      logger,
		httpClient:  &http.Client{Timeout: timeout},
	}, nil
}

// SetHTTPClient устанавливает кастомный HTTPClient (для тестирования).
func (t *TelegramAlerter) SetHTTPClient(client HTTPClient) {
	t.httpClient = client
}

// Send отправляет алерт в Telegram.
// Применяет rate limiting по ErrorCode.
// При ошибке Telegram API — логирует ошибку и возвращает nil (приложение продолжает работу).
func (t *TelegramAlerter) Send(ctx context.Context, alert Alert) error {
	// Проверяем rate limiting.
	// Примечание: при создании через factory rateLimiter=nil (rate limiting на уровне
	// MultiChannelAlerter). Guard оставлен для прямого использования TelegramAlerter.
	if t.rateLimiter != nil && !t.rateLimiter.Allow(alert.ErrorCode) {
		t.logger.Debug("алерт подавлен rate limiter",
			"error_code", alert.ErrorCode,
			"channel", ChannelTelegram,
		)
		return nil // Rate limited — не ошибка
	}

	// Форматируем сообщение
	message := t.formatMessage(alert)

	// Отправляем во все чаты
	successCount := 0
	for i, chatID := range t.config.ChatIDs {
		// Проверяем контекст перед каждой отправкой (M1 fix)
		select {
		case <-ctx.Done():
			t.logger.Debug("отправка telegram алерта отменена",
				"error_code", alert.ErrorCode,
				"remaining_chats", len(t.config.ChatIDs)-i,
			)
			return nil // Отмена — не ошибка, логируем и выходим
		default:
		}

		if err := t.sendToChat(ctx, chatID, message); err != nil {
			t.logger.Error("ошибка отправки telegram алерта",
				"error", err.Error(),
				"chat_id", chatID,
				"error_code", alert.ErrorCode,
			)
			// Продолжаем отправку в другие чаты
		} else {
			successCount++
		}
	}

	if successCount > 0 {
		t.logger.Info("telegram алерт отправлен",
			"error_code", alert.ErrorCode,
			"severity", alert.Severity.String(),
			"chats_success", successCount,
			"chats_total", len(t.config.ChatIDs),
		)
	} else if len(t.config.ChatIDs) > 0 {
		// H-2/Review #9: предупреждение при полном отказе доставки во все чаты.
		t.logger.Warn("telegram алерт не доставлен ни в один чат",
			"error_code", alert.ErrorCode,
			"chats_total", len(t.config.ChatIDs),
		)
	}

	return nil
}

// formatMessage форматирует алерт в Markdown для Telegram.
func (t *TelegramAlerter) formatMessage(alert Alert) string {
	var sb strings.Builder

	sb.WriteString("🚨 *apk-ci Alert*\n\n")

	sb.WriteString("*Error:* `")
	sb.WriteString(escapeMarkdown(alert.ErrorCode))
	sb.WriteString("`\n")

	sb.WriteString("*Severity:* ")
	sb.WriteString(escapeMarkdown(alert.Severity.String()))
	sb.WriteString("\n")

	sb.WriteString("*Command:* ")
	sb.WriteString(escapeMarkdown(alert.Command))
	sb.WriteString("\n")

	if alert.Infobase != "" {
		sb.WriteString("*Infobase:* ")
		sb.WriteString(escapeMarkdown(alert.Infobase))
		sb.WriteString("\n")
	}

	sb.WriteString("\n*Message:*\n")
	sb.WriteString(escapeMarkdown(alert.Message))
	sb.WriteString("\n\n")

	sb.WriteString("_Trace ID:_ `")
	sb.WriteString(escapeMarkdown(alert.TraceID))
	sb.WriteString("`\n")

	sb.WriteString("_Time:_ ")
	sb.WriteString(escapeMarkdown(alert.Timestamp.Format(time.RFC3339)))

	return sb.String()
}

// markdownReplacer — переиспользуемый replacer для экранирования символов Markdown v1.
// Создаётся один раз на уровне пакета для избежания аллокаций при каждом вызове.
// Backslash экранируется ПЕРВЫМ, чтобы не удваивать экранирование остальных символов.
// M-1/Review #10: добавлен ">" для защиты от цитирования в Markdown v1.
// markdownReplacer is effectively constant (initialized once, never reassigned).
var markdownReplacer = strings.NewReplacer(
	`\`, `\\`,
	"_", "\\_",
	"*", "\\*",
	"`", "\\`",
	"[", "\\[",
	"]", "\\]",
	"(", "\\(",
	")", "\\)",
	">", "\\>",
)

// escapeMarkdown экранирует специальные символы Markdown v1 для Telegram.
// Экранируются все символы, которые могут сломать форматирование Markdown v1:
// _ * ` [ ] ( )
// Скобки "()" экранируются для защиты от инъекции inline ссылок [text](url).
//
// TODO: при миграции на MarkdownV2 необходимо экранировать дополнительные символы:
// ! # - . = > { } | ~
// См. https://core.telegram.org/bots/api#markdownv2-style
func escapeMarkdown(s string) string {
	return markdownReplacer.Replace(s)
}

// telegramRequest представляет запрос к Telegram API sendMessage.
type telegramRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// telegramResponse представляет ответ Telegram API.
type telegramResponse struct {
	OK          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code,omitempty"`
	Description string `json:"description,omitempty"`
}

// sendToChat отправляет сообщение в конкретный чат.
func (t *TelegramAlerter) sendToChat(ctx context.Context, chatID, message string) error {
	url := fmt.Sprintf("%s%s/sendMessage", TelegramAPIBaseURL, t.config.BotToken)

	reqBody := telegramRequest{
		ChatID:    chatID,
		Text:      message,
		ParseMode: TelegramParseMode,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		// Санитизируем ошибку: Go stdlib включает URL (с BotToken) в текст ошибки
		sanitizedErr := strings.ReplaceAll(err.Error(), t.config.BotToken, "[REDACTED]")
		return fmt.Errorf("HTTP request failed: %s", sanitizedErr)
	}
	defer resp.Body.Close()

	// Ограничиваем размер ответа для защиты от DoS (аналогично webhook.go)
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxTelegramResponseSize))
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var telegramResp telegramResponse
	if err := json.Unmarshal(body, &telegramResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !telegramResp.OK {
		return fmt.Errorf("Telegram API error %d: %s", telegramResp.ErrorCode, telegramResp.Description)
	}

	return nil
}
