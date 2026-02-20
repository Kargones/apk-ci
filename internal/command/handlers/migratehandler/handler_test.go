package migratehandler

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/pkg/output"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestWorkflow создаёт тестовый workflow файл.
func createTestWorkflow(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

// TestMigrateHandler_DryRun проверяет что в dry-run режиме файлы не изменяются.
func TestMigrateHandler_DryRun(t *testing.T) {
	tmpDir := t.TempDir()

	original := `env:
  BR_COMMAND: service-mode-enable
`
	filePath := createTestWorkflow(t, tmpDir, "deploy.yml", original)

	legacyToNR := map[string]string{
		"service-mode-enable": "nr-service-mode-enable",
	}

	// Сканируем файл
	replacements, err := scanFile(filePath, legacyToNR)
	require.NoError(t, err)
	require.Len(t, replacements, 1)

	// Симулируем dry-run: НЕ применяем замены
	// (dry-run логика в Execute — тут проверяем что без applyReplacements файл не меняется)
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, original, string(content), "файл не должен быть изменён в dry-run режиме")
}

// TestMigrateHandler_Backup проверяет создание .bak файлов.
func TestMigrateHandler_Backup(t *testing.T) {
	tmpDir := t.TempDir()

	original := `env:
  BR_COMMAND: dbrestore
`
	filePath := createTestWorkflow(t, tmpDir, "restore.yml", original)

	// Создаём backup
	err := backupFile(filePath)
	require.NoError(t, err)

	// Проверяем что .bak создан
	backupPath := filePath + ".bak"
	_, statErr := os.Stat(backupPath)
	require.NoError(t, statErr, ".bak файл должен существовать")

	// Содержимое backup совпадает с оригиналом
	backupContent, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, original, string(backupContent))

	// Применяем замену
	replacements := []Replacement{
		{Line: 2, OldCommand: "dbrestore", NewCommand: "nr-dbrestore"},
	}
	err = applyReplacements(filePath, replacements)
	require.NoError(t, err)

	// Оригинал изменён
	modified, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Contains(t, string(modified), "nr-dbrestore")

	// Backup по-прежнему содержит оригинал
	backupContent2, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, original, string(backupContent2))
}

// TestMigrateHandler_TextReport проверяет текстовый формат отчёта.
func TestMigrateHandler_TextReport(t *testing.T) {
	report := &MigrationReport{
		ScannedFiles:      5,
		ModifiedFiles:     2,
		TotalReplacements: 3,
		DryRun:            false,
		Replacements: []ReplacementInfo{
			{File: ".gitea/workflows/deploy.yml", Line: 12, OldCommand: "service-mode-enable", NewCommand: "nr-service-mode-enable"},
			{File: ".gitea/workflows/deploy.yml", Line: 28, OldCommand: "dbrestore", NewCommand: "nr-dbrestore"},
			{File: ".gitea/workflows/quality.yml", Line: 8, OldCommand: "sq-scan-branch", NewCommand: "nr-sq-scan-branch"},
		},
		BackupFiles: []string{
			".gitea/workflows/deploy.yml.bak",
			".gitea/workflows/quality.yml.bak",
		},
	}

	var buf bytes.Buffer
	err := writeTextReport(&buf, report)
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "Migration Report")
	assert.Contains(t, result, "Scanned: 5 files")
	assert.Contains(t, result, "Modified: 2 files")
	assert.Contains(t, result, "Total replacements: 3")
	assert.Contains(t, result, "deploy.yml")
	assert.Contains(t, result, "service-mode-enable")
	assert.Contains(t, result, "nr-service-mode-enable")
	assert.Contains(t, result, "Backup files created:")
	assert.Contains(t, result, "deploy.yml.bak")
}

// TestMigrateHandler_TextReport_DryRun проверяет текстовый формат отчёта в dry-run режиме.
func TestMigrateHandler_TextReport_DryRun(t *testing.T) {
	report := &MigrationReport{
		ScannedFiles:      3,
		ModifiedFiles:     1,
		TotalReplacements: 1,
		DryRun:            true,
		Replacements: []ReplacementInfo{
			{File: "deploy.yml", Line: 5, OldCommand: "dbrestore", NewCommand: "nr-dbrestore"},
		},
	}

	var buf bytes.Buffer
	err := writeTextReport(&buf, report)
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "[DRY-RUN]")
	assert.Empty(t, report.BackupFiles)
}

