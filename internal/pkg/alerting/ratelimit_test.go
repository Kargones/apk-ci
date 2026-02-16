package alerting

import (
	"testing"
	"time"
)

func TestRateLimiter_Allow_FirstCall(t *testing.T) {
	limiter := NewRateLimiter(5 * time.Minute)

	// Первый вызов должен пройти
	if !limiter.Allow("TEST_ERROR") {
		t.Error("Allow() = false, want true for first call")
	}
}

func TestRateLimiter_Allow_RateLimited(t *testing.T) {
	limiter := NewRateLimiter(5 * time.Minute)

	// Первый вызов
	if !limiter.Allow("TEST_ERROR") {
		t.Fatal("Allow() first call = false, want true")
	}

	// Повторный вызов в пределах window должен быть заблокирован
	if limiter.Allow("TEST_ERROR") {
		t.Error("Allow() = true, want false for repeated call within window")
	}
}

func TestRateLimiter_Allow_DifferentErrorCodes(t *testing.T) {
	limiter := NewRateLimiter(5 * time.Minute)

	// Первый код ошибки
	if !limiter.Allow("ERROR_A") {
		t.Error("Allow(ERROR_A) = false, want true")
	}

	// Другой код ошибки должен проходить независимо
	if !limiter.Allow("ERROR_B") {
		t.Error("Allow(ERROR_B) = false, want true")
	}

	// Первый код всё ещё заблокирован
	if limiter.Allow("ERROR_A") {
		t.Error("Allow(ERROR_A) repeated = true, want false")
	}
}

func TestRateLimiter_Allow_AfterWindowExpires(t *testing.T) {
	limiter := NewRateLimiter(100 * time.Millisecond)

	// Первый вызов
	if !limiter.Allow("TEST_ERROR") {
		t.Fatal("Allow() first call = false, want true")
	}

	// Используем mock времени для ускорения теста
	baseTime := time.Now()
	limiter.SetNowFunc(func() time.Time {
		return baseTime.Add(200 * time.Millisecond) // После истечения window
	})

	// После истечения window должен проходить
	if !limiter.Allow("TEST_ERROR") {
		t.Error("Allow() after window = false, want true")
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	limiter := NewRateLimiter(5 * time.Minute)

	// Первый вызов
	if !limiter.Allow("TEST_ERROR") {
		t.Fatal("Allow() first call = false, want true")
	}

	// Заблокировано
	if limiter.Allow("TEST_ERROR") {
		t.Fatal("Allow() repeated = true, want false")
	}

	// Reset
	limiter.Reset("TEST_ERROR")

	// Снова должен проходить
	if !limiter.Allow("TEST_ERROR") {
		t.Error("Allow() after Reset = false, want true")
	}
}

func TestRateLimiter_ResetAll(t *testing.T) {
	limiter := NewRateLimiter(5 * time.Minute)

	// Заполняем несколько кодов
	limiter.Allow("ERROR_A")
	limiter.Allow("ERROR_B")
	limiter.Allow("ERROR_C")

	// Все заблокированы
	if limiter.Allow("ERROR_A") || limiter.Allow("ERROR_B") || limiter.Allow("ERROR_C") {
		t.Fatal("Expected all to be rate limited before ResetAll")
	}

	// ResetAll
	limiter.ResetAll()

	// Все снова должны проходить
	if !limiter.Allow("ERROR_A") {
		t.Error("Allow(ERROR_A) after ResetAll = false, want true")
	}
	if !limiter.Allow("ERROR_B") {
		t.Error("Allow(ERROR_B) after ResetAll = false, want true")
	}
	if !limiter.Allow("ERROR_C") {
		t.Error("Allow(ERROR_C) after ResetAll = false, want true")
	}
}

func TestRateLimiter_CleanupExpiredEntries(t *testing.T) {
	limiter := NewRateLimiter(100 * time.Millisecond)

	baseTime := time.Now()
	limiter.SetNowFunc(func() time.Time { return baseTime })

	// Заполняем больше cleanupThreshold записей
	for i := 0; i < cleanupThreshold+50; i++ {
		limiter.Allow("ERROR_" + time.Now().String() + string(rune(i)))
	}

	// Сдвигаем время за пределы window
	limiter.SetNowFunc(func() time.Time {
		return baseTime.Add(200 * time.Millisecond)
	})

	// Следующий вызов Allow() должен запустить cleanup
	limiter.Allow("TRIGGER_CLEANUP")

	limiter.mu.Lock()
	mapLen := len(limiter.sent)
	limiter.mu.Unlock()

	// После cleanup должна остаться только запись TRIGGER_CLEANUP
	// (все старые записи с expired window удалены)
	if mapLen > cleanupThreshold {
		t.Errorf("После cleanup ожидалось <= %d записей, получено %d", cleanupThreshold, mapLen)
	}
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewRateLimiter(5 * time.Minute)
	done := make(chan bool)

	// Запускаем несколько горутин
	for i := 0; i < 100; i++ {
		go func(errorCode string) {
			limiter.Allow(errorCode)
			done <- true
		}("ERROR_" + string(rune('A'+i%26)))
	}

	// Ждём завершения всех горутин
	for i := 0; i < 100; i++ {
		<-done
	}
	// Тест успешен если не было паники
}
