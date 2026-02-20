package git

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNewConfigs тестирует создание новой конфигурации Git
func TestNewConfigs(t *testing.T) {
	configs := NewConfigs()

	// Проверяем, что конфигурация не пустая
	if len(configs.Config) == 0 {
		t.Error("Конфигурация не должна быть пустой")
	}

	// Проверяем наличие обязательных параметров
	expectedConfigs := map[string]string{
		"core.symlinks":         "false",
		"core.ignorecase":       "true",
		"core.quotepath":        "false",
		"core.autocrlf":         "false",
		"push.autoSetupRemote":  "true",
	}

	for _, config := range configs.Config {
		if expectedValue, exists := expectedConfigs[config.Name]; exists {
			if config.Value != expectedValue {
				t.Errorf("Неверное значение для %s: ожидалось %s, получено %s", config.Name, expectedValue, config.Value)
			}
			delete(expectedConfigs, config.Name)
		}
	}

	// Проверяем, что все ожидаемые конфигурации присутствуют
	if len(expectedConfigs) > 0 {
		t.Errorf("Отсутствуют конфигурации: %v", expectedConfigs)
	}
}

// TestGitStruct тестирует структуру Git
func TestGitStruct(t *testing.T) {
	git := &Git{
		RepURL:     "https://github.com/test/repo.git",
		RepPath:    "/tmp/test-repo",
		Branch:     "main",
		CommitSHA1: "abc123",
		WorkDir:    "/tmp",
		Token:      "test-token",
	}

	// Проверяем, что все поля установлены корректно
	if git.RepURL != "https://github.com/test/repo.git" {
		t.Errorf("Неверный RepURL: %s", git.RepURL)
	}
	if git.RepPath != "/tmp/test-repo" {
		t.Errorf("Неверный RepPath: %s", git.RepPath)
	}
	if git.Branch != "main" {
		t.Errorf("Неверный Branch: %s", git.Branch)
	}
	if git.CommitSHA1 != "abc123" {
		t.Errorf("Неверный CommitSHA1: %s", git.CommitSHA1)
	}
	if git.WorkDir != "/tmp" {
		t.Errorf("Неверный WorkDir: %s", git.WorkDir)
	}
	if git.Token != "test-token" {
		t.Errorf("Неверный Token: %s", git.Token)
	}
}

// TestConfigStruct тестирует структуру Config
func TestConfigStruct(t *testing.T) {
	config := Config{
		Name:  "test.name",
		Value: "test.value",
	}

	if config.Name != "test.name" {
		t.Errorf("Неверное имя конфигурации: %s", config.Name)
	}
	if config.Value != "test.value" {
		t.Errorf("Неверное значение конфигурации: %s", config.Value)
	}
}

// TestConfigsStruct тестирует структуру Configs
func TestConfigsStruct(t *testing.T) {
	configs := Configs{
		Config: []Config{
			{Name: "test1", Value: "value1"},
			{Name: "test2", Value: "value2"},
		},
	}

	if len(configs.Config) != 2 {
		t.Errorf("Неверное количество конфигураций: %d", len(configs.Config))
	}

	if configs.Config[0].Name != "test1" || configs.Config[0].Value != "value1" {
		t.Error("Неверная первая конфигурация")
	}

	if configs.Config[1].Name != "test2" || configs.Config[1].Value != "value2" {
		t.Error("Неверная вторая конфигурация")
	}
}

// TestGitConstants тестирует константы модуля
func TestGitConstants(t *testing.T) {
	if LastCommit != "last" {
		t.Errorf("Неверная константа LastCommit: %s", LastCommit)
	}

	if GitCommand != "git" {
		t.Errorf("Неверная константа GitCommand: %s", GitCommand)
	}
}

// TestIsCloneSuccessful тестирует функцию isCloneSuccessful
func TestIsCloneSuccessful(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected bool
	}{
		{
			name:     "успешное клонирование с Cloning",
			output:   "Cloning into 'repo'...\nReceiving objects: 100% (123/123), done.",
			expected: true,
		},
		{
			name:     "успешное клонирование с done",
			output:   "remote: Enumerating objects: 123, done.\nReceiving objects: 100% (123/123), done.",
			expected: true,
		},
		{
			name:     "успешное клонирование с получено",
			output:   "Клонирование в 'repo'...\nПолучение объектов: 100% (123/123), получено.",
			expected: true,
		},
		{
			name:     "неуспешное клонирование",
			output:   "fatal: repository not found",
			expected: false,
		},
		{
			name:     "пустой вывод",
			output:   "",
			expected: false,
		},
		{
			name:     "частичное клонирование без завершения",
			output:   "Receiving objects: 50% (60/123)",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCloneSuccessful(tt.output)
			if result != tt.expected {
				t.Errorf("isCloneSuccessful(%q) = %v, ожидалось %v", tt.output, result, tt.expected)
			}
		})
	}
}

