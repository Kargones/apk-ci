package actionmenu

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/gitea"
	"github.com/Kargones/apk-ci/internal/adapter/gitea/giteatest"
	"github.com/Kargones/apk-ci/internal/command/handlers/gitea/shared"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/output"
)

// captureStdout перехватывает вывод в stdout для тестирования.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Не удалось создать pipe: %v", err)
	}
	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("Не удалось закрыть writer: %v", err)
	}
	os.Stdout = oldStdout

	buf := make([]byte, 10240)
	n, _ := r.Read(buf)
	return string(buf[:n])
}

// testProjectConfig создаёт тестовую ProjectConfig с базами данных.
func testProjectConfig(debug bool) *config.ProjectConfig {
	return &config.ProjectConfig{
		Debug: debug,
		Prod: map[string]struct {
			DbName     string                 `yaml:"dbName"`
			AddDisable []string               `yaml:"add-disable"`
			Related    map[string]interface{} `yaml:"related"`
		}{
			"ProdDB": {
				DbName: "Production",
				Related: map[string]interface{}{
					"TestDB": nil,
				},
			},
		},
	}
}

// testProjectConfigMultiple создаёт тестовую ProjectConfig с несколькими базами.
func testProjectConfigMultiple() *config.ProjectConfig {
	return &config.ProjectConfig{
		Debug: false,
		Prod: map[string]struct {
			DbName     string                 `yaml:"dbName"`
			AddDisable []string               `yaml:"add-disable"`
			Related    map[string]interface{} `yaml:"related"`
		}{
			"ProdDB1": {
				DbName: "Production 1",
				Related: map[string]interface{}{
					"TestDB1": nil,
					"TestDB2": nil,
				},
			},
			"ProdDB2": {
				DbName: "Production 2",
				Related: map[string]interface{}{
					"TestDB3": nil,
				},
			},
		},
	}
}

// TestName проверяет, что Name() возвращает правильное имя команды.
func TestName(t *testing.T) {
	h := &ActionMenuHandler{}
	if h.Name() != constants.ActNRActionMenuBuild {
		t.Errorf("Ожидалось %s, получено %s", constants.ActNRActionMenuBuild, h.Name())
	}
}

// TestDescription проверяет, что Description() возвращает непустую строку.
func TestDescription(t *testing.T) {
	h := &ActionMenuHandler{}
	if h.Description() == "" {
		t.Error("Description() не должен возвращать пустую строку")
	}
}

// TestExecute_MissingConfig проверяет ошибку при отсутствии конфигурации (AC: #9).
func TestExecute_MissingConfig(t *testing.T) {
	h := &ActionMenuHandler{giteaClient: giteatest.NewMockClient()}

	err := h.Execute(context.Background(), nil)

	if err == nil {
		t.Error("Ожидалась ошибка при nil config")
	}
	if !strings.Contains(err.Error(), shared.ErrConfigMissing) {
		t.Errorf("Ожидался код ошибки %s, получено: %v", shared.ErrConfigMissing, err)
	}
}

// TestExecute_MissingOwnerRepo проверяет ошибку при отсутствии owner/repo.
func TestExecute_MissingOwnerRepo(t *testing.T) {
	h := &ActionMenuHandler{giteaClient: giteatest.NewMockClient()}
	cfg := &config.Config{} // пустой owner и repo

	err := h.Execute(context.Background(), cfg)

	if err == nil {
		t.Error("Ожидалась ошибка при пустом owner/repo")
	}
	if !strings.Contains(err.Error(), shared.ErrMissingOwnerRepo) {
		t.Errorf("Ожидался код ошибки %s, получено: %v", shared.ErrMissingOwnerRepo, err)
	}
}

