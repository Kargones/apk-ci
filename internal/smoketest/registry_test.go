package smoketest

import (
	"testing"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/constants"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kargones/apk-ci/internal/command/handlers"
)

// allNRCommands — полный список NR-команд и help (24 шт.: 23 NR + help).
// Каждый элемент: {константа из constants.go, ожидаемое строковое значение}.
func init() {
	if err := handlers.RegisterAll(); err != nil {
		panic("smoketest: failed to register handlers: " + err.Error())
	}
}

var allNRCommands = []struct {
	constant string
	name     string
}{
	{constants.ActNRVersion, "nr-version"},
	{constants.ActNRServiceModeStatus, "nr-service-mode-status"},
	{constants.ActNRServiceModeEnable, "nr-service-mode-enable"},
	{constants.ActNRServiceModeDisable, "nr-service-mode-disable"},
	{constants.ActNRForceDisconnectSessions, "nr-force-disconnect-sessions"},
	{constants.ActNRDbrestore, "nr-dbrestore"},
	{constants.ActNRDbupdate, "nr-dbupdate"},
	{constants.ActNRCreateTempDb, "nr-create-temp-db"},
	{constants.ActNRStore2db, "nr-store2db"},
	{constants.ActNRStorebind, "nr-storebind"},
	{constants.ActNRCreateStores, "nr-create-stores"},
	{constants.ActNRConvert, "nr-convert"},
	{constants.ActNRGit2store, "nr-git2store"},
	{constants.ActNRExecuteEpf, "nr-execute-epf"},
	{constants.ActNRSQScanBranch, "nr-sq-scan-branch"},
	{constants.ActNRSQScanPR, "nr-sq-scan-pr"},
	{constants.ActNRSQReportBranch, "nr-sq-report-branch"},
	{constants.ActNRSQProjectUpdate, "nr-sq-project-update"},
	{constants.ActNRTestMerge, "nr-test-merge"},
	{constants.ActNRActionMenuBuild, "nr-action-menu-build"},
	{constants.ActNRMigrate, "nr-migrate"},
	{constants.ActNRDeprecatedAudit, "nr-deprecated-audit"},
	{constants.ActNRExtensionPublish, "nr-extension-publish"},
	{constants.ActHelp, "help"},
}

// deprecatedAliases — полный список deprecated alias → NR-команда (18 шт.).
// Review #33: 5 команд без deprecated aliases: nr-version, nr-force-disconnect-sessions,
// nr-migrate, nr-deprecated-audit и help (см. noLegacyCommands).
var deprecatedAliases = []struct {
	deprecated string
	newName    string
}{
	{constants.ActServiceModeStatus, constants.ActNRServiceModeStatus},
	{constants.ActServiceModeEnable, constants.ActNRServiceModeEnable},
	{constants.ActServiceModeDisable, constants.ActNRServiceModeDisable},
	{constants.ActDbrestore, constants.ActNRDbrestore},
	{constants.ActDbupdate, constants.ActNRDbupdate},
	{constants.ActCreateTempDb, constants.ActNRCreateTempDb},
	{constants.ActStore2db, constants.ActNRStore2db},
	{constants.ActStoreBind, constants.ActNRStorebind},
	{constants.ActCreateStores, constants.ActNRCreateStores},
	{constants.ActConvert, constants.ActNRConvert},
	{constants.ActGit2store, constants.ActNRGit2store},
	{constants.ActExecuteEpf, constants.ActNRExecuteEpf},
	{constants.ActSQScanBranch, constants.ActNRSQScanBranch},
	{constants.ActSQScanPR, constants.ActNRSQScanPR},
	{constants.ActSQReportBranch, constants.ActNRSQReportBranch},
	{constants.ActSQProjectUpdate, constants.ActNRSQProjectUpdate},
	{constants.ActTestMerge, constants.ActNRTestMerge},
	{constants.ActionMenuBuildName, constants.ActNRActionMenuBuild},
	{constants.ActExtensionPublish, constants.ActNRExtensionPublish},
}