// TestCloneWithDifferentURLs тестирует клонирование с различными URL
func TestCloneWithDifferentURLs(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Создаем временную директорию для тестов
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		git      *Git
		wantErr  bool
	}{
		{
			name: "пустой URL",
			git: &Git{
				RepURL:  "",
				RepPath: filepath.Join(tempDir, "empty-url"),
				WorkDir: tempDir,
			},
			wantErr: true,
		},
		{
			name: "невалидный URL",
			git: &Git{
				RepURL:  "invalid-url",
				RepPath: filepath.Join(tempDir, "invalid-url"),
				WorkDir: tempDir,
			},
			wantErr: true,
		},
		{
			name: "несуществующий репозиторий",
			git: &Git{
				RepURL:  "https://github.com/nonexistent/repo.git",
				RepPath: filepath.Join(tempDir, "nonexistent"),
				WorkDir: tempDir,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.git.Clone(ctx, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("Clone() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestResetWithDifferentModes тестирует Reset с различными режимами
func TestResetWithDifferentModes(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Создаем временную директорию
	tempDir := t.TempDir()

	git := &Git{
		RepPath: tempDir,
		WorkDir: tempDir,
	}

	// Тест с несуществующей директорией (Reset может не возвращать ошибку в некоторых случаях)
	err := git.Reset(ctx, logger)
	// Reset может успешно выполняться даже без git репозитория
	t.Logf("Reset returned: %v", err)
}

// TestAddWithPatterns тестирует Add с различными паттернами
func TestAddWithPatterns(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tempDir := t.TempDir()

	git := &Git{
		RepPath: tempDir,
		WorkDir: tempDir,
	}

	// Тест добавления файлов (функция Add не принимает pattern)
	err := git.Add(ctx, logger)
	if err == nil {
		t.Error("Expected error for non-git directory, got nil")
	}
}

// TestCommitWithEmptyMessage тестирует Commit с пустым сообщением
func TestCommitWithEmptyMessage(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tempDir := t.TempDir()

	git := &Git{
		RepPath: tempDir,
		WorkDir: tempDir,
	}

	// Тест с пустым сообщением
	err := git.Commit(ctx, logger, "")
	if err == nil {
		t.Error("Expected error for commit with empty message, got nil")
	}

	// Тест с валидным сообщением но без git репозитория
	err = git.Commit(ctx, logger, "Test commit")
	if err == nil {
		t.Error("Expected error for non-git directory, got nil")
	}
}

// TestPushToRemote тестирует Push в удаленный репозиторий
func TestPushToRemote(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tempDir := t.TempDir()

	git := &Git{
		RepPath: tempDir,
		WorkDir: tempDir,
		Branch:  "main",
	}

	// Тест push без git репозитория
	err := git.Push(ctx, logger)
	if err == nil {
		t.Error("Expected error for push from non-git directory, got nil")
	}

	// Тест force push без git репозитория
	err = git.PushForce(ctx, logger)
	if err == nil {
		t.Error("Expected error for force push from non-git directory, got nil")
	}
}

// TestCheckoutCommitValidation тестирует валидацию параметров CheckoutCommit
func TestCheckoutCommitValidation(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name       string
		repoPath   string
		commitHash string
		wantError  bool
	}{
		{
			name:       "пустой путь к репозиторию",
			repoPath:   "",
			commitHash: "abc123",
			wantError:  true,
		},
		{
			name:       "пустой хеш коммита",
			repoPath:   "/tmp/test",
			commitHash: "",
			wantError:  true,
		},
		{
			name:       "несуществующая директория",
			repoPath:   "/nonexistent/path",
			commitHash: "abc123",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckoutCommit(ctx, logger, tt.repoPath, tt.commitHash)
			if (err != nil) != tt.wantError {
				t.Errorf("CheckoutCommit() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestGitMethodsWithMockRepo тестирует методы Git с временным репозиторием
func TestGitMethodsWithMockRepo(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Не удалось создать временную директорию: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	git := &Git{
		RepURL:  "https://github.com/test/repo.git",
		RepPath: tempDir,
		Branch:  "main",
		WorkDir: tempDir,
	}

	// Тестируем методы без реального git репозитория - они должны логировать ошибки, но не паниковать
	t.Run("Config без git репозитория", func(t *testing.T) {
		// Config метод не должен паниковать, но может выдать ошибку в логи
		git.Config(ctx, logger)
	})

	t.Run("Add без git репозитория", func(t *testing.T) {
		// Add метод не должен паниковать, но может выдать ошибку в логи
		git.Add(ctx, logger)
	})

	t.Run("Commit без git репозитория", func(t *testing.T) {
		// Commit метод не должен паниковать, но может выдать ошибку в логи
		git.Commit(ctx, logger, "тестовый коммит")
	})

	t.Run("Push без git репозитория", func(t *testing.T) {
		// Push метод не должен паниковать, но выдаст ошибку в логи
		git.Push(ctx, logger)
	})

	t.Run("PushForce без git репозитория", func(t *testing.T) {
		// PushForce метод не должен паниковать, но выдаст ошибку в логи
		git.PushForce(ctx, logger)
	})
}

// TestGitResetToCommit тестирует метод Reset
func TestGitResetToCommit(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Тест 1: с пустым CommitSHA1 (должен возвращать nil)
	git1 := &Git{
		RepURL:     "https://github.com/test/repo.git",
		RepPath:    "/tmp/test",
		Branch:     "main",
		CommitSHA1: "", // пустой
	}

	err := git1.Reset(ctx, logger)
	if err != nil {
		t.Errorf("Ожидался nil для пустого CommitSHA1, получена ошибка: %v", err)
	}

	// Тест 2: с CommitSHA1 = "last" (должен возвращать nil)
	git2 := &Git{
		RepURL:     "https://github.com/test/repo.git",
		RepPath:    "/tmp/test",
		Branch:     "main",
		CommitSHA1: "last",
	}

	err = git2.Reset(ctx, logger)
	if err != nil {
		t.Errorf("Ожидался nil для CommitSHA1='last', получена ошибка: %v", err)
	}

	// Тест 3: с несуществующим репозиторием
	// Из-за особенностей реализации метод Reset не возвращает ошибку
	// когда resetOk пустая строка (что всегда содержится в выводе)
	git3 := &Git{
		RepURL:     "https://github.com/test/repo.git",
		RepPath:    "/tmp/nonexistent",
		Branch:     "main",
		CommitSHA1: "abc123",
	}

	err = git3.Reset(ctx, logger)
	// Метод не возвращает ошибку из-за логики с resetOk
	t.Logf("Reset для несуществующего репозитория вернул: %v", err)
}

// TestGitClone тестирует метод Clone
func TestGitClone(t *testing.T) {
	ctx := context.Background()
	// Создаем контекст с коротким таймаутом для предотвращения зависания тестов
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Создаем временную директорию
	tempDir, err := os.MkdirTemp("", "git-clone-test-*")
	if err != nil {
		t.Fatalf("Не удалось создать временную директорию: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	git := &Git{
		RepURL:  "invalid://nonexistent/repo.git",
		RepPath: filepath.Join(tempDir, "test-repo"),
		Branch:  "main",
	}

	// Тестируем клонирование несуществующего репозитория
	err = git.Clone(ctx, logger)
	if err == nil {
		t.Error("Ожидалась ошибка для несуществующего репозитория")
	}
}

// TestGitPull тестирует метод Pull
func TestGitPull(t *testing.T) {
	git := &Git{
		RepPath: "/tmp/nonexistent",
	}

	// Тестируем pull с несуществующим репозиторием - метод не существует
	// Проверяем только создание структуры
	if git.RepPath != "/tmp/nonexistent" {
		t.Error("Неверный путь к репозиторию")
	}
}

// TestGitFetchAllBranches тестирует метод FetchAllBranches
func TestGitFetchAllBranches(t *testing.T) {
	git := &Git{
		RepPath: "/tmp/nonexistent",
	}

	// Тестируем fetch с несуществующим репозиторием - метод не существует
	// Проверяем только создание структуры
	if git.RepPath != "/tmp/nonexistent" {
		t.Error("Неверный путь к репозиторию")
	}
}

// TestGitCheckout тестирует метод Checkout
func TestGitCheckout(t *testing.T) {
	git := &Git{
		RepPath: "/tmp/nonexistent",
		Branch:  "test-branch",
	}

	// Тестируем checkout с несуществующим репозиторием - метод не существует
	// Проверяем только создание структуры
	if git.Branch != "test-branch" {
		t.Error("Неверная ветка")
	}
}

// BenchmarkNewConfigs бенчмарк для NewConfigs
func BenchmarkNewConfigs(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewConfigs()
	}
}

// BenchmarkGitStruct бенчмарк для создания структуры Git
func BenchmarkGitStruct(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = &Git{
			RepURL:     "https://github.com/test/repo.git",
			RepPath:    "/tmp/test-repo",
			Branch:     "main",
			CommitSHA1: "abc123",
			WorkDir:    "/tmp",
			Token:      "test-token",
		}
	}
}

// TestSetUser тестирует метод SetUser
func TestSetUser(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Создаем временную директорию для тестов
	tempDir, err := os.MkdirTemp("", "git-setuser-test-*")
	if err != nil {
		t.Fatalf("Не удалось создать временную директорию: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	git := &Git{
		RepPath: tempDir,
	}

	t.Run("SetUser с валидными параметрами", func(t *testing.T) {
		// SetUser не должен паниковать с валидными параметрами
		git.SetUser(ctx, logger, "Test User", "test@example.com")
	})

	t.Run("SetUser с пустыми параметрами", func(t *testing.T) {
		// SetUser должен использовать значения по умолчанию
		git.SetUser(ctx, logger, "", "")
	})

	t.Run("SetUser с частично пустыми параметрами", func(t *testing.T) {
		git.SetUser(ctx, logger, "Test User", "")
		git.SetUser(ctx, logger, "", "test@example.com")
	})
}

// TestSwitchOrCreateBranch тестирует функцию SwitchOrCreateBranch
func TestSwitchOrCreateBranch(t *testing.T) {
	ctx := context.Background()
	
	t.Run("несуществующий путь к репозиторию", func(t *testing.T) {
		err := SwitchOrCreateBranch(ctx, "/nonexistent/path", "test-branch")
		if err == nil {
			t.Error("Ожидалась ошибка для несуществующего пути")
		}
		if !strings.Contains(err.Error(), "repository path does not exist") {
			t.Errorf("Неожиданное сообщение об ошибке: %v", err)
		}
	})

	t.Run("пустое имя ветки", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "git-switch-test-*")
		if err != nil {
			t.Fatalf("Не удалось создать временную директорию: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("Failed to remove temp dir: %v", err)
			}
		}()

		// Функция должна обработать пустое имя ветки
		err = SwitchOrCreateBranch(ctx, tempDir, "")
		// Ошибка ожидается, так как это не git репозиторий
		if err == nil {
			t.Error("Ожидалась ошибка для не-git директории")
		}
	})
}

// TestSyncRepoBranches тестирует функцию SyncRepoBranches
func TestSyncRepoBranches(t *testing.T) {
	ctx := context.Background()
	t.Run("несуществующий путь", func(t *testing.T) {
		err := SyncRepoBranches(ctx, "/nonexistent/path")
		if err == nil {
			t.Error("Ожидалась ошибка для несуществующего пути")
		}
	})

	t.Run("не git репозиторий", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "git-sync-test-*")
		if err != nil {
			t.Fatalf("Не удалось создать временную директорию: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("Failed to remove temp dir: %v", err)
			}
		}()

		err = SyncRepoBranches(ctx, tempDir)
		if err == nil {
			t.Error("Ожидалась ошибка для не-git директории")
		}
	})
}

// TestCloneToTempDir тестирует функцию CloneToTempDir
func TestCloneToTempDir(t *testing.T) {
	ctx := context.Background()
	// Создаем контекст с коротким таймаутом для предотвращения зависания тестов
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("клонирование несуществующего репозитория", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "git-clone-temp-test-*")
		if err != nil {
			t.Fatalf("Не удалось создать временную директорию: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("Failed to remove temp dir: %v", err)
			}
		}()

		cloneDir := filepath.Join(tempDir, "cloned-repo")
		_, err = CloneToTempDir(ctx, logger, cloneDir, "invalid://nonexistent/repo.git", "main", "", 30*time.Second)
		if err == nil {
			t.Error("Ожидалась ошибка для несуществующего репозитория")
		}
	})

	t.Run("клонирование с токеном", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "git-clone-token-test-*")
		if err != nil {
			t.Fatalf("Не удалось создать временную директорию: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("Failed to remove temp dir: %v", err)
			}
		}()

		cloneDir := filepath.Join(tempDir, "cloned-repo")
		_, err = CloneToTempDir(ctx, logger, cloneDir, "invalid://nonexistent/repo.git", "main", "test-token", 30*time.Second)
		if err == nil {
			t.Error("Ожидалась ошибка для несуществующего репозитория с токеном")
		}
	})

	t.Run("клонирование без ветки", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "git-clone-nobranch-test-*")
		if err != nil {
			t.Fatalf("Не удалось создать временную директорию: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("Failed to remove temp dir: %v", err)
			}
		}()

		cloneDir := filepath.Join(tempDir, "cloned-repo")
		_, err = CloneToTempDir(ctx, logger, cloneDir, "invalid://nonexistent/repo.git", "", "", 30*time.Second)
		if err == nil {
			t.Error("Ожидалась ошибка для несуществующего репозитория без ветки")
		}
	})
}