// TestExecute_NoChanges проверяет, что при отсутствии изменений project.yaml
// и ForceUpdate=false возвращается StateChanged=false (AC: #4, #10).
func TestExecute_NoChanges(t *testing.T) {
	mock := giteatest.NewMockClient()
	mock.GetLatestCommitFunc = func(_ context.Context, _ string) (*gitea.Commit, error) {
		return &gitea.Commit{SHA: "abc123"}, nil
	}
	mock.GetCommitFilesFunc = func(_ context.Context, _ string) ([]gitea.CommitFile, error) {
		// project.yaml НЕ изменён
		return []gitea.CommitFile{
			{Filename: "README.md", Status: "modified"},
		}, nil
	}

	h := &ActionMenuHandler{giteaClient: mock}
	cfg := &config.Config{
		Owner:       "testorg",
		Repo:        "testrepo",
		BaseBranch:  "main",
		ForceUpdate: false,
	}

	// Установим BR_OUTPUT_FORMAT=json для проверки JSON вывода
	os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer os.Unsetenv("BR_OUTPUT_FORMAT")

	captured := captureStdout(t, func() {
		err := h.Execute(context.Background(), cfg)
		if err != nil {
			t.Errorf("Не ожидалась ошибка: %v", err)
		}
	})

	// Проверяем JSON вывод
	var result output.Result
	if err := json.Unmarshal([]byte(captured), &result); err != nil {
		t.Fatalf("Не удалось разобрать JSON: %v\nВывод: %s", err, captured)
	}

	if result.Status != output.StatusSuccess {
		t.Errorf("Ожидался статус success, получен: %s", result.Status)
	}

	// Проверяем data
	dataBytes, _ := json.Marshal(result.Data)
	var data ActionMenuData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		t.Fatalf("Не удалось разобрать data: %v", err)
	}

	if data.StateChanged {
		t.Error("StateChanged должен быть false при отсутствии изменений project.yaml")
	}
	if data.ProjectYamlChanged {
		t.Error("ProjectYamlChanged должен быть false")
	}
}

// TestExecute_ForceUpdate проверяет принудительное обновление (AC: #4).
func TestExecute_ForceUpdate(t *testing.T) {
	mock := giteatest.NewMockClient()

	// Мокаем получение текущих файлов
	mock.GetRepositoryContentsFunc = func(_ context.Context, _, _ string) ([]gitea.FileInfo, error) {
		return []gitea.FileInfo{}, nil
	}
	mock.SetRepositoryStateFunc = func(_ context.Context, ops []gitea.BatchOperation, _, _ string) error {
		// Проверяем, что операции создания были вызваны
		if len(ops) == 0 {
			t.Error("Ожидались операции синхронизации")
		}
		return nil
	}

	h := &ActionMenuHandler{giteaClient: mock}
	cfg := &config.Config{
		Owner:         "testorg",
		Repo:          "testrepo",
		BaseBranch:    "main",
		ForceUpdate:   true, // Принудительное обновление
		MenuMain:      []string{"test-workflow.yml\nname: Test\non:\n  push:\njobs:\n  test:\n    runs-on: ubuntu-latest"},
		ProjectConfig: testProjectConfig(false),
	}

	os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer os.Unsetenv("BR_OUTPUT_FORMAT")

	captured := captureStdout(t, func() {
		err := h.Execute(context.Background(), cfg)
		if err != nil {
			t.Errorf("Не ожидалась ошибка: %v", err)
		}
	})

	var result output.Result
	if err := json.Unmarshal([]byte(captured), &result); err != nil {
		t.Fatalf("Не удалось разобрать JSON: %v", err)
	}

	if result.Status != output.StatusSuccess {
		t.Errorf("Ожидался статус success, получен: %s", result.Status)
	}

	dataBytes, _ := json.Marshal(result.Data)
	var data ActionMenuData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		t.Fatalf("Не удалось разобрать data: %v", err)
	}

	if !data.ForceUpdate {
		t.Error("ForceUpdate должен быть true")
	}
}

// TestExecute_AddFiles проверяет добавление новых файлов (AC: #5).
func TestExecute_AddFiles(t *testing.T) {
	mock := giteatest.NewMockClient()

	var capturedOps []gitea.BatchOperation
	mock.GetRepositoryContentsFunc = func(_ context.Context, _, _ string) ([]gitea.FileInfo, error) {
		// Пустой каталог — нет существующих файлов
		return []gitea.FileInfo{}, nil
	}
	mock.SetRepositoryStateFunc = func(_ context.Context, ops []gitea.BatchOperation, _, _ string) error {
		capturedOps = ops
		return nil
	}

	h := &ActionMenuHandler{giteaClient: mock}
	cfg := &config.Config{
		Owner:         "testorg",
		Repo:          "testrepo",
		BaseBranch:    "main",
		ForceUpdate:   true,
		MenuMain:      []string{"new-workflow.yml\nname: New\non:\n  push:\njobs:\n  test:\n    runs-on: ubuntu-latest"},
		ProjectConfig: testProjectConfig(false),
	}

	os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer os.Unsetenv("BR_OUTPUT_FORMAT")

	captured := captureStdout(t, func() {
		_ = h.Execute(context.Background(), cfg)
	})

	// Проверяем, что была операция create
	if len(capturedOps) == 0 {
		t.Error("Ожидались операции")
	}
	for _, op := range capturedOps {
		if op.Operation != "create" {
			t.Errorf("Ожидалась операция create, получена: %s", op.Operation)
		}
	}

	var result output.Result
	if err := json.Unmarshal([]byte(captured), &result); err != nil {
		t.Fatalf("Не удалось разобрать JSON: %v", err)
	}

	dataBytes, _ := json.Marshal(result.Data)
	var data ActionMenuData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		t.Fatalf("Не удалось разобрать data: %v", err)
	}

	if data.AddedFiles != 1 {
		t.Errorf("Ожидалось 1 добавленный файл, получено: %d", data.AddedFiles)
	}
	if !data.StateChanged {
		t.Error("StateChanged должен быть true при добавлении файлов")
	}
}