// TestMigrateHandler_JSONReport проверяет JSON формат отчёта.
func TestMigrateHandler_JSONReport(t *testing.T) {
	report := &MigrationReport{
		ScannedFiles:      3,
		ModifiedFiles:     1,
		TotalReplacements: 2,
		DryRun:            false,
		Replacements: []ReplacementInfo{
			{File: "deploy.yml", Line: 12, OldCommand: "service-mode-enable", NewCommand: "nr-service-mode-enable"},
			{File: "deploy.yml", Line: 28, OldCommand: "dbrestore", NewCommand: "nr-dbrestore"},
		},
		BackupFiles: []string{"deploy.yml.bak"},
	}

	var buf bytes.Buffer
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: "nr-migrate",
		Data:    report,
		Metadata: &output.Metadata{
			DurationMs: 42,
			TraceID:    "test-trace-123",
			APIVersion: "v1",
		},
	}

	writer := output.NewWriter(output.FormatJSON)
	err := writer.Write(&buf, result)
	require.NoError(t, err)

	// Парсим JSON
	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "success", parsed["status"])
	assert.Equal(t, "nr-migrate", parsed["command"])

	data, ok := parsed["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(3), data["scanned_files"])
	assert.Equal(t, float64(1), data["modified_files"])
	assert.Equal(t, float64(2), data["total_replacements"])

	replacements, ok := data["replacements"].([]any)
	require.True(t, ok)
	assert.Len(t, replacements, 2)

	first := replacements[0].(map[string]any)
	assert.Equal(t, "deploy.yml", first["file"])
	assert.Equal(t, float64(12), first["line"])
	assert.Equal(t, "service-mode-enable", first["old_command"])
	assert.Equal(t, "nr-service-mode-enable", first["new_command"])
}

// TestMigrateHandler_NoModifiedFiles проверяет отчёт когда нет файлов для замены.
func TestMigrateHandler_NoModifiedFiles(t *testing.T) {
	report := &MigrationReport{
		ScannedFiles:      3,
		ModifiedFiles:     0,
		TotalReplacements: 0,
		DryRun:            false,
	}

	var buf bytes.Buffer
	err := writeTextReport(&buf, report)
	require.NoError(t, err)

	result := buf.String()
	assert.Contains(t, result, "Modified: 0 files")
	assert.Contains(t, result, "Total replacements: 0")
}

// TestWriteSuccess_TextFormat проверяет writeSuccess в текстовом формате.
func TestWriteSuccess_TextFormat(t *testing.T) {
	report := &MigrationReport{
		ScannedFiles:      1,
		ModifiedFiles:     1,
		TotalReplacements: 1,
		Replacements: []ReplacementInfo{
			{File: "test.yml", Line: 1, OldCommand: "convert", NewCommand: "nr-convert"},
		},
	}

	var buf bytes.Buffer
	err := writeSuccess(&buf, output.FormatText, "nr-migrate", "trace-1", time.Now(), report)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "Migration Report")
}

// TestWriteSuccess_JSONFormat проверяет writeSuccess в JSON формате.
func TestWriteSuccess_JSONFormat(t *testing.T) {
	report := &MigrationReport{
		ScannedFiles:      1,
		ModifiedFiles:     1,
		TotalReplacements: 1,
		Replacements: []ReplacementInfo{
			{File: "test.yml", Line: 1, OldCommand: "convert", NewCommand: "nr-convert"},
		},
	}

	var buf bytes.Buffer
	err := writeSuccess(&buf, output.FormatJSON, "nr-migrate", "trace-1", time.Now(), report)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "success", parsed["status"])
}

// TestWriteError_TextFormat проверяет writeError в текстовом формате.
// В текстовом формате writeError не пишет в writer — ошибку обрабатывает main.go.
func TestWriteError_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	err := writeError(&buf, output.FormatText, "nr-migrate", "trace-1", time.Now(), assert.AnError)
	assert.ErrorIs(t, err, assert.AnError, "writeError должен вернуть оригинальную ошибку")
	assert.Empty(t, buf.String(), "текстовый формат не должен писать в writer")
}

