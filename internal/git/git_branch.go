package git

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SwitchOrCreateBranch переключается на ветку или создает новую
// SwitchOrCreateBranch переключается на ветку или создает её при отсутствии.
// Пытается переключиться на указанную ветку, а если она не существует,
// создает новую ветку и переключается на неё.
// Параметры:
//   - ctx: контекст выполнения операции (может содержать таймаут)
//   - repoPath: путь к локальному репозиторию
//   - branchName: имя ветки для переключения или создания
//
// Возвращает:
//   - error: ошибка переключения/создания ветки или nil при успехе
func SwitchOrCreateBranch(ctx context.Context, repoPath, branchName string) error {
	slog.Debug("Начинаем переключение или создание ветки",
		slog.String("repoPath", repoPath),
		slog.String("branchName", branchName))

	// Проверяем, существует ли директория репозитория
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		slog.Error("Путь к репозиторию не существует",
			slog.String("repoPath", repoPath))
		return fmt.Errorf("repository path does not exist: %s", repoPath)
	}
	slog.Debug("Путь к репозиторию существует", slog.String("repoPath", repoPath))

	// Ожидаем освобождения возможного заблокированного index.lock файла (60 минут согласно требованиям)
	slog.Debug("Ожидаем освобождения блокировки Git",
		slog.String("repoPath", repoPath),
		slog.Duration("timeout", 60*time.Minute))
	if err := waitForGitLockRelease(repoPath, 60*time.Minute); err != nil {
		slog.Warn("Failed to wait for git index.lock release", slog.String("error", err.Error()))
	} else {
		slog.Debug("Блокировка Git освобождена или отсутствовала")
	}

	// Сохраняем текущую рабочую директорию
	originalDir, err := os.Getwd()
	if err != nil {
		slog.Error("Не удалось получить текущую рабочую директорию",
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to get current working directory: %w", err)
	}
	slog.Debug("Сохранили текущую рабочую директорию", slog.String("originalDir", originalDir))

	// Переходим в директорию репозитория
	slog.Debug("Переходим в директорию репозитория",
		slog.String("from", originalDir),
		slog.String("to", repoPath))
	if chdirErr := os.Chdir(repoPath); chdirErr != nil {
		slog.Error("Не удалось перейти в директорию репозитория",
			slog.String("repoPath", repoPath),
			slog.String("error", chdirErr.Error()))
		return fmt.Errorf("failed to change to repository directory: %w", chdirErr)
	}

	// Используем defer для восстановления исходной директории
	defer func() {
		if restoreErr := os.Chdir(originalDir); restoreErr != nil {
			// Логируем ошибку, но не возвращаем её, чтобы не перезаписать основную ошибку
			// Используем slog вместо fmt.Printf для соответствия forbidigo
			slog.Warn("Failed to restore original directory", slog.String("error", restoreErr.Error()))
		}
	}()

	// Используем переданный контекст или создаем новый с увеличенным таймаутом для больших репозиториев
	var cancel context.CancelFunc
	if ctx == nil {
		
	}

	// Если контекст не имеет дедлайна, устанавливаем таймаут 60 минут для больших репозиториев (до 80 ГБ)
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		slog.Debug("Устанавливаем таймаут для операций с большими репозиториями",
			slog.Duration("timeout", 60*time.Minute))
		ctx, cancel = context.WithTimeout(ctx, 60*time.Minute)
		defer cancel()
	} else {
		if deadline, _ := ctx.Deadline(); !deadline.IsZero() {
			slog.Debug("Используем существующий таймаут контекста",
				slog.Time("deadline", deadline))
		}
	}

	// Проверяем существование локальной ветки
	slog.Debug("Проверяем существование локальной ветки", slog.String("branch", branchName))
	// #nosec G204 - GitCommand is a constant, branchName from trusted Git refs
	cmdCheck := exec.CommandContext(ctx, GitCommand, "rev-parse", "--verify", branchName)
	if runErr := cmdCheck.Run(); runErr == nil {
		// Локальная ветка существует, переключаемся на неё
		slog.Debug("Локальная ветка существует, переключаемся на неё", slog.String("branch", branchName))
		if checkoutErr := executeGitCommandWithRetry(repoPath, []string{"checkout", branchName}); checkoutErr != nil {
			slog.Error("Не удалось переключиться на существующую ветку",
				slog.String("branch", branchName),
				slog.String("error", checkoutErr.Error()))
			return fmt.Errorf("failed to switch to branch %s: %w", branchName, checkoutErr)
		}
		slog.Info("Успешно переключились на существующую ветку", slog.String("branch", branchName))
	} else {
		// Локальная ветка не существует - проверяем удаленную
		slog.Debug("Локальная ветка не существует, проверяем удаленную ветку",
			slog.String("branch", branchName),
			slog.String("checkError", runErr.Error()))

		// Проверяем существование удаленной ветки origin/branchName
		// #nosec G204 - GitCommand is a constant, branchName from trusted Git refs
		cmdCheckRemote := exec.CommandContext(ctx, GitCommand, "rev-parse", "--verify", "origin/"+branchName)
		if remoteErr := cmdCheckRemote.Run(); remoteErr == nil {
			// Удаленная ветка существует - создаем локальную от неё
			slog.Debug("Удаленная ветка существует, создаем локальную ветку от неё",
				slog.String("branch", branchName),
				slog.String("remote", "origin/"+branchName))

			if createErr := executeGitCommandWithRetry(repoPath, []string{"checkout", "-b", branchName, "origin/" + branchName}); createErr != nil {
				slog.Error("Не удалось создать локальную ветку от удаленной",
					slog.String("branch", branchName),
					slog.String("remote", "origin/"+branchName),
					slog.String("error", createErr.Error()))
				return fmt.Errorf("failed to create branch %s from origin/%s: %w", branchName, branchName, createErr)
			}
			slog.Info("Успешно создали локальную ветку от удаленной",
				slog.String("branch", branchName),
				slog.String("remote", "origin/"+branchName))
		} else {
			// Ни локальная, ни удаленная ветка не существуют - создаем новую
			slog.Debug("Ни локальная, ни удаленная ветка не существуют, создаем новую ветку",
				slog.String("branch", branchName),
				slog.String("remoteCheckError", remoteErr.Error()))

			if newBranchErr := executeGitCommandWithRetry(repoPath, []string{"checkout", "-b", branchName}); newBranchErr != nil {
				slog.Error("Не удалось создать новую ветку",
					slog.String("branch", branchName),
					slog.String("error", newBranchErr.Error()))
				return fmt.Errorf("failed to create and switch to branch %s: %w", branchName, newBranchErr)
			}
			slog.Info("Успешно создали новую ветку", slog.String("branch", branchName))
		}
	}

	// Ожидаем синхронизации состояния каталога с git
	slog.Debug("Ожидаем синхронизации Git репозитория", slog.String("repoPath", repoPath))
	if err := waitForGitSync(ctx, repoPath); err != nil {
		slog.Error("Не удалось дождаться синхронизации Git",
			slog.String("repoPath", repoPath),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to wait for git sync: %w", err)
	}
	slog.Debug("Git синхронизация завершена успешно")

	slog.Debug("Переключение/создание ветки завершено успешно",
		slog.String("branch", branchName),
		slog.String("repoPath", repoPath))
	return nil
}

