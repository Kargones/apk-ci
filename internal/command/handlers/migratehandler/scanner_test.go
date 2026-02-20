package migratehandler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScanFile_TableDriven — table-driven тесты для scanFile().
// Покрывает все форматы: env-блок, step-env, inline-run, кавычки, уже-NR, non-BR_COMMAND.
func TestScanFile_TableDriven(t *testing.T) {
	legacyToNR := map[string]string{
		"service-mode-enable":  "nr-service-mode-enable",
		"service-mode-disable": "nr-service-mode-disable",
		"service-mode-status":  "nr-service-mode-status",
		"dbrestore":            "nr-dbrestore",
		"sq-scan-branch":       "nr-sq-scan-branch",
		"convert":              "nr-convert",
	}

	tests := []struct {
		name         string
		content      string
		wantCount    int
		wantCommands []string // ожидаемые OldCommand значения
	}{
		{
			name: "env-блок без кавычек",
			content: `env:
  BR_COMMAND: service-mode-enable
  BR_INFOBASE_NAME: MyBase`,
			wantCount:    1,
			wantCommands: []string{"service-mode-enable"},
		},
		{
			name: "env-блок с двойными кавычками",
			content: `env:
  BR_COMMAND: "sq-scan-branch"`,
			wantCount:    1,
			wantCommands: []string{"sq-scan-branch"},
		},
		{
			name: "env-блок с одинарными кавычками",
			content: `env:
  BR_COMMAND: 'convert'`,
			wantCount:    1,
			wantCommands: []string{"convert"},
		},
		{
			name: "step-env блок",
			content: `- name: Включить сервисный режим
  env:
    BR_COMMAND: service-mode-enable`,
			wantCount:    1,
			wantCommands: []string{"service-mode-enable"},
		},
		{
			name: "inline в run",
			content: `- name: Запуск команды
  run: BR_COMMAND=dbrestore ./apk-ci`,
			wantCount:    1,
			wantCommands: []string{"dbrestore"},
		},
		{
			name: "уже NR-команда — не заменять",
			content: `env:
  BR_COMMAND: nr-service-mode-enable`,
			wantCount:    0,
			wantCommands: nil,
		},
		{
			name:         "без BR_COMMAND — quick skip",
			content:      `name: test workflow`,
			wantCount:    0,
			wantCommands: nil,
		},
		{
			name: "комментарий с BR_COMMAND — пропуск",
			content: `# BR_COMMAND: service-mode-enable
env:
  BR_COMMAND: dbrestore`,
			wantCount:    1,
			wantCommands: []string{"dbrestore"},
		},
		{
			name: "extension-publish — нет NR-аналога",
			content: `env:
  BR_COMMAND: extension-publish`,
			wantCount:    0,
			wantCommands: nil,
		},
		{
			name: "множественные замены в одном файле",
			content: `jobs:
  deploy:
    steps:
      - env:
          BR_COMMAND: service-mode-enable
      - env:
          BR_COMMAND: dbrestore
      - env:
          BR_COMMAND: service-mode-disable`,
			wantCount:    3,
			wantCommands: []string{"service-mode-enable", "dbrestore", "service-mode-disable"},
		},
		{
			name: "inline с пробелами и контекстом",
			content: `- name: Run
  run: |
    export BR_COMMAND=service-mode-status
    ./apk-ci`,
			wantCount:    1,
			wantCommands: []string{"service-mode-status"},
		},
		{
			name: "env-блок с inline-комментарием (без кавычек)",
			content: `env:
  BR_COMMAND: service-mode-enable  # Включить сервисный режим`,
			wantCount:    1,
			wantCommands: []string{"service-mode-enable"},
		},
		{
			name: "env-блок с двойными кавычками и inline-комментарием",
			content: `env:
  BR_COMMAND: "sq-scan-branch"  # Сканирование ветки`,
			wantCount:    1,
			wantCommands: []string{"sq-scan-branch"},
		},
		{
			name: "env-блок с одинарными кавычками и inline-комментарием",
			content: `env:
  BR_COMMAND: 'convert'  # Конвертация`,
			wantCount:    1,
			wantCommands: []string{"convert"},
		},
		{
			name: "inline с двойными кавычками",
			content: `- name: Запуск команды
  run: BR_COMMAND="dbrestore" ./apk-ci`,
			wantCount:    1,
			wantCommands: []string{"dbrestore"},
		},
		{
			name: "inline с одинарными кавычками",
			content: `- name: Запуск команды
  run: BR_COMMAND='service-mode-status' ./apk-ci`,
			wantCount:    1,
			wantCommands: []string{"service-mode-status"},
		},
		{
			name: "inline с асимметричными кавычками — пропуск",
			content: `- name: Запуск команды
  run: BR_COMMAND="dbrestore' ./apk-ci`,
			wantCount:    0,
			wantCommands: nil,
		},
		{
			name: "env-блок с асимметричными двойными кавычками — пропуск",
			content: `env:
  BR_COMMAND: "service-mode-enable`,
			wantCount:    0,
			wantCommands: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "test.yml")
			require.NoError(t, os.WriteFile(filePath, []byte(tc.content), 0644))

			replacements, err := scanFile(filePath, legacyToNR)
			require.NoError(t, err)

			assert.Len(t, replacements, tc.wantCount, "количество замен должно быть %d", tc.wantCount)

			if tc.wantCommands != nil {
				for i, cmd := range tc.wantCommands {
					assert.Equal(t, cmd, replacements[i].OldCommand, "OldCommand для замены %d", i)
					assert.Equal(t, legacyToNR[cmd], replacements[i].NewCommand, "NewCommand для замены %d", i)
				}
			}
		})
	}
}

