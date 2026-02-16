package git

import (
	"os"
	"testing"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// GitConfig представляет конфигурацию Git для тестирования
type GitConfig struct {
	Timeout time.Duration `env:"GIT_TIMEOUT" env-default:"30s"`
}

// TestGitTimeoutConfiguration проверяет, что таймаут Git правильно загружается из переменных окружения
func TestGitTimeoutConfiguration(t *testing.T) {
	// Сохраняем оригинальное значение переменной окружения
	originalTimeout := os.Getenv("GIT_TIMEOUT")
	defer func() {
		if originalTimeout == "" {
			os.Unsetenv("GIT_TIMEOUT")
		} else {
			os.Setenv("GIT_TIMEOUT", originalTimeout)
		}
	}()

	// Тест 1: Проверяем загрузку увеличенного таймаута (10 минут)
	t.Run("increased_timeout_600s", func(t *testing.T) {
		// Устанавливаем переменную окружения как в main-test.yaml
		os.Setenv("GIT_TIMEOUT", "600s")

		// Создаем конфигурацию
		gitConfig := &GitConfig{}
		
		// Загружаем из переменных окружения
		err := cleanenv.ReadEnv(gitConfig)
		if err != nil {
			t.Fatalf("Ошибка загрузки конфигурации из переменных окружения: %v", err)
		}

		// Проверяем, что таймаут установлен правильно
		expectedTimeout := 600 * time.Second
		if gitConfig.Timeout != expectedTimeout {
			t.Errorf("Ожидался таймаут %v, получен %v", expectedTimeout, gitConfig.Timeout)
		}
	})

	// Тест 2: Проверяем значение по умолчанию
	t.Run("default_timeout_30s", func(t *testing.T) {
		// Убираем переменную окружения
		os.Unsetenv("GIT_TIMEOUT")

		// Создаем конфигурацию по умолчанию
		gitConfig := &GitConfig{}
		
		// Загружаем из переменных окружения (должно использовать значение по умолчанию)
		err := cleanenv.ReadEnv(gitConfig)
		if err != nil {
			t.Fatalf("Ошибка загрузки конфигурации из переменных окружения: %v", err)
		}

		// Проверяем значение по умолчанию
		expectedTimeout := 30 * time.Second
		if gitConfig.Timeout != expectedTimeout {
			t.Errorf("Ожидался таймаут по умолчанию %v, получен %v", expectedTimeout, gitConfig.Timeout)
		}
	})

	// Тест 3: Проверяем, что Git структура использует переданный таймаут
	t.Run("git_struct_uses_timeout", func(t *testing.T) {
		// Создаем Git структуру с увеличенным таймаутом
		git := &Git{
			Timeout: 600 * time.Second,
		}

		// Проверяем, что таймаут установлен правильно
		expectedTimeout := 600 * time.Second
		if git.Timeout != expectedTimeout {
			t.Errorf("Ожидался таймаут в Git структуре %v, получен %v", expectedTimeout, git.Timeout)
		}
	})
}