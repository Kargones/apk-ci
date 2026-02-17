package output

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestWritePlanOnlyResult(t *testing.T) {
	plan := &DryRunPlan{
		Command:          "test-command",
		Summary:          "Test summary",
		ValidationPassed: true,
		Steps: []PlanStep{
			{
				Order:      1,
				Operation:  "test-operation",
				Parameters: map[string]any{"key": "value"},
			},
		},
	}

	tests := []struct {
		name   string
		format string
	}{
		{name: "text format", format: FormatText},
		{name: "json format", format: FormatJSON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WritePlanOnlyResult(&buf, tt.format, "test-cmd", "trace-123", "v1", time.Now(), plan)
			if err != nil {
				t.Errorf("WritePlanOnlyResult() error = %v", err)
			}
			if buf.Len() == 0 {
				t.Error("WritePlanOnlyResult() produced empty output")
			}
		})
	}
}

func TestWriteDryRunResult(t *testing.T) {
	plan := &DryRunPlan{
		Command:          "dryrun-command",
		Summary:          "DryRun summary",
		ValidationPassed: false,
		Steps: []PlanStep{
			{
				Order:      1,
				Operation:  "dryrun-op",
				Parameters: map[string]any{"param": "val"},
			},
		},
	}

	tests := []struct {
		name   string
		format string
	}{
		{name: "text format", format: FormatText},
		{name: "json format", format: FormatJSON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteDryRunResult(&buf, tt.format, "dryrun-cmd", "trace-456", "v1", time.Now(), plan)
			if err != nil {
				t.Errorf("WriteDryRunResult() error = %v", err)
			}
			if buf.Len() == 0 {
				t.Error("WriteDryRunResult() produced empty output")
			}
		})
	}
}

func TestWritePlanResultJSONPlanOnlyFlag(t *testing.T) {
	plan := &DryRunPlan{
		Command:          "cmd",
		ValidationPassed: true,
		Steps:            []PlanStep{},
	}

	var buf bytes.Buffer
	err := writePlanResult(&buf, FormatJSON, "cmd", "trace", "v1", time.Now(), plan, true)
	if err != nil {
		t.Errorf("writePlanResult() error = %v", err)
	}

	// Check that plan_only is set in JSON output
	output := buf.String()
	if !strings.Contains(output, `"plan_only": true`) {
		t.Errorf("Expected plan_only: true in JSON output, got: %s", output)
	}
}

func TestWritePlanResultJSONDryRunFlag(t *testing.T) {
	plan := &DryRunPlan{
		Command:          "cmd",
		ValidationPassed: true,
		Steps:            []PlanStep{},
	}

	var buf bytes.Buffer
	err := writePlanResult(&buf, FormatJSON, "cmd", "trace", "v1", time.Now(), plan, false)
	if err != nil {
		t.Errorf("writePlanResult() error = %v", err)
	}

	// Check that dry_run is set in JSON output
	output := buf.String()
	if !strings.Contains(output, `"dry_run": true`) {
		t.Errorf("Expected dry_run: true in JSON output, got: %s", output)
	}
}

func TestWritePlanOnlyResultTextHeader(t *testing.T) {
	plan := &DryRunPlan{
		Command:          "cmd",
		ValidationPassed: true,
		Steps:            []PlanStep{},
	}

	var buf bytes.Buffer
	err := WritePlanOnlyResult(&buf, FormatText, "cmd", "trace", "v1", time.Now(), plan)
	if err != nil {
		t.Errorf("WritePlanOnlyResult() error = %v", err)
	}

	// Check for OPERATION PLAN header (plan-only mode)
	if !bytes.Contains(buf.Bytes(), []byte("=== OPERATION PLAN ===")) {
		t.Error("Expected OPERATION PLAN header in text output")
	}
}

func TestWriteDryRunResultTextHeader(t *testing.T) {
	plan := &DryRunPlan{
		Command:          "cmd",
		ValidationPassed: true,
		Steps:            []PlanStep{},
	}

	var buf bytes.Buffer
	err := WriteDryRunResult(&buf, FormatText, "cmd", "trace", "v1", time.Now(), plan)
	if err != nil {
		t.Errorf("WriteDryRunResult() error = %v", err)
	}

	// Check for DRY RUN header
	if !bytes.Contains(buf.Bytes(), []byte("=== DRY RUN ===")) {
		t.Error("Expected DRY RUN header in text output")
	}
}

func TestWritePlanResultWithSkippedSteps(t *testing.T) {
	plan := &DryRunPlan{
		Command:          "cmd",
		ValidationPassed: true,
		Steps: []PlanStep{
			{
				Order:      1,
				Operation:  "skipped-op",
				Skipped:    true,
				SkipReason: "dependency not met",
			},
		},
	}

	var buf bytes.Buffer
	err := WritePlanOnlyResult(&buf, FormatText, "cmd", "trace", "v1", time.Now(), plan)
	if err != nil {
		t.Errorf("WritePlanOnlyResult() error = %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("[SKIP]")) {
		t.Error("Expected [SKIP] marker in output")
	}
	if !bytes.Contains(buf.Bytes(), []byte("dependency not met")) {
		t.Error("Expected skip reason in output")
	}
}