// TestScanDirectory_SkipsNonYaml проверяет что scanDirectory пропускает non-yaml файлы.
func TestScanDirectory_SkipsNonYaml(t *testing.T) {
	tmpDir := t.TempDir()

	// Создаём файлы разных типов
	files := map[string]string{
		"deploy.yml":   "BR_COMMAND: test",
		"quality.yaml": "BR_COMMAND: test",
		"readme.md":    "Some markdown",
		"script.sh":    "#!/bin/bash",
		"config.json":  "{}",
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644))
	}

	result, err := scanDirectory(tmpDir)
	require.NoError(t, err)

	assert.Len(t, result, 2, "должны быть найдены только .yml и .yaml файлы")

	// Проверяем что найдены правильные файлы
	var names []string //nolint:prealloc // test helper
	for _, f := range result {
		names = append(names, filepath.Base(f))
	}
	assert.Contains(t, names, "deploy.yml")
	assert.Contains(t, names, "quality.yaml")
}

// TestScanDirectory_Recursive проверяет рекурсивный поиск.
func TestScanDirectory_Recursive(t *testing.T) {
	tmpDir := t.TempDir()

	// Создаём вложенную структуру
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.MkdirAll(subDir, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yml"), []byte("test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "nested.yaml"), []byte("test"), 0644))

	result, err := scanDirectory(tmpDir)
	require.NoError(t, err)

	assert.Len(t, result, 2, "должны быть найдены файлы и во вложенных директориях")
}

// TestScanDirectory_NonExistent проверяет ошибку для несуществующей директории.
func TestScanDirectory_NonExistent(t *testing.T) {
	_, err := scanDirectory("/nonexistent/path")
	assert.Error(t, err, "должна быть ошибка для несуществующей директории")
}

// TestBackupFile проверяет создание .bak файлов.
func TestBackupFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.yml")
	content := "original content"
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	err := backupFile(filePath)
	require.NoError(t, err)

	// Проверяем backup
	backupPath := filePath + ".bak"
	backupContent, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(backupContent), "backup должен содержать оригинальное содержимое")
}

// TestApplyReplacements_Atomic проверяет атомарность замены (temp file → rename).
func TestApplyReplacements_Atomic(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.yml")

	original := `env:
  BR_COMMAND: service-mode-enable
  BR_INFOBASE_NAME: MyBase
`
	require.NoError(t, os.WriteFile(filePath, []byte(original), 0644))

	replacements := []Replacement{
		{Line: 2, OldCommand: "service-mode-enable", NewCommand: "nr-service-mode-enable"},
	}

	err := applyReplacements(filePath, replacements)
	require.NoError(t, err)

	// Проверяем результат
	result, err := os.ReadFile(filePath)
	require.NoError(t, err)

	expected := `env:
  BR_COMMAND: nr-service-mode-enable
  BR_INFOBASE_NAME: MyBase
`
	assert.Equal(t, expected, string(result))

	// Проверяем что temp файлы были удалены
	matches, _ := filepath.Glob(filepath.Join(tmpDir, ".migrate-*.tmp"))
	assert.Empty(t, matches, "временные файлы должны быть удалены после rename")
}

// TestApplyReplacements_WithQuotes проверяет замену с кавычками.
func TestApplyReplacements_WithQuotes(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.yml")

	original := `env:
  BR_COMMAND: "sq-scan-branch"
`
	require.NoError(t, os.WriteFile(filePath, []byte(original), 0644))

	replacements := []Replacement{
		{Line: 2, OldCommand: "sq-scan-branch", NewCommand: "nr-sq-scan-branch"},
	}

	err := applyReplacements(filePath, replacements)
	require.NoError(t, err)

	result, err := os.ReadFile(filePath)
	require.NoError(t, err)

	expected := `env:
  BR_COMMAND: "nr-sq-scan-branch"
`
	assert.Equal(t, expected, string(result))
}