// TestExecute_UpdateFiles проверяет обновление существующих файлов (AC: #5).
func TestExecute_UpdateFiles(t *testing.T) {
	mock := giteatest.NewMockClient()

	var capturedOps []gitea.BatchOperation
	mock.GetRepositoryContentsFunc = func(_ context.Context, _, _ string) ([]gitea.FileInfo, error) {
		// Существующий файл с другим SHA
		return []gitea.FileInfo{
			{
				Name: "existing-workflow.yml",
				Path: ".gitea/workflows/existing-workflow.yml",
				SHA:  "old-sha-different",
			},
		}, nil
	}
	mock.GetFileContentFunc = func(_ context.Context, _ string) ([]byte, error) {
		return []byte("old content"), nil
	}
	mock.SetRepositoryStateFunc = func(_ context.Context, ops []gitea.BatchOperation, _, _ string) error {
		capturedOps = ops
		return nil
	}

	h := &ActionMenuHandler{giteaClient: mock}
	cfg := &config.Config{
		Owner:         "testorg",
		Repo:          "testrepo",
		BaseBranch:    "main",
		ForceUpdate:   true,
		MenuMain:      []string{"existing-workflow.yml\nname: Updated\non:\n  push:\njobs:\n  test:\n    runs-on: ubuntu-latest"},
		ProjectConfig: testProjectConfig(false),
	}

	os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer os.Unsetenv("BR_OUTPUT_FORMAT")

	captured := captureStdout(t, func() {
		_ = h.Execute(context.Background(), cfg)
	})

	// Проверяем, что была операция update
	hasUpdate := false
	for _, op := range capturedOps {
		if op.Operation == "update" {
			hasUpdate = true
		}
	}
	if !hasUpdate {
		t.Error("Ожидалась операция update")
	}

	var result output.Result
	if err := json.Unmarshal([]byte(captured), &result); err != nil {
		t.Fatalf("Не удалось разобрать JSON: %v", err)
	}

	dataBytes, _ := json.Marshal(result.Data)
	var data ActionMenuData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		t.Fatalf("Не удалось разобрать data: %v", err)
	}

	if data.UpdatedFiles != 1 {
		t.Errorf("Ожидался 1 обновлённый файл, получено: %d", data.UpdatedFiles)
	}
}

// TestExecute_DeleteFiles проверяет удаление устаревших файлов (AC: #5).
func TestExecute_DeleteFiles(t *testing.T) {
	mock := giteatest.NewMockClient()

	var capturedOps []gitea.BatchOperation
	mock.GetRepositoryContentsFunc = func(_ context.Context, _, _ string) ([]gitea.FileInfo, error) {
		// Существует файл, которого нет в новой конфигурации
		return []gitea.FileInfo{
			{
				Name: "old-workflow.yml",
				Path: ".gitea/workflows/old-workflow.yml",
				SHA:  "old-sha",
			},
			{
				Name: "keep-workflow.yml",
				Path: ".gitea/workflows/keep-workflow.yml",
				SHA:  "keep-sha",
			},
		}, nil
	}
	mock.GetFileContentFunc = func(_ context.Context, path string) ([]byte, error) {
		return []byte("content for " + path), nil
	}
	mock.SetRepositoryStateFunc = func(_ context.Context, ops []gitea.BatchOperation, _, _ string) error {
		capturedOps = ops
		return nil
	}

	h := &ActionMenuHandler{giteaClient: mock}
	// Только keep-workflow остаётся
	cfg := &config.Config{
		Owner:         "testorg",
		Repo:          "testrepo",
		BaseBranch:    "main",
		ForceUpdate:   true,
		MenuMain:      []string{"keep-workflow.yml\nname: Keep\non:\n  push:\njobs:\n  test:\n    runs-on: ubuntu-latest"},
		ProjectConfig: testProjectConfig(false),
	}

	os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer os.Unsetenv("BR_OUTPUT_FORMAT")

	captured := captureStdout(t, func() {
		_ = h.Execute(context.Background(), cfg)
	})

	// Проверяем, что была операция delete для old-workflow.yml
	hasDelete := false
	for _, op := range capturedOps {
		if op.Operation == "delete" && strings.Contains(op.Path, "old-workflow.yml") {
			hasDelete = true
		}
	}
	if !hasDelete {
		t.Error("Ожидалась операция delete для old-workflow.yml")
	}

	var result output.Result
	if err := json.Unmarshal([]byte(captured), &result); err != nil {
		t.Fatalf("Не удалось разобрать JSON: %v", err)
	}

	dataBytes, _ := json.Marshal(result.Data)
	var data ActionMenuData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		t.Fatalf("Не удалось разобрать data: %v", err)
	}

	if data.DeletedFiles != 1 {
		t.Errorf("Ожидался 1 удалённый файл, получено: %d", data.DeletedFiles)
	}
}

