package deprecatedaudithandler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kargones/apk-ci/internal/constants"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Blank imports для регистрации handlers (необходимо для TestCollectDeprecatedAliases).
	// deprecatedaudithandler не включён — он регистрируется автоматически при загрузке
	// текущего тестового пакета (init() из handler.go вызывается первым).
	_ "github.com/Kargones/apk-ci/internal/command/handlers/converthandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/createstoreshandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/createtempdbhandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/dbrestorehandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/dbupdatehandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/executeepfhandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/forcedisconnecthandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/git2storehandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/gitea/actionmenu"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/gitea/testmerge"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/help"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/migratehandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/servicemodedisablehandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/servicemodeenablehandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/servicemodestatushandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/sonarqube/projectupdate"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/sonarqube/reportbranch"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/sonarqube/scanbranch"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/sonarqube/scanpr"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/store2dbhandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/storebindhandler"
	_ "github.com/Kargones/apk-ci/internal/command/handlers/version"
)

func TestCollectDeprecatedAliases(t *testing.T) {
	aliases := collectDeprecatedAliases()

	// Должно быть ровно 18 deprecated aliases
	assert.Len(t, aliases, 18, "ожидается 18 deprecated aliases")

	// Проверяем что каждый alias имеет все поля
	for _, a := range aliases {
		assert.NotEmpty(t, a.DeprecatedName, "DeprecatedName не должен быть пустым")
		assert.NotEmpty(t, a.NRName, "NRName не должен быть пустым")
		assert.NotEmpty(t, a.HandlerPackage, "HandlerPackage не должен быть пустым")
		assert.NotEqual(t, "unknown", a.HandlerPackage,
			"HandlerPackage для %s не должен быть 'unknown'", a.NRName)
	}

	// Проверяем наличие конкретных aliases
	aliasMap := make(map[string]string)
	for _, a := range aliases {
		aliasMap[a.DeprecatedName] = a.NRName
	}
	assert.Equal(t, "nr-service-mode-status", aliasMap["service-mode-status"])
	assert.Equal(t, "nr-dbrestore", aliasMap["dbrestore"])
	assert.Equal(t, "nr-git2store", aliasMap["git2store"])
	assert.Equal(t, "nr-sq-scan-branch", aliasMap["sq-scan-branch"])
}

func TestScanTodoComments(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string // относительный путь → содержимое
		wantTags []string          // ожидаемые теги найденных элементов
		wantLen  int
	}{
		{
			name: "TODO(H-7) найден",
			files: map[string]string{
				"handler.go": `package foo
// TODO(H-7): Deprecated alias будет удалён в v2.0.0
func Bar() {}
`,
			},
			wantTags: []string{"H-7"},
			wantLen:  1,
		},
		{
			name: "@deprecated найден",
			files: map[string]string{
				"old.go": `package foo
// @deprecated используйте NewFunc вместо OldFunc
func OldFunc() {}
`,
			},
			wantTags: []string{"@deprecated"},
			wantLen:  1,
		},
		{
			name: "Deprecated: Go маркер",
			files: map[string]string{
				"legacy.go": `package foo
// Deprecated: используйте NewAPI
func LegacyAPI() {}
`,
			},
			wantTags: []string{"Deprecated"},
			wantLen:  1,
		},
		{
			name: "Нет маркеров",
			files: map[string]string{
				"clean.go": `package foo
// Обычный комментарий
func Clean() {}
`,
			},
			wantLen: 0,
		},
		{
			name: "Тестовый файл исключается",
			files: map[string]string{
				"handler_test.go": `package foo
// TODO(H-7): В тесте — не должен попадать в отчёт
func TestBar() {}
`,
			},
			wantLen: 0,
		},
		{
			name: "Несколько маркеров в одном файле",
			files: map[string]string{
				"multi.go": `package foo
// TODO(H-7): Первый TODO
func A() {}
// TODO(H-7): Второй TODO
func B() {}
// @deprecated Третий маркер
func C() {}
`,
			},
			wantTags: []string{"H-7", "H-7", "@deprecated"},
			wantLen:  3,
		},
		{
			name: "TODO(H-7) инлайн-комментарий",
			files: map[string]string{
				"inline.go": "package foo\nvar x = 1 // TODO(H-7): Inline TODO\nfunc F() {}\n",
			},
			wantTags: []string{"H-7"},
			wantLen:  1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			for relPath, content := range tc.files {
				fullPath := filepath.Join(tmpDir, relPath)
				require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
				require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o644))
			}

			todos, err := scanTodoComments(tmpDir)
			require.NoError(t, err)
			assert.Len(t, todos, tc.wantLen)

			for i, tag := range tc.wantTags {
				if i < len(todos) {
					assert.Equal(t, tag, todos[i].Tag)
				}
			}
		})
	}
}

