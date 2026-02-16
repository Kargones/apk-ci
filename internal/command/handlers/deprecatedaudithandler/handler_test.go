package deprecatedaudithandler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/pkg/output"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditReport_TextFormat(t *testing.T) {
	report := &AuditReport{
		DeprecatedAliases: []DeprecatedAliasInfo{
			{DeprecatedName: "dbrestore", NRName: "nr-dbrestore", HandlerPackage: "dbrestorehandler"},
			{DeprecatedName: "convert", NRName: "nr-convert", HandlerPackage: "converthandler"},
		},
		TodoComments: []TodoInfo{
			{File: "handler.go", Line: 40, Text: "Удалить в v2.0.0", Tag: "H-7"},
		},
		LegacyCases: []LegacyCaseInfo{
			{File: "main.go", Line: 300, CaseValue: "dbrestore", Note: "deprecated alias for nr-dbrestore"},
		},
		Summary: AuditSummary{
			TotalDeprecatedAliases: 2,
			TotalTodoH7:            1,
			TotalLegacyCases:       1,
			ReadyForRemoval:        "no",
			Message:                "2 deprecated aliases, 1 TODO(H-7) комментариев, 1 legacy case-веток",
		},
	}

	var buf bytes.Buffer
	err := writeTextReport(&buf, report)
	require.NoError(t, err)

	text := buf.String()
	assert.Contains(t, text, "Deprecated Code Audit Report")
	assert.Contains(t, text, "Deprecated Aliases (2)")
	assert.Contains(t, text, "dbrestore")
	assert.Contains(t, text, "nr-dbrestore")
	assert.Contains(t, text, "dbrestorehandler")
	assert.Contains(t, text, "TODO(H-7) Comments (1)")
	assert.Contains(t, text, "handler.go:40")
	assert.Contains(t, text, "Legacy Switch Cases (1)")
	assert.Contains(t, text, "main.go:300")
	assert.Contains(t, text, "Ready for removal: no")
}

func TestAuditReport_JSONFormat(t *testing.T) {
	// Используем buildAuditReport для консистентности значений summary
	aliases := []DeprecatedAliasInfo{
		{DeprecatedName: "dbrestore", NRName: "nr-dbrestore", HandlerPackage: "dbrestorehandler"},
	}
	todos := []TodoInfo{
		{File: "handler.go", Line: 40, Text: "Удалить в v2.0.0", Tag: "H-7"},
	}
	legacyCases := []LegacyCaseInfo{
		{File: "main.go", Line: 300, CaseValue: "dbrestore", Note: "deprecated alias for nr-dbrestore"},
	}
	report := buildAuditReport(aliases, todos, legacyCases)

	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: "nr-deprecated-audit",
		Data:    report,
	}

	var buf bytes.Buffer
	writer := output.NewWriter("json")
	err := writer.Write(&buf, result)
	require.NoError(t, err)

	// Парсим JSON для проверки структуры
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))

	assert.Equal(t, "success", parsed["status"])
	assert.Equal(t, "nr-deprecated-audit", parsed["command"])

	data, ok := parsed["data"].(map[string]any)
	require.True(t, ok, "data должен быть объектом")

	parsedAliases, aliasOk := data["deprecated_aliases"].([]any)
	require.True(t, aliasOk, "deprecated_aliases должен быть массивом")
	assert.Len(t, parsedAliases, 1)

	summary, summaryOk := data["summary"].(map[string]any)
	require.True(t, summaryOk, "summary должен быть объектом")
	assert.Equal(t, "no", summary["ready_for_removal"])
}

func TestBuildAuditReport(t *testing.T) {
	aliases := []DeprecatedAliasInfo{
		{DeprecatedName: "a1", NRName: "nr-a1", HandlerPackage: "pkg1"},
		{DeprecatedName: "a2", NRName: "nr-a2", HandlerPackage: "pkg2"},
	}
	todos := []TodoInfo{
		{File: "f.go", Line: 1, Text: "remove", Tag: "H-7"},
		{File: "f.go", Line: 5, Text: "old api", Tag: "Deprecated"},
	}
	legacyCases := []LegacyCaseInfo{
		{File: "main.go", Line: 10, CaseValue: "a1"},
	}

	report := buildAuditReport(aliases, todos, legacyCases)

	assert.Equal(t, 2, report.Summary.TotalDeprecatedAliases)
	assert.Equal(t, 1, report.Summary.TotalTodoH7) // Только H-7, не Deprecated
	assert.Equal(t, 1, report.Summary.TotalLegacyCases)
	// "no" — есть legacy cases, значит ещё не готово к удалению
	assert.Equal(t, "no", report.Summary.ReadyForRemoval)
	assert.Contains(t, report.Summary.Message, "2 deprecated aliases")
}