// TestSmoke_AllNRCommandsRegistered проверяет что все 22 NR-команд + help
// зарегистрированы в глобальном реестре через command.Get().
func TestSmoke_AllNRCommandsRegistered(t *testing.T) {
	for _, tc := range allNRCommands {
		t.Run(tc.name, func(t *testing.T) {
			h, ok := command.Get(tc.name)
			require.True(t, ok, "команда %q должна быть зарегистрирована в registry", tc.name)
			assert.Equal(t, tc.constant, tc.name,
				"значение константы constants.Act* должно совпадать с ожидаемым строковым именем команды")
			assert.NotNil(t, h, "handler не должен быть nil")
		})
	}
}

// TestSmoke_DeprecatedAliasesRegistered проверяет что все 19 deprecated aliases
// зарегистрированы и реализуют интерфейс command.Deprecatable.
func TestSmoke_DeprecatedAliasesRegistered(t *testing.T) {
	for _, tc := range deprecatedAliases {
		t.Run(tc.deprecated+"->"+tc.newName, func(t *testing.T) {
			h, ok := command.Get(tc.deprecated)
			require.True(t, ok, "deprecated alias %q должен быть зарегистрирован", tc.deprecated)

			dep, isDep := h.(command.Deprecatable)
			require.True(t, isDep, "handler для %q должен реализовывать Deprecatable", tc.deprecated)
			assert.True(t, dep.IsDeprecated(), "%q должен быть deprecated", tc.deprecated)
			assert.Equal(t, tc.newName, dep.NewName(),
				"NewName() для %q должен быть %q", tc.deprecated, tc.newName)
		})
	}
}

// TestSmoke_HandlersNameDescription проверяет что каждый handler возвращает
// непустые Name() и Description(), и Name() соответствует константе из constants.go.
func TestSmoke_HandlersNameDescription(t *testing.T) {
	for _, tc := range allNRCommands {
		t.Run(tc.name, func(t *testing.T) {
			h, ok := command.Get(tc.name)
			require.True(t, ok, "команда %q должна быть зарегистрирована", tc.name)

			assert.NotEmpty(t, h.Name(), "Name() не должен быть пустым для %q", tc.name)
			assert.NotEmpty(t, h.Description(), "Description() не должен быть пустым для %q", tc.name)
			assert.Equal(t, tc.name, h.Name(), "Name() должен совпадать с именем регистрации для %q", tc.name)
		})
	}
}

// TestSmoke_NamesUnique проверяет что все имена в command.Names() уникальны —
// нет дубликатов в реестре.
func TestSmoke_NamesUnique(t *testing.T) {
	names := command.Names()
	seen := make(map[string]bool, len(names))

	for _, name := range names {
		assert.False(t, seen[name], "обнаружен дубликат имени команды: %q", name)
		seen[name] = true
	}
}

// TestSmoke_NamesDeterministic проверяет что command.Names() возвращает
// одинаковый отсортированный результат при повторных вызовах.
func TestSmoke_NamesDeterministic(t *testing.T) {
	first := command.Names()
	for i := 0; i < 10; i++ {
		current := command.Names()
		assert.Equal(t, first, current, "Names() должен возвращать детерминированный результат (итерация %d)", i)
	}
}

// TestDeprecatedBridge_RollbackScenario проверяет что вызов deprecated-алиаса
// выполняет тот же handler что и NR-команда.
// Story 7.4 AC1: deprecated alias делегирует на NR handler.
func TestDeprecatedBridge_RollbackScenario(t *testing.T) {
	for _, tc := range deprecatedAliases {
		t.Run(tc.deprecated+"->"+tc.newName, func(t *testing.T) {
			// Получаем NR handler
			nrHandler, ok := command.Get(tc.newName)
			require.True(t, ok, "NR-команда %q должна быть зарегистрирована", tc.newName)

			// Получаем deprecated handler
			depHandler, ok := command.Get(tc.deprecated)
			require.True(t, ok, "deprecated alias %q должен быть зарегистрирован", tc.deprecated)

			// Deprecated handler должен реализовывать Deprecatable
			dep, isDep := depHandler.(command.Deprecatable)
			require.True(t, isDep, "%q должен реализовывать Deprecatable", tc.deprecated)
			assert.True(t, dep.IsDeprecated())
			assert.Equal(t, tc.newName, dep.NewName())

			// Оба handler.Description() должны совпадать — bridge делегирует на NR
			assert.Equal(t, nrHandler.Description(), depHandler.Description(),
				"Description() deprecated alias %q должен совпадать с NR handler %q",
				tc.deprecated, tc.newName)
		})
	}
}