func TestScanTodoComments_VendorExcluded(t *testing.T) {
	tmpDir := t.TempDir()

	// Файл в vendor/ — не должен сканироваться
	vendorDir := filepath.Join(tmpDir, "vendor", "pkg")
	require.NoError(t, os.MkdirAll(vendorDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(vendorDir, "lib.go"),
		[]byte("package pkg\n// TODO(H-7): В vendor — исключить\nfunc F() {}\n"),
		0o644,
	))

	todos, err := scanTodoComments(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, todos, "файлы из vendor/ не должны сканироваться")
}

func TestScanTodoComments_BmadExcluded(t *testing.T) {
	tmpDir := t.TempDir()

	// Файл в _bmad/ — не должен сканироваться
	bmadDir := filepath.Join(tmpDir, "_bmad-output")
	require.NoError(t, os.MkdirAll(bmadDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(bmadDir, "tool.go"),
		[]byte("package tool\n// TODO(H-7): В _bmad — исключить\nfunc T() {}\n"),
		0o644,
	))

	todos, err := scanTodoComments(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, todos, "файлы из _bmad* не должны сканироваться")
}

func TestScanTodoComments_IDEDirsExcluded(t *testing.T) {
	tmpDir := t.TempDir()

	// Файлы в IDE-директориях — не должны сканироваться
	ideDirs := []string{".cursor", ".windsurf", ".claude", ".idea", ".vscode"}
	for _, dir := range ideDirs {
		dirPath := filepath.Join(tmpDir, dir)
		require.NoError(t, os.MkdirAll(dirPath, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(dirPath, "config.go"),
			[]byte("package config\n// TODO(H-7): В IDE-директории — исключить\nfunc F() {}\n"),
			0o644,
		))
	}

	todos, err := scanTodoComments(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, todos, "файлы из IDE-директорий (.cursor, .windsurf и т.д.) не должны сканироваться")
}

func TestScanLegacySwitchCases(t *testing.T) {
	tmpDir := t.TempDir()
	mainGo := filepath.Join(tmpDir, "main.go")

	content := `package main

func run() int {
	switch cfg.Command {
	case constants.ActStore2db:
		err = app.Store2DbWithConfig(&ctx, l, cfg)
	case constants.ActConvert:
		err = app.Convert(&ctx, l, cfg)
	// NOTE: Кастомная обработка extension-publish
	case "extension-publish":
		err = app.ExtPublish(&ctx, l, cfg)
	case constants.ActDbrestore:
		err = app.DbRestoreWithConfig(&ctx, l, cfg, cfg.InfobaseName)
	case constants.ActTestMerge:
		err = app.TestMerge(&ctx, l, cfg)
	default:
		return 2
	}
	return 0
}
`
	require.NoError(t, os.WriteFile(mainGo, []byte(content), 0o644))

	aliases := []DeprecatedAliasInfo{
		{DeprecatedName: "store2db", NRName: "nr-store2db"},
		{DeprecatedName: "convert", NRName: "nr-convert"},
		{DeprecatedName: "dbrestore", NRName: "nr-dbrestore"},
		{DeprecatedName: "test-merge", NRName: "nr-test-merge"},
	}

	cases, err := scanLegacySwitchCases(mainGo, tmpDir, aliases)
	require.NoError(t, err)

	assert.Len(t, cases, 5)

	// store2db — deprecated alias
	assert.Equal(t, "store2db", cases[0].CaseValue)
	assert.Equal(t, "main.go", cases[0].File)
	assert.Greater(t, cases[0].Line, 0)
	assert.Equal(t, "deprecated alias for nr-store2db", cases[0].Note)

	// convert — deprecated alias
	assert.Equal(t, "convert", cases[1].CaseValue)
	assert.Equal(t, "deprecated alias for nr-convert", cases[1].Note)

	// extension-publish — не deprecated alias, но с NOTE-комментарием
	assert.Equal(t, "extension-publish", cases[2].CaseValue)
	assert.Equal(t, "Кастомная обработка extension-publish", cases[2].Note)

	// dbrestore — deprecated alias
	assert.Equal(t, "dbrestore", cases[3].CaseValue)
	assert.Equal(t, "deprecated alias for nr-dbrestore", cases[3].Note)

	// test-merge — deprecated alias
	assert.Equal(t, "test-merge", cases[4].CaseValue)
	assert.Equal(t, "deprecated alias for nr-test-merge", cases[4].Note)
}

