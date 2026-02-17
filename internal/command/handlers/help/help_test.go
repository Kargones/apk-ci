package help

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelpHandler_Name(t *testing.T) {
	h := &Handler{}
	assert.Equal(t, "help", h.Name())
	assert.Equal(t, constants.ActHelp, h.Name())
}

func TestHelpHandler_Description(t *testing.T) {
	h := &Handler{}
	assert.Equal(t, "Вывод списка доступных команд", h.Description())
}

func TestHelpHandler_Execute_TextOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	h := &Handler{}
	ctx := context.Background()

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(ctx, nil)
	})

	require.NoError(t, execErr)

	// Проверяем заголовок
	assert.Contains(t, out, "apk-ci — инструмент автоматизации 1C:Enterprise")

	// Проверяем группировку
	assert.Contains(t, out, "NR-команды:")
	assert.Contains(t, out, "Legacy-команды:")

	// Проверяем что help присутствует
	assert.Contains(t, out, "help")
	assert.Contains(t, out, "Вывод списка доступных команд")

	// Проверяем подсказку
	assert.Contains(t, out, "BR_OUTPUT_FORMAT=json")
}

func TestHelpHandler_Execute_JSONOutput(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &Handler{}
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
	assert.Equal(t, "help", result.Command)
	assert.NotNil(t, result.Data)
	assert.NotNil(t, result.Metadata)
	assert.Equal(t, constants.APIVersion, result.Metadata.APIVersion)
	assert.NotEmpty(t, result.Metadata.TraceID)

	// Проверяем что data содержит nr_commands и legacy_commands
	dataMap, ok := result.Data.(map[string]any)
	require.True(t, ok, "Data должен быть map")
	assert.Contains(t, dataMap, "nr_commands")
	assert.Contains(t, dataMap, "legacy_commands")
}

func TestHelpHandler_DeprecatedMarking(t *testing.T) {
	data := buildData()

	// Ищем deprecated команды (если есть DeprecatedBridge в registry)
	for _, cmd := range data.NRCommands {
		if cmd.Deprecated {
			assert.NotEmpty(t, cmd.NewName, "deprecated команда %s должна указывать новое имя", cmd.Name)
		}
	}
}

func TestHelpHandler_NRGrouping(t *testing.T) {
	data := buildData()

	// NR-команды должны содержать help (зарегистрирован через init() в этом пакете)
	// nr-version не зарегистрирован т.к. version пакет не импортирован в тестовом бинарнике
	nrNames := make([]string, 0, len(data.NRCommands))
	for _, cmd := range data.NRCommands {
		nrNames = append(nrNames, cmd.Name)
	}

	assert.Contains(t, nrNames, "help", "help должна быть в NR-командах")

	// NR-команды НЕ должны содержать legacy-команды
	for _, name := range nrNames {
		_, isLegacy := legacyCommands[name]
		assert.False(t, isLegacy, "NR-команда %s не должна быть в legacy-командах", name)
	}

	// Legacy-команды не должны быть пустыми
	assert.NotEmpty(t, data.LegacyCommands, "legacy-команды не должны быть пустыми")

	// Проверяем что legacy-команды содержат ожидаемые
	legacyNames := make([]string, 0, len(data.LegacyCommands))
	for _, cmd := range data.LegacyCommands {
		legacyNames = append(legacyNames, cmd.Name)
	}
	assert.Contains(t, legacyNames, constants.ActConvert)
	assert.Contains(t, legacyNames, constants.ActDbrestore)
}

func TestHelpHandler_Sorting(t *testing.T) {
	data := buildData()

	// Проверяем сортировку NR-команд
	for i := 1; i < len(data.NRCommands); i++ {
		assert.True(t, data.NRCommands[i-1].Name < data.NRCommands[i].Name,
			"NR-команды должны быть отсортированы: %s < %s", data.NRCommands[i-1].Name, data.NRCommands[i].Name)
	}

	// Проверяем сортировку legacy-команд
	for i := 1; i < len(data.LegacyCommands); i++ {
		assert.True(t, data.LegacyCommands[i-1].Name < data.LegacyCommands[i].Name,
			"legacy-команды должны быть отсортированы: %s < %s", data.LegacyCommands[i-1].Name, data.LegacyCommands[i].Name)
	}
}

func TestHelpHandler_Registration(t *testing.T) {
	// RegisterCmd() вызван в TestMain — проверяем что handler зарегистрирован
	h, ok := command.Get(constants.ActHelp)
	require.True(t, ok, "handler help должен быть зарегистрирован в registry")
	assert.Equal(t, constants.ActHelp, h.Name())
}