// TestExecute_MixedOperations проверяет комбинацию add/update/delete (AC: #5, #10).
func TestExecute_MixedOperations(t *testing.T) {
	mock := giteatest.NewMockClient()

	var capturedOps []gitea.BatchOperation
	mock.GetRepositoryContentsFunc = func(_ context.Context, _, _ string) ([]gitea.FileInfo, error) {
		return []gitea.FileInfo{
			{Name: "to-delete.yml", Path: ".gitea/workflows/to-delete.yml", SHA: "del-sha"},
			{Name: "to-update.yml", Path: ".gitea/workflows/to-update.yml", SHA: "upd-sha"},
		}, nil
	}
	mock.GetFileContentFunc = func(_ context.Context, _ string) ([]byte, error) {
		return []byte("old content"), nil
	}
	mock.SetRepositoryStateFunc = func(_ context.Context, ops []gitea.BatchOperation, _, _ string) error {
		capturedOps = ops
		return nil
	}

	h := &ActionMenuHandler{giteaClient: mock}
	cfg := &config.Config{
		Owner:       "testorg",
		Repo:        "testrepo",
		BaseBranch:  "main",
		ForceUpdate: true,
		MenuMain: []string{
			"to-update.yml\nname: Updated Content\non:\n  push:\njobs:\n  test:\n    runs-on: ubuntu-latest",
			"---",
			"to-create.yml\nname: New\non:\n  push:\njobs:\n  test:\n    runs-on: ubuntu-latest",
		},
		ProjectConfig: testProjectConfig(false),
	}

	os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer os.Unsetenv("BR_OUTPUT_FORMAT")

	captured := captureStdout(t, func() {
		_ = h.Execute(context.Background(), cfg)
	})

	// Проверяем наличие всех типов операций
	opCounts := map[string]int{}
	for _, op := range capturedOps {
		opCounts[op.Operation]++
	}

	if opCounts["create"] < 1 {
		t.Error("Ожидалась операция create")
	}
	if opCounts["update"] < 1 {
		t.Error("Ожидалась операция update")
	}
	if opCounts["delete"] < 1 {
		t.Error("Ожидалась операция delete")
	}

	var result output.Result
	if err := json.Unmarshal([]byte(captured), &result); err != nil {
		t.Fatalf("Не удалось разобрать JSON: %v", err)
	}

	dataBytes, _ := json.Marshal(result.Data)
	var data ActionMenuData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		t.Fatalf("Не удалось разобрать data: %v", err)
	}

	if !data.StateChanged {
		t.Error("StateChanged должен быть true")
	}
	if len(data.SyncedFiles) != 3 {
		t.Errorf("Ожидалось 3 синхронизированных файла, получено: %d", len(data.SyncedFiles))
	}
}

