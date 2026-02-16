// Package git предоставляет функциональность для работы с Git репозиториями
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

const (
	// LastCommit представляет последний коммит
	LastCommit = "last"
	// GitCommand команда для работы с git
	GitCommand = "git"
)

var (
	// cloneOk = "Cloning into"
	resetOk = "HEAD is now at"
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

// NewConfigs создает новый экземпляр конфигурации Git.
// Инициализирует структуру с настройками по умолчанию для работы
// с Git репозиториями и командами.
// Возвращает:
//   - Configs: новую структуру конфигурации Git
//
// ToDo: Добавить команду SetBranch
func NewConfigs() Configs {
	gca := []Config{
		{"core.symlinks", "false"},
		{"core.ignorecase", "true"},
		{"core.quotepath", "false"},
		{"core.autocrlf", "false"},
		{"push.autoSetupRemote", "true"},
	}
	gcs := Configs{}
	gcs.Config = append(gcs.Config, gca...)
	return gcs
}

// Reset сбрасывает состояние репозитория к указанному коммиту.
// Выполняет жесткий сброс (hard reset) рабочей директории и индекса
// к состоянию указанного коммита, отменяя все последующие изменения.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений
//
// Возвращает:
//   - error: ошибка выполнения сброса или nil при успехе
func (g *Git) Reset(ctx context.Context, l *slog.Logger) error {
	if g.CommitSHA1 == "" || g.CommitSHA1 == LastCommit {
		return nil
	}
	// Создаем контекст с конфигурируемым таймаутом
	timeout := g.Timeout
	if timeout == 0 {
		timeout = 60 * time.Minute // Значение по умолчанию для push операций
	}
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Формируем параметры команды
	args := []string{"reset", "--hard", g.CommitSHA1}

	// Выполняем команду git reset с таймаутом
	// #nosec G204 - GitCommand является константой, args формируется из проверенных значений
	cmd := exec.CommandContext(cmdCtx, GitCommand, args...)
	cmd.Dir = g.RepPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			l.Error("Таймаут выполнения git reset",
				slog.Group("Параметры команды",
					slog.String("URL репозитория", g.RepURL),
					slog.String("Целевой каталог", g.RepPath),
					slog.String("SHA1 коммит", g.CommitSHA1),
					slog.String("Команда", GitCommand+" "+strings.Join(args, " ")),
				),
			)
			return fmt.Errorf("таймаут выполнения git reset: %w", ctx.Err())
		}
		return err
	}

	// Проверяем успешность reset
	if !strings.Contains(string(output), resetOk) {
		l.Error("Ошибка установки коммита",
			slog.Group("Параметры установки коммита",
				slog.String("URL репозитория", g.RepURL),
				slog.String("Целевой каталог", g.RepPath),
				slog.String("Ветка", g.Branch),
				slog.String("SHA1 коммит", g.CommitSHA1),
				slog.String("Вывод консоли", string(output)),
				slog.Bool("Результат проверки", strings.Contains(string(output), resetOk)),
				slog.String("Проверяемая подстрока", resetOk),
			),
		)
		return fmt.Errorf("ошибка установки коммита")
	}
	return nil
}

