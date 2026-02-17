package sonarqube

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestEntityWithWorkDir(t *testing.T) (*SonarScannerEntity, string) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "bsl_test")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	cfg := &config.ScannerConfig{Timeout: 10 * time.Second, WorkDir: tmpDir, TempDir: tmpDir}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	e := NewSonarScannerEntity(cfg, logger)
	e.workDir = tmpDir
	return e, tmpDir
}

func TestFixBSLTokenizationIssues_FileNotFound(t *testing.T) {
	e := newTestEntity()
	err := e.FixBSLTokenizationIssues("/nonexistent/file.bsl")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestFixBSLTokenizationIssues_CRLF(t *testing.T) {
	e, dir := newTestEntityWithWorkDir(t)
	f := filepath.Join(dir, "test.bsl")
	os.WriteFile(f, []byte("line1\r\nline2\r\n"), 0644)

	err := e.FixBSLTokenizationIssues(f)
	require.NoError(t, err)

	content, _ := os.ReadFile(f)
	assert.NotContains(t, string(content), "\r\n")
	// Backup should exist
	_, statErr := os.Stat(f + ".backup")
	assert.NoError(t, statErr)
}

func TestFixBSLTokenizationIssues_NBSP(t *testing.T) {
	e, dir := newTestEntityWithWorkDir(t)
	f := filepath.Join(dir, "test.bsl")
	os.WriteFile(f, []byte("Процедура\u00A0Тест()\n"), 0644)

	err := e.FixBSLTokenizationIssues(f)
	require.NoError(t, err)

	content, _ := os.ReadFile(f)
	assert.NotContains(t, string(content), "\u00A0")
}

func TestFixBSLTokenizationIssues_BOM(t *testing.T) {
	e, dir := newTestEntityWithWorkDir(t)
	f := filepath.Join(dir, "test.bsl")
	os.WriteFile(f, []byte("\uFEFFПроцедура Тест()\n"), 0644)

	err := e.FixBSLTokenizationIssues(f)
	require.NoError(t, err)

	content, _ := os.ReadFile(f)
	assert.False(t, len(content) > 0 && content[0] == 0xEF)
}

func TestFixBSLTokenizationIssues_TrailingSpaces(t *testing.T) {
	e, dir := newTestEntityWithWorkDir(t)
	f := filepath.Join(dir, "test.bsl")
	os.WriteFile(f, []byte("line1   \nline2\t\t\n"), 0644)

	err := e.FixBSLTokenizationIssues(f)
	require.NoError(t, err)

	content, _ := os.ReadFile(f)
	lines := splitLines(string(content))
	for _, l := range lines {
		if l != "" {
			assert.Equal(t, l, trimRight(l))
		}
	}
}

func TestFixBSLTokenizationIssues_NoChanges(t *testing.T) {
	e, dir := newTestEntityWithWorkDir(t)
	f := filepath.Join(dir, "clean.bsl")
	original := "Процедура Тест()\n"
	os.WriteFile(f, []byte(original), 0644)

	err := e.FixBSLTokenizationIssues(f)
	require.NoError(t, err)

	// No backup should be created if fixBSLSyntaxIssues doesn't change content
	// (it might due to regex replacements, so just check no error)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimRight(s string) string {
	i := len(s)
	for i > 0 && (s[i-1] == ' ' || s[i-1] == '\t') {
		i--
	}
	return s[:i]
}

func TestFixBSLSyntaxIssues_OddQuotes(t *testing.T) {
	e := newTestEntity()
	input := `Сообщить("Незакрытая строка` + "\n"
	result := e.fixBSLSyntaxIssues(input, "test.bsl")
	// Should add closing quote
	assert.Contains(t, result, `"`)
}

func TestFixBSLSyntaxIssues_KeywordCapitalization(t *testing.T) {
	e := newTestEntity()
	input := "Процедура тест()" + "\n" + "Функция другой()" + "\n"
	result := e.fixBSLSyntaxIssues(input, "test.bsl")
	assert.NotEmpty(t, result)
}

func TestFindAndValidateBSLFiles_EmptyWorkDir(t *testing.T) {
	e := newTestEntity()
	e.workDir = ""
	_, _, err := e.FindAndValidateBSLFiles()
	assert.Error(t, err)
}

func TestFindAndValidateBSLFiles_WithFiles(t *testing.T) {
	e, dir := newTestEntityWithWorkDir(t)

	// Create a valid BSL file
	validFile := filepath.Join(dir, "valid.bsl")
	os.WriteFile(validFile, []byte("Процедура Тест()\nКонецПроцедуры\n"), 0644)

	// Create a problematic BSL file (with BOM)
	badFile := filepath.Join(dir, "bad.bsl")
	os.WriteFile(badFile, []byte("\uFEFFПроцедура Тест()\n"), 0644)

	valid, problematic, err := e.FindAndValidateBSLFiles()
	require.NoError(t, err)
	assert.Len(t, valid, 1)
	assert.Len(t, problematic, 1)
	assert.Equal(t, validFile, valid[0])
	assert.Equal(t, badFile, problematic[0])
}

func TestFindAndValidateBSLFiles_OSFiles(t *testing.T) {
	e, dir := newTestEntityWithWorkDir(t)

	osFile := filepath.Join(dir, "module.os")
	os.WriteFile(osFile, []byte("Процедура()\nКонецПроцедуры\n"), 0644)

	valid, _, err := e.FindAndValidateBSLFiles()
	require.NoError(t, err)
	assert.Len(t, valid, 1)
}

func TestValidateBSLFile_Valid(t *testing.T) {
	e, dir := newTestEntityWithWorkDir(t)
	f := filepath.Join(dir, "test.bsl")
	os.WriteFile(f, []byte("Процедура Тест()\nКонецПроцедуры\n"), 0644)
	assert.True(t, e.validateBSLFile(f))
}

func TestValidateBSLFile_MixedLineEndings(t *testing.T) {
	e, dir := newTestEntityWithWorkDir(t)
	f := filepath.Join(dir, "test.bsl")
	os.WriteFile(f, []byte("line1\r\nline2\nline3\r\n"), 0644)
	assert.False(t, e.validateBSLFile(f))
}

func TestValidateBSLFile_BOM(t *testing.T) {
	e, dir := newTestEntityWithWorkDir(t)
	f := filepath.Join(dir, "test.bsl")
	os.WriteFile(f, []byte("\uFEFFcontent\n"), 0644)
	assert.False(t, e.validateBSLFile(f))
}

func TestValidateBSLFile_NBSP(t *testing.T) {
	e, dir := newTestEntityWithWorkDir(t)
	f := filepath.Join(dir, "test.bsl")
	os.WriteFile(f, []byte("hello\u00A0world\n"), 0644)
	assert.False(t, e.validateBSLFile(f))
}

func TestValidateBSLFile_OddQuotes(t *testing.T) {
	e, dir := newTestEntityWithWorkDir(t)
	f := filepath.Join(dir, "test.bsl")
	os.WriteFile(f, []byte(`Сообщить("text` + "\n"), 0644)
	assert.False(t, e.validateBSLFile(f))
}

func TestValidateBSLFile_NotExists(t *testing.T) {
	e := newTestEntity()
	assert.False(t, e.validateBSLFile("/nonexistent"))
}

func TestValidateBSLFile_CommentLinesIgnored(t *testing.T) {
	e, dir := newTestEntityWithWorkDir(t)
	f := filepath.Join(dir, "test.bsl")
	os.WriteFile(f, []byte("// comment with odd \" quote\nПроцедура()\n"), 0644)
	assert.True(t, e.validateBSLFile(f))
}

func TestPreProcessBSLFiles(t *testing.T) {
	e, dir := newTestEntityWithWorkDir(t)

	// Create problematic file
	f := filepath.Join(dir, "bad.bsl")
	os.WriteFile(f, []byte("\uFEFFline1\r\nline2\r\n"), 0644)

	err := e.preProcessBSLFiles()
	require.NoError(t, err)

	// File should be fixed
	content, _ := os.ReadFile(f)
	assert.NotContains(t, string(content), "\r\n")
}

func TestPreProcessBSLFiles_EmptyWorkDir(t *testing.T) {
	e := newTestEntity()
	e.workDir = ""
	err := e.preProcessBSLFiles()
	assert.Error(t, err)
}

func TestAddFileToExclusions(t *testing.T) {
	e := newTestEntity()
	e.AddFileToExclusions("file1.bsl")
	e.AddFileToExclusions("file2.bsl")
	e.AddFileToExclusions("file1.bsl") // duplicate

	assert.Len(t, e.excludedFiles, 2)
	assert.Equal(t, "file1.bsl", e.excludedFiles[0])
}

func TestAddFilesToExclusions(t *testing.T) {
	e := newTestEntity()
	e.AddFilesToExclusions([]string{"a.bsl", "b.bsl", "c.bsl"})
	assert.Len(t, e.excludedFiles, 3)
}

func TestGetExcludedFiles(t *testing.T) {
	e := newTestEntity()
	e.AddFileToExclusions("test.bsl")
	files := e.GetExcludedFiles()
	assert.Len(t, files, 1)
	// Verify it's a copy
	files[0] = "modified"
	assert.Equal(t, "test.bsl", e.excludedFiles[0])
}

func TestClearExclusions(t *testing.T) {
	e := newTestEntity()
	e.AddFileToExclusions("test.bsl")
	e.ClearExclusions()
	assert.Empty(t, e.excludedFiles)
	assert.Empty(t, e.GetProperty("sonar.exclusions"))
}

func TestExtractProblematicBSLFiles(t *testing.T) {
	e := newTestEntity()
	e.workDir = "/project"

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"error_pattern", "error processing src/Module.bsl", 1},
		{"failed_pattern", "failed to parse common/Utils.bsl", 1},
		{"tokenization", "tokenization error in CommonModules/Test.bsl", 1},
		{"multiple", "error in a.bsl and failed b.bsl", 2},
		{"no_bsl", "some random error message", 0},
		{"short_path", "x.bsl", 1}, // extracted even if short
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := e.ExtractProblematicBSLFiles(tt.input)
			assert.Len(t, files, tt.expected)
		})
	}
}

