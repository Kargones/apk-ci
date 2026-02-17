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
		return fmt.Errorf("branch switch failed: %w", err)
	}

	l.Info("Branch switch operation completed successfully", slog.String("branch", g.Branch))
	return nil
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