// Clone клонирует удаленный репозиторий в локальную директорию.
// Создает полную копию репозитория с указанной ветки для локальной
// разработки и работы с кодом.
// Параметры:
//   - ctx: указатель на контекст выполнения операции
//   - l: логгер для записи сообщений
//
// Возвращает:
//   - error: ошибка клонирования или nil при успехе
func (g *Git) Clone(ctx context.Context, l *slog.Logger) error {
	// Создаем контекст с конфигурируемым таймаутом для клонирования
	timeout := g.Timeout
	if timeout == 0 {
		timeout = 60 * time.Minute // Значение по умолчанию, если таймаут не задан
	}
	ctxTimeout, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Формируем параметры команды
	args := []string{"clone"}
	if g.Branch != "" {
		args = append(args, "-b", g.Branch)
	}
	args = append(args, g.RepURL)
	//ToDo:	Переделать, если каталог есть создавать новый
	args = append(args, g.RepPath)

	// Выполняем команду git clone с таймаутом
	// #nosec G204 - GitCommand является константой, args формируется из проверенных значений
	cmd := exec.CommandContext(ctxTimeout, GitCommand, args...)
	cmd.Dir = g.WorkDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctxTimeout.Err() == context.DeadlineExceeded {
			l.Error("Таймаут выполнения git clone",
				slog.Group("Параметры команды",
					slog.String("URL репозитория", g.RepURL),
					slog.String("Целевой каталог", g.RepPath),
					slog.String("Ветка", g.Branch),
					slog.String("Команда", GitCommand+" "+strings.Join(args, " ")),
				),
			)
			return fmt.Errorf("таймаут выполнения git clone: %w", ctxTimeout.Err())
		}
		return err
	}

	// Проверяем успешность клонирования
	if !isCloneSuccessful(string(output)) {
		l.Error("Ошибка клонирования",
			slog.Group("Параметры клонирования",
				slog.String("URL репозитория", g.RepURL),
				slog.String("Целевой каталог", g.RepPath),
				slog.String("Вывод консоли", string(output)),
				slog.Bool("Результат проверки", isCloneSuccessful(string(output))),
			),
		)
		return fmt.Errorf("ошибка клонирования")
	}
	// err = SyncRepoBranches(g.RepPath)
	// if err != nil {
	// 	l.Error("Ошибка синхронизации веток",
	// 		slog.String("URL репозитория", g.RepURL),
	// 		slog.String("Целевой каталог", g.RepPath),
	// 		slog.String("Ветка", g.Branch),
	// 		slog.String("SHA1 коммит", g.CommitSHA1),
	// 	)
	// 	return err
	// }
	// Установка на конкретный коммит
	// if err := g.Reset(ctx, l); err != nil {
	// 	l.Error("Failed to reset git repository", slog.String("error", err.Error()))
	// 	return err
	// }
	return nil
}

// Config устанавливает конфигурацию Git
// Config настраивает параметры Git репозитория.
// Устанавливает пользовательские настройки Git, такие как имя пользователя
// и email, необходимые для создания коммитов.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений
func (g *Git) Config(ctx context.Context, l *slog.Logger) error {
	gcs := NewConfigs()
	for _, gc := range gcs.Config {
		// Создаем контекст с таймаутом для каждой команды config
		cmdCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()

		args := []string{"config", "--local", gc.Name, gc.Value}
		// #nosec G204 - GitCommand является константой, args формируется из проверенных значений
		cmd := exec.CommandContext(cmdCtx, GitCommand, args...)
		cmd.Dir = g.RepPath

		output, err := cmd.CombinedOutput()
		if err != nil {
			if cmdCtx.Err() == context.DeadlineExceeded {
				l.Error("Таймаут при установке параметра репозитория",
					slog.String("repURL", g.RepURL),
					slog.String("repPath", g.RepPath),
					slog.String("configName", gc.Name),
					slog.String("configValue", gc.Value),
					slog.Duration("timeout", 15*time.Second))
				return fmt.Errorf("таймаут при установке параметра %s", gc.Name)
			}
			l.Error("Ошибка установки параметра репозитория",
				slog.String("repURL", g.RepURL),
				slog.String("repPath", g.RepPath),
				slog.String("configName", gc.Name),
				slog.String("configValue", gc.Value),
				slog.String("output", string(output)),
				slog.String("error", err.Error()))
			return fmt.Errorf("ошибка установки параметра %s: %w", gc.Name, err)
		}
	}
	return nil
}