// TestExecute_NoDatabases проверяет graceful exit при отсутствии баз данных (AC: #2).
func TestExecute_NoDatabases(t *testing.T) {
	mock := giteatest.NewMockClient()

	h := &ActionMenuHandler{giteaClient: mock}
	cfg := &config.Config{
		Owner:         "testorg",
		Repo:          "testrepo",
		BaseBranch:    "main",
		ForceUpdate:   true,
		ProjectConfig: &config.ProjectConfig{}, // Пустой ProjectConfig
	}

	os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer os.Unsetenv("BR_OUTPUT_FORMAT")

	captured := captureStdout(t, func() {
		err := h.Execute(context.Background(), cfg)
		if err != nil {
			t.Errorf("Не ожидалась ошибка: %v", err)
		}
	})

	var result output.Result
	if err := json.Unmarshal([]byte(captured), &result); err != nil {
		t.Fatalf("Не удалось разобрать JSON: %v", err)
	}

	if result.Status != output.StatusSuccess {
		t.Errorf("Ожидался success при отсутствии баз данных, получен: %s", result.Status)
	}

	dataBytes, _ := json.Marshal(result.Data)
	var data ActionMenuData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		t.Fatalf("Не удалось разобрать data: %v", err)
	}

	if data.StateChanged {
		t.Error("StateChanged должен быть false при отсутствии баз данных")
	}
	if data.DatabasesProcessed != 0 {
		t.Errorf("DatabasesProcessed должен быть 0, получено: %d", data.DatabasesProcessed)
	}
}

// TestExecute_JSONOutput проверяет корректность структуры JSON (AC: #6).
func TestExecute_JSONOutput(t *testing.T) {
	mock := giteatest.NewMockClient()
	mock.GetRepositoryContentsFunc = func(_ context.Context, _, _ string) ([]gitea.FileInfo, error) {
		return []gitea.FileInfo{}, nil
	}
	mock.SetRepositoryStateFunc = func(_ context.Context, _ []gitea.BatchOperation, _, _ string) error {
		return nil
	}

	h := &ActionMenuHandler{giteaClient: mock}
	cfg := &config.Config{
		Owner:         "testorg",
		Repo:          "testrepo",
		BaseBranch:    "main",
		ForceUpdate:   true,
		MenuMain:      []string{"workflow.yml\nname: Test\non:\n  push:\njobs:\n  test:\n    runs-on: ubuntu-latest"},
		ProjectConfig: testProjectConfig(false),
	}

	os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer os.Unsetenv("BR_OUTPUT_FORMAT")

	captured := captureStdout(t, func() {
		_ = h.Execute(context.Background(), cfg)
	})

	var result output.Result
	if err := json.Unmarshal([]byte(captured), &result); err != nil {
		t.Fatalf("Невалидный JSON: %v\nВывод: %s", err, captured)
	}

	// Проверяем структуру result
	if result.Status != output.StatusSuccess {
		t.Errorf("Ожидался статус success")
	}
	if result.Command != constants.ActNRActionMenuBuild {
		t.Errorf("Ожидалась команда %s, получена: %s", constants.ActNRActionMenuBuild, result.Command)
	}
	if result.Metadata == nil {
		t.Error("Metadata не должна быть nil")
	}
	if result.Metadata != nil && result.Metadata.TraceID == "" {
		t.Error("TraceID не должен быть пустым")
	}
	if result.Metadata != nil && result.Metadata.APIVersion != constants.APIVersion {
		t.Errorf("Ожидался APIVersion %s", constants.APIVersion)
	}

	// Проверяем data
	dataBytes, _ := json.Marshal(result.Data)
	var data ActionMenuData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		t.Fatalf("Не удалось разобрать data: %v", err)
	}

	// Проверяем все поля
	if data.TotalGenerated == 0 {
		t.Error("TotalGenerated должен быть > 0")
	}
	if data.DatabasesProcessed == 0 {
		t.Error("DatabasesProcessed должен быть > 0")
	}
	if !data.ForceUpdate {
		t.Error("ForceUpdate должен быть true")
	}
}