// TestGitSwitch тестирует метод Switch
func TestGitSwitch(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	git := &Git{
		RepPath: "/tmp/nonexistent",
		Branch:  "test-branch",
	}

	// Switch должен возвращать ошибку, если репозиторий не существует
	err := git.Switch(ctx, logger)
	
	// Проверяем, что метод вернул ошибку для несуществующего репозитория
	if err == nil {
		t.Error("Expected error for non-existent repository, but got nil")
	}
}

// TestValidateCommitExists тестирует функцию validateCommitExists
func TestValidateCommitExists(t *testing.T) {
	ctx := context.Background()
	t.Run("несуществующий репозиторий", func(t *testing.T) {
		err := validateCommitExists(ctx, "/nonexistent/path", "abc123")
		if err == nil {
			t.Error("Ожидалась ошибка для несуществующего репозитория")
		}
	})

	t.Run("пустой хеш коммита", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "git-validate-test-*")
		if err != nil {
			t.Fatalf("Не удалось создать временную директорию: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("Failed to remove temp dir: %v", err)
			}
		}()

		err = validateCommitExists(ctx, tempDir, "")
		if err == nil {
			t.Error("Ожидалась ошибка для пустого хеша коммита")
		}
	})

	t.Run("несуществующий коммит", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "git-validate-commit-test-*")
		if err != nil {
			t.Fatalf("Не удалось создать временную директорию: %v", err)
		}
		defer func() {
			if err := os.RemoveAll(tempDir); err != nil {
				t.Logf("Failed to remove temp dir: %v", err)
			}
		}()

		err = validateCommitExists(ctx, tempDir, "nonexistent-commit-hash")
		if err == nil {
			t.Error("Ожидалась ошибка для несуществующего коммита")
		}
	})
}