// Add добавляет файлы в индекс Git
// Add добавляет файлы в индекс Git для последующего коммита.
// Подготавливает изменения в рабочей директории для включения
// в следующий коммит.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений
func (g *Git) Add(ctx context.Context, l *slog.Logger) error {
	// Создаем контекст с конфигурируемым таймаутом
	timeout := g.Timeout
	if timeout == 0 {
		timeout = 30 * time.Minute // Значение по умолчанию
	}
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{"add", "."}
	// #nosec G204 - GitCommand является константой, args формируется из проверенных значений
	cmd := exec.CommandContext(cmdCtx, GitCommand, args...)
	cmd.Dir = g.RepPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			l.Error("Таймаут при добавлении файлов в индекс",
				slog.String("repURL", g.RepURL),
				slog.String("repPath", g.RepPath),
				slog.Duration("timeout", timeout))
			return fmt.Errorf("таймаут при добавлении файлов в индекс")
		}
		l.Error("Ошибка добавления файлов в индекс",
			slog.String("repURL", g.RepURL),
			slog.String("repPath", g.RepPath),
			slog.String("output", string(output)),
			slog.String("error", err.Error()))
		return fmt.Errorf("ошибка добавления файлов в индекс: %w", err)
	}
	return nil
}

// Switch переключается на указанную ветку
// Switch переключается на указанную ветку в репозитории.
// Изменяет активную ветку рабочей директории на указанную ветку,
// обновляя файлы в соответствии с состоянием целевой ветки.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений
//
// Возвращает:
//   - error: ошибка переключения ветки или nil при успехе
func (g *Git) Switch(ctx context.Context, l *slog.Logger) error {
	l.Debug("Starting branch switch operation",
		slog.String("branch", g.Branch),
		slog.String("repo_path", g.RepPath))

	if err := SwitchOrCreateBranch(ctx, g.RepPath, g.Branch); err != nil {
		l.Error("Failed to switch or create branch",
			slog.String("error", err.Error()),
			slog.String("branch", g.Branch),
			slog.String("repo_path", g.RepPath))
		return fmt.Errorf("branch switch failed: %v", err)
	}

	l.Info("Branch switch operation completed successfully", slog.String("branch", g.Branch))
	return nil
}

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
		return fmt.Errorf("failed to get current working directory: %v", err)
	}
	slog.Debug("Сохранили текущую рабочую директорию", slog.String("originalDir", originalDir))

	// Переходим в директорию репозитория
	slog.Debug("Переходим в директорию репозитория",
		slog.String("from", originalDir),
		slog.String("to", repoPath))
	if err := os.Chdir(repoPath); err != nil {
		slog.Error("Не удалось перейти в директорию репозитория",
			slog.String("repoPath", repoPath),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to change to repository directory: %v", err)
	}

	// Используем defer для восстановления исходной директории
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			// Логируем ошибку, но не возвращаем её, чтобы не перезаписать основную ошибку
			// Используем slog вместо fmt.Printf для соответствия forbidigo
			slog.Warn("Failed to restore original directory", slog.String("error", err.Error()))
		}
	}()

	// Используем переданный контекст или создаем новый с увеличенным таймаутом для больших репозиториев
	var cancel context.CancelFunc
	if ctx == nil {
		ctx = context.Background()
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
	cmdCheck := exec.CommandContext(ctx, GitCommand, "rev-parse", "--verify", branchName)
	if err := cmdCheck.Run(); err == nil {
		// Локальная ветка существует, переключаемся на неё
		slog.Debug("Локальная ветка существует, переключаемся на неё", slog.String("branch", branchName))
		if err := executeGitCommandWithRetry(repoPath, []string{"checkout", branchName}); err != nil {
			slog.Error("Не удалось переключиться на существующую ветку",
				slog.String("branch", branchName),
				slog.String("error", err.Error()))
			return fmt.Errorf("failed to switch to branch %s: %v", branchName, err)
		}
		slog.Info("Успешно переключились на существующую ветку", slog.String("branch", branchName))
	} else {
		// Локальная ветка не существует - проверяем удаленную
		slog.Debug("Локальная ветка не существует, проверяем удаленную ветку",
			slog.String("branch", branchName),
			slog.String("checkError", err.Error()))

		// Проверяем существование удаленной ветки origin/branchName
		cmdCheckRemote := exec.CommandContext(ctx, GitCommand, "rev-parse", "--verify", "origin/"+branchName)
		if err := cmdCheckRemote.Run(); err == nil {
			// Удаленная ветка существует - создаем локальную от неё
			slog.Debug("Удаленная ветка существует, создаем локальную ветку от неё",
				slog.String("branch", branchName),
				slog.String("remote", "origin/"+branchName))

			if err := executeGitCommandWithRetry(repoPath, []string{"checkout", "-b", branchName, "origin/" + branchName}); err != nil {
				slog.Error("Не удалось создать локальную ветку от удаленной",
					slog.String("branch", branchName),
					slog.String("remote", "origin/"+branchName),
					slog.String("error", err.Error()))
				return fmt.Errorf("failed to create branch %s from origin/%s: %v", branchName, branchName, err)
			}
			slog.Info("Успешно создали локальную ветку от удаленной",
				slog.String("branch", branchName),
				slog.String("remote", "origin/"+branchName))
		} else {
			// Ни локальная, ни удаленная ветка не существуют - создаем новую
			slog.Debug("Ни локальная, ни удаленная ветка не существуют, создаем новую ветку",
				slog.String("branch", branchName),
				slog.String("remoteCheckError", err.Error()))

			if err := executeGitCommandWithRetry(repoPath, []string{"checkout", "-b", branchName}); err != nil {
				slog.Error("Не удалось создать новую ветку",
					slog.String("branch", branchName),
					slog.String("error", err.Error()))
				return fmt.Errorf("failed to create and switch to branch %s: %v", branchName, err)
			}
			slog.Info("Успешно создали новую ветку", slog.String("branch", branchName))
		}
	}

	// Ожидаем синхронизации состояния каталога с git
	slog.Debug("Ожидаем синхронизации Git репозитория", slog.String("repoPath", repoPath))
	if err := waitForGitSync(repoPath); err != nil {
		slog.Error("Не удалось дождаться синхронизации Git",
			slog.String("repoPath", repoPath),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to wait for git sync: %v", err)
	}
	slog.Debug("Git синхронизация завершена успешно")

	slog.Debug("Переключение/создание ветки завершено успешно",
		slog.String("branch", branchName),
		slog.String("repoPath", repoPath))
	return nil
}

// waitForGitSync ожидает синхронизации состояния каталога с git
// getGitStatus возвращает статус Git репозитория для диагностики
func getGitStatus(repoPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, GitCommand, "status", "--porcelain")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %w", err)
	}

	return string(output), nil
}