// SyncRepoBranches синхронизирует ветки репозитория
// SyncRepoBranches синхронизирует ветки репозитория с удаленным сервером.
// Получает информацию о всех удаленных ветках и создает локальные
// отслеживающие ветки для каждой удаленной ветки.
// Параметры:
//   - repoPath: путь к локальному репозиторию
//
// Возвращает:
//   - error: ошибка синхронизации или nil при успехе
func SyncRepoBranches(ctx context.Context, repoPath string) error {
	// Сохраняем текущую директорию для восстановления в конце
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("ошибка получения текущей директории: %w", err)
	}
	defer func() {
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
slog.Warn("failed to restore working directory", slog.String("error", chdirErr.Error()))
		}
	}()

	// Переходим в целевой каталог
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("ошибка преобразования пути: %w", err)
	}

	err = os.Chdir(absPath)
	if err != nil {
		return fmt.Errorf("не удалось перейти в каталог %s: %w", absPath, err)
	}

	// Получаем все изменения с удаленного репозитория
	err = runGitCommand(ctx, 30*time.Minute, "fetch", "--all", "--prune")
	if err != nil {
		return fmt.Errorf("ошибка при выполнении git fetch: %w", err)
	}

	// Получаем список всех удаленных веток
	remoteBranches, err := getRemoteBranches(ctx, 10 * time.Minute)
	if err != nil {
		return fmt.Errorf("ошибка получения списка веток: %w", err)
	}

	// Создаем локальные ветки для каждой удаленной
	for _, branch := range remoteBranches {
		if strings.HasPrefix(branch, "HEAD ->") {
			continue
		}

		localBranch := strings.TrimPrefix(branch, "origin/")
		if err := createTrackingBranch(ctx, localBranch, branch); err != nil {
slog.Warn("failed to create tracking branch", slog.String("branch", localBranch), slog.String("error", err.Error()))
		}
	}

	// Обновляем все локальные ветки
	if err := runGitCommand(ctx, 30*time.Minute, "pull", "--all"); err != nil {
		return fmt.Errorf("ошибка при выполнении git pull: %w", err)
	}

	return nil
}

// getRemoteBranches получает список удаленных веток репозитория.
// Выполняет git команду для получения всех удаленных веток
// и возвращает их в виде списка строк.
// Возвращает:
//   - []string: список имен удаленных веток
//   - error: ошибка получения веток или nil при успехе
func getRemoteBranches(ctx context.Context, timeout time.Duration) ([]string, error) {
	// Создаем контекст с переданным таймаутом для команды git
	if timeout == 0 {
		timeout = 10 * time.Minute // Значение по умолчанию
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// #nosec G204 - GitCommand is a constant, all arguments are hardcoded
	cmd := exec.CommandContext(ctx, GitCommand, "branch", "-r")
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("таймаут при получении удаленных веток")
		}
		return nil, fmt.Errorf("ошибка получения веток: %s", string(output))
	}

	var branches []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

// createTrackingBranch создает локальную отслеживающую ветку.
// Создает новую локальную ветку, которая отслеживает указанную
// удаленную ветку для синхронизации изменений.
// Параметры:
//   - local: имя локальной ветки для создания
//   - remote: имя удаленной ветки для отслеживания
//
// Возвращает:
//   - error: ошибка создания ветки или nil при успехе
func createTrackingBranch(ctx context.Context, local, remote string) error {
	// Проверяем существование локальной ветки
	if err := runGitCommand(ctx, 10*time.Minute, "show-ref", "--verify", "--quiet", "refs/heads/"+local); err == nil {
		return fmt.Errorf("ветка %s уже существует", local)
	}

	// Создаем новую ветку с трекингом
	if err := runGitCommand(ctx, 30*time.Minute, "checkout", "-b", local, "--track", remote); err != nil {
		return fmt.Errorf("не удалось создать ветку %s: %w", local, err)
	}

	// Branch created: local -> remote (logging handled by caller)
	return nil
}
