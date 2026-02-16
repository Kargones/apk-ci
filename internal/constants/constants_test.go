// Package constants содержит тесты для констант проекта apk-ci.
package constants

import (
	"strings"
	"testing"
)

// TestMessageConstants проверяет корректность констант сообщений
func TestMessageConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"MsgAppExit", MsgAppExit, "Завершение работы програмы"},
		{"MsgErrProcessing", MsgErrProcessing, "Обработка ошибки"},
		{"MsgSource", MsgSource, "Исходный"},
		{"MsgDistination", MsgDistination, "Конечный"},
		{"ПроверкаПеременных", ПроверкаПеременных, "Проверка переменных"},
		{"ПеременныеСреды", ПеременныеСреды, "Переменные среды"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Константа %s = %q, ожидалось %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestGitBranchConstants проверяет корректность констант веток Git
func TestGitBranchConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"EdtBranch", EdtBranch, "main"},
		{"OneCBranch", OneCBranch, "xml"},
		{"BaseBranch", BaseBranch, "main"},
		{"TestBranch", TestBranch, "testMerge"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Константа %s = %q, ожидалось %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestPathConstants проверяет корректность констант путей
func TestPathConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"StoreRoot", StoreRoot, "tcp://prod-1c-repo.apkholding.ru/gitops/"},
		{"GiteaWorkflowsPath", GiteaWorkflowsPath, ".gitea/workflows"},
		{"LocalBase", LocalBase, "local"},
		{"WorkDir", WorkDir, "/tmp/4del"},
		{"TempDir", TempDir, "/tmp/4del/temp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Константа %s = %q, ожидалось %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestActionConstants проверяет корректность констант действий
func TestActionConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"ActConvert", ActConvert, "convert"},
		{"ActGit2store", ActGit2store, "git2store"},
		{"ActDbrestore", ActDbrestore, "dbrestore"},
		{"ActIssuetask", ActIssuetask, "issuetask"},
		{"ActServiceModeEnable", ActServiceModeEnable, "service-mode-enable"},
		{"ActServiceModeDisable", ActServiceModeDisable, "service-mode-disable"},
		{"ActServiceModeStatus", ActServiceModeStatus, "service-mode-status"},
		{"ActStore2db", ActStore2db, "store2db"},
		{"ActStoreBind", ActStoreBind, "storebind"},
		{"ActDbupdate", ActDbupdate, "dbupdate"},
		{"ActAnalyzeProject", ActAnalyzeProject, "analyze-project"},
		{"ActionMenuBuildName", ActionMenuBuildName, "action-menu-build"},
		{"ActCreateTempDb", ActCreateTempDb, "create-temp-db"},
		{"ActCreateStores", ActCreateStores, "create-stores"},
		{"ActExecuteEpf", ActExecuteEpf, "execute-epf"},
		{"ActSQScanBranch", ActSQScanBranch, "sq-scan-branch"},
		{"ActSQScanPR", ActSQScanPR, "sq-scan-pr"},
		{"ActSQProjectUpdate", ActSQProjectUpdate, "sq-project-update"},
		{"ActSQReportBranch", ActSQReportBranch, "sq-report-branch"},
		{"ActTestMerge", ActTestMerge, "test-merge"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Константа %s = %q, ожидалось %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestTaskConstants проверяет корректность констант задач
func TestTaskConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		contains string
	}{
		{"TaskDbRestore", TaskDbRestore, "Восстановление базы"},
		{"TaskStore2DbTest", TaskStore2DbTest, "Загрузка конфигурации"},
		{"TaskUpdateDbTest", TaskUpdateDbTest, "Применение обновленной"},
		{"TaskLoadAndUpdateDbTest", TaskLoadAndUpdateDbTest, "Загрузка конфигурации"},
		{"TaskStore2DbProd", TaskStore2DbProd, "Загрузка конфигурации"},
		{"TaskUpdateDbProd", TaskUpdateDbProd, "Применение обновленной"},
		{"TaskLoadAndUpdateDbProd", TaskLoadAndUpdateDbProd, "Загрузка конфигурации"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.constant, tt.contains) {
				t.Errorf("Константа %s = %q, должна содержать %q", tt.name, tt.constant, tt.contains)
			}
		})
	}
}