func TestScanLegacySwitchCases_FileNotFound(t *testing.T) {
	_, err := scanLegacySwitchCases("/nonexistent/main.go", "/nonexistent", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ошибка открытия файла")
}

func TestDeriveHandlerPackage(t *testing.T) {
	tests := []struct {
		nrName   string
		expected string
	}{
		{"nr-service-mode-status", "servicemodestatushandler"},
		{"nr-dbrestore", "dbrestorehandler"},
		{"nr-sq-scan-branch", "scanbranch"},
		{"nr-action-menu-build", "actionmenu"},
		{"nr-unknown-command", "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.nrName, func(t *testing.T) {
			assert.Equal(t, tc.expected, deriveHandlerPackage(tc.nrName))
		})
	}
}

func TestBuildConstToCmdMap(t *testing.T) {
	m := buildConstToCmdMap()

	assert.Equal(t, "store2db", m["ActStore2db"])
	assert.Equal(t, "dbrestore", m["ActDbrestore"])
	assert.Equal(t, "action-menu-build", m["ActionMenuBuildName"])
	assert.Equal(t, "test-merge", m["ActTestMerge"])
}

// TestBuildConstToCmdMap_SyncWithConstants проверяет что значения в buildConstToCmdMap
// соответствуют реальным значениям констант из пакета constants.
// Защита от рассинхронизации хардкод-маппинга при переименовании констант.
func TestBuildConstToCmdMap_SyncWithConstants(t *testing.T) {
	m := buildConstToCmdMap()

	// Маппинг имени константы → ожидаемое значение из constants пакета
	expected := map[string]string{
		"ActStore2db":          constants.ActStore2db,
		"ActConvert":           constants.ActConvert,
		"ActGit2store":         constants.ActGit2store,
		"ActDbupdate":          constants.ActDbupdate,
		"ActDbrestore":         constants.ActDbrestore,
		"ActionMenuBuildName":  constants.ActionMenuBuildName,
		"ActStoreBind":         constants.ActStoreBind,
		"ActCreateTempDb":      constants.ActCreateTempDb,
		"ActCreateStores":      constants.ActCreateStores,
		"ActExecuteEpf":        constants.ActExecuteEpf,
		"ActSQScanBranch":      constants.ActSQScanBranch,
		"ActSQScanPR":          constants.ActSQScanPR,
		"ActSQProjectUpdate":   constants.ActSQProjectUpdate,
		"ActSQReportBranch":    constants.ActSQReportBranch,
		"ActTestMerge":         constants.ActTestMerge,
		"ActExtensionPublish":  constants.ActExtensionPublish,
		"ActServiceModeEnable":  constants.ActServiceModeEnable,
		"ActServiceModeDisable": constants.ActServiceModeDisable,
		"ActServiceModeStatus":  constants.ActServiceModeStatus,
	}

	for constName, expectedValue := range expected {
		actual, ok := m[constName]
		require.True(t, ok, "buildConstToCmdMap должен содержать %q", constName)
		assert.Equal(t, expectedValue, actual,
			"buildConstToCmdMap[%q] = %q, но constants.%s = %q — обновите маппинг в scanner.go",
			constName, actual, constName, expectedValue)
	}

	// Review #32: обратная проверка — каждая запись в map должна быть в expected.
	// Защита от "лишних" записей с невалидными ключами.
	for constName := range m {
		_, ok := expected[constName]
		assert.True(t, ok,
			"buildConstToCmdMap содержит неизвестную константу %q — удалите или добавьте в expected", constName)
	}

	// Количество записей должно совпадать — защита от лишних/пропущенных
	assert.Equal(t, len(expected), len(m),
		"buildConstToCmdMap содержит %d записей, ожидается %d — проверьте синхронизацию",
		len(m), len(expected))
}

// TestDeriveHandlerPackage_Completeness проверяет что все deprecated aliases
// из registry имеют корректный маппинг в deriveHandlerPackage (не "unknown").
// Защита от рассинхронизации hardcoded маппинга при добавлении новых handlers.
func TestDeriveHandlerPackage_Completeness(t *testing.T) {
	aliases := collectDeprecatedAliases()
	require.NotEmpty(t, aliases, "должен быть хотя бы один deprecated alias")

	for _, a := range aliases {
		pkg := deriveHandlerPackage(a.NRName)
		assert.NotEqual(t, "unknown", pkg,
			"deriveHandlerPackage(%q) не должен возвращать 'unknown' — обновите packageMap в scanner.go", a.NRName)
	}
}
