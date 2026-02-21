package convertpipelinehandler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
)

// mockExecutor для тестов.
type mockExecutor struct {
	calls   []string
	failAt  string
	failErr error
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

func TestStageToCode(t *testing.T) {
	tests := []struct {
		stage string
		want  string
	}{
		{StageConvert, "CONVERT"},
		{StageGit2Store, "GIT2STORE"},
		{StageExtensionPublish, "EXTENSION_PUBLISH"},
		{"unknown", "UNKNOWN"},
	}
	for _, tc := range tests {
		if got := stageToCode(tc.stage); got != tc.want {
			t.Errorf("stageToCode(%q) = %q, want %q", tc.stage, got, tc.want)
		}
	}
}

// --- Pipeline execution tests ---

func setupEnv(t *testing.T) {
	t.Helper()
	os.Setenv("BR_OUTPUT_FORMAT", "text")
	os.Unsetenv("BR_PIPELINE_SKIP_STAGES")
	os.Unsetenv("BR_PIPELINE_TIMEOUT")
	t.Cleanup(func() {
		os.Unsetenv("BR_OUTPUT_FORMAT")
		os.Unsetenv("BR_PIPELINE_SKIP_STAGES")
		os.Unsetenv("BR_PIPELINE_TIMEOUT")
	})
}

func TestExecuteAllStages(t *testing.T) {
	setupEnv(t)
	mock := &mockExecutor{}
	h := &ConvertPipelineHandler{executor: mock}
	cfg := &config.Config{AddArray: []string{"ext1"}}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.calls) != 3 {
		t.Errorf("expected 3 calls, got %d: %v", len(mock.calls), mock.calls)
	}
	// Порядок: convert → git2store → extension-publish
	expected := []string{StageConvert, StageGit2Store, StageExtensionPublish}
	for i, e := range expected {
		if i < len(mock.calls) && mock.calls[i] != e {
			t.Errorf("call[%d] = %q, want %q", i, mock.calls[i], e)
		}
	}
}

func TestExecuteSkipByEnv(t *testing.T) {
	setupEnv(t)
	os.Setenv("BR_PIPELINE_SKIP_STAGES", "extension-publish")
	mock := &mockExecutor{}
	h := &ConvertPipelineHandler{executor: mock}
	cfg := &config.Config{AddArray: []string{"ext1"}}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.calls) != 2 {
		t.Errorf("expected 2 calls, got %d: %v", len(mock.calls), mock.calls)
	}
}

func TestExecuteNoExtensionsSkipsPublish(t *testing.T) {
	setupEnv(t)
	mock := &mockExecutor{}
	h := &ConvertPipelineHandler{executor: mock}
	cfg := &config.Config{AddArray: nil}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// extension-publish skipped by ShouldRun
	if len(mock.calls) != 2 {
		t.Errorf("expected 2 calls (convert+git2store), got %d: %v", len(mock.calls), mock.calls)
	}
}

func TestExecuteStageFailStopsPipeline(t *testing.T) {
	setupEnv(t)
	mock := &mockExecutor{failAt: StageGit2Store, failErr: fmt.Errorf("store error")}
	h := &ConvertPipelineHandler{executor: mock}
	cfg := &config.Config{AddArray: []string{"ext1"}}

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	// convert called + git2store called (failed), extension-publish NOT called
	if len(mock.calls) != 2 {
		t.Errorf("expected 2 calls, got %d: %v", len(mock.calls), mock.calls)
	}
}

func TestPipelineContextPassthrough(t *testing.T) {
	// Тест: AfterRun сохраняет данные из env в PipelineContext
	pctx := &PipelineContext{Cfg: &config.Config{}}

	// Подменяем getenv
	origGetenv := getenv
	defer func() { getenv = origGetenv }()
	getenv = func(key string) string {
		switch key {
		case "BR_SOURCE":
			return "/src"
		case "BR_TARGET":
			return "/out/xml"
		case "BR_DIRECTION":
			return "edt2xml"
		}
		return ""
	}

	stages := buildStages()
	log := slog.Default()

	// AfterRun convert сохраняет результат
	stages[0].AfterRun(pctx, log)
	if pctx.Convert == nil {
		t.Fatal("Convert result should not be nil after AfterRun")
	}
	if pctx.Convert.TargetPath != "/out/xml" {
		t.Errorf("expected /out/xml, got %s", pctx.Convert.TargetPath)
	}
	if pctx.Convert.Direction != "edt2xml" {
		t.Errorf("expected edt2xml, got %s", pctx.Convert.Direction)
	}

	// BeforeRun git2store может прочитать результат convert
	stages[1].BeforeRun(pctx, log) // не должен паниковать
}

func TestPipelineContextData(t *testing.T) {
	// Проверяем buildContextData
	pctx := &PipelineContext{}
	if cd := buildContextData(pctx); cd != nil {
		t.Error("empty context should return nil")
	}

	pctx.Convert = &ConvertStageResult{TargetPath: "/out"}
	cd := buildContextData(pctx)
	if cd == nil {
		t.Fatal("expected non-nil context data")
	}
	if cd.Convert.TargetPath != "/out" {
		t.Errorf("expected /out, got %s", cd.Convert.TargetPath)
	}
}

func TestRegisterCmd(t *testing.T) {
	_ = RegisterCmd()
	h, ok := command.Get(constants.ActNRConvertPipeline)
	if !ok {
		t.Fatal("handler not found after RegisterCmd")
	}
	if h.Name() != constants.ActNRConvertPipeline {
		t.Errorf("expected %s, got %s", constants.ActNRConvertPipeline, h.Name())
	}
}
