package version

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVersionHandler_GoldenJSON проверяет что JSON вывод соответствует ожидаемой структуре golden file.
// Сравниваются поля и типы, а не конкретные значения (версия, trace_id и т.д. динамические).
func TestVersionHandler_GoldenJSON(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &VersionHandler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.NoError(t, execErr)

	// Парсим фактический вывод
	var actual map[string]any
	err := json.Unmarshal([]byte(out), &actual)
	require.NoError(t, err, "вывод должен быть валидным JSON")

	// Загружаем golden file
	goldenData, err := os.ReadFile("testdata/version_json_output.golden")
	require.NoError(t, err, "golden file должен существовать")

	var golden map[string]any
	err = json.Unmarshal(goldenData, &golden)
	require.NoError(t, err, "golden file должен быть валидным JSON")

	// Сравниваем структуру: наличие полей и совпадение типов
	for key := range golden {
		assert.Contains(t, actual, key, "JSON должен содержать поле '%s'", key)
	}

	// Проверяем что нет лишних top-level полей
	for key := range actual {
		assert.Contains(t, golden, key, "JSON содержит неожиданное поле '%s'", key)
	}

	// Проверяем структуру и типы полей data
	goldenDataMap, ok := golden["data"].(map[string]any)
	require.True(t, ok)
	actualData, ok := actual["data"].(map[string]any)
	require.True(t, ok)
	for key := range goldenDataMap {
		assert.Contains(t, actualData, key, "data должен содержать поле '%s'", key)
	}
	for key := range actualData {
		assert.Contains(t, goldenDataMap, key, "data содержит неожиданное поле '%s'", key)
	}
	// Проверяем типы полей data
	for key, val := range actualData {
		switch key {
		case "rollback_mapping":
			// Story 7.4 AC4: rollback_mapping — массив объектов
			_, isArray := val.([]any)
			assert.True(t, isArray, "data.%s должен быть массивом, получен %T", key, val)
		default:
			_, isString := val.(string)
			assert.True(t, isString, "data.%s должен быть строкой, получен %T", key, val)
		}
	}

	// Проверяем структуру и типы полей metadata
	goldenMeta, ok := golden["metadata"].(map[string]any)
	require.True(t, ok)
	actualMeta, ok := actual["metadata"].(map[string]any)
	require.True(t, ok)
	for key := range goldenMeta {
		assert.Contains(t, actualMeta, key, "metadata должен содержать поле '%s'", key)
	}
	for key := range actualMeta {
		assert.Contains(t, goldenMeta, key, "metadata содержит неожиданное поле '%s'", key)
	}
	// Проверяем типы metadata
	_, isFloat := actualMeta["duration_ms"].(float64)
	assert.True(t, isFloat, "metadata.duration_ms должен быть числом")
	_, isString := actualMeta["trace_id"].(string)
	assert.True(t, isString, "metadata.trace_id должен быть строкой")
	_, isString = actualMeta["api_version"].(string)
	assert.True(t, isString, "metadata.api_version должен быть строкой")
}

// TestVersionHandler_StdoutOnlyJSON проверяет что stdout содержит ТОЛЬКО JSON (нет логов, нет лишнего текста).
func TestVersionHandler_StdoutOnlyJSON(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &VersionHandler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.NoError(t, execErr)
	require.NotEmpty(t, out)

	// Проверяем что stdout — это ровно один JSON объект (без лишнего текста)
	var result output.Result
	decoder := json.NewDecoder(bytes.NewReader([]byte(out)))
	err := decoder.Decode(&result)
	require.NoError(t, err, "stdout должен начинаться с валидного JSON")

	// Проверяем что после JSON нет ничего (кроме пробелов/переводов строк)
	var remaining bytes.Buffer
	remaining.ReadFrom(decoder.Buffered())
	trimmed := bytes.TrimSpace(remaining.Bytes())
	assert.Empty(t, trimmed, "после JSON в stdout не должно быть лишнего текста")
}
