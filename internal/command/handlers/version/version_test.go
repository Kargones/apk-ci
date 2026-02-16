package version

import (
	"context"
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/testutil"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionHandler_Name(t *testing.T) {
	h := &VersionHandler{}
	assert.Equal(t, "nr-version", h.Name())
	assert.Equal(t, constants.ActNRVersion, h.Name())
}

func TestVersionHandler_Execute_Success(t *testing.T) {
	h := &VersionHandler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.NoError(t, execErr)
	assert.NotEmpty(t, out)
}

func TestVersionHandler_Execute_JSONOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &VersionHandler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.NoError(t, execErr)

	// Проверяем что вывод — валидный JSON
	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err, "stdout должен содержать валидный JSON")

	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "nr-version", result.Command)
	assert.NotNil(t, result.Data)

	// Проверяем поля data
	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok, "Data должен быть map")
	assert.NotEmpty(t, dataMap["version"])
	assert.Equal(t, runtime.Version(), dataMap["go_version"])
	assert.NotEmpty(t, dataMap["commit"])
}

func TestVersionHandler_Execute_TextOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	h := &VersionHandler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.NoError(t, execErr)

	// Проверяем человекочитаемый формат версии (AC3)
	assert.Contains(t, out, "apk-ci version")
	assert.Contains(t, out, "Go:")
	assert.Contains(t, out, runtime.Version())
	assert.Contains(t, out, "Commit:")
}

func TestVersionHandler_Execute_Metadata(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &VersionHandler{}
	traceID := tracing.GenerateTraceID()
	ctx := tracing.WithTraceID(context.Background(), traceID)

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.NoError(t, execErr)

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)

	require.NotNil(t, result.Metadata)
	assert.Equal(t, traceID, result.Metadata.TraceID, "trace_id должен совпадать с переданным в context")
	assert.GreaterOrEqual(t, result.Metadata.DurationMs, int64(0))
	assert.Equal(t, constants.APIVersion, result.Metadata.APIVersion)
}

func TestVersionHandler_Execute_MetadataGeneratesTraceID(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &VersionHandler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.NoError(t, execErr)

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)

	require.NotNil(t, result.Metadata)
	assert.NotEmpty(t, result.Metadata.TraceID, "trace_id должен быть сгенерирован если не задан в context")
	assert.Len(t, result.Metadata.TraceID, 32, "trace_id должен быть 32-символьным hex")
}

func TestVersionHandler_Registration(t *testing.T) {
	// init() уже вызван при импорте пакета — проверяем что handler зарегистрирован
	h, ok := command.Get(constants.ActNRVersion)
	require.True(t, ok, "handler nr-version должен быть зарегистрирован в registry")
	assert.Equal(t, constants.ActNRVersion, h.Name())
}

func TestBuildVersionData_Fallback(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		commit      string
		wantVersion string
		wantCommit  string
	}{
		{
			name:        "оба пустые — fallback",
			version:     "",
			commit:      "",
			wantVersion: "dev",
			wantCommit:  "unknown",
		},
		{
			name:        "version пустой — fallback version",
			version:     "",
			commit:      "abc1234",
			wantVersion: "dev",
			wantCommit:  "abc1234",
		},
		{
			name:        "commit пустой — fallback commit",
			version:     "1.0.0",
			commit:      "",
			wantVersion: "1.0.0",
			wantCommit:  "unknown",
		},
		{
			name:        "оба заданы — без fallback",
			version:     "1.0.0",
			commit:      "abc1234",
			wantVersion: "1.0.0",
			wantCommit:  "abc1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := buildVersionData(tt.version, tt.commit)
			assert.Equal(t, tt.wantVersion, data.Version)
			assert.Equal(t, tt.wantCommit, data.Commit)
			assert.Equal(t, runtime.Version(), data.GoVersion)
		})
	}
}

func TestVersionData_WriteText(t *testing.T) {
	data := &VersionData{
		Version:   "1.2.3",
		GoVersion: "go1.25.1",
		Commit:    "abc1234",
	}

	var buf strings.Builder
	err := data.writeText(&buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "apk-ci version 1.2.3")
	assert.Contains(t, out, "Go:     go1.25.1")
	assert.Contains(t, out, "Commit: abc1234")
}

func TestVersionHandler_Execute_TextOutput_ContainsRollbackMapping(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	h := &VersionHandler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.NoError(t, execErr)

	// Story 7.4 AC4: текстовый вывод содержит Rollback Mapping
	assert.Contains(t, out, "Rollback Mapping:", "текстовый вывод должен содержать секцию Rollback Mapping")
	// Должен содержать хотя бы одну NR-команду
	assert.Contains(t, out, "nr-", "текстовый вывод должен содержать NR-команды")
	assert.Contains(t, out, "→", "текстовый вывод должен содержать стрелки маппинга")
}

func TestVersionHandler_Execute_JSONOutput_ContainsRollbackMapping(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &VersionHandler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.NoError(t, execErr)

	// Проверяем JSON
	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err, "stdout должен содержать валидный JSON")

	// Извлекаем data
	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok, "Data должен быть map")

	// Story 7.4 AC4: JSON содержит rollback_mapping
	rollbackMapping, ok := dataMap["rollback_mapping"]
	require.True(t, ok, "Data должен содержать rollback_mapping")

	mappingList, ok := rollbackMapping.([]any)
	require.True(t, ok, "rollback_mapping должен быть массивом")
	assert.NotEmpty(t, mappingList, "rollback_mapping не должен быть пустым")

	// Проверяем структуру первого элемента
	firstEntry, ok := mappingList[0].(map[string]any)
	require.True(t, ok, "элемент rollback_mapping должен быть map")
	assert.Contains(t, firstEntry, "nr_command", "элемент должен содержать nr_command")
	assert.Contains(t, firstEntry, "legacy_alias", "элемент должен содержать legacy_alias")
}

func TestBuildRollbackMapping_ReturnsEntries(t *testing.T) {
	entries := buildRollbackMapping()

	// В контексте этого тестового пакета зарегистрирован как минимум nr-version
	assert.NotEmpty(t, entries, "rollback маппинг не должен быть пустым")

	// Проверяем что nr-version присутствует в маппинге
	hasVersion := false
	for _, entry := range entries {
		if entry.NRCommand == constants.ActNRVersion {
			hasVersion = true
			break
		}
	}
	assert.True(t, hasVersion, "маппинг должен содержать nr-version")
}

func TestVersionData_WriteText_ContainsRollbackMapping(t *testing.T) {
	data := &VersionData{
		Version:   "1.2.3",
		GoVersion: "go1.25.1",
		Commit:    "abc1234",
		RollbackMapping: []RollbackEntry{
			{NRCommand: "nr-service-mode-status", LegacyAlias: "service-mode-status"},
			{NRCommand: "nr-version", LegacyAlias: ""},
		},
	}

	var buf strings.Builder
	err := data.writeText(&buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Rollback Mapping:")
	assert.Contains(t, out, "nr-service-mode-status")
	assert.Contains(t, out, "service-mode-status")
	assert.Contains(t, out, "nr-version")
	assert.Contains(t, out, "(нет)", "команда без алиаса должна показывать (нет)")
}

func TestVersionHandler_PlanOnly(t *testing.T) {
	t.Setenv("BR_PLAN_ONLY", "true")

	h := &VersionHandler{}

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(context.Background(), nil)
	})

	require.NoError(t, execErr)
	assert.Contains(t, out, "не поддерживает отображение плана операций")
	assert.Contains(t, out, constants.ActNRVersion)
}