func waitForGitSync(repoPath string) error {
	maxAttempts := 10
	slog.Debug("Начинаем ожидание Git синхронизации",
		slog.String("repoPath", repoPath),
		slog.Int("maxAttempts", maxAttempts))

	// Получаем начальный статус репозитория для диагностики
	if status, err := getGitStatus(repoPath); err == nil {
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
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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

// Commit создает коммит с указанным сообщением
// ToDo: Возвращать ошибку "nothing to commit, working tree clean"
// Commit создает коммит с указанным сообщением.
// Фиксирует изменения из индекса в истории репозитория
// с описательным сообщением о внесенных изменениях.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений
//   - comment: сообщение коммита
func (g *Git) Commit(ctx context.Context, l *slog.Logger, comment string) error {
	// Создаем контекст с конфигурируемым таймаутом
	timeout := g.Timeout
	if timeout == 0 {
		timeout = 30 * time.Minute // Значение по умолчанию
	}
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{"commit", "-m", comment}
	// #nosec G204 - GitCommand является константой, args формируется из проверенных значений
	cmd := exec.CommandContext(cmdCtx, GitCommand, args...)
	cmd.Dir = g.RepPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			l.Error("Таймаут при создании коммита",
				slog.String("repURL", g.RepURL),
				slog.String("repPath", g.RepPath),
				slog.String("comment", comment),
				slog.Duration("timeout", timeout))
			return fmt.Errorf("таймаут при создании коммита")
		}
		l.Error("Ошибка создания коммита",
			slog.String("repURL", g.RepURL),
			slog.String("repPath", g.RepPath),
			slog.String("comment", comment),
			slog.String("output", string(output)),
			slog.String("error", err.Error()))
		return fmt.Errorf("ошибка создания коммита: %w", err)
	}
	return nil
}