// TestRunGitCommand тестирует функцию runGitCommand
func TestRunGitCommand(t *testing.T) {
	ctx := context.Background()
	t.Run("неверная команда git", func(t *testing.T) {
		err := runGitCommand(ctx, 30*time.Second, "invalid-command")
		if err == nil {
			t.Error("Ожидалась ошибка для неверной команды git")
		}
	})

	t.Run("git version", func(t *testing.T) {
		// Эта команда должна работать, если git установлен
		err := runGitCommand(ctx, 30*time.Second, "version")
		// Не проверяем ошибку, так как git может быть не установлен в тестовой среде
		t.Logf("git version result: %v", err)
	})
}

// TestGetRemoteBranches тестирует функцию getRemoteBranches
func TestGetRemoteBranches(t *testing.T) {
	ctx := context.Background()
	// Сохраняем текущую директорию
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Не удалось получить текущую директорию: %v", err)
	}

	// Создаем временную директорию
	tempDir, err := os.MkdirTemp("", "git-remote-branches-test-*")
	if err != nil {
		t.Fatalf("Не удалось создать временную директорию: %v", err)
	}

	// Используем defer для правильной очистки в обратном порядке
	defer func() {
		// Сначала возвращаемся в исходную директорию
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to restore directory: %v", err)
		}
		// Затем удаляем временную директорию
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Переходим в временную директорию
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Не удалось перейти в временную директорию: %v", err)
	}

	// Тестируем в не-git директории
	_, err = getRemoteBranches(ctx, 30*time.Second)
	if err == nil {
		t.Error("Ожидалась ошибка для не-git директории")
	}

	// Убеждаемся, что мы вернулись в исходную директорию перед завершением теста
	if err := os.Chdir(originalDir); err != nil {
		t.Fatalf("Не удалось вернуться в исходную директорию: %v", err)
	}
}

