package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// isCloneSuccessful проверяет успешность клонирования по выводу git clone
// Поддерживает различные локали
func isCloneSuccessful(output string) bool {
	// Список сообщений о успешном клонировании в разных локалях
	successMessages := []string{
		"Cloning into",   // English
		"Клонирование в", // Russian
		"Clonage dans",   // French
		"Klonen nach",    // German
		"Clonando en",    // Spanish
	}

	for _, msg := range successMessages {
		if strings.Contains(output, msg) {
			return true
		}
	}

	// Дополнительная проверка: если есть процент загрузки, то клонирование точно идет
	if strings.Contains(output, "Updating files:") ||
		strings.Contains(output, "готово") ||
		strings.Contains(output, "done") {
		return true
	}

	return false
}

// Git представляет структуру для работы с Git репозиторием.
// Содержит всю необходимую информацию для выполнения операций
// с удаленным и локальным репозиторием.
type Git struct {
	// RepURL - URL удаленного репозитория
	RepURL string
	// RepPath - путь к локальному репозиторию
	RepPath string
	// Branch - имя ветки для работы
	Branch string
	// CommitSHA1 - SHA1 хеш коммита для сброса
	CommitSHA1 string
	// WorkDir - рабочая директория для выполнения команд
	WorkDir string
	// Token - токен авторизации для доступа к репозиторию
	Token string
	// Timeout - таймаут для Git операций
	Timeout time.Duration
}

// Configs содержит список конфигураций Git.
// Используется для группировки и управления настройками Git.
type Configs struct {
	// Config - массив конфигурационных параметров Git
	Config []Config
}

// Config представляет одну конфигурацию Git.
// Содержит пару ключ-значение для настройки Git.
type Config struct {
	// Name - имя конфигурационного параметра
	Name string
	// Value - значение конфигурационного параметра
	Value string
}

// runGitCommand выполняет команду Git с указанными аргументами.
// Запускает git команду и возвращает её вывод или ошибку.
// Параметры:
//   - args: аргументы команды git
//
// Возвращает:
//   - error: ошибка выполнения или nil при успехе
func runGitCommand(ctx context.Context, timeout time.Duration, args ...string) error {
	// Создаем контекст с переданным таймаутом для команды git
	if timeout == 0 {
		timeout = 30 * time.Minute // Значение по умолчанию
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// #nosec G204 - GitCommand является константой, args формируется из проверенных значений
	cmd := exec.CommandContext(ctx, GitCommand, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("git %s: command timeout", strings.Join(args, " "))
		}
		return fmt.Errorf("git %s: %s", strings.Join(args, " "), stderr.String())
	}
	return nil
}
