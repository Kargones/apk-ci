package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseReport_Additional(t *testing.T) {
	t.Run("valid report with multiple versions", func(t *testing.T) {
		// Format: each version block ends when next "Версия:" line appears
		content := `Версия:	1
Версия конфигурации:	1.0.0.1
Пользователь:	User1
Дата создания:	01.01.2024
Время создания:	12:00:00
Комментарий:
Initial commit
Версия:	2
Версия конфигурации:	1.0.0.2
Пользователь:	User2
Дата создания:	02.01.2024
Время создания:	13:00:00
Комментарий:
Second commit
Версия:	3
Версия конфигурации:	1.0.0.3
Пользователь:	User3
Дата создания:	03.01.2024
Время создания:	14:00:00
`
		tmpDir := t.TempDir()
		reportPath := filepath.Join(tmpDir, "report.txt")
		err := os.WriteFile(reportPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		records, users, maxVer, err := ParseReport(reportPath)
		if err != nil {
			t.Fatalf("ParseReport() error = %v", err)
		}

		// Last record is added when scanner finishes, previous records when new version line appears
		if len(records) != 3 {
			t.Errorf("ParseReport() got %d records, want 3. Records: %+v", len(records), records)
		}
		if len(users) != 3 {
			t.Errorf("ParseReport() got %d users, want 3. Users: %+v", len(users), users)
		}
		if maxVer != 3 {
			t.Errorf("ParseReport() got maxVer %d, want 3", maxVer)
		}
	})

	t.Run("report with same user multiple times", func(t *testing.T) {
		content := `Версия:	1
Пользователь:	SameUser
Дата создания:	01.01.2024
Время создания:	12:00:00
Версия:	2
Пользователь:	SameUser
Дата создания:	02.01.2024
Время создания:	13:00:00
`
		tmpDir := t.TempDir()
		reportPath := filepath.Join(tmpDir, "report.txt")
		err := os.WriteFile(reportPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		_, users, _, err := ParseReport(reportPath)
		if err != nil {
			t.Fatalf("ParseReport() error = %v", err)
		}

		if len(users) != 1 {
			t.Errorf("ParseReport() got %d users, want 1", len(users))
		}
	})
}

func TestParseReport_InvalidPaths(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"empty path", ""},
		{"path traversal", "../outside/file.txt"},
		{"path traversal complex", "dir/../../../etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := ParseReport(tt.path)
			if err == nil {
				t.Error("ParseReport() expected error for invalid path")
			}
		})
	}
}

func TestParseReport_NonExistentFile(t *testing.T) {
	_, _, _, err := ParseReport("/non/existent/path/report.txt")
	if err == nil {
		t.Error("ParseReport() expected error for non-existent file")
	}
}

func TestParseReport_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "empty.txt")
	err := os.WriteFile(reportPath, []byte{}, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	records, users, maxVer, err := ParseReport(reportPath)
	if err != nil {
		t.Fatalf("ParseReport() error = %v", err)
	}

	if len(records) != 0 {
		t.Errorf("ParseReport() got %d records, want 0", len(records))
	}
	if len(users) != 0 {
		t.Errorf("ParseReport() got %d users, want 0", len(users))
	}
	if maxVer != 0 {
		t.Errorf("ParseReport() got maxVer %d, want 0", maxVer)
	}
}

func TestParseReport_SingleRecord(t *testing.T) {
	content := `Версия:	42
Версия конфигурации:	2.1.3.4
Пользователь:	TestUser
Дата создания:	15.06.2024
Время создания:	14:30:45
`
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.txt")
	err := os.WriteFile(reportPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	records, users, maxVer, err := ParseReport(reportPath)
	if err != nil {
		t.Fatalf("ParseReport() error = %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("ParseReport() got %d records, want 1", len(records))
	}

	if records[0].Version != 42 {
		t.Errorf("ParseReport() version = %d, want 42", records[0].Version)
	}
	if records[0].ConfVersion != "2.1.3.4" {
		t.Errorf("ParseReport() confVersion = %s, want 2.1.3.4", records[0].ConfVersion)
	}
	if records[0].User != "TestUser" {
		t.Errorf("ParseReport() user = %s, want TestUser", records[0].User)
	}
	if records[0].Date.IsZero() {
		t.Error("ParseReport() date not parsed correctly")
	}
	if maxVer != 42 {
		t.Errorf("ParseReport() maxVer = %d, want 42", maxVer)
	}
	if len(users) != 1 {
		t.Errorf("ParseReport() got %d users, want 1", len(users))
	}
}

func TestParseReport_CommentParsing(t *testing.T) {
	content := `Версия:	1
Пользователь:	User1
Дата создания:	01.01.2024
Время создания:	12:00:00
Комментарий:
This is the comment
`
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.txt")
	err := os.WriteFile(reportPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	records, _, _, err := ParseReport(reportPath)
	if err != nil {
		t.Fatalf("ParseReport() error = %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("ParseReport() got %d records, want 1", len(records))
	}

	expectedComment := "This is the comment"
	if records[0].Comment != expectedComment {
		t.Errorf("ParseReport() comment = %q, want %q", records[0].Comment, expectedComment)
	}
}

func TestParseReport_MalformedVersionSkipped(t *testing.T) {
	// "invalid" version is skipped, valid version after it is parsed
	content := `Версия:	invalid
Версия:	5
Пользователь:	User1
Дата создания:	01.01.2024
Время создания:	12:00:00
`
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.txt")
	err := os.WriteFile(reportPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	records, _, maxVer, err := ParseReport(reportPath)
	if err != nil {
		t.Fatalf("ParseReport() error = %v", err)
	}

	// Should parse version 5 (invalid version is skipped)
	if maxVer != 5 {
		t.Errorf("ParseReport() maxVer = %d, want 5", maxVer)
	}
	if len(records) != 1 {
		t.Errorf("ParseReport() got %d records, want 1", len(records))
	}
}

func TestParseReport_InvalidDateFormat(t *testing.T) {
	content := `Версия:	1
Пользователь:	User1
Дата создания:	invalid-date
Время создания:	invalid-time
`
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "report.txt")
	err := os.WriteFile(reportPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	records, _, _, err := ParseReport(reportPath)
	if err != nil {
		t.Fatalf("ParseReport() error = %v", err)
	}

	// Record should still be parsed, but date should be zero
	if len(records) != 1 {
		t.Fatalf("ParseReport() got %d records, want 1", len(records))
	}
}
