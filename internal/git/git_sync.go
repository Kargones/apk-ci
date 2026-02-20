package git

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// waitForGitLockRelease ожидает освобождения блокировки Git index.lock
// Вместо принудительного удаления файла блокировки, ожидаем его естественного освобождения
func waitForGitLockRelease(repoPath string, timeout time.Duration) error {
	lockFile := filepath.Join(repoPath, ".git", "index.lock")
	slog.Debug("Начинаем ожидание освобождения файла блокировки Git index",
		slog.String("Путь", lockFile),
		slog.Duration("Максимальное время ожидания", timeout))

	start := time.Now()
	checkCount := 0

	for {
		checkCount++
		if _, err := os.Stat(lockFile); os.IsNotExist(err) {
			elapsed := time.Since(start)
			slog.Debug("Файл блокировки Git index не найден или освобожден",
				slog.String("Путь", lockFile),
				slog.Duration("Время ожидания", elapsed),
				slog.Int("Количество проверок", checkCount))
			return nil
		}

		elapsed := time.Since(start)
		if elapsed > timeout {
			slog.Error("Превышен таймаут ожидания освобождения файла блокировки Git index",
				slog.String("Путь", lockFile),
				slog.Duration("Время ожидания", elapsed),
				slog.Duration("Максимальное время", timeout),
				slog.Int("Количество проверок", checkCount))
			return fmt.Errorf("timeout waiting for git index.lock to be released: %s", lockFile)
		}

		// Логируем прогресс каждые 30 секунд для длительных операций
		if checkCount%30 == 0 {
			slog.Debug("Продолжаем ожидание освобождения файла блокировки Git index",
				slog.String("Путь", lockFile),
				slog.Duration("Прошло времени", elapsed),
				slog.Duration("Осталось времени", timeout-elapsed),
				slog.Int("Количество проверок", checkCount))
		}

		time.Sleep(1 * time.Second)
	}
}

// executeGitCommandWithRetry выполняет Git команду с повторными попытками при ошибке index.lock
func executeGitCommandWithRetry(repoPath string, args []string) error {
	maxRetries := 3
	slog.Debug("Выполняем Git команду с повторными попытками",
		slog.String("Команда", strings.Join(args, " ")),
		slog.String("Путь", repoPath),
		slog.Int("Максимум попыток", maxRetries))

	for attempt := 1; attempt <= maxRetries; attempt++ {
		slog.Debug("Попытка выполнения Git команды",
			slog.Int("Попытка", attempt),
			slog.String("Команда", strings.Join(args, " ")))

		// #nosec G204 - command is "git" (hardcoded), args from programmatic construction
		cmd := exec.Command("git", args...)
		cmd.Dir = repoPath
		output, err := cmd.CombinedOutput()

		if err == nil {
			slog.Debug("Git команда выполнена успешно",
				slog.Int("Попытка", attempt),
				slog.String("Команда", strings.Join(args, " ")))
			return nil
		}

		slog.Debug("Git команда завершилась с ошибкой",
			slog.Int("Попытка", attempt),
			slog.String("Команда", strings.Join(args, " ")),
			slog.String("Вывод", string(output)),
			slog.String("Ошибка", err.Error()))

		// Проверяем, связана ли ошибка с index.lock
		if strings.Contains(string(output), "index.lock") && attempt < maxRetries {
			slog.Info("Обнаружена ошибка index.lock, ожидаем освобождения блокировки",
				slog.Int("Попытка", attempt),
				slog.Int("Максимум попыток", maxRetries),
				slog.String("Команда", strings.Join(args, " ")),
				slog.String("Путь репозитория", repoPath))

			// Ожидаем освобождения файла блокировки (60 минут согласно требованиям)
			slog.Debug("Начинаем ожидание освобождения блокировки index.lock",
				slog.Duration("Максимальное время ожидания", 60*time.Minute),
				slog.Int("Попытка", attempt))

			if lockErr := waitForGitLockRelease(repoPath, 60*time.Minute); lockErr != nil {
				slog.Error("Не удалось дождаться освобождения файла блокировки",
					slog.String("Ошибка", lockErr.Error()),
					slog.String("Путь", repoPath),
					slog.Int("Попытка", attempt),
					slog.String("Команда", strings.Join(args, " ")))
			} else {
				slog.Debug("Блокировка index.lock успешно освобождена",
					slog.String("Путь", repoPath),
					slog.Int("Попытка", attempt))
			}

			// Ждем немного перед повтором
			slog.Debug("Ожидание перед повторной попыткой",
				slog.Duration("Задержка", 500*time.Millisecond),
				slog.Int("Попытка", attempt))
			time.Sleep(500 * time.Millisecond)
			continue
		}

		slog.Error("Git команда завершилась с ошибкой (не связанной с index.lock)",
			slog.Int("Попытка", attempt),
			slog.String("Команда", strings.Join(args, " ")),
			slog.String("Вывод", string(output)))
		return fmt.Errorf("git command failed: %s", string(output))
	}

	slog.Error("Git команда не выполнена после всех попыток",
		slog.Int("Попыток", maxRetries),
		slog.String("Команда", strings.Join(args, " ")))
	return fmt.Errorf("git command failed after %d attempts", maxRetries)
}

