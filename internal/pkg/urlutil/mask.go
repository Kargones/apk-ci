// Package urlutil предоставляет утилиты для безопасной работы с URL.
package urlutil

import "net/url"

// MaskURL маскирует URL для безопасного логирования.
// Скрывает path и query параметры, которые могут содержать токены или credentials.
// Пример: "https://hooks.slack.com/services/XXX/YYY/ZZZ" → "https://hooks.slack.com/***"
func MaskURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "***invalid-url***"
	}
	// Показываем только scheme и host
	return u.Scheme + "://" + u.Host + "/***"
}