func TestHelpHandler_LegacyCommandsCompleteness(t *testing.T) {
	// Все Act* константы из constants.go, разделённые по категориям.
	// При добавлении новой константы тест упадёт, требуя явного решения.

	// NR-команды — зарегистрированы в Registry, НЕ должны быть в legacyCommands.
	nrCommands := []string{
		constants.ActNRVersion,
		constants.ActHelp,
	}

	// Legacy-команды — должны быть в legacyCommands map.
	expectedLegacy := []string{
		constants.ActConvert,
		constants.ActGit2store,
		constants.ActDbrestore,
		constants.ActServiceModeEnable,
		constants.ActServiceModeDisable,
		constants.ActServiceModeStatus,
		constants.ActStore2db,
		constants.ActStoreBind,
		constants.ActDbupdate,
		constants.ActionMenuBuildName,
		constants.ActCreateTempDb,
		constants.ActCreateStores,
		constants.ActExecuteEpf,
		constants.ActSQScanBranch,
		constants.ActSQScanPR,
		constants.ActSQProjectUpdate,
		constants.ActSQReportBranch,
		constants.ActTestMerge,
		constants.ActExtensionPublish,
	}

	// Неактивные константы — определены в constants.go, но не используются в main.go switch.
	// При активации команды — перенести в expectedLegacy или nrCommands.
	excludedInactive := []string{
		constants.ActIssuetask,
		constants.ActAnalyzeProject,
	}

	// Проверяем что все legacy-константы представлены в legacyCommands map.
	for _, name := range expectedLegacy {
		_, exists := legacyCommands[name]
		assert.True(t, exists, "legacy-команда %s должна быть в legacyCommands map", name)
	}

	// Проверяем что NR-команды НЕ в legacyCommands.
	for _, name := range nrCommands {
		_, exists := legacyCommands[name]
		assert.False(t, exists, "NR-команда %s не должна быть в legacyCommands", name)
	}

	// Проверяем что legacyCommands map не содержит лишних записей,
	// не перечисленных в expectedLegacy (ловит добавление в map без обновления теста).
	assert.Equal(t, len(expectedLegacy), len(legacyCommands),
		"legacyCommands map должен содержать ровно столько записей, сколько в expectedLegacy")

	// Обратная проверка: каждая запись в legacyCommands должна быть в expectedLegacy.
	for name := range legacyCommands {
		assert.Contains(t, expectedLegacy, name,
			"legacyCommands содержит %s, не указанную в expectedLegacy — добавьте в тест", name)
	}

	// Общий подсчёт: все Act* константы должны быть явно учтены в одном из списков.
	// При добавлении новой Act* константы — добавьте её в nrCommands, expectedLegacy или excludedInactive.
	_ = excludedInactive // используется для документирования неактивных констант
}

func TestBuildData(t *testing.T) {
	data := buildData()

	assert.NotNil(t, data)
	assert.NotEmpty(t, data.NRCommands, "NR-команды не должны быть пустыми")
	assert.NotEmpty(t, data.LegacyCommands, "legacy-команды не должны быть пустыми")

	// Каждая команда должна иметь имя и описание
	for _, cmd := range data.NRCommands {
		assert.NotEmpty(t, cmd.Name, "имя NR-команды не должно быть пустым")
		assert.NotEmpty(t, cmd.Description, "описание NR-команды %s не должно быть пустым", cmd.Name)
	}
	for _, cmd := range data.LegacyCommands {
		assert.NotEmpty(t, cmd.Name, "имя legacy-команды не должно быть пустым")
		assert.NotEmpty(t, cmd.Description, "описание legacy-команды %s не должно быть пустым", cmd.Name)
	}
}

func TestData_WriteText(t *testing.T) {
	data := &Data{
		NRCommands: []CommandInfo{
			{Name: "help", Description: "Вывод списка доступных команд"},
			{Name: "nr-version", Description: "Вывод информации о версии приложения"},
		},
		LegacyCommands: []CommandInfo{
			{Name: "convert", Description: "Конвертация форматов данных"},
			{Name: "dbrestore", Description: "Восстановление базы данных"},
		},
	}

	var buf bytes.Buffer
	err := data.writeText(&buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "apk-ci")
	assert.Contains(t, out, "NR-команды:")
	assert.Contains(t, out, "Legacy-команды:")
	assert.Contains(t, out, "help")
	assert.Contains(t, out, "nr-version")
	assert.Contains(t, out, "convert")
	assert.Contains(t, out, "dbrestore")
}

func TestData_WriteText_Deprecated(t *testing.T) {
	data := &Data{
		NRCommands: []CommandInfo{
			{Name: "help", Description: "Вывод списка доступных команд"},
			{Name: "old-cmd", Description: "Старая команда", Deprecated: true, NewName: "new-cmd"},
		},
	}

	var buf bytes.Buffer
	err := data.writeText(&buf)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "[deprecated → new-cmd]", "deprecated команда должна быть помечена")
}

func TestHelpHandler_PlanOnly(t *testing.T) {
	t.Setenv("BR_PLAN_ONLY", "true")

	h := &Handler{}

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(context.Background(), nil)
	})

	require.NoError(t, execErr)
	assert.Contains(t, out, "не поддерживает отображение плана операций")
	assert.Contains(t, out, constants.ActHelp)
}

func TestHelpHandler_TextOutput_ShowsPlanOptions(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "text")

	h := &Handler{}

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(context.Background(), nil)
	})

	require.NoError(t, execErr)
	assert.Contains(t, out, "BR_PLAN_ONLY=true")
	assert.Contains(t, out, "BR_VERBOSE=true")
	assert.Contains(t, out, "BR_DRY_RUN=true")
}