// waitForGitSync ожидает синхронизации состояния каталога с git
// getGitStatus возвращает статус Git репозитория для диагностики
func getGitStatus(ctx context.Context, repoPath string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// #nosec G204 - GitCommand is a constant, all arguments are hardcoded
	cmd := exec.CommandContext(ctx, GitCommand, "status", "--porcelain")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %w", err)
	}

	return string(output), nil
}

func waitForGitSync(ctx context.Context, repoPath string) error {
	maxAttempts := 10
	slog.Debug("Начинаем ожидание Git синхронизации",
		slog.String("repoPath", repoPath),
		slog.Int("maxAttempts", maxAttempts))

	// Получаем начальный статус репозитория для диагностики
	if status, err := getGitStatus(ctx, repoPath); err == nil {
		slog.Debug("Начальный статус Git репозитория",
			slog.String("status", status))
	} else {
		slog.Debug("Не удалось получить статус Git репозитория",
			slog.String("error", err.Error()))
	}

	for i := 0; i < maxAttempts; i++ {
		slog.Debug("Попытка Git синхронизации",
			slog.Int("attempt", i+1),
			slog.Int("maxAttempts", maxAttempts))

		// Проверяем, что Git репозиторий готов к работе
		// Используем простую команду git status для проверки готовности
		ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		// #nosec G204 - GitCommand is a constant, all arguments are hardcoded
		cmd := exec.CommandContext(ctx, GitCommand, "status", "--porcelain")
		cmd.Dir = repoPath

		slog.Debug("Выполняем команду git status --porcelain",
			slog.String("workDir", repoPath),
			slog.Duration("timeout", 2*time.Minute))

		// Захватываем вывод для диагностики
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		cancel() // Освобождаем ресурсы контекста

		// Если это ошибка таймаута, прерываем попытки
		if ctx.Err() == context.DeadlineExceeded {
			slog.Error("Git команда превысила таймаут",
				slog.Int("attempt", i+1),
				slog.Duration("timeout", 2*time.Minute),
				slog.String("stdout", stdout.String()),
				slog.String("stderr", stderr.String()))
			return fmt.Errorf("git command timeout after %d attempts", i+1)
		}

		if err == nil {
			slog.Debug("Git синхронизация завершена успешно - репозиторий готов",
				slog.Int("attempt", i+1),
				slog.String("status", stdout.String()))
			return nil
		}

		// Проверяем тип ошибки
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode := exitError.ExitCode()
			slog.Debug("Git status команда завершилась с кодом возврата",
				slog.Int("attempt", i+1),
				slog.Int("exitCode", exitCode),
				slog.String("stdout", stdout.String()),
				slog.String("stderr", stderr.String()))
		} else {
			// Логируем другие типы ошибок
			slog.Debug("Git status команда завершилась с ошибкой",
				slog.Int("attempt", i+1),
				slog.String("error", err.Error()),
				slog.String("stdout", stdout.String()),
				slog.String("stderr", stderr.String()))
		}

		slog.Debug("Ожидание перед следующей попыткой",
			slog.Duration("delay", 100*time.Millisecond))
		time.Sleep(100 * time.Millisecond)
	}

	slog.Error("Git синхронизация не завершена после всех попыток",
		slog.Int("maxAttempts", maxAttempts))
	return fmt.Errorf("git sync timeout after %d attempts", maxAttempts)
}
