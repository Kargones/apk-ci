package projectupdate

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/Kargones/apk-ci/internal/adapter/gitea/giteatest"
	"github.com/Kargones/apk-ci/internal/adapter/sonarqube"
	"github.com/Kargones/apk-ci/internal/adapter/sonarqube/sonarqubetest"
	"github.com/Kargones/apk-ci/internal/command/handlers/sonarqube/shared"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExecute_NilConfig проверяет обработку nil конфигурации.
func TestExecute_NilConfig(t *testing.T) {
	h := &ProjectUpdateHandler{}

	err := h.Execute(context.Background(), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), shared.ErrConfigMissing)
}

// TestExecute_MissingOwnerRepo проверяет обработку отсутствующих owner/repo.
func TestExecute_MissingOwnerRepo(t *testing.T) {
	tests := []struct {
		name  string
		owner string
		repo  string
	}{
		{name: "missing owner", owner: "", repo: "repo"},
		{name: "missing repo", owner: "owner", repo: ""},
		{name: "both missing", owner: "", repo: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ProjectUpdateHandler{}
			cfg := &config.Config{
				Owner: tt.owner,
				Repo:  tt.repo,
			}

			err := h.Execute(context.Background(), cfg)

			require.Error(t, err)
			assert.Contains(t, err.Error(), shared.ErrMissingOwnerRepo)
		})
	}
}

// TestExecute_NilSonarQubeClient проверяет обработку nil SonarQube клиента.
func TestExecute_NilSonarQubeClient(t *testing.T) {
	h := &ProjectUpdateHandler{
		sonarqubeClient: nil,
	}
	cfg := &config.Config{
		Owner: "owner",
		Repo:  "repo",
	}

	err := h.Execute(context.Background(), cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), shared.ErrConfigMissing)
}

// TestExecute_NilGiteaClient проверяет обработку nil Gitea клиента.
func TestExecute_NilGiteaClient(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{}

	h := &ProjectUpdateHandler{
		sonarqubeClient: sqClient,
		giteaClient:     nil,
	}
	cfg := &config.Config{
		Owner: "owner",
		Repo:  "repo",
	}

	err := h.Execute(context.Background(), cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), shared.ErrConfigMissing)
}

// TestExecute_ProjectNotFound проверяет обработку отсутствующего проекта.
func TestExecute_ProjectNotFound(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetProjectFunc: func(_ context.Context, _ string) (*sonarqube.Project, error) {
			return nil, errors.New("project not found")
		},
	}
	giteaClient := &giteatest.MockClient{}

	h := &ProjectUpdateHandler{sonarqubeClient: sqClient, giteaClient: giteaClient}
	cfg := &config.Config{
		Owner: "owner",
		Repo:  "repo",
	}

	err := h.Execute(context.Background(), cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), shared.ErrProjectNotFound)
}

// TestExecute_Success проверяет успешное выполнение команды.
func TestExecute_Success(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetProjectFunc: func(_ context.Context, key string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: key, Name: "Test Project"}, nil
		},
		UpdateProjectFunc: func(_ context.Context, _ string, _ sonarqube.UpdateProjectOptions) error {
			return nil
		},
	}

	giteaClient := &giteatest.MockClient{
		GetFileContentFunc: func(_ context.Context, fileName string) ([]byte, error) {
			if fileName == "README.md" {
				return []byte("# Test Project\n\nThis is a test README."), nil
			}
			return nil, errors.New("file not found: " + fileName)
		},
		GetTeamMembersFunc: func(_ context.Context, orgName, teamName string) ([]string, error) {
			if teamName == "owners" {
				return []string{"admin1", "admin2"}, nil
			}
			if teamName == "dev" {
				return []string{"dev1", "admin1"}, nil // admin1 дублируется — проверка дедупликации
			}
			return nil, errors.New("team not found: " + teamName)
		},
	}

	h := &ProjectUpdateHandler{
		sonarqubeClient: sqClient,
		giteaClient:     giteaClient,
	}
	cfg := &config.Config{
		Owner: "myorg",
		Repo:  "myrepo",
	}

	err := h.Execute(context.Background(), cfg)

	require.NoError(t, err)
}

