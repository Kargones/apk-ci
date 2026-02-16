package output

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "обновить golden files")

func TestNewJSONWriter(t *testing.T) {
	writer := NewJSONWriter()
	assert.NotNil(t, writer)
}

func TestJSONWriter_ImplementsWriter(_ *testing.T) {
	var _ Writer = (*JSONWriter)(nil)
}

func TestJSONWriter_Write_SuccessResult(t *testing.T) {
	result := &Result{
		Status:  StatusSuccess,
		Command: "test-command",
		Data:    map[string]string{"version": "1.0.0"},
		Metadata: &Metadata{
			DurationMs: 150,
			APIVersion: "v1",
		},
	}

	writer := NewJSONWriter()
	var buf bytes.Buffer
	err := writer.Write(&buf, result)
	require.NoError(t, err)

	goldenPath := filepath.Join("testdata", "golden", "result_success.json")
	if *update {
		err = os.WriteFile(goldenPath, buf.Bytes(), 0600)
		require.NoError(t, err)
	}

	expected, err := os.ReadFile(goldenPath) //nolint:gosec // golden files в testdata — безопасны
	require.NoError(t, err)

	assert.Equal(t, string(expected), buf.String())
}

func TestJSONWriter_Write_ErrorResult(t *testing.T) {
	result := &Result{
		Status:  StatusError,
		Command: "test-command",
		Error: &ErrorInfo{
			Code:    "CONFIG.LOAD_FAILED",
			Message: "не удалось загрузить конфигурацию",
		},
		Metadata: &Metadata{
			DurationMs: 50,
			APIVersion: "v1",
		},
	}

	writer := NewJSONWriter()
	var buf bytes.Buffer
	err := writer.Write(&buf, result)
	require.NoError(t, err)

	goldenPath := filepath.Join("testdata", "golden", "result_error.json")
	if *update {
		err = os.WriteFile(goldenPath, buf.Bytes(), 0600)
		require.NoError(t, err)
	}

	expected, err := os.ReadFile(goldenPath) //nolint:gosec // golden files в testdata — безопасны
	require.NoError(t, err)

	assert.Equal(t, string(expected), buf.String())
}

func TestJSONWriter_Write_MinimalResult(t *testing.T) {
	result := &Result{
		Status:  StatusSuccess,
		Command: "test-command",
	}

	writer := NewJSONWriter()
	var buf bytes.Buffer
	err := writer.Write(&buf, result)
	require.NoError(t, err)

	goldenPath := filepath.Join("testdata", "golden", "result_minimal.json")
	if *update {
		err = os.WriteFile(goldenPath, buf.Bytes(), 0600)
		require.NoError(t, err)
	}

	expected, err := os.ReadFile(goldenPath) //nolint:gosec // golden files в testdata — безопасны
	require.NoError(t, err)

	assert.Equal(t, string(expected), buf.String())
}

func TestJSONWriter_Write_ValidJSON(t *testing.T) {
	result := &Result{
		Status:  StatusSuccess,
		Command: "test-command",
		Data:    map[string]string{"key": "value"},
	}

	writer := NewJSONWriter()
	var buf bytes.Buffer
	err := writer.Write(&buf, result)
	require.NoError(t, err)

	// Проверяем что результат — валидный JSON
	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "success", parsed["status"])
	assert.Equal(t, "test-command", parsed["command"])
}

// loadSchema загружает JSON Schema из файла для валидации.
func loadSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	schemaPath := filepath.Join("testdata", "schema", "result.schema.json")
	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(schemaPath)
	require.NoError(t, err, "не удалось загрузить JSON Schema")
	return schema
}

func TestJSONWriter_Write_SchemaValidation_Success(t *testing.T) {
	schema := loadSchema(t)

	result := &Result{
		Status:  StatusSuccess,
		Command: "test-command",
		Data:    map[string]string{"version": "1.0.0"},
		Metadata: &Metadata{
			DurationMs: 150,
			APIVersion: "v1",
		},
	}

	writer := NewJSONWriter()
	var buf bytes.Buffer
	err := writer.Write(&buf, result)
	require.NoError(t, err)

	var jsonData any
	err = json.Unmarshal(buf.Bytes(), &jsonData)
	require.NoError(t, err)

	err = schema.Validate(jsonData)
	assert.NoError(t, err, "успешный Result должен соответствовать JSON Schema")
}