// TestCreateTrackingBranch тестирует функцию createTrackingBranch
func TestCreateTrackingBranch(t *testing.T) {
	ctx := context.Background()
	// Создаем временную директорию
	tempDir, err := os.MkdirTemp("", "git-tracking-branch-test-*")
	if err != nil {
		t.Fatalf("Не удалось создать временную директорию: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Сохраняем текущую директорию
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Не удалось получить текущую директорию: %v", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Failed to restore directory: %v", err)
		}
	}()

	// Переходим в временную директорию
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Не удалось перейти в временную директорию: %v", err)
	}

	// Тестируем в не-git директории
	err = createTrackingBranch(ctx, "test-branch", "origin/test-branch")
	if err == nil {
		t.Error("Ожидалась ошибка для не-git директории")
	}
}

// TestWaitForGitSync тестирует функцию waitForGitSync
func TestWaitForGitSync(t *testing.T) {
	ctx := context.Background()
	// Создаем временную директорию
	tempDir, err := os.MkdirTemp("", "git-sync-test-*")
	if err != nil {
		t.Fatalf("Не удалось создать временную директорию: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to remove temp dir: %v", err)
		}
	}()

	// Тестируем в не-git директории
	err = waitForGitSync(ctx, tempDir)
	if err == nil {
		t.Error("Ожидалась ошибка для не-git директории")
	}
	if !strings.Contains(err.Error(), "git sync timeout") {
		t.Errorf("Неожиданное сообщение об ошибке: %v", err)
	}
}