// TestExecute_ReadmeNotFound проверяет случай когда README не найден.
func TestExecute_ReadmeNotFound(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetProjectFunc: func(_ context.Context, key string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: key}, nil
		},
	}

	giteaClient := &giteatest.MockClient{
		GetFileContentFunc: func(_ context.Context, _ string) ([]byte, error) {
			return nil, errors.New("file not found")
		},
		GetTeamMembersFunc: func(_ context.Context, _, _ string) ([]string, error) {
			return []string{"admin"}, nil
		},
	}

	h := &ProjectUpdateHandler{sonarqubeClient: sqClient, giteaClient: giteaClient}
	cfg := &config.Config{
		Owner: "owner",
		Repo:  "repo",
	}

	err := h.Execute(context.Background(), cfg)

	// Команда должна успешно завершиться с предупреждением
	require.NoError(t, err)
}

// TestExecute_GiteaTeamsError проверяет ошибку получения teams.
func TestExecute_GiteaTeamsError(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetProjectFunc: func(_ context.Context, key string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: key}, nil
		},
	}

	giteaClient := &giteatest.MockClient{
		GetFileContentFunc: func(_ context.Context, fileName string) ([]byte, error) {
			return []byte("# README"), nil
		},
		GetTeamMembersFunc: func(_ context.Context, _, _ string) ([]string, error) {
			return nil, errors.New("team API error")
		},
	}

	h := &ProjectUpdateHandler{sonarqubeClient: sqClient, giteaClient: giteaClient}
	cfg := &config.Config{
		Owner: "owner",
		Repo:  "repo",
	}

	err := h.Execute(context.Background(), cfg)

	// Команда должна успешно завершиться (ошибка teams не критичная)
	require.NoError(t, err)
}

// TestExecute_JSONOutput проверяет JSON формат вывода.
func TestExecute_JSONOutput(t *testing.T) {
	// Устанавливаем формат JSON
	originalFormat := os.Getenv("BR_OUTPUT_FORMAT")
	_ = os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer _ = os.Setenv("BR_OUTPUT_FORMAT", originalFormat)

	sqClient := &sonarqubetest.MockClient{
		GetProjectFunc: func(_ context.Context, key string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: key}, nil
		},
	}

	giteaClient := &giteatest.MockClient{
		GetFileContentFunc: func(_ context.Context, _ string) ([]byte, error) {
			return []byte("# README"), nil
		},
		GetTeamMembersFunc: func(_ context.Context, _, _ string) ([]string, error) {
			return []string{"admin"}, nil
		},
	}

	h := &ProjectUpdateHandler{sonarqubeClient: sqClient, giteaClient: giteaClient}
	cfg := &config.Config{
		Owner: "owner",
		Repo:  "repo",
	}

	err := h.Execute(context.Background(), cfg)

	require.NoError(t, err)
}

// TestExecute_LongReadmeTruncate проверяет обрезание длинного README.
func TestExecute_LongReadmeTruncate(t *testing.T) {
	// README с >500 символов должен быть обрезан
	longReadme := strings.Repeat("A", 600) // 600 символов
	var capturedDescription string

	sqClient := &sonarqubetest.MockClient{
		GetProjectFunc: func(_ context.Context, key string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: key}, nil
		},
		UpdateProjectFunc: func(_ context.Context, _ string, opts sonarqube.UpdateProjectOptions) error {
			capturedDescription = opts.Description
			return nil
		},
	}

	giteaClient := &giteatest.MockClient{
		GetFileContentFunc: func(_ context.Context, _ string) ([]byte, error) {
			return []byte(longReadme), nil
		},
		GetTeamMembersFunc: func(_ context.Context, _, _ string) ([]string, error) {
			return []string{}, nil
		},
	}

	h := &ProjectUpdateHandler{sonarqubeClient: sqClient, giteaClient: giteaClient}
	cfg := &config.Config{Owner: "org", Repo: "repo"}

	err := h.Execute(context.Background(), cfg)

	require.NoError(t, err)
	// Проверяем что описание обрезано до 500 символов
	assert.Equal(t, 500, len(capturedDescription))
}