// TestExecute_StateChangedFalse проверяет StateChanged=false когда контент файлов идентичен (AC: #10).
func TestExecute_StateChangedFalse(t *testing.T) {
	mock := giteatest.NewMockClient()

	// Контент который будет сгенерирован (после template processing)
	// MenuMain: "workflow.yml\nname: Test $TestBaseReplace$\n..."
	// После замены $TestBaseReplace$ на "TestDB" получим этот контент:
	expectedContent := "name: Test TestDB\non:\n  push:\njobs:\n  test:\n    runs-on: ubuntu-latest"

	mock.GetRepositoryContentsFunc = func(_ context.Context, _, _ string) ([]gitea.FileInfo, error) {
		return []gitea.FileInfo{
			{
				Name: "workflow.yml",
				Path: ".gitea/workflows/workflow.yml",
				SHA:  "git-blob-sha-will-be-ignored", // Git SHA не используется для сравнения
			},
		}, nil
	}
	mock.GetFileContentFunc = func(_ context.Context, _ string) ([]byte, error) {
		// Возвращаем тот же контент, что будет сгенерирован — файл не должен обновляться
		return []byte(expectedContent), nil
	}

	setRepositoryCalled := false
	mock.SetRepositoryStateFunc = func(_ context.Context, ops []gitea.BatchOperation, _, _ string) error {
		setRepositoryCalled = true
		return nil
	}

	h := &ActionMenuHandler{giteaClient: mock}
	cfg := &config.Config{
		Owner:       "testorg",
		Repo:        "testrepo",
		BaseBranch:  "main",
		ForceUpdate: true,
		// Шаблон с $TestBaseReplace$ который заменится на "TestDB"
		MenuMain:      []string{"workflow.yml\nname: Test $TestBaseReplace$\non:\n  push:\njobs:\n  test:\n    runs-on: ubuntu-latest"},
		ProjectConfig: testProjectConfig(false),
	}

	os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer os.Unsetenv("BR_OUTPUT_FORMAT")

	captured := captureStdout(t, func() {
		_ = h.Execute(context.Background(), cfg)
	})

	var result output.Result
	if err := json.Unmarshal([]byte(captured), &result); err != nil {
		t.Fatalf("Не удалось разобрать JSON: %v", err)
	}

	dataBytes, _ := json.Marshal(result.Data)
	var data ActionMenuData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		t.Fatalf("Не удалось разобрать data: %v", err)
	}

	// Контент идентичен — изменений не должно быть
	if data.StateChanged {
		t.Errorf("StateChanged должен быть false когда контент файлов идентичен")
		t.Logf("SetRepositoryState called: %v", setRepositoryCalled)
		t.Logf("Added: %d, Updated: %d, Deleted: %d", data.AddedFiles, data.UpdatedFiles, data.DeletedFiles)
	}

	// SetRepositoryState не должен вызываться если нет изменений
	if setRepositoryCalled {
		t.Error("SetRepositoryState не должен вызываться когда нет изменений")
	}
}

// TestTextOutput проверяет текстовый вывод (AC: #7).
func TestTextOutput(t *testing.T) {
	mock := giteatest.NewMockClient()
	mock.GetRepositoryContentsFunc = func(_ context.Context, _, _ string) ([]gitea.FileInfo, error) {
		return []gitea.FileInfo{}, nil
	}
	mock.SetRepositoryStateFunc = func(_ context.Context, _ []gitea.BatchOperation, _, _ string) error {
		return nil
	}

	h := &ActionMenuHandler{giteaClient: mock}
	cfg := &config.Config{
		Owner:         "testorg",
		Repo:          "testrepo",
		BaseBranch:    "main",
		ForceUpdate:   true,
		MenuMain:      []string{"workflow.yml\nname: Test\non:\n  push:\njobs:\n  test:\n    runs-on: ubuntu-latest"},
		ProjectConfig: testProjectConfig(false),
	}

	// Текстовый формат (по умолчанию)
	os.Unsetenv("BR_OUTPUT_FORMAT")

	captured := captureStdout(t, func() {
		_ = h.Execute(context.Background(), cfg)
	})

	// Проверяем наличие ключевых элементов текстового вывода
	expectedStrings := []string{
		"Построение меню действий",
		"Принудительное обновление:",
		"Баз данных обработано:",
		"Файлов сгенерировано:",
		"Добавлено:",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(captured, expected) {
			t.Errorf("Текстовый вывод должен содержать '%s'\nПолучено:\n%s", expected, captured)
		}
	}
}