// Push отправляет изменения в удаленный репозиторий
// Push отправляет локальные коммиты в удаленный репозиторий.
// Синхронизирует локальную ветку с удаленной, загружая
// новые коммиты на сервер.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений
func (g *Git) Push(ctx context.Context, l *slog.Logger) error {
	// Создаем контекст с конфигурируемым таймаутом
	timeout := g.Timeout
	if timeout == 0 {
		timeout = 60 * time.Minute // Значение по умолчанию для push операций
	}
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{"push", "origin", g.Branch}
	// #nosec G204 - GitCommand является константой, args формируется из проверенных значений
	cmd := exec.CommandContext(cmdCtx, GitCommand, args...)
	cmd.Dir = g.RepPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			l.Error("Таймаут при отправке изменений",
				slog.String("repURL", g.RepURL),
				slog.String("repPath", g.RepPath),
				slog.String("branch", g.Branch),
				slog.Duration("timeout", timeout))
			return fmt.Errorf("таймаут при отправке изменений")
		}
		l.Error("Ошибка отправки изменений",
			slog.String("repURL", g.RepURL),
			slog.String("repPath", g.RepPath),
			slog.String("branch", g.Branch),
			slog.String("output", string(output)),
			slog.String("error", err.Error()))
		return fmt.Errorf("ошибка отправки изменений: %w", err)
	}
	return nil
}

// PushForce принудительно отправляет изменения в удаленный репозиторий
// PushForce принудительно отправляет изменения в удаленный репозиторий.
// Перезаписывает историю удаленной ветки локальными изменениями,
// игнорируя конфликты. Используется с осторожностью.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений
func (g *Git) PushForce(ctx context.Context, l *slog.Logger) error {
	// Создаем контекст с конфигурируемым таймаутом
	timeout := g.Timeout
	if timeout == 0 {
		timeout = 60 * time.Minute // Значение по умолчанию для push операций
	}
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := []string{"push", "--force", "origin", g.Branch}
	// #nosec G204 - GitCommand является константой, args формируется из проверенных значений
	cmd := exec.CommandContext(cmdCtx, GitCommand, args...)
	cmd.Dir = g.RepPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			l.Error("Таймаут при принудительной отправке изменений",
				slog.String("repURL", g.RepURL),
				slog.String("repPath", g.RepPath),
				slog.String("branch", g.Branch),
				slog.Duration("timeout", timeout))
			return fmt.Errorf("таймаут при принудительной отправке изменений")
		}
		l.Error("Ошибка принудительной отправки изменений",
			slog.String("repURL", g.RepURL),
			slog.String("repPath", g.RepPath),
			slog.String("branch", g.Branch),
			slog.String("output", string(output)),
			slog.String("error", err.Error()))
		return fmt.Errorf("ошибка принудительной отправки изменений: %w", err)
	}
	return nil
}