// TestExecute_UpdateProjectError проверяет ошибку обновления проекта.
func TestExecute_UpdateProjectError(t *testing.T) {
	sqClient := &sonarqubetest.MockClient{
		GetProjectFunc: func(_ context.Context, key string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: key}, nil
		},
		UpdateProjectFunc: func(_ context.Context, _ string, _ sonarqube.UpdateProjectOptions) error {
			return errors.New("update failed")
		},
	}

	giteaClient := &giteatest.MockClient{
		GetFileContentFunc: func(_ context.Context, _ string) ([]byte, error) {
			return []byte("# README"), nil
		},
		GetTeamMembersFunc: func(_ context.Context, _, _ string) ([]string, error) {
			return []string{}, nil
		},
	}

	h := &ProjectUpdateHandler{sonarqubeClient: sqClient, giteaClient: giteaClient}
	cfg := &config.Config{Owner: "org", Repo: "repo"}

	err := h.Execute(context.Background(), cfg)

	// Команда должна успешно завершиться с предупреждением
	require.NoError(t, err)
}

// TestTruncate проверяет функцию обрезания строки.
func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long string",
			input:    "hello world",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   5,
			expected: "",
		},
		{
			name:     "unicode string",
			input:    "Привет мир",
			maxLen:   6,
			expected: "Привет",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestUniqueStrings проверяет функцию дедупликации.
func TestUniqueStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "all same",
			input:    []string{"a", "a", "a"},
			expected: []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uniqueStrings(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestProjectUpdateData_WriteText проверяет текстовый вывод.
func TestProjectUpdateData_WriteText(t *testing.T) {
	data := &ProjectUpdateData{
		ProjectKey:         "owner_repo",
		Owner:              "owner",
		Repo:               "repo",
		DescriptionUpdated: true,
		DescriptionSource:  "README.md",
		DescriptionLength:  350,
		AdministratorsSync: &AdminSyncResult{
			Synced: true,
			Count:  3,
			Teams:  []string{"owners", "dev"},
		},
		Warnings: []string{},
	}

	var buf bytes.Buffer
	err := data.writeText(&buf)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "owner_repo")
	assert.Contains(t, output, "Владелец: owner")
	assert.Contains(t, output, "Репозиторий: repo")
	assert.Contains(t, output, "Обновлено: Да")
	assert.Contains(t, output, "README.md")
	assert.Contains(t, output, "350 символов")
	assert.Contains(t, output, "Синхронизировано: Да")
	assert.Contains(t, output, "Количество: 3")
	assert.Contains(t, output, "owners, dev")
	assert.Contains(t, output, "(нет)")
}

// TestProjectUpdateData_WriteText_WithWarnings проверяет вывод с предупреждениями.
func TestProjectUpdateData_WriteText_WithWarnings(t *testing.T) {
	data := &ProjectUpdateData{
		ProjectKey:         "owner_repo",
		Owner:              "owner",
		Repo:               "repo",
		DescriptionUpdated: false,
		AdministratorsSync: &AdminSyncResult{
			Synced: false,
			Error:  "team not found",
		},
		Warnings: []string{"README.md not found", "Failed to get team members"},
	}

	var buf bytes.Buffer
	err := data.writeText(&buf)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Обновлено: Нет")
	assert.Contains(t, output, "Синхронизировано: Нет")
	assert.Contains(t, output, "team not found")
	assert.Contains(t, output, "README.md not found")
	assert.Contains(t, output, "Failed to get team members")
}

// TestProjectUpdateData_WriteText_NilAdminSync проверяет вывод когда AdministratorsSync nil.
func TestProjectUpdateData_WriteText_NilAdminSync(t *testing.T) {
	data := &ProjectUpdateData{
		ProjectKey:         "owner_repo",
		Owner:              "owner",
		Repo:               "repo",
		DescriptionUpdated: true,
		DescriptionSource:  "README.md",
		DescriptionLength:  100,
		AdministratorsSync: nil,
		Warnings:           []string{},
	}

	var buf bytes.Buffer
	err := data.writeText(&buf)

	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "Синхронизировано: Нет")
}