func TestBuildAuditReport_Empty(t *testing.T) {
	report := buildAuditReport(nil, nil, nil)

	assert.Equal(t, 0, report.Summary.TotalDeprecatedAliases)
	assert.Equal(t, 0, report.Summary.TotalTodoH7)
	assert.Equal(t, 0, report.Summary.TotalLegacyCases)
	assert.Equal(t, "yes", report.Summary.ReadyForRemoval)
}

func TestBuildAuditReport_AllCoveredByTodo(t *testing.T) {
	aliases := []DeprecatedAliasInfo{
		{DeprecatedName: "a1", NRName: "nr-a1", HandlerPackage: "pkg1"},
		{DeprecatedName: "a2", NRName: "nr-a2", HandlerPackage: "pkg2"},
	}
	todos := []TodoInfo{
		{File: "f1.go", Line: 1, Text: "remove a1", Tag: "H-7"},
		{File: "f2.go", Line: 5, Text: "remove a2", Tag: "H-7"},
	}
	legacyCases := []LegacyCaseInfo{
		{File: "main.go", Line: 10, CaseValue: "a1"},
	}

	report := buildAuditReport(aliases, todos, legacyCases)

	assert.Equal(t, 2, report.Summary.TotalDeprecatedAliases)
	assert.Equal(t, 2, report.Summary.TotalTodoH7)
	assert.Equal(t, 1, report.Summary.TotalLegacyCases)
	// "no" — legacy case-ветки ещё есть в switch, даже если aliases покрыты TODO
	assert.Equal(t, "no", report.Summary.ReadyForRemoval)
}

func TestBuildAuditReport_PartialWhenNoLegacyCases(t *testing.T) {
	aliases := []DeprecatedAliasInfo{
		{DeprecatedName: "a1", NRName: "nr-a1", HandlerPackage: "pkg1"},
	}
	todos := []TodoInfo{
		{File: "f.go", Line: 1, Text: "remove", Tag: "H-7"},
	}
	// Нет legacy case-веток — aliases + TODO, но legacy уже вынесен в registry
	report := buildAuditReport(aliases, todos, nil)

	assert.Equal(t, 1, report.Summary.TotalDeprecatedAliases)
	assert.Equal(t, 1, report.Summary.TotalTodoH7)
	assert.Equal(t, 0, report.Summary.TotalLegacyCases)
	// "partial" — aliases покрыты TODO, legacy уже удалён
	assert.Equal(t, "partial", report.Summary.ReadyForRemoval)
}