// TestWriteError_JSONFormat проверяет writeError в JSON формате.
// JSON формат выводит структурированную ошибку И возвращает оригинальную ошибку.
func TestWriteError_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	err := writeError(&buf, output.FormatJSON, "nr-migrate", "trace-1", time.Now(), assert.AnError)
	assert.ErrorIs(t, err, assert.AnError, "writeError должен вернуть оригинальную ошибку")

	var parsed map[string]any
	parseErr := json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, parseErr)
	assert.Equal(t, "error", parsed["status"])
	assert.Equal(t, "MIGRATE.FAILED", parsed["error"].(map[string]any)["code"])
}

// TestMigrateHandler_Execute_NonExistentPath_ReturnsError проверяет что Execute
// возвращает ошибку при несуществующем пути (текстовый формат).
// Регрессионный тест: writeError ранее возвращал nil вместо оригинальной ошибки.
func TestMigrateHandler_Execute_NonExistentPath_ReturnsError(t *testing.T) {
	t.Setenv("BR_MIGRATE_PATH", "/nonexistent/path/to/workflows")
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	h := &MigrateHandler{}
	ctx := context.Background()

	err := h.Execute(ctx, nil)
	assert.Error(t, err, "Execute с несуществующим путём должен вернуть ошибку")
}

// TestMigrateHandler_Execute_NonExistentPath_ReturnsError_JSON проверяет что Execute
// возвращает ошибку при несуществующем пути (JSON формат).
func TestMigrateHandler_Execute_NonExistentPath_ReturnsError_JSON(t *testing.T) {
	t.Setenv("BR_MIGRATE_PATH", "/nonexistent/path/to/workflows")
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &MigrateHandler{}
	ctx := context.Background()

	err := h.Execute(ctx, nil)
	assert.Error(t, err, "Execute с несуществующим путём должен вернуть ошибку (JSON)")
}

// TestMigrateHandler_Execute_DryRun_Success проверяет полный flow Execute в dry-run режиме.
func TestMigrateHandler_Execute_DryRun_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Создаём тестовый workflow — NR-команда, которая не должна заменяться
	createTestWorkflow(t, tmpDir, "deploy.yml", "env:\n  BR_COMMAND: nr-service-mode-enable\n")

	t.Setenv("BR_MIGRATE_PATH", tmpDir)
	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	h := &MigrateHandler{}
	ctx := context.Background()

	err := h.Execute(ctx, nil)
	require.NoError(t, err, "Execute в dry-run без замен должен завершиться успешно")
}

// TestMigrateHandler_Execute_FullFlow_WithBackup проверяет полный flow Execute:
// scan → backup → apply → report. Использует прямой вызов internal-функций,
// т.к. в контексте unit-тестов buildLegacyToNRMapping() возвращает пустой маппинг
// (зарегистрирован только migratehandler без deprecated-алиаса).
// Полный integration тест с реальным маппингом всех 18 команд — в smoketest.
func TestMigrateHandler_Execute_FullFlow_WithBackup(t *testing.T) {
	tmpDir := t.TempDir()

	original := "env:\n  BR_COMMAND: service-mode-enable\n  BR_INFOBASE_NAME: TestBase\n"
	filePath := createTestWorkflow(t, tmpDir, "deploy.yml", original)

	legacyToNR := map[string]string{
		"service-mode-enable": "nr-service-mode-enable",
	}

	// 1. Scan
	replacements, err := scanFile(filePath, legacyToNR)
	require.NoError(t, err)
	require.Len(t, replacements, 1)
	assert.Equal(t, "service-mode-enable", replacements[0].OldCommand)

	// 2. Backup
	err = backupFile(filePath)
	require.NoError(t, err)
	_, statErr := os.Stat(filePath + ".bak")
	require.NoError(t, statErr, ".bak файл должен существовать")

	// 3. Apply
	err = applyReplacements(filePath, replacements)
	require.NoError(t, err)

	// 4. Verify результат
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "nr-service-mode-enable")
	assert.NotContains(t, string(content), "  BR_COMMAND: service-mode-enable\n")
	assert.Contains(t, string(content), "BR_INFOBASE_NAME: TestBase", "другие env не должны измениться")

	// 5. Verify backup содержит оригинал
	backup, err := os.ReadFile(filePath + ".bak")
	require.NoError(t, err)
	assert.Equal(t, original, string(backup))

	// 6. Verify report
	report := &MigrationReport{
		ScannedFiles:      1,
		ModifiedFiles:     1,
		TotalReplacements: 1,
		Replacements: []ReplacementInfo{
			{File: filePath, Line: 2, OldCommand: "service-mode-enable", NewCommand: "nr-service-mode-enable"},
		},
		BackupFiles: []string{filePath + ".bak"},
	}
	var buf bytes.Buffer
	err = writeTextReport(&buf, report)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "nr-service-mode-enable")
}

