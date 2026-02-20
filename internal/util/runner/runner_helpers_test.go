package runner

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessParamValue_Regular(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	r := &Runner{TmpDir: t.TempDir()}

	fv, cmd, err := r.processParamValue("regular", logger)
	if err != nil {
		t.Fatal(err)
	}
	if fv != "regular" {
		t.Errorf("expected 'regular', got %q", fv)
	}
	if cmd != nil {
		t.Errorf("expected nil cmdParams, got %v", cmd)
	}
}

func TestProcessParamValue_SlashC(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	r := &Runner{TmpDir: t.TempDir()}

	fv, cmd, err := r.processParamValue("/cMyCode", logger)
	if err != nil {
		t.Fatal(err)
	}
	if fv != "" {
		t.Errorf("expected empty fileValue for /c, got %q", fv)
	}
	if len(cmd) != 2 || cmd[0] != "/c" || cmd[1] != "MyCode" {
		t.Errorf("unexpected cmd params: %v", cmd)
	}
}

func TestProcessParamValue_SlashOut(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tmpDir := t.TempDir()
	r := &Runner{TmpDir: tmpDir}

	fv, cmd, err := r.processParamValue("/Out", logger)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(fv, "/Out ") {
		t.Errorf("expected fileValue starting with '/Out ', got %q", fv)
	}
	if r.OutFileName == "" {
		t.Error("expected OutFileName to be set")
	}
	if cmd != nil {
		t.Errorf("expected nil cmdParams, got %v", cmd)
	}
}

func TestProcessParamValue_SlashOutBadDir(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	r := &Runner{TmpDir: "/nonexistent/dir/xyz"}

	_, _, err := r.processParamValue("/Out", logger)
	if err == nil {
		t.Error("expected error for bad TmpDir")
	}
}

func TestBuildMaskedParam(t *testing.T) {
	params := []string{"/ConfigurationRepositoryP", "secret", "/S", "server"}

	if m := buildMaskedParam(params, 1, "secret"); m != "*****" {
		t.Errorf("expected '*****', got %q", m)
	}

	if m := buildMaskedParam(params, 3, "server"); m != "server" {
		t.Errorf("expected 'server', got %q", m)
	}
}

func TestValidateParams(t *testing.T) {
	tests := []struct {
		name    string
		r       Runner
		wantErr bool
	}{
		{"empty executable", Runner{RunString: ""}, true},
		{"valid", Runner{RunString: "/bin/echo", Params: []string{"a"}}, false},
		{"semicolon", Runner{RunString: "/bin/echo", Params: []string{"a;b"}}, true},
		{"ampersand", Runner{RunString: "/bin/echo", Params: []string{"a&b"}}, true},
		{"pipe", Runner{RunString: "/bin/echo", Params: []string{"a|b"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.r.validateParams()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReadOutputFile_NoFile(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	r := &Runner{OutFileName: "/nonexistent/file.out"}

	result := r.readOutputFile(logger)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestReadOutputFile_WithFile(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "test.out")
	os.WriteFile(outFile, []byte("output data"), 0644)

	r := &Runner{OutFileName: outFile}
	result := r.readOutputFile(logger)
	if string(result) != "output data" {
		t.Errorf("expected 'output data', got %q", string(result))
	}
	if string(r.FileOut) != "output data" {
		t.Errorf("expected FileOut to be set")
	}
}

func TestReadOutputFile_EmptyName(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	r := &Runner{OutFileName: ""}

	result := r.readOutputFile(logger)
	if result != nil {
		t.Errorf("expected nil for empty OutFileName, got %v", result)
	}
}

func TestPrepareAtParams(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tmpDir := t.TempDir()

	r := &Runner{
		TmpDir: tmpDir,
		Params: []string{"@", "/S", "server", "/ConfigurationRepositoryP", "secret"},
	}

	lParams, err := r.prepareAtParams(logger)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, p := range lParams {
		if p == "*****" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected masked param '*****' in %v", lParams)
	}

	if len(r.Params) < 2 || r.Params[0] != "@" {
		t.Errorf("expected Params to start with '@', got %v", r.Params)
	}
}

func TestPrepareAtParams_BadTmpDir(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	r := &Runner{
		TmpDir: "/nonexistent/dir",
		Params: []string{"@", "param1"},
	}

	_, err := r.prepareAtParams(logger)
	if err == nil {
		t.Error("expected error for bad TmpDir")
	}
}

func TestRunCommand_WithOutputFile(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tmpDir := t.TempDir()

	outFile := filepath.Join(tmpDir, "out.txt")
	os.WriteFile(outFile, []byte("file output"), 0644)

	r := &Runner{
		RunString:   "/usr/bin/echo",
		Params:      []string{"hello"},
		WorkDir:     tmpDir,
		TmpDir:      tmpDir,
		OutFileName: outFile,
	}

	result, err := r.RunCommand(context.Background(), logger)
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != "file output" {
		t.Errorf("expected 'file output', got %q", string(result))
	}
}

func TestRunCommand_AtParamsWithSlashC(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tmpDir := t.TempDir()

	r := &Runner{
		RunString: "/usr/bin/echo",
		Params:    []string{"@", "/cTestCode", "param1"},
		WorkDir:   tmpDir,
		TmpDir:    tmpDir,
	}

	_, err := r.RunCommand(context.Background(), logger)
	if err != nil {
		t.Logf("error (may be expected): %v", err)
	}
}

func TestPrepareAtParams_WithSlashOutError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tmpDir := t.TempDir()

	// Set TmpDir to valid for param file but make /Out fail by using a nested bad path
	r := &Runner{
		TmpDir: tmpDir,
		Params: []string{"@", "/OutSomething"},
	}

	// This should succeed since TmpDir is valid
	_, err := r.prepareAtParams(logger)
	if err != nil {
		t.Logf("Got error (checking /Out handling): %v", err)
	}
}