func TestDeprecatedAuditHandler_PlanOnly(t *testing.T) {
	t.Setenv("BR_PLAN_ONLY", "true")
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	h := &DeprecatedAuditHandler{}
	err := h.Execute(t.Context(), nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	require.NoError(t, err)
	text := buf.String()
	assert.Contains(t, text, "не поддерживает отображение плана")
	// Не должен содержать отчёт аудита
	assert.NotContains(t, text, "Deprecated Code Audit Report")
}

func TestWriteError_JSON(t *testing.T) {
	var buf bytes.Buffer
	err := writeError(&buf, output.FormatJSON, "test-trace", time.Now(), fmt.Errorf("test error"))

	require.Error(t, err)
	assert.Equal(t, "test error", err.Error())

	// JSON ошибка должна быть записана в writer
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Equal(t, "error", parsed["status"])
	assert.Equal(t, "nr-deprecated-audit", parsed["command"])
}

func TestWriteError_Text(t *testing.T) {
	var buf bytes.Buffer
	err := writeError(&buf, "text", "test-trace", time.Now(), fmt.Errorf("test error"))

	require.Error(t, err)
	assert.Equal(t, "test error", err.Error())
	// В текстовом формате вывод ошибки делегируется main.go
	assert.Empty(t, buf.String())
}

func TestDeprecatedAuditHandler_Execute_Text(t *testing.T) {
	// Создаём временную директорию с тестовыми Go-файлами
	tmpDir := t.TempDir()

	goFile := filepath.Join(tmpDir, "test.go")
	require.NoError(t, os.WriteFile(goFile, []byte(`package test
// TODO(H-7): Удалить deprecated alias
func Foo() {}
`), 0o644))

	// Создаём mock main.go для legacy scan
	mainDir := filepath.Join(tmpDir, "cmd", "apk-ci")
	require.NoError(t, os.MkdirAll(mainDir, 0o755))
	mainGo := filepath.Join(mainDir, "main.go")
	require.NoError(t, os.WriteFile(mainGo, []byte(`package main
func run() int {
	switch cfg.Command {
	case constants.ActStore2db:
		return 0
	default:
		return 2
	}
}
`), 0o644))

	// Устанавливаем env для аудита
	t.Setenv("BR_AUDIT_ROOT", tmpDir)
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	// Перехватываем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	h := &DeprecatedAuditHandler{}
	err := h.Execute(t.Context(), nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	require.NoError(t, err)
	text := buf.String()
	assert.Contains(t, text, "Deprecated Code Audit Report")
	assert.Contains(t, text, "Deprecated Aliases")
	assert.Contains(t, text, "TODO(H-7) Comments")
	assert.Contains(t, text, "Legacy Switch Cases")
	assert.Contains(t, text, "Summary")
}

func TestDeprecatedAuditHandler_Execute_JSON(t *testing.T) {
	tmpDir := t.TempDir()

	goFile := filepath.Join(tmpDir, "test.go")
	require.NoError(t, os.WriteFile(goFile, []byte(`package test
// TODO(H-7): Удалить
func Bar() {}
`), 0o644))

	// Создаём mock main.go
	mainDir := filepath.Join(tmpDir, "cmd", "apk-ci")
	require.NoError(t, os.MkdirAll(mainDir, 0o755))
	mainGo := filepath.Join(mainDir, "main.go")
	require.NoError(t, os.WriteFile(mainGo, []byte(`package main
func run() int {
	switch cfg.Command {
	default:
		return 2
	}
}
`), 0o644))

	t.Setenv("BR_AUDIT_ROOT", tmpDir)
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	h := &DeprecatedAuditHandler{}
	err := h.Execute(t.Context(), nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Equal(t, "success", parsed["status"])
	assert.Equal(t, "nr-deprecated-audit", parsed["command"])

	data, ok := parsed["data"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, data, "deprecated_aliases")
	assert.Contains(t, data, "todo_comments")
	assert.Contains(t, data, "legacy_cases")
	assert.Contains(t, data, "summary")
}

func TestDeprecatedAuditHandler_NameAndDescription(t *testing.T) {
	h := &DeprecatedAuditHandler{}
	assert.Equal(t, "nr-deprecated-audit", h.Name())
	assert.NotEmpty(t, h.Description())
}

// TestDeprecatedAuditHandler_PathTraversal проверяет что BR_AUDIT_ROOT
// с выходом за пределы рабочей директории через ".." возвращает ошибку.
// Review #32: защита от path traversal.
func TestDeprecatedAuditHandler_PathTraversal(t *testing.T) {
	t.Setenv("BR_AUDIT_ROOT", "../../etc")
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	h := &DeprecatedAuditHandler{}
	err := h.Execute(t.Context(), nil)
	assert.Error(t, err, "path traversal через '..' должен быть отклонён")
	assert.Contains(t, err.Error(), "недопустимый путь")
}

// TestDeprecatedAuditHandler_DryRunOverPlanOnly проверяет приоритет dry-run
// над plan-only в handler (Story 7.3 AC-11).
// Review #32: тест приоритета режимов на уровне handler.
func TestDeprecatedAuditHandler_DryRunOverPlanOnly(t *testing.T) {
	tmpDir := t.TempDir()

	// Создаём минимальный Go-файл и main.go
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.go"),
		[]byte("package test\n"), 0o644))
	mainDir := filepath.Join(tmpDir, "cmd", "apk-ci")
	require.NoError(t, os.MkdirAll(mainDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "main.go"),
		[]byte("package main\nfunc run() int {\n\tswitch cfg.Command {\n\tdefault:\n\t\treturn 2\n\t}\n}\n"),
		0o644))

	t.Setenv("BR_AUDIT_ROOT", tmpDir)
	t.Setenv("BR_OUTPUT_FORMAT", "text")
	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_PLAN_ONLY", "true")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	h := &DeprecatedAuditHandler{}
	err := h.Execute(t.Context(), nil)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	// Audit не поддерживает dry-run, поэтому при BR_DRY_RUN=true + BR_PLAN_ONLY=true
	// plan-only НЕ должен перехватить — dry-run имеет приоритет и handler выполняется полностью.
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Deprecated Code Audit Report",
		"при dry-run+plan-only должен выполниться полный аудит, а не plan-only unsupported")
}