// SetUser устанавливает пользователя Git
// SetUser устанавливает пользователя Git для репозитория.
// Настраивает имя пользователя и email, которые будут использоваться
// в коммитах для данного репозитория.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений
//   - name: имя пользователя Git
//   - email: email пользователя Git
func (g *Git) SetUser(ctx context.Context, l *slog.Logger, name, email string) error {
	if name == "" {
		name = "anonimus"
	}
	if email == "" {
		email = "anonimus@anonimus.org"
	}

	// Устанавливаем имя пользователя
	ctx1, cancel1 := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel1()

	// #nosec G204 - GitCommand является константой, args формируется из проверенных значений
	cmd1 := exec.CommandContext(ctx1, GitCommand, "config", "--local", "user.name", name)
	cmd1.Dir = g.RepPath

	output1, err := cmd1.CombinedOutput()
	if err != nil {
		if ctx1.Err() == context.DeadlineExceeded {
			l.Error("Таймаут при установке имени пользователя", slog.String("repURL", g.RepURL), slog.String("repPath", g.RepPath))
			return fmt.Errorf("таймаут при установке имени пользователя: %w", ctx1.Err())
		}
		l.Error("Ошибка установки имени репозитория", slog.String("repURL", g.RepURL), slog.String("repPath", g.RepPath), slog.String("error", err.Error()), slog.String("output", string(output1)))
		return fmt.Errorf("ошибка установки имени репозитория: %w", err)
	}

	// Устанавливаем email пользователя
	ctx2, cancel2 := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel2()

	// #nosec G204 - GitCommand является константой, args формируется из проверенных значений
	cmd2 := exec.CommandContext(ctx2, GitCommand, "config", "--local", "user.email", email)
	cmd2.Dir = g.RepPath

	output2, err := cmd2.CombinedOutput()
	if err != nil {
		if ctx2.Err() == context.DeadlineExceeded {
			l.Error("Таймаут при установке email пользователя", slog.String("repURL", g.RepURL), slog.String("repPath", g.RepPath))
			return fmt.Errorf("таймаут при установке email пользователя: %w", ctx2.Err())
		}
		l.Error("Ошибка установки e-mail репозитория", slog.String("repURL", g.RepURL), slog.String("repPath", g.RepPath), slog.String("error", err.Error()), slog.String("output", string(output2)))
		return fmt.Errorf("ошибка установки e-mail репозитория: %w", err)
	}

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
func SyncRepoBranches(repoPath string) error {
	// Сохраняем текущую директорию для восстановления в конце
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("ошибка получения текущей директории: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			// Игнорируем ошибку восстановления директории
			_ = chdirErr
		}
	}()

	// Переходим в целевой каталог
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("ошибка преобразования пути: %v", err)
	}

	err = os.Chdir(absPath)
	if err != nil {
		return fmt.Errorf("не удалось перейти в каталог %s: %v", absPath, err)
	}

	// Получаем все изменения с удаленного репозитория
	err = runGitCommand(30*time.Minute, "fetch", "--all", "--prune")
	if err != nil {
		return fmt.Errorf("ошибка при выполнении git fetch: %v", err)
	}

	// Получаем список всех удаленных веток
	remoteBranches, err := getRemoteBranches(10 * time.Minute)
	if err != nil {
		return fmt.Errorf("ошибка получения списка веток: %v", err)
	}

	// Создаем локальные ветки для каждой удаленной
	for _, branch := range remoteBranches {
		if strings.HasPrefix(branch, "HEAD ->") {
			continue
		}

		localBranch := strings.TrimPrefix(branch, "origin/")
		if err := createTrackingBranch(localBranch, branch); err != nil {
			// Игнорируем ошибку создания tracking branch
			_ = err
		}
	}

	// Обновляем все локальные ветки
	if err := runGitCommand(30*time.Minute, "pull", "--all"); err != nil {
		return fmt.Errorf("ошибка при выполнении git pull: %v", err)
	}

	return nil
}