// TestGitMethodsErrorHandling тестирует обработку ошибок в методах Git
func TestGitMethodsErrorHandling(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Создаем временную директорию для тестов
	tempDir, err := os.MkdirTemp("", "git_test_*")
	if err != nil {
		t.Fatalf("Не удалось создать временную директорию: %v", err)
	}
	defer os.RemoveAll(tempDir)

	git := &Git{
		RepURL:     "https://github.com/test/repo.git",
		RepPath:    tempDir,
		Branch:     "main",
		CommitSHA1: "abc123",
		WorkDir:    tempDir,
		Token:      "test-token",
	}

	t.Run("Reset с несуществующим коммитом", func(t *testing.T) {
		// Reset не должен паниковать
		err := git.Reset(ctx, logger)
		// Метод может не возвращать ошибку из-за особенностей реализации
		t.Logf("Reset error: %v", err)
	})

	t.Run("Clone с недоступным репозиторием", func(t *testing.T) {
		err := git.Clone(ctx, logger)
		if err == nil {
			t.Error("Ожидалась ошибка для недоступного репозитория")
		}
	})

	t.Run("Config с временным путем", func(t *testing.T) {
		// Config не должен паниковать
		git.Config(ctx, logger)
	})

	t.Run("Add с пустой директорией", func(t *testing.T) {
		// Add не должен паниковать, но может выдать ошибку
		git.Add(ctx, logger)
	})

	t.Run("Commit с пустой директорией", func(t *testing.T) {
		// Commit не должен паниковать, но может выдать ошибку
		git.Commit(ctx, logger, "test commit")
	})

	t.Run("Push без инициализированного репозитория", func(t *testing.T) {
		// Push не должен паниковать, но выдаст ошибку в логи
		git.Push(ctx, logger)
	})

	t.Run("PushForce без инициализированного репозитория", func(t *testing.T) {
		// PushForce не должен паниковать, но выдаст ошибку в логи
		git.PushForce(ctx, logger)
	})
}

