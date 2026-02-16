// Package shared содержит общую логику для SonarQube команд.
package shared

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kargones/apk-ci/internal/adapter/gitea"
	"github.com/Kargones/apk-ci/internal/adapter/gitea/giteatest"
	"github.com/Kargones/apk-ci/internal/adapter/sonarqube"
	"github.com/Kargones/apk-ci/internal/adapter/sonarqube/sonarqubetest"
)

// TestHasRelevantChangesInCommit проверяет определение релевантных изменений в коммите.
func TestHasRelevantChangesInCommit(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		projectStructure []string
		commitFiles    []gitea.CommitFile
		expectedResult bool
		expectedError  bool
	}{
		{
			name:             "changes in main config",
			projectStructure: []string{"MyConfig"},
			commitFiles: []gitea.CommitFile{
				{Filename: "MyConfig/src/Module.bsl"},
			},
			expectedResult: true,
			expectedError:  false,
		},
		{
			name:             "changes in extension",
			projectStructure: []string{"MyConfig", "Extension1"},
			commitFiles: []gitea.CommitFile{
				{Filename: "MyConfig.Extension1/src/Module.bsl"},
			},
			expectedResult: true,
			expectedError:  false,
		},
		{
			name:             "no changes in config dirs",
			projectStructure: []string{"MyConfig"},
			commitFiles: []gitea.CommitFile{
				{Filename: "README.md"},
				{Filename: "docs/guide.md"},
			},
			expectedResult: false,
			expectedError:  false,
		},
		{
			name:             "empty project structure - all changes relevant",
			projectStructure: []string{},
			commitFiles: []gitea.CommitFile{
				{Filename: "any/file.txt"},
			},
			expectedResult: true,
			expectedError:  false,
		},
		{
			name:             "mixed changes - some relevant",
			projectStructure: []string{"Config"},
			commitFiles: []gitea.CommitFile{
				{Filename: "README.md"},
				{Filename: "Config/src/Module.bsl"},
			},
			expectedResult: true,
			expectedError:  false,
		},
		{
			name:             "file in root of config dir",
			projectStructure: []string{"Config"},
			commitFiles: []gitea.CommitFile{
				{Filename: "Config/Module.bsl"}, // Файл в корне config dir
			},
			expectedResult: true, // Любой файл с префиксом "Config/" считается релевантным
			expectedError:  false,
		},
		{
			name:             "file without slash after config dir",
			projectStructure: []string{"Config"},
			commitFiles: []gitea.CommitFile{
				{Filename: "ConfigFile.txt"}, // Имя начинается с Config, но нет /
			},
			expectedResult: false, // Не соответствует Config/
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &giteatest.MockClient{
				AnalyzeProjectStructureFunc: func(ctx context.Context, branch string) ([]string, error) {
					return tt.projectStructure, nil
				},
				GetCommitFilesFunc: func(ctx context.Context, sha string) ([]gitea.CommitFile, error) {
					return tt.commitFiles, nil
				},
			}

			result, err := HasRelevantChangesInCommit(ctx, mock, "main", "abc123")

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

// TestHasRelevantChangesInCommit_APIErrors проверяет обработку ошибок API.
func TestHasRelevantChangesInCommit_APIErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("AnalyzeProjectStructure error", func(t *testing.T) {
		mock := &giteatest.MockClient{
			AnalyzeProjectStructureFunc: func(ctx context.Context, branch string) ([]string, error) {
				return nil, errors.New("API error")
			},
		}

		_, err := HasRelevantChangesInCommit(ctx, mock, "main", "abc123")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка анализа структуры проекта")
	})

	t.Run("GetCommitFiles error", func(t *testing.T) {
		mock := &giteatest.MockClient{
			AnalyzeProjectStructureFunc: func(ctx context.Context, branch string) ([]string, error) {
				return []string{"Config"}, nil
			},
			GetCommitFilesFunc: func(ctx context.Context, sha string) ([]gitea.CommitFile, error) {
				return nil, errors.New("API error")
			},
		}

		_, err := HasRelevantChangesInCommit(ctx, mock, "main", "abc123")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка получения файлов коммита")
	})
}

// TestWaitForAnalysisCompletion проверяет ожидание завершения анализа.
func TestWaitForAnalysisCompletion(t *testing.T) {
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("immediate success", func(t *testing.T) {
		mock := &sonarqubetest.MockClient{
			GetAnalysisStatusFunc: func(ctx context.Context, taskID string) (*sonarqube.AnalysisStatus, error) {
				return &sonarqube.AnalysisStatus{
					Status:     "SUCCESS",
					AnalysisID: "analysis-123",
				}, nil
			},
		}

		status, err := WaitForAnalysisCompletion(ctx, mock, "task-123", log)
		require.NoError(t, err)
		assert.Equal(t, "SUCCESS", status.Status)
		assert.Equal(t, "analysis-123", status.AnalysisID)
	})

	t.Run("immediate failure", func(t *testing.T) {
		mock := &sonarqubetest.MockClient{
			GetAnalysisStatusFunc: func(ctx context.Context, taskID string) (*sonarqube.AnalysisStatus, error) {
				return &sonarqube.AnalysisStatus{
					Status:       "FAILED",
					ErrorMessage: "Analysis failed",
				}, nil
			},
		}

		status, err := WaitForAnalysisCompletion(ctx, mock, "task-123", log)
		require.NoError(t, err)
		assert.Equal(t, "FAILED", status.Status)
	})

	t.Run("canceled status", func(t *testing.T) {
		mock := &sonarqubetest.MockClient{
			GetAnalysisStatusFunc: func(ctx context.Context, taskID string) (*sonarqube.AnalysisStatus, error) {
				return &sonarqube.AnalysisStatus{
					Status: "CANCELED",
				}, nil
			},
		}

		status, err := WaitForAnalysisCompletion(ctx, mock, "task-123", log)
		require.NoError(t, err)
		assert.Equal(t, "CANCELED", status.Status)
	})

	t.Run("API error", func(t *testing.T) {
		mock := &sonarqubetest.MockClient{
			GetAnalysisStatusFunc: func(ctx context.Context, taskID string) (*sonarqube.AnalysisStatus, error) {
				return nil, errors.New("connection failed")
			},
		}

		_, err := WaitForAnalysisCompletion(ctx, mock, "task-123", log)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка получения статуса анализа")
	})

	t.Run("unknown status", func(t *testing.T) {
		mock := &sonarqubetest.MockClient{
			GetAnalysisStatusFunc: func(ctx context.Context, taskID string) (*sonarqube.AnalysisStatus, error) {
				return &sonarqube.AnalysisStatus{
					Status: "UNKNOWN_STATUS",
				}, nil
			},
		}

		_, err := WaitForAnalysisCompletion(ctx, mock, "task-123", log)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "неизвестный статус анализа")
	})

	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Отменяем сразу

		mock := &sonarqubetest.MockClient{
			GetAnalysisStatusFunc: func(ctx context.Context, taskID string) (*sonarqube.AnalysisStatus, error) {
				return &sonarqube.AnalysisStatus{Status: "PENDING"}, nil
			},
		}

		_, err := WaitForAnalysisCompletion(ctx, mock, "task-123", log)
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("success after pending", func(t *testing.T) {
		callCount := 0
		mock := &sonarqubetest.MockClient{
			GetAnalysisStatusFunc: func(ctx context.Context, taskID string) (*sonarqube.AnalysisStatus, error) {
				callCount++
				if callCount < 3 {
					return &sonarqube.AnalysisStatus{Status: "PENDING"}, nil
				}
				return &sonarqube.AnalysisStatus{Status: "SUCCESS", AnalysisID: "analysis-123"}, nil
			},
		}

		// Используем короткий контекст для быстрого теста
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		status, err := WaitForAnalysisCompletion(ctx, mock, "task-123", log)
		require.NoError(t, err)
		assert.Equal(t, "SUCCESS", status.Status)
		assert.GreaterOrEqual(t, callCount, 3)
	})
}