// runGitCommand выполняет команду Git с указанными аргументами.
// Запускает git команду и возвращает её вывод или ошибку.
// Параметры:
//   - args: аргументы команды git
//
// Возвращает:
//   - error: ошибка выполнения или nil при успехе
func runGitCommand(timeout time.Duration, args ...string) error {
	// Создаем контекст с переданным таймаутом для команды git
	if timeout == 0 {
		timeout = 30 * time.Minute // Значение по умолчанию
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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

// getRemoteBranches получает список удаленных веток репозитория.
// Выполняет git команду для получения всех удаленных веток
// и возвращает их в виде списка строк.
// Возвращает:
//   - []string: список имен удаленных веток
//   - error: ошибка получения веток или nil при успехе
func getRemoteBranches(timeout time.Duration) ([]string, error) {
	// Создаем контекст с переданным таймаутом для команды git
	if timeout == 0 {
		timeout = 10 * time.Minute // Значение по умолчанию
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

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
func createTrackingBranch(local, remote string) error {
	// Проверяем существование локальной ветки
	if err := runGitCommand(10*time.Minute, "show-ref", "--verify", "--quiet", "refs/heads/"+local); err == nil {
		return fmt.Errorf("ветка %s уже существует", local)
	}

	// Создаем новую ветку с трекингом
	if err := runGitCommand(30*time.Minute, "checkout", "-b", local, "--track", remote); err != nil {
		return fmt.Errorf("не удалось создать ветку %s: %v", local, err)
	}

	// Branch created: local -> remote (logging handled by caller)
	return nil
}

// CloneToTempDir клонирует репозиторий в временный каталог.
// Создает временный каталог и клонирует указанную ветку репозитория.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений
//   - repoURL: URL репозитория для клонирования
//   - branch: имя ветки для клонирования
//   - token: токен авторизации (опционально)
//
// Возвращает:
//   - string: путь к созданному временному каталогу
//   - error: ошибка выполнения операции
func CloneToTempDir(ctx context.Context, l *slog.Logger, tempDir, repoURL, branch, token string, timeout time.Duration) (string, error) {
	// Формируем URL с токеном если он предоставлен
	cloneURL := repoURL
	if token != "" {
		// Добавляем токен в URL для аутентификации
		if strings.Contains(repoURL, "://") {
			parts := strings.SplitN(repoURL, "://", 2)
			if len(parts) == 2 {
				cloneURL = fmt.Sprintf("%s://%s@%s", parts[0], token, parts[1])
			}
		}
	}

	l.Debug("Cloning repository to temp directory",
		slog.String("repo_url", repoURL),
		slog.String("branch", branch),
		slog.String("temp_dir", tempDir),
	)

	// Используем переданный контекст или создаем с таймаутом если контекст без таймаута
	var ctxWithTimeout context.Context
	var cancel context.CancelFunc

	if _, ok := ctx.Deadline(); ok {
		// Контекст уже имеет таймаут, используем его
		ctxWithTimeout = ctx
		cancel = func() {} // Пустая функция, так как контекст уже управляется вызывающей стороной
	} else {
		// Контекст без таймаута, добавляем переданный таймаут
		if timeout == 0 {
			timeout = 60 * time.Minute // Значение по умолчанию для clone операций
		}
		ctxWithTimeout, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	// Формируем параметры команды
	args := []string{"clone"}
	if branch != "" {
		args = append(args, "-b", branch)
	}
	args = append(args, cloneURL, tempDir)

	// Выполняем команду git clone с таймаутом
	// #nosec G204 - GitCommand является константой, args формируется из проверенных значений
	cmd := exec.CommandContext(ctxWithTimeout, GitCommand, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		if ctxWithTimeout.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("git clone timeout")
		}
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	if !isCloneSuccessful(string(output)) {
		l.Error("Ошибка клонирования",
			slog.String("URL репозитория", cloneURL),
			slog.String("Целевой каталог", tempDir),
			slog.String("Вывод консоли", string(output)),
		)
		return "", fmt.Errorf("ошибка клонирования")
	}

	l.Debug("Repository cloned successfully",
		slog.String("repo_path", tempDir),
	)

	return tempDir, nil
}

// CheckoutCommit переключает git-репозиторий к указанному коммиту.
// Выполняет атомарную операцию checkout с проверками валидности.
//
// Parameters:
//   - ctx: контекст выполнения
//   - l: логгер для записи событий
//   - repoPath: путь к git-репозиторию
//   - commitHash: хеш коммита для переключения
//
// Returns:
//   - error: ошибка если операция не удалась
func CheckoutCommit(ctx context.Context, l *slog.Logger, repoPath, commitHash string) error {
	// Проверяем валидность входных параметров
	if repoPath == "" {
		return fmt.Errorf("путь к репозиторию не может быть пустым")
	}
	if commitHash == "" {
		return fmt.Errorf("хеш коммита не может быть пустым")
	}

	// Проверяем существование директории репозитория
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		l.Error("Директория репозитория не существует",
			slog.String("repoPath", repoPath))
		return fmt.Errorf("директория репозитория не существует: %s", repoPath)
	}

	// Проверяем, что это git-репозиторий
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		l.Error("Указанная директория не является git-репозиторием",
			slog.String("repoPath", repoPath),
			slog.String("gitDir", gitDir))
		return fmt.Errorf("директория %s не является git-репозиторием", repoPath)
	}

	// Проверяем существование коммита в репозитории
	if err := validateCommitExists(repoPath, commitHash); err != nil {
		l.Error("Коммит не найден в репозитории",
			slog.String("repoPath", repoPath),
			slog.String("commitHash", commitHash),
			slog.String("error", err.Error()))
		return fmt.Errorf("коммит %s не найден в репозитории: %w", commitHash, err)
	}

	// Выполняем checkout коммита с таймаутом
	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	args := []string{"checkout", commitHash}
	// #nosec G204 - GitCommand является константой, args формируется из проверенных значений
	cmd := exec.CommandContext(cmdCtx, GitCommand, args...)
	cmd.Dir = repoPath

	// DEBUG: Список файлов до checkout'а
	if files, err := filepath.Glob(filepath.Join(repoPath, "*")); err == nil {
		l.Debug("DEBUG: Files in repo before checkout",
			slog.String("repoPath", repoPath),
			slog.Any("files", files),
			slog.Int("fileCount", len(files)))
	}

	l.Info("Переключение к коммиту",
		slog.String("repoPath", repoPath),
		slog.String("commitHash", commitHash))

	output, err := cmd.CombinedOutput()
	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			l.Error("Таймаут при переключении к коммиту",
				slog.String("repoPath", repoPath),
				slog.String("commitHash", commitHash),
				slog.Duration("timeout", 30*time.Minute))
			return fmt.Errorf("таймаут при переключении к коммиту %s", commitHash)
		}
		l.Error("Ошибка переключения к коммиту",
			slog.Group("Параметры операции",
				slog.String("Путь к репозиторию", repoPath),
				slog.String("Хеш коммита", commitHash),
				slog.String("Вывод консоли", string(output)),
				slog.String("Ошибка", err.Error()),
			))
		return fmt.Errorf("ошибка переключения к коммиту %s: %w", commitHash, err)
	}

	// DEBUG: Список файлов после checkout'а
	if files, err := filepath.Glob(filepath.Join(repoPath, "*")); err == nil {
		l.Debug("DEBUG: Files in repo after checkout",
			slog.String("repoPath", repoPath),
			slog.String("commitHash", commitHash),
			slog.Any("files", files),
			slog.Int("fileCount", len(files)))
	}

	l.Info("Успешно переключились к коммиту",
		slog.String("repoPath", repoPath),
		slog.String("commitHash", commitHash))

	return nil
}

// validateCommitExists проверяет существование коммита в репозитории
func validateCommitExists(repoPath, commitHash string) error {
	// Создаем контекст с таймаутом для команды git
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// #nosec G204 - GitCommand является константой, args формируется из проверенных значений
	cmd := exec.CommandContext(ctx, GitCommand, "cat-file", "-e", commitHash)
	cmd.Dir = repoPath

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("таймаут при проверке коммита %s", commitHash)
		}
		return fmt.Errorf("коммит не существует: %s", stderr.String())
	}
	return nil
}