// noLegacyCommands — NR-команды без legacy-аналога.
// Review #32: вынесено в пакетную переменную для переиспользования и удобства обновления.
var noLegacyCommands = map[string]bool{
	constants.ActNRVersion:                 true,
	constants.ActNRForceDisconnectSessions: true,
	constants.ActNRMigrate:                 true,
	constants.ActNRDeprecatedAudit:         true,
	constants.ActHelp:                      true,
}

// TestRegistry_AllNRCommandsHaveDeprecatedAlias проверяет что все NR-команды
// (кроме nr-version, help, nr-force-disconnect-sessions, nr-migrate, nr-deprecated-audit)
// имеют deprecated-алиас.
// Story 7.4 AC1: защита от регрессии при добавлении новых NR-команд.
func TestRegistry_AllNRCommandsHaveDeprecatedAlias(t *testing.T) {
	allWithAliases := command.ListAllWithAliases()

	for _, info := range allWithAliases {
		t.Run(info.Name, func(t *testing.T) {
			if noLegacyCommands[info.Name] {
				assert.Empty(t, info.DeprecatedAlias,
					"команда %q не должна иметь deprecated-алиас (нет legacy-аналога)", info.Name)
				return
			}
			assert.NotEmpty(t, info.DeprecatedAlias,
				"NR-команда %q должна иметь deprecated-алиас для rollback", info.Name)
		})
	}
}

// TestListAllWithAliases_MatchesDeprecatedAliases проверяет что ListAllWithAliases
// возвращает маппинг, совпадающий с известным списком deprecated-алиасов.
// Story 7.4 AC4: полный rollback-маппинг.
func TestListAllWithAliases_MatchesDeprecatedAliases(t *testing.T) {
	allWithAliases := command.ListAllWithAliases()

	// Строим маппинг из ListAllWithAliases
	aliasMap := make(map[string]string)
	for _, info := range allWithAliases {
		if info.DeprecatedAlias != "" {
			aliasMap[info.Name] = info.DeprecatedAlias
		}
	}

	// Должна быть хотя бы одна команда с непустым LegacyAlias
	assert.NotEmpty(t, aliasMap,
		"маппинг должен содержать хотя бы одну команду с непустым deprecated-алиасом")

	// Проверяем соответствие с известным списком deprecated aliases
	for _, tc := range deprecatedAliases {
		alias, ok := aliasMap[tc.newName]
		assert.True(t, ok, "ListAllWithAliases должен содержать %q", tc.newName)
		assert.Equal(t, tc.deprecated, alias,
			"alias для %q должен быть %q, получен %q", tc.newName, tc.deprecated, alias)
	}

	// Количество alias-записей должно совпадать
	assert.Equal(t, len(deprecatedAliases), len(aliasMap),
		"количество deprecated-алиасов должно совпадать")
}

// TestSmoke_TotalCommandCount проверяет общее количество зарегистрированных команд.
// 23 основных (22 NR + help) + 18 deprecated = 41.
func TestSmoke_TotalCommandCount(t *testing.T) {
	all := command.All()
	names := command.Names()

	// Количества должны совпадать
	assert.Equal(t, len(all), len(names), "All() и Names() должны возвращать одинаковое количество")

	// Точное количество: 24 основных + 19 deprecated = 41
	expected := len(allNRCommands) + len(deprecatedAliases)
	assert.Equal(t, expected, len(all),
		"ожидается %d команд (24 основных + 19 deprecated), получено %d",
		expected, len(all))
}