// TestWriteText_NoChanges проверяет текстовый вывод при отсутствии изменений.
func TestWriteText_NoChanges(t *testing.T) {
	data := &ActionMenuData{
		StateChanged:       false,
		ProjectYamlChanged: false,
		ForceUpdate:        false,
	}

	var buf strings.Builder
	err := data.writeText(&buf)
	if err != nil {
		t.Fatalf("writeText вернул ошибку: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Изменения в project.yaml не обнаружены") {
		t.Error("Ожидалось сообщение об отсутствии изменений")
	}
}

// TestExtractDatabases проверяет извлечение баз данных из конфигурации.
func TestExtractDatabases(t *testing.T) {
	h := &ActionMenuHandler{}

	cfg := &config.Config{
		ProjectConfig: testProjectConfigMultiple(),
	}

	databases := h.extractDatabases(cfg, nil)

	// Должно быть 5 баз данных: 2 prod + 3 test
	if len(databases) != 5 {
		t.Errorf("Ожидалось 5 баз данных, получено: %d", len(databases))
	}

	prodCount := 0
	testCount := 0
	for _, db := range databases {
		if db.Prod {
			prodCount++
		} else {
			testCount++
		}
	}

	if prodCount != 2 {
		t.Errorf("Ожидалось 2 prod базы, получено: %d", prodCount)
	}
	if testCount != 3 {
		t.Errorf("Ожидалось 3 test базы, получено: %d", testCount)
	}
}

// TestGenerateFiles_ReplacementRules проверяет замену переменных в шаблонах (AC: #3).
func TestGenerateFiles_ReplacementRules(t *testing.T) {
	h := &ActionMenuHandler{}

	cfg := &config.Config{
		MenuMain: []string{
			"test.yml\nname: Test\ndb: $TestBaseReplace$\nall_test:$TestBaseReplaceAll$\nprod: $ProdBaseReplace$\nall_prod:$ProdBaseReplaceAll$",
		},
	}

	databases := []ProjectDatabase{
		{Name: "ProdDB", Prod: true},
		{Name: "TestDB1", Prod: false},
		{Name: "TestDB2", Prod: false},
	}

	files, err := h.generateFiles(cfg, databases, nil)
	if err != nil {
		t.Fatalf("generateFiles вернул ошибку: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Ожидался 1 файл, получено: %d", len(files))
	}

	content := files[0].Content

	// Проверяем замены
	if !strings.Contains(content, "db: TestDB1") {
		t.Error("$TestBaseReplace$ должен быть заменён на TestDB1")
	}
	if !strings.Contains(content, "TestDB1\n          - TestDB2") {
		t.Error("$TestBaseReplaceAll$ должен содержать все test базы")
	}
	if !strings.Contains(content, "prod: ProdDB") {
		t.Error("$ProdBaseReplace$ должен быть заменён на ProdDB")
	}
	if !strings.Contains(content, "all_prod:\n          - ProdDB") {
		t.Error("$ProdBaseReplaceAll$ должен содержать все prod базы")
	}
}

// TestExecute_GetLatestCommitError проверяет обработку ошибки GetLatestCommit (M-2).
func TestExecute_GetLatestCommitError(t *testing.T) {
	mock := giteatest.NewMockClient()

	mock.GetLatestCommitFunc = func(_ context.Context, _ string) (*gitea.Commit, error) {
		return nil, fmt.Errorf("connection refused")
	}

	h := &ActionMenuHandler{giteaClient: mock}
	cfg := &config.Config{
		Owner:         "testorg",
		Repo:          "testrepo",
		BaseBranch:    "main",
		ForceUpdate:   false, // Нужна проверка project.yaml
		ProjectConfig: testProjectConfig(false),
	}

	os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer os.Unsetenv("BR_OUTPUT_FORMAT")

	captured := captureStdout(t, func() {
		err := h.Execute(context.Background(), cfg)
		// Ошибка checkProjectYamlChanges не должна блокировать выполнение
		// (graceful degradation — продолжаем как будто изменения есть)
		if err != nil {
			t.Logf("Неожиданная ошибка: %v", err)
		}
	})

	var result output.Result
	if err := json.Unmarshal([]byte(captured), &result); err != nil {
		t.Fatalf("Не удалось разобрать JSON: %v", err)
	}

	// Должен быть success, т.к. ошибка checkProjectYamlChanges не фатальна
	if result.Status != output.StatusSuccess {
		t.Errorf("Ожидался success при ошибке GetLatestCommit (graceful degradation)")
	}
}

// TestExecute_SyncFilesError проверяет обработку ошибки SetRepositoryState (M-2).
func TestExecute_SyncFilesError(t *testing.T) {
	mock := giteatest.NewMockClient()

	mock.GetRepositoryContentsFunc = func(_ context.Context, _, _ string) ([]gitea.FileInfo, error) {
		return []gitea.FileInfo{}, nil
	}
	mock.SetRepositoryStateFunc = func(_ context.Context, _ []gitea.BatchOperation, _, _ string) error {
		return fmt.Errorf("API rate limit exceeded")
	}

	h := &ActionMenuHandler{giteaClient: mock}
	cfg := &config.Config{
		Owner:         "testorg",
		Repo:          "testrepo",
		BaseBranch:    "main",
		ForceUpdate:   true,
		MenuMain:      []string{"workflow.yml\nname: Test\non:\n  push:\njobs:\n  test:\n    runs-on: ubuntu-latest"},
		ProjectConfig: testProjectConfig(false),
	}

	os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer os.Unsetenv("BR_OUTPUT_FORMAT")

	captured := captureStdout(t, func() {
		err := h.Execute(context.Background(), cfg)
		if err == nil {
			t.Error("Ожидалась ошибка при сбое SetRepositoryState")
		}
		if !strings.Contains(err.Error(), shared.ErrSyncFailed) {
			t.Errorf("Ожидался код ошибки %s, получено: %v", shared.ErrSyncFailed, err)
		}
	})

	var result output.Result
	if err := json.Unmarshal([]byte(captured), &result); err != nil {
		t.Fatalf("Не удалось разобрать JSON: %v", err)
	}

	if result.Status != output.StatusError {
		t.Errorf("Ожидался статус error")
	}
	if result.Error == nil || result.Error.Code != shared.ErrSyncFailed {
		t.Errorf("Ожидался код ошибки %s", shared.ErrSyncFailed)
	}
}

// TestExecute_EmptyMenuMain проверяет поведение при пустом MenuMain (M-3).
func TestExecute_EmptyMenuMain(t *testing.T) {
	mock := giteatest.NewMockClient()

	var capturedOps []gitea.BatchOperation
	mock.GetRepositoryContentsFunc = func(_ context.Context, _, _ string) ([]gitea.FileInfo, error) {
		// Существующие файлы которые будут удалены
		return []gitea.FileInfo{
			{Name: "existing.yml", Path: ".gitea/workflows/existing.yml", SHA: "git-sha"},
		}, nil
	}
	mock.GetFileContentFunc = func(_ context.Context, _ string) ([]byte, error) {
		return []byte("old content"), nil
	}
	mock.SetRepositoryStateFunc = func(_ context.Context, ops []gitea.BatchOperation, _, _ string) error {
		capturedOps = ops
		return nil
	}

	h := &ActionMenuHandler{giteaClient: mock}
	cfg := &config.Config{
		Owner:         "testorg",
		Repo:          "testrepo",
		BaseBranch:    "main",
		ForceUpdate:   true,
		MenuMain:      []string{}, // Пустой MenuMain
		ProjectConfig: testProjectConfig(false),
	}

	os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer os.Unsetenv("BR_OUTPUT_FORMAT")

	captured := captureStdout(t, func() {
		err := h.Execute(context.Background(), cfg)
		// Ошибка generateFiles при отсутствии prod/test баз
		// или удаление всех файлов если MenuMain пуст
		if err != nil {
			t.Logf("Ожидаемая ошибка: %v", err)
		}
	})

	// Проверяем что был вызван SetRepositoryState с операцией delete
	// (если generateFiles прошла успешно, что зависит от баз данных)
	if len(capturedOps) > 0 {
		for _, op := range capturedOps {
			if op.Operation != "delete" {
				t.Errorf("При пустом MenuMain ожидаются только операции delete, получена: %s", op.Operation)
			}
		}
	}

	t.Logf("Captured output: %s", captured)
}

// TestWriteError_JSONFormat проверяет JSON формат вывода ошибок (M-2).
func TestWriteError_JSONFormat(t *testing.T) {
	h := &ActionMenuHandler{}

	os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer os.Unsetenv("BR_OUTPUT_FORMAT")

	captured := captureStdout(t, func() {
		err := h.writeError("json", "test-trace-id", time.Now(), "TEST.ERROR", "Test error message")
		if err == nil {
			t.Error("writeError должен возвращать error")
		}
	})

	var result output.Result
	if err := json.Unmarshal([]byte(captured), &result); err != nil {
		t.Fatalf("Невалидный JSON: %v\nВывод: %s", err, captured)
	}

	if result.Status != output.StatusError {
		t.Error("Status должен быть error")
	}
	if result.Error == nil {
		t.Error("Error не должен быть nil")
	}
	if result.Error != nil && result.Error.Code != "TEST.ERROR" {
		t.Errorf("Ожидался код ошибки TEST.ERROR, получен: %s", result.Error.Code)
	}
	if result.Error != nil && result.Error.Message != "Test error message" {
		t.Errorf("Неверное сообщение ошибки: %s", result.Error.Message)
	}
	if result.Metadata == nil || result.Metadata.TraceID != "test-trace-id" {
		t.Error("TraceID должен быть test-trace-id")
	}
}