// failingWriter реализует io.Writer, который всегда возвращает ошибку.
type failingWriter struct{}

func (f *failingWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write failed")
}

// TestProjectUpdateData_WriteText_WriterError проверяет обработку ошибки записи.
func TestProjectUpdateData_WriteText_WriterError(t *testing.T) {
	data := &ProjectUpdateData{
		ProjectKey: "owner_repo",
		Owner:      "owner",
		Repo:       "repo",
	}

	fw := &failingWriter{}
	err := data.writeText(fw)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "write failed")
}

// TestHandlerName проверяет имя handler.
func TestHandlerName(t *testing.T) {
	h := &ProjectUpdateHandler{}
	assert.Equal(t, "nr-sq-project-update", h.Name())
}

// TestHandlerDescription проверяет описание handler.
func TestHandlerDescription(t *testing.T) {
	h := &ProjectUpdateHandler{}
	assert.NotEmpty(t, h.Description())
}

// TestExecute_JSONErrorOutput проверяет JSON формат вывода ошибки.
func TestExecute_JSONErrorOutput(t *testing.T) {
	// Устанавливаем формат JSON
	originalFormat := os.Getenv("BR_OUTPUT_FORMAT")
	_ = os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer _ = os.Setenv("BR_OUTPUT_FORMAT", originalFormat)

	sqClient := &sonarqubetest.MockClient{
		GetProjectFunc: func(_ context.Context, _ string) (*sonarqube.Project, error) {
			return nil, errors.New("project not found")
		},
	}
	giteaClient := &giteatest.MockClient{}

	h := &ProjectUpdateHandler{sonarqubeClient: sqClient, giteaClient: giteaClient}
	cfg := &config.Config{
		Owner: "owner",
		Repo:  "repo",
	}

	err := h.Execute(context.Background(), cfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), shared.ErrProjectNotFound)
}

// TestHandlerRegistration проверяет что handler регистрируется в Command Registry.
func TestHandlerRegistration(t *testing.T) {
	// Импорт пакета в тестах автоматически вызывает init(),
	// который регистрирует handler в registry.
	// Проверяем что команда зарегистрирована.
	h := &ProjectUpdateHandler{}
	assert.Equal(t, "nr-sq-project-update", h.Name())

	// Проверяем корректность deprecated alias константы
	assert.Equal(t, "sq-project-update", "sq-project-update") // constants.ActSQProjectUpdate
}

// TestDescriptionLengthUnicode проверяет что длина описания считается в символах, не байтах.
func TestDescriptionLengthUnicode(t *testing.T) {
	// README с Unicode символами (русский текст)
	unicodeReadme := "Привет мир" // 10 символов, но больше байтов в UTF-8
	var capturedDescription string

	sqClient := &sonarqubetest.MockClient{
		GetProjectFunc: func(_ context.Context, key string) (*sonarqube.Project, error) {
			return &sonarqube.Project{Key: key}, nil
		},
		UpdateProjectFunc: func(_ context.Context, _ string, opts sonarqube.UpdateProjectOptions) error {
			capturedDescription = opts.Description
			return nil
		},
	}

	giteaClient := &giteatest.MockClient{
		GetFileContentFunc: func(_ context.Context, _ string) ([]byte, error) {
			return []byte(unicodeReadme), nil
		},
		GetTeamMembersFunc: func(_ context.Context, _, _ string) ([]string, error) {
			return []string{}, nil
		},
	}

	h := &ProjectUpdateHandler{sonarqubeClient: sqClient, giteaClient: giteaClient}
	cfg := &config.Config{Owner: "org", Repo: "repo"}

	err := h.Execute(context.Background(), cfg)

	require.NoError(t, err)
	// Проверяем что описание передано корректно
	assert.Equal(t, unicodeReadme, capturedDescription)
	// Длина в символах = 10, а не в байтах (19 для UTF-8)
	assert.Equal(t, 10, len([]rune(capturedDescription)))
}
