package convertpipelinehandler

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
)

// mockExecutor реализует StageExecutor для тестов.
type mockExecutor struct {
	calls    []string
	failAt   string
	failErr  error
}

func (m *mockExecutor) ExecuteStage(_ context.Context, stageName string, _ *config.Config) error {
	m.calls = append(m.calls, stageName)
	if stageName == m.failAt {
		return m.failErr
	}
	return nil
}

func TestName(t *testing.T) {
	h := &ConvertPipelineHandler{}
	if h.Name() != constants.ActNRConvertPipeline {
		t.Errorf("expected %s, got %s", constants.ActNRConvertPipeline, h.Name())
	}
}

func TestDescription(t *testing.T) {
	h := &ConvertPipelineHandler{}
	if h.Description() == "" {
		t.Error("description should not be empty")
	}
}

func TestParseSkipStages(t *testing.T) {
	tests := []struct {
		input string
		want  map[string]bool
	}{
		{"", map[string]bool{}},
		{"convert", map[string]bool{"convert": true}},
		{"convert, extension-publish", map[string]bool{"convert": true, "extension-publish": true}},
		{" git2store , convert ", map[string]bool{"git2store": true, "convert": true}},
	}
	for _, tc := range tests {
		got := parseSkipStages(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("parseSkipStages(%q) = %v, want %v", tc.input, got, tc.want)
			continue
		}
		for k := range tc.want {
			if !got[k] {
				t.Errorf("parseSkipStages(%q) missing key %s", tc.input, k)
			}
		}
	}
}

func TestStageToCommand(t *testing.T) {
	tests := []struct {
		stage string
		want  string
	}{
		{StageConvert, constants.ActNRConvert},
		{StageGit2Store, constants.ActNRGit2store},
		{StageExtensionPublish, constants.ActNRExtensionPublish},
		{"unknown", "unknown"},
	}
	for _, tc := range tests {
		if got := stageToCommand(tc.stage); got != tc.want {
			t.Errorf("stageToCommand(%q) = %q, want %q", tc.stage, got, tc.want)
		}
	}
}

func TestExecuteAllStagesSuccess(t *testing.T) {
	mock := &mockExecutor{}
	h := &ConvertPipelineHandler{executor: mock}

	cfg := &config.Config{
		AddArray: []string{"ext1"},
	}

	// Убедимся что output не ломается
	os.Setenv("BR_OUTPUT_FORMAT", "text")
	defer os.Unsetenv("BR_OUTPUT_FORMAT")
	os.Unsetenv("BR_PIPELINE_SKIP_STAGES")

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(mock.calls) != 3 {
		t.Errorf("expected 3 stage calls, got %d: %v", len(mock.calls), mock.calls)
	}
}

func TestExecuteSkipExtensionPublish(t *testing.T) {
	mock := &mockExecutor{}
	h := &ConvertPipelineHandler{executor: mock}
	cfg := &config.Config{AddArray: []string{"ext1"}}

	os.Setenv("BR_OUTPUT_FORMAT", "text")
	os.Setenv("BR_PIPELINE_SKIP_STAGES", "extension-publish")
	defer os.Unsetenv("BR_OUTPUT_FORMAT")
	defer os.Unsetenv("BR_PIPELINE_SKIP_STAGES")

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.calls) != 2 {
		t.Errorf("expected 2 calls (convert, git2store), got %d: %v", len(mock.calls), mock.calls)
	}
}

func TestExecuteNoExtensionsSkipsPublish(t *testing.T) {
	mock := &mockExecutor{}
	h := &ConvertPipelineHandler{executor: mock}
	cfg := &config.Config{AddArray: nil}

	os.Setenv("BR_OUTPUT_FORMAT", "text")
	defer os.Unsetenv("BR_OUTPUT_FORMAT")
	os.Unsetenv("BR_PIPELINE_SKIP_STAGES")

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only run convert + git2store
	if len(mock.calls) != 2 {
		t.Errorf("expected 2 calls, got %d: %v", len(mock.calls), mock.calls)
	}
}

func TestExecuteStageFailure(t *testing.T) {
	mock := &mockExecutor{
		failAt:  StageGit2Store,
		failErr: fmt.Errorf("git2store failed"),
	}
	h := &ConvertPipelineHandler{executor: mock}
	cfg := &config.Config{AddArray: []string{"ext1"}}

	os.Setenv("BR_OUTPUT_FORMAT", "text")
	defer os.Unsetenv("BR_OUTPUT_FORMAT")
	os.Unsetenv("BR_PIPELINE_SKIP_STAGES")

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error from git2store stage")
	}

	// Only convert should have been called before failure
	if len(mock.calls) != 2 {
		t.Errorf("expected 2 calls (convert + git2store), got %d: %v", len(mock.calls), mock.calls)
	}
}

func TestRegisterCmd(t *testing.T) {
	// Clear registry for isolated test
	command.Register(nil) // will fail, that's fine — we just need to verify RegisterCmd works
	
	// RegisterCmd should work (may fail if already registered in this test process)
	_ = RegisterCmd()
	
	h, ok := command.Get(constants.ActNRConvertPipeline)
	if !ok {
		t.Fatal("handler not found after RegisterCmd")
	}
	if h.Name() != constants.ActNRConvertPipeline {
		t.Errorf("expected %s, got %s", constants.ActNRConvertPipeline, h.Name())
	}
}