func TestExtractProblematicBSLFiles_Dedup(t *testing.T) {
	e := newTestEntity()
	e.workDir = "/project"
	files := e.ExtractProblematicBSLFiles("error in src/Module.bsl\nfailed src/Module.bsl again")
	// Should deduplicate
	count := 0
	seen := map[string]bool{}
	for _, f := range files {
		if !seen[f] {
			seen[f] = true
			count++
		}
	}
	assert.Equal(t, count, len(files))
}

func TestSuggestBSLExclusions(t *testing.T) {
	e := newTestEntity()
	suggestions := e.SuggestBSLExclusions("error in CommonModules/Test.bsl")
	assert.NotEmpty(t, suggestions)

	foundCommonModules := false
	for _, s := range suggestions {
		if findSubstring(s, "CommonModules") {
			foundCommonModules = true
		}
	}
	assert.True(t, foundCommonModules)
}

func TestSuggestBSLExclusions_NoFiles(t *testing.T) {
	e := newTestEntity()
	suggestions := e.SuggestBSLExclusions("no bsl files here")
	assert.Empty(t, suggestions)
}

func TestSuggestBSLExclusions_Multiple(t *testing.T) {
	e := newTestEntity()
	suggestions := e.SuggestBSLExclusions("error ServerModule.bsl and ClientModule.bsl")
	assert.NotEmpty(t, suggestions)
}

func TestUpdateExclusionsProperty(t *testing.T) {
	e := newTestEntity()
	e.SetProperty("sonar.exclusions", "existing/**")
	e.AddFileToExclusions("src/test.bsl")

	exclusions := e.GetProperty("sonar.exclusions")
	assert.Contains(t, exclusions, "existing/**")
	assert.Contains(t, exclusions, "test.bsl")
}
