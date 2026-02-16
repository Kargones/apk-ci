package alerting

import (
	"sync"
	"time"
)

// RateLimiter контролирует частоту отправки алертов.
// Использует in-memory хранение (error_code → last_sent_time).
// Thread-safe через sync.Mutex.
//
// ВАЖНО: Rate limiting работает только В ПРЕДЕЛАХ ОДНОГО ЗАПУСКА процесса.
// Для CLI-приложения (короткоживущий процесс) это означает, что rate limiting
// НЕ работает между запусками. Каждый новый запуск CLI получает пустой RateLimiter.
//
// Для полноценного rate limiting между запусками требуется внешнее хранилище
// (файл, Redis, etc.), что выходит за рамки текущей реализации.
//
// Текущее поведение приемлемо для:
// - Долгоживущих процессов (daemon mode)
// - Нескольких ошибок в одном запуске CLI
//
// M-4/Review Epic-6: Для CLI rate limiter фактически бесполезен —
// каждый запуск получает пустой map, первый Send() всегда проходит.
//
// Для защиты от email storm при частых запусках рекомендуется:
// - Настроить rate limiting на уровне SMTP сервера
// - Использовать внешний alerting сервис (PagerDuty, OpsGenie) с built-in deduplication
type RateLimiter struct {
	mu     sync.Mutex
	window time.Duration
	sent   map[string]time.Time
	// now используется для тестирования (позволяет mock времени)
	now func() time.Time
}

// NewRateLimiter создаёт RateLimiter с указанным интервалом.
// Window определяет минимальный интервал между алертами одного типа.
//
// Пример:
//
//	limiter := NewRateLimiter(5 * time.Minute)
//	if limiter.Allow("DB_CONN_FAIL") {
//	    // Можно отправить алерт
//	}
func NewRateLimiter(window time.Duration) *RateLimiter {
	return &RateLimiter{
		window: window,
		sent:   make(map[string]time.Time),
		now:    time.Now,
	}
}

// cleanupThreshold — порог количества записей, после которого запускается очистка.
const cleanupThreshold = 100

// Allow проверяет можно ли отправить алерт с данным errorCode.
// Возвращает true если алерт можно отправить (прошло достаточно времени
// с последней отправки или алерт с таким кодом ещё не отправлялся).
//
// При возврате true — помечает errorCode как отправленный с текущим временем.
// Это атомарная операция: проверка и обновление выполняются под mutex.
//
// Периодически очищает expired entries для предотвращения утечки памяти
// при большом количестве уникальных error codes.
func (r *RateLimiter) Allow(errorCode string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()

	// Очищаем expired entries при превышении порога
	if len(r.sent) > cleanupThreshold {
		r.cleanupExpiredLocked(now)
	}

	if lastSent, ok := r.sent[errorCode]; ok {
		if now.Sub(lastSent) < r.window {
			return false // rate limited
		}
	}
	r.sent[errorCode] = now
	return true
}

// cleanupExpiredLocked удаляет записи с истёкшим window.
// Вызывается под mutex.
func (r *RateLimiter) cleanupExpiredLocked(now time.Time) {
	for code, lastSent := range r.sent {
		if now.Sub(lastSent) >= r.window {
			delete(r.sent, code)
		}
	}
}

// Reset сбрасывает состояние rate limiter для указанного errorCode.
// Используется в основном для тестирования.
func (r *RateLimiter) Reset(errorCode string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sent, errorCode)
}

// ResetAll сбрасывает состояние rate limiter для всех errorCode.
// Используется в основном для тестирования.
func (r *RateLimiter) ResetAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = make(map[string]time.Time)
}

// SetNowFunc устанавливает функцию получения текущего времени.
// Используется для тестирования.
func (r *RateLimiter) SetNowFunc(fn func() time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.now = fn
}