// TestAPIConstants проверяет корректность констант API
func TestAPIConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"APIVersion", APIVersion, "v1"},
		{"GroupName", GroupName, "qa"},
		{"GitOpsSystemUser", GitOpsSystemUser, "gitops"},
		{"DebugUser", DebugUser, "xor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Константа %s = %q, ожидалось %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestCommitConstants проверяет корректность констант коммитов
func TestCommitConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"DefaultCommitAuthorName", DefaultCommitAuthorName, "GitOps Bot"},
		{"DefaultCommitAuthorEmail", DefaultCommitAuthorEmail, "gitops@apkholding.ru"},
		{"LastCommit", LastCommit, "last"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Константа %s = %q, ожидалось %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestLogLevelConstants проверяет корректность констант уровней логирования
func TestLogLevelConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"LogLevelDebug", LogLevelDebug, "Debug"},
		{"LogLevelInfo", LogLevelInfo, "Info"},
		{"LogLevelWarn", LogLevelWarn, "Warn"},
		{"LogLevelError", LogLevelError, "Error"},
		{"LogLevelDefault", LogLevelDefault, LogLevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Константа %s = %q, ожидалось %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestDefaultCredentials проверяет корректность констант учетных данных по умолчанию
func TestDefaultCredentials(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"DefaultUser", DefaultUser, "gitops"},
		{"DefaultPass", DefaultPass, "gitops"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Константа %s = %q, ожидалось %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestSearchMessages проверяет корректность констант поисковых сообщений
func TestSearchMessages(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		contains string
	}{
		{"SearchMsgStoreLockOk", SearchMsgStoreLockOk, "Захват объектов"},
		{"SearchMsgStoreMergeOk", SearchMsgStoreMergeOk, "Объединение конфигураций"},
		{"SearchMsgStoreBindOk", SearchMsgStoreBindOk, "Подключение информационной базы"},
		{"SearchMsgStoreUnBindOk", SearchMsgStoreUnBindOk, "Отключение от хранилища"},
		{"SearchMsgStoreCommitOk", SearchMsgStoreCommitOk, "Помещение изменений"},
		{"SearchMsgStoreCreateOk", SearchMsgStoreCreateOk, "Создание хранилища"},
		{"SearchMsgBaseCreateOk", SearchMsgBaseCreateOk, "Создание информационной базы"},
		{"SearchMsgBaseCreateOkEn", SearchMsgBaseCreateOkEn, "Infobase created"},
		{"SearchMsgBaseAddOk", SearchMsgBaseAddOk, "Обновление конфигурации"},
		{"SearchMsgBaseLoadOk", SearchMsgBaseLoadOk, "Обновление конфигурации"},
		{"SearchMsgBaseDumpOk", SearchMsgBaseDumpOk, "Сохранение конфигурации"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.constant, tt.contains) {
				t.Errorf("Константа %s = %q, должна содержать %q", tt.name, tt.constant, tt.contains)
			}
		})
	}
}

// TestSpecialConstants проверяет корректность специальных констант
func TestSpecialConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"SearchMsgEmptyFile", SearchMsgEmptyFile, "\ufeff"},
		{"InvalidLink", InvalidLink, "неверная ссылка"},
		{"DefaultServiceModeMessage", DefaultServiceModeMessage, "Система находится в режиме обслуживания"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Константа %s = %q, ожидалось %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestMergeSettingsString проверяет корректность XML строки настроек слияния
func TestMergeSettingsString(t *testing.T) {
	if !strings.Contains(MergeSettingsString, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>") {
		t.Error("MergeSettingsString должна содержать XML заголовок")
	}
	
	if !strings.Contains(MergeSettingsString, "<AllowMainConfigurationObjectDeletion>true</AllowMainConfigurationObjectDeletion>") {
		t.Error("MergeSettingsString должна содержать настройку AllowMainConfigurationObjectDeletion")
	}
	
	if !strings.Contains(MergeSettingsString, "xmlns=\"http://v8.1c.ru/8.3/config/merge/settings\"") {
		t.Error("MergeSettingsString должна содержать правильное пространство имен")
	}
}

// TestConstantsNotEmpty проверяет, что все константы не пустые
func TestConstantsNotEmpty(t *testing.T) {
	constants := map[string]string{
		"MsgAppExit":                    MsgAppExit,
		"MsgErrProcessing":              MsgErrProcessing,
		"MsgSource":                     MsgSource,
		"MsgDistination":                MsgDistination,
		"EdtBranch":                     EdtBranch,
		"OneCBranch":                    OneCBranch,
		"BaseBranch":                    BaseBranch,
		"TestBranch":                    TestBranch,
		"StoreRoot":                     StoreRoot,
		"GiteaWorkflowsPath":            GiteaWorkflowsPath,
		"LocalBase":                     LocalBase,
		"ActConvert":                    ActConvert,
		"APIVersion":                    APIVersion,
		"GroupName":                     GroupName,
		"GitOpsSystemUser":              GitOpsSystemUser,
		"DefaultCommitAuthorName":       DefaultCommitAuthorName,
		"DefaultCommitAuthorEmail":      DefaultCommitAuthorEmail,
		"LogLevelDebug":                 LogLevelDebug,
		"LogLevelInfo":                  LogLevelInfo,
		"LogLevelWarn":                  LogLevelWarn,
		"LogLevelError":                 LogLevelError,
		"DefaultUser":                   DefaultUser,
		"DefaultPass":                   DefaultPass,
		"DefaultServiceModeMessage":     DefaultServiceModeMessage,
		"LastCommit":                    LastCommit,
		"WorkDir":                       WorkDir,
		"TempDir":                       TempDir,
	}

	for name, value := range constants {
		t.Run(name, func(t *testing.T) {
			if value == "" {
				t.Errorf("Константа %s не должна быть пустой", name)
			}
		})
	}
}