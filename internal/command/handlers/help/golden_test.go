package help

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHelpHandler_GoldenJSON проверяет структуру JSON вывода help.
// Golden file содержит ожидаемую JSON структуру (проверяются ключи и типы, не конкретные значения).
func TestHelpHandler_GoldenJSON(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &Handler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.NoError(t, execErr)

	// Проверяем что вывод — валидный JSON с правильной структурой
	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err, "stdout должен содержать валидный JSON")

	// Проверяем верхнеуровневые поля
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "help", result.Command)
	assert.NotNil(t, result.Metadata)
	assert.NotEmpty(t, result.Metadata.TraceID)
	assert.Equal(t, "v1", result.Metadata.APIVersion)

	// Проверяем структуру data
	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok, "Data должен быть map")

	nrCmds, ok := dataMap["nr_commands"].([]any)
	require.True(t, ok, "nr_commands должен быть массивом")
	assert.NotEmpty(t, nrCmds, "nr_commands не должен быть пустым")

	legacyCmds, ok := dataMap["legacy_commands"].([]any)
	require.True(t, ok, "legacy_commands должен быть массивом")
	assert.NotEmpty(t, legacyCmds, "legacy_commands не должен быть пустым")

	// Проверяем структуру каждой NR-команды
	for _, cmdAny := range nrCmds {
		cmd, ok := cmdAny.(map[string]any)
		require.True(t, ok, "каждая команда должна быть объектом")
		assert.Contains(t, cmd, "name", "команда должна иметь поле name")
		assert.Contains(t, cmd, "description", "команда должна иметь поле description")
	}

	// Проверяем структуру каждой legacy-команды
	for _, cmdAny := range legacyCmds {
		cmd, ok := cmdAny.(map[string]any)
		require.True(t, ok, "каждая команда должна быть объектом")
		assert.Contains(t, cmd, "name", "команда должна иметь поле name")
		assert.Contains(t, cmd, "description", "команда должна иметь поле description")
	}

	// Golden file: сравнение структуры JSON с эталоном.
	goldenPath := filepath.Join("testdata", "help_json_output.golden")

	// Форматируем JSON для читаемости
	var prettyJSON bytes.Buffer
	err = json.Indent(&prettyJSON, []byte(out), "", "  ")
	require.NoError(t, err)

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		err = os.WriteFile(goldenPath, prettyJSON.Bytes(), 0600)
		require.NoError(t, err, "не удалось записать golden file")
		t.Log("Golden file обновлён")
	} else {
		// Сравниваем структуру JSON с golden file
		goldenData, readErr := os.ReadFile(goldenPath) //nolint:gosec // путь фиксированный: testdata/
		require.NoError(t, readErr, "не удалось прочитать golden file; запустите с UPDATE_GOLDEN=1")

		var goldenResult output.Result
		require.NoError(t, json.Unmarshal(goldenData, &goldenResult),
			"golden file содержит невалидный JSON")

		// Сравниваем структуру (ключи и типы), не конкретные значения
		assert.Equal(t, goldenResult.Status, result.Status, "status должен совпадать с golden")
		assert.Equal(t, goldenResult.Command, result.Command, "command должен совпадать с golden")
		assert.NotNil(t, result.Metadata, "metadata должна присутствовать как в golden")

		goldenDataMap, ok := goldenResult.Data.(map[string]any)
		require.True(t, ok, "golden Data должен быть map")
		actualDataMap, ok := result.Data.(map[string]any)
		require.True(t, ok, "actual Data должен быть map")

		// Проверяем что ключи в data совпадают
		for key := range goldenDataMap {
			assert.Contains(t, actualDataMap, key, "ключ %s из golden должен присутствовать в actual", key)
		}
		for key := range actualDataMap {
			assert.Contains(t, goldenDataMap, key, "ключ %s из actual должен присутствовать в golden", key)
		}
	}
}