// TestApplyReplacements_SingleQuotes проверяет замену с одинарными кавычками.
func TestApplyReplacements_SingleQuotes(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.yml")

	original := `env:
  BR_COMMAND: 'convert'
`
	require.NoError(t, os.WriteFile(filePath, []byte(original), 0644))

	replacements := []Replacement{
		{Line: 2, OldCommand: "convert", NewCommand: "nr-convert"},
	}

	err := applyReplacements(filePath, replacements)
	require.NoError(t, err)

	result, err := os.ReadFile(filePath)
	require.NoError(t, err)

	expected := `env:
  BR_COMMAND: 'nr-convert'
`
	assert.Equal(t, expected, string(result))
}

// TestApplyReplacements_Inline проверяет замену inline формата BR_COMMAND=value.
func TestApplyReplacements_Inline(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.yml")

	original := `- name: Run
  run: BR_COMMAND=dbrestore ./apk-ci
`
	require.NoError(t, os.WriteFile(filePath, []byte(original), 0644))

	replacements := []Replacement{
		{Line: 2, OldCommand: "dbrestore", NewCommand: "nr-dbrestore"},
	}

	err := applyReplacements(filePath, replacements)
	require.NoError(t, err)

	result, err := os.ReadFile(filePath)
	require.NoError(t, err)

	expected := `- name: Run
  run: BR_COMMAND=nr-dbrestore ./apk-ci
`
	assert.Equal(t, expected, string(result))
}

// TestApplyReplacements_PreservesInlineComment проверяет что inline-комментарии
// сохраняются после замены команды.
func TestApplyReplacements_PreservesInlineComment(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.yml")

	original := "env:\n  BR_COMMAND: service-mode-enable  # Включить сервисный режим\n"
	require.NoError(t, os.WriteFile(filePath, []byte(original), 0644))

	replacements := []Replacement{
		{Line: 2, OldCommand: "service-mode-enable", NewCommand: "nr-service-mode-enable"},
	}

	err := applyReplacements(filePath, replacements)
	require.NoError(t, err)

	result, err := os.ReadFile(filePath)
	require.NoError(t, err)

	expected := "env:\n  BR_COMMAND: nr-service-mode-enable  # Включить сервисный режим\n"
	assert.Equal(t, expected, string(result), "inline-комментарий должен быть сохранён")
}

// TestApplyReplacements_PreservesInlineComment_Quotes проверяет сохранение
// inline-комментариев для значений в кавычках.
func TestApplyReplacements_PreservesInlineComment_Quotes(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.yml")

	original := "env:\n  BR_COMMAND: \"sq-scan-branch\"  # Сканирование\n"
	require.NoError(t, os.WriteFile(filePath, []byte(original), 0644))

	replacements := []Replacement{
		{Line: 2, OldCommand: "sq-scan-branch", NewCommand: "nr-sq-scan-branch"},
	}

	err := applyReplacements(filePath, replacements)
	require.NoError(t, err)

	result, err := os.ReadFile(filePath)
	require.NoError(t, err)

	expected := "env:\n  BR_COMMAND: \"nr-sq-scan-branch\"  # Сканирование\n"
	assert.Equal(t, expected, string(result), "inline-комментарий и кавычки должны быть сохранены")
}

// TestApplyReplacements_InlineWithDoubleQuotes проверяет замену inline BR_COMMAND="value".
func TestApplyReplacements_InlineWithDoubleQuotes(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.yml")

	original := "- name: Run\n  run: BR_COMMAND=\"dbrestore\" ./apk-ci\n"
	require.NoError(t, os.WriteFile(filePath, []byte(original), 0644))

	replacements := []Replacement{
		{Line: 2, OldCommand: "dbrestore", NewCommand: "nr-dbrestore"},
	}

	err := applyReplacements(filePath, replacements)
	require.NoError(t, err)

	result, err := os.ReadFile(filePath)
	require.NoError(t, err)

	expected := "- name: Run\n  run: BR_COMMAND=\"nr-dbrestore\" ./apk-ci\n"
	assert.Equal(t, expected, string(result), "кавычки должны быть сохранены")
}