// TestMigrateHandler_Execute_FullFlow_JSON проверяет Execute с JSON-выводом.
func TestMigrateHandler_Execute_FullFlow_JSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Файл с NR-командой — замен не будет, но Execute должен пройти полный path
	createTestWorkflow(t, tmpDir, "deploy.yml", "env:\n  BR_COMMAND: nr-service-mode-enable\n")

	t.Setenv("BR_MIGRATE_PATH", tmpDir)
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &MigrateHandler{}
	err := h.Execute(context.Background(), nil)
	require.NoError(t, err, "Execute JSON path должен завершиться успешно")
}

// TestMigrateHandler_NameDescription проверяет Name() и Description().
func TestMigrateHandler_NameDescription(t *testing.T) {
	h := &MigrateHandler{}
	assert.Equal(t, "nr-migrate", h.Name())
	assert.NotEmpty(t, h.Description())
}

// testLegacyHandler — mock handler для тестирования Execute с deprecated alias.
type testLegacyHandler struct{}

func (h *testLegacyHandler) Name() string                                          { return "nr-test-legacy-cmd" }
func (h *testLegacyHandler) Description() string                                   { return "test handler" }
func (h *testLegacyHandler) Execute(_ context.Context, _ *config.Config) error      { return nil }

// TestMigrateHandler_Execute_RealApply_NoBackup проверяет полный Execute flow
// с реальными заменами, без dry-run, с noBackup=true.
func TestMigrateHandler_Execute_RealApply_NoBackup(t *testing.T) {
	// Регистрируем тестовый handler с deprecated alias для buildLegacyToNRMapping
	command.RegisterWithAlias(&testLegacyHandler{}, "test-legacy-cmd")

	tmpDir := t.TempDir()
	createTestWorkflow(t, tmpDir, "deploy.yml",
		"env:\n  BR_COMMAND: test-legacy-cmd\n  BR_INFOBASE_NAME: TestBase\n")

	t.Setenv("BR_MIGRATE_PATH", tmpDir)
	t.Setenv("BR_OUTPUT_FORMAT", "text")
	t.Setenv("BR_DRY_RUN", "")
	t.Setenv("BR_PLAN_ONLY", "")
	t.Setenv("BR_MIGRATE_NO_BACKUP", "true")

	h := &MigrateHandler{}
	err := h.Execute(context.Background(), nil)
	require.NoError(t, err)

	// Проверяем что файл был изменён
	content, readErr := os.ReadFile(filepath.Join(tmpDir, "deploy.yml"))
	require.NoError(t, readErr)
	assert.Contains(t, string(content), "nr-test-legacy-cmd")
	assert.NotContains(t, string(content), "  BR_COMMAND: test-legacy-cmd\n")

	// Проверяем что .bak НЕ создан (noBackup=true)
	_, statErr := os.Stat(filepath.Join(tmpDir, "deploy.yml.bak"))
	assert.True(t, os.IsNotExist(statErr), ".bak не должен создаваться при noBackup=true")
}

// TestMigrateHandler_Execute_RealApply_WithBackup проверяет Execute flow
// с реальными заменами, без dry-run, с backup.
func TestMigrateHandler_Execute_RealApply_WithBackup(t *testing.T) {
	// Регистрация тестового handler уже выполнена в предыдущем тесте (глобальный registry)
	tmpDir := t.TempDir()
	original := "env:\n  BR_COMMAND: test-legacy-cmd\n"
	createTestWorkflow(t, tmpDir, "workflow.yml", original)

	t.Setenv("BR_MIGRATE_PATH", tmpDir)
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	t.Setenv("BR_DRY_RUN", "")
	t.Setenv("BR_PLAN_ONLY", "")
	t.Setenv("BR_MIGRATE_NO_BACKUP", "")

	h := &MigrateHandler{}
	err := h.Execute(context.Background(), nil)
	require.NoError(t, err)

	// Проверяем что файл был изменён
	content, readErr := os.ReadFile(filepath.Join(tmpDir, "workflow.yml"))
	require.NoError(t, readErr)
	assert.Contains(t, string(content), "nr-test-legacy-cmd")

	// Проверяем что .bak создан
	backupContent, bakErr := os.ReadFile(filepath.Join(tmpDir, "workflow.yml.bak"))
	require.NoError(t, bakErr)
	assert.Equal(t, original, string(backupContent))
}