func TestJSONWriter_Write_SchemaValidation_Error(t *testing.T) {
	schema := loadSchema(t)

	result := &Result{
		Status:  StatusError,
		Command: "test-command",
		Error: &ErrorInfo{
			Code:    "CONFIG.LOAD_FAILED",
			Message: "не удалось загрузить конфигурацию",
		},
		Metadata: &Metadata{
			DurationMs: 50,
			APIVersion: "v1",
		},
	}

	writer := NewJSONWriter()
	var buf bytes.Buffer
	err := writer.Write(&buf, result)
	require.NoError(t, err)

	var jsonData any
	err = json.Unmarshal(buf.Bytes(), &jsonData)
	require.NoError(t, err)

	err = schema.Validate(jsonData)
	assert.NoError(t, err, "Result с ошибкой должен соответствовать JSON Schema")
}

func TestJSONWriter_Write_SchemaValidation_Minimal(t *testing.T) {
	schema := loadSchema(t)

	result := &Result{
		Status:  StatusSuccess,
		Command: "test-command",
	}

	writer := NewJSONWriter()
	var buf bytes.Buffer
	err := writer.Write(&buf, result)
	require.NoError(t, err)

	var jsonData any
	err = json.Unmarshal(buf.Bytes(), &jsonData)
	require.NoError(t, err)

	err = schema.Validate(jsonData)
	assert.NoError(t, err, "минимальный Result должен соответствовать JSON Schema")
}

func TestJSONWriter_Write_SchemaValidation_PlanOnly(t *testing.T) {
	schema := loadSchema(t)

	result := &Result{
		Status:   StatusSuccess,
		Command:  "nr-dbupdate",
		PlanOnly: true,
		Plan: &DryRunPlan{
			Command: "nr-dbupdate",
			Steps: []PlanStep{{
				Order:      1,
				Operation:  "Обновление БД",
				Parameters: map[string]any{"database": "TestDB"},
			}},
			ValidationPassed: true,
		},
		Metadata: &Metadata{
			DurationMs: 10,
			APIVersion: "v1",
		},
	}

	writer := NewJSONWriter()
	var buf bytes.Buffer
	err := writer.Write(&buf, result)
	require.NoError(t, err)

	var jsonData any
	err = json.Unmarshal(buf.Bytes(), &jsonData)
	require.NoError(t, err)

	err = schema.Validate(jsonData)
	assert.NoError(t, err, "PlanOnly Result должен соответствовать JSON Schema")
}

func TestJSONWriter_Write_PlanOnlyResult(t *testing.T) {
	result := &Result{
		Status:   StatusSuccess,
		Command:  "nr-dbupdate",
		PlanOnly: true,
		Plan: &DryRunPlan{
			Command:          "nr-dbupdate",
			Steps:            []PlanStep{{Order: 1, Operation: "Обновление БД"}},
			ValidationPassed: true,
		},
		Metadata: &Metadata{
			DurationMs: 10,
			APIVersion: "v1",
		},
	}

	writer := NewJSONWriter()
	var buf bytes.Buffer
	err := writer.Write(&buf, result)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	assert.Equal(t, true, parsed["plan_only"], "JSON должен содержать plan_only: true")
	assert.NotNil(t, parsed["plan"], "JSON должен содержать plan")
	assert.Nil(t, parsed["data"], "JSON не должен содержать data при plan_only")
	assert.Nil(t, parsed["dry_run"], "JSON не должен содержать dry_run при plan_only")
}

func TestJSONWriter_Write_PlanOnlyFalseOmitted(t *testing.T) {
	result := &Result{
		Status:  StatusSuccess,
		Command: "test-command",
		Data:    map[string]string{"key": "value"},
	}

	writer := NewJSONWriter()
	var buf bytes.Buffer
	err := writer.Write(&buf, result)
	require.NoError(t, err)

	// PlanOnly=false не должен появляться в JSON (omitempty)
	assert.NotContains(t, buf.String(), "plan_only")
}

func TestJSONWriter_Write_NilResult(t *testing.T) {
	writer := NewJSONWriter()
	var buf bytes.Buffer
	err := writer.Write(&buf, nil)
	require.NoError(t, err)

	// nil result сериализуется как "null"
	assert.Equal(t, "null\n", buf.String())
}