// TestApplyReplacements_InlineWithSingleQuotes проверяет замену inline BR_COMMAND='value'.
func TestApplyReplacements_InlineWithSingleQuotes(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.yml")

	original := "- name: Run\n  run: BR_COMMAND='service-mode-status' ./apk-ci\n"
	require.NoError(t, os.WriteFile(filePath, []byte(original), 0644))

	replacements := []Replacement{
		{Line: 2, OldCommand: "service-mode-status", NewCommand: "nr-service-mode-status"},
	}

	err := applyReplacements(filePath, replacements)
	require.NoError(t, err)

	result, err := os.ReadFile(filePath)
	require.NoError(t, err)

	expected := "- name: Run\n  run: BR_COMMAND='nr-service-mode-status' ./apk-ci\n"
	assert.Equal(t, expected, string(result), "одинарные кавычки должны быть сохранены")
}

// TestBuildLegacyToNRMapping проверяет реальный вызов buildLegacyToNRMapping().
// В контексте unit-тестов зарегистрирован только migratehandler (без deprecated-алиаса),
// поэтому маппинг пуст. Тест проверяет корректность работы функции.
func TestBuildLegacyToNRMapping(t *testing.T) {
	result := buildLegacyToNRMapping()
	assert.NotNil(t, result, "buildLegacyToNRMapping не должен возвращать nil")
	// NR-команды без deprecated alias не попадают в маппинг
	_, exists := result["nr-migrate"]
	assert.False(t, exists, "NR-команда без legacy-аналога не должна быть в маппинге")
}

// Review #33: тесты error paths для backupFile — повышают покрытие с 66.7% до ~85%.

// TestBackupFile_NonExistentFile проверяет что backupFile возвращает ошибку
// при попытке backup несуществующего файла.
func TestBackupFile_NonExistentFile(t *testing.T) {
	err := backupFile("/nonexistent/path/to/file.yml")
	assert.Error(t, err, "backupFile должен вернуть ошибку для несуществующего файла")
	assert.Contains(t, err.Error(), "ошибка чтения файла для backup")
}

// TestBackupFile_ReadOnlyDir проверяет что backupFile возвращает ошибку
// при невозможности записи backup (readonly директория).
func TestBackupFile_ReadOnlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping: root ignores file permissions")
	}
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.yml")
	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0644))

	// Делаем директорию read-only после создания файла
	require.NoError(t, os.Chmod(tmpDir, 0555))
	t.Cleanup(func() { os.Chmod(tmpDir, 0755) })

	err := backupFile(filePath)
	assert.Error(t, err, "backupFile должен вернуть ошибку при read-only директории")
	assert.Contains(t, err.Error(), "ошибка создания backup")
}

// Review #33: тесты error paths для applyReplacements — повышают покрытие с 71.7% до ~85%.

// TestApplyReplacements_NonExistentFile проверяет что applyReplacements возвращает ошибку
// при попытке замены в несуществующем файле.
func TestApplyReplacements_NonExistentFile(t *testing.T) {
	replacements := []Replacement{
		{Line: 1, OldCommand: "test", NewCommand: "nr-test"},
	}
	err := applyReplacements("/nonexistent/path/to/file.yml", replacements)
	assert.Error(t, err, "applyReplacements должен вернуть ошибку для несуществующего файла")
	assert.Contains(t, err.Error(), "ошибка чтения файла")
}

// TestApplyReplacements_ReadOnlyDir проверяет что applyReplacements возвращает ошибку
// при невозможности создания temp-файла (readonly директория).
func TestApplyReplacements_ReadOnlyDir(t *testing.T) {
	tmpDir := t.TempDir()
	if os.Getuid() == 0 {
		t.Skip("skipping: root ignores file permissions")
	}
	filePath := filepath.Join(tmpDir, "test.yml")
	require.NoError(t, os.WriteFile(filePath, []byte("env:\n  BR_COMMAND: dbrestore\n"), 0644))

	replacements := []Replacement{
		{Line: 2, OldCommand: "dbrestore", NewCommand: "nr-dbrestore"},
	}

	// Делаем директорию read-only после создания файла
	require.NoError(t, os.Chmod(tmpDir, 0555))
	t.Cleanup(func() { os.Chmod(tmpDir, 0755) })

	err := applyReplacements(filePath, replacements)
	assert.Error(t, err, "applyReplacements должен вернуть ошибку при read-only директории")
	assert.Contains(t, err.Error(), "ошибка создания временного файла")
}

// TestScanFile_Idempotent проверяет идемпотентность — повторный запуск не ломает файлы.
func TestScanFile_Idempotent(t *testing.T) {
	legacyToNR := map[string]string{
		"service-mode-enable": "nr-service-mode-enable",
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.yml")

	// Файл уже содержит NR-команду
	content := `env:
  BR_COMMAND: nr-service-mode-enable`
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	replacements, err := scanFile(filePath, legacyToNR)
	require.NoError(t, err)
	assert.Empty(t, replacements, "NR-команды не должны заменяться повторно")
}