// TestMigrateHandler_Execute_DryRun_WithReplacements проверяет dry-run с найденными заменами.
// Файлы НЕ должны изменяться.
func TestMigrateHandler_Execute_DryRun_WithReplacements(t *testing.T) {
	tmpDir := t.TempDir()
	original := "env:\n  BR_COMMAND: test-legacy-cmd\n"
	createTestWorkflow(t, tmpDir, "deploy.yml", original)

	t.Setenv("BR_MIGRATE_PATH", tmpDir)
	t.Setenv("BR_OUTPUT_FORMAT", "text")
	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_PLAN_ONLY", "")

	h := &MigrateHandler{}
	err := h.Execute(context.Background(), nil)
	require.NoError(t, err)

	// Проверяем что файл НЕ был изменён
	content, readErr := os.ReadFile(filepath.Join(tmpDir, "deploy.yml"))
	require.NoError(t, readErr)
	assert.Equal(t, original, string(content), "файл не должен быть изменён в dry-run режиме")
}

// TestMigrateHandler_Execute_PlanOnly проверяет что plan-only режим
// возвращает unsupported без выполнения миграции.
func TestMigrateHandler_Execute_PlanOnly(t *testing.T) {
	t.Setenv("BR_PLAN_ONLY", "true")
	t.Setenv("BR_MIGRATE_PATH", "/nonexistent")

	h := &MigrateHandler{}
	// plan-only должен вернуть nil (unsupported message пишется в stdout)
	err := h.Execute(context.Background(), nil)
	assert.NoError(t, err, "plan-only режим не должен возвращать ошибку")
}

// TestMigrateHandler_Execute_DryRunOverPlanOnly проверяет приоритет dry-run
// над plan-only в handler (Story 7.3 AC-11). При BR_DRY_RUN=true и BR_PLAN_ONLY=true
// handler должен выполнить dry-run (показать замены без применения),
// а не plan-only unsupported.
// Review #32: тест приоритета режимов на уровне handler.
func TestMigrateHandler_Execute_DryRunOverPlanOnly(t *testing.T) {
	tmpDir := t.TempDir()
	original := "env:\n  BR_COMMAND: test-legacy-cmd\n"
	createTestWorkflow(t, tmpDir, "deploy.yml", original)

	t.Setenv("BR_MIGRATE_PATH", tmpDir)
	t.Setenv("BR_OUTPUT_FORMAT", "json")
	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_PLAN_ONLY", "true")

	// Перехватываем stdout для проверки вывода
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	h := &MigrateHandler{}
	err := h.Execute(context.Background(), nil)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	require.NoError(t, err)

	// Файл НЕ должен быть изменён (dry-run)
	content, readErr := os.ReadFile(filepath.Join(tmpDir, "deploy.yml"))
	require.NoError(t, readErr)
	assert.Equal(t, original, string(content),
		"файл не должен быть изменён — dry-run имеет приоритет над plan-only")

	// Вывод должен быть JSON с отчётом миграции (dry-run), а не plan-only unsupported
	output := buf.String()
	assert.NotContains(t, output, "не поддерживает отображение плана",
		"при dry-run+plan-only НЕ должен выводиться plan-only unsupported")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed),
		"вывод должен быть валидным JSON (отчёт миграции)")
	assert.Equal(t, "success", parsed["status"],
		"dry-run должен завершиться успешно")
}

// TestMigrateHandler_Execute_PathTraversal проверяет что BR_MIGRATE_PATH
// с выходом за пределы рабочей директории через ".." возвращает ошибку.
// Review #32: защита от path traversal.
func TestMigrateHandler_Execute_PathTraversal(t *testing.T) {
	t.Setenv("BR_MIGRATE_PATH", "../../etc/")
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	h := &MigrateHandler{}
	err := h.Execute(context.Background(), nil)
	assert.Error(t, err, "path traversal через '..' должен быть отклонён")
	assert.Contains(t, err.Error(), "недопустимый путь")
}