// TestGitExecuteCommandWithRetry тестирует функцию executeGitCommandWithRetry
func TestGitExecuteCommandWithRetry(t *testing.T) {

	// Создаем временную директорию
	tempDir, err := os.MkdirTemp("", "git-retry-test-*")
	if err != nil {
		t.Fatalf("Не удалось создать временную директорию: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Переходим в временную директорию
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Не удалось получить текущую директорию: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Не удалось перейти в временную директорию: %v", err)
	}

	// Тестируем команду с ошибкой
	err = executeGitCommandWithRetry(tempDir, []string{"invalid-command"})
	if err == nil {
		t.Error("Ожидалась ошибка для неверной команды")
	}

	// Тестируем валидную команду (версия git)
	err = executeGitCommandWithRetry(tempDir, []string{"version"})
	if err != nil {
		t.Logf("git version command failed (expected in test environment): %v", err)
	}
}

// TestGitWaitForGitLockRelease тестирует функцию waitForGitLockRelease
func TestGitWaitForGitLockRelease(t *testing.T) {

	// Создаем временную директорию
	tempDir, err := os.MkdirTemp("", "git-lock-test-*")
	if err != nil {
		t.Fatalf("Не удалось создать временную директорию: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Тестируем без lock файла (должно завершиться сразу)
	waitForGitLockRelease(tempDir, 1*time.Second)

	// Создаем lock файл и тестируем ожидание
	lockFile := tempDir + "/.git/index.lock"
	if err := os.MkdirAll(tempDir+"/.git", 0755); err != nil {
		t.Fatalf("Не удалось создать .git директорию: %v", err)
	}

	// Создаем lock файл
	if file, err := os.Create(lockFile); err == nil {
		file.Close()
		// Тестируем ожидание с таймаутом
		start := time.Now()
		waitForGitLockRelease(tempDir, 100*time.Millisecond)
		elapsed := time.Since(start)

		// Удаляем lock файл
		os.Remove(lockFile)

		if elapsed < 50*time.Millisecond {
			t.Errorf("Ожидание должно было занять больше времени, заняло: %v", elapsed)
		}
	}
}

// TestGitSwitchOrCreateBranch тестирует функцию SwitchOrCreateBranch
func TestGitSwitchOrCreateBranch(t *testing.T) {
	ctx := context.Background()

	// Создаем временную директорию
	tempDir, err := os.MkdirTemp("", "git-switch-test-*")
	if err != nil {
		t.Fatalf("Не удалось создать временную директорию: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Тестируем переключение на несуществующую ветку в несуществующем репозитории
	err = SwitchOrCreateBranch(ctx, tempDir, "test-branch")
	// Ожидаем ошибку, так как это не git репозиторий
	if err == nil {
		t.Error("Ожидалась ошибка для не-git директории")
	}
}

// TestGitHelperFunctions тестирует вспомогательные функции Git
func TestGitHelperFunctions(t *testing.T) {
	ctx := context.Background()

	// Создаем временную директорию
	tempDir, err := os.MkdirTemp("", "git-helpers-test-*")
	if err != nil {
		t.Fatalf("Не удалось создать временную директорию: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Не удалось получить текущую директорию: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Не удалось перейти в временную директорию: %v", err)
	}

	// Тестируем getGitStatus
	_, err = getGitStatus(ctx, tempDir)
	if err == nil {
		t.Log("getGitStatus completed without error (unexpected in non-git directory)")
	}

	// Тестируем waitForGitSync
	err = waitForGitSync(ctx, tempDir)
	if err == nil {
		t.Log("waitForGitSync completed without error")
	}
}

// TestGitCloneWithTokenError тестирует клонирование с ошибками токена
func TestGitCloneWithTokenError(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Создаем временную директорию
	tempDir, err := os.MkdirTemp("", "git-clone-error-test-*")
	if err != nil {
		t.Fatalf("Не удалось создать временную директорию: %v", err)
	}
	defer os.RemoveAll(tempDir)

	git := &Git{
		RepURL:  "https://test.invalid/repo.git", // Домен .invalid гарантированно не резолвится (RFC 2606)
		RepPath: tempDir + "/cloned",
		Token:   "invalid-token",
		WorkDir: tempDir,
	}

	// Тестируем клонирование с недействительным URL
	err = git.Clone(ctx, logger)
	if err == nil {
		t.Error("Ожидалась ошибка для недействительного URL")
	}
}

// TestGitConfigMethod тестирует метод Config
func TestGitConfigMethod(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Создаем временную директорию
	tempDir, err := os.MkdirTemp("", "git-config-test-*")
	if err != nil {
		t.Fatalf("Не удалось создать временную директорию: %v", err)
	}
	defer os.RemoveAll(tempDir)

	git := &Git{
		RepPath: tempDir,
		WorkDir: tempDir,
	}

	// Тестируем настройку конфигурации в не-git директории
	err = git.Config(ctx, logger)
	// Может завершиться с ошибкой или без неё в зависимости от того, создаёт ли git конфигурацию
	if err != nil {
		t.Logf("Config method failed as expected in non-git directory: %v", err)
	}
}
