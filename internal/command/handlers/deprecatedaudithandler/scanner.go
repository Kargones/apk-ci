package deprecatedaudithandler

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Kargones/apk-ci/internal/command"
)

// todoH7Pattern ищет TODO(H-7) комментарии в Go-файлах.
var todoH7Pattern = regexp.MustCompile(`//\s*TODO\s*\(H-7\)(.*)`)

// deprecatedGoPattern ищет стандартный Go маркер deprecated.
var deprecatedGoPattern = regexp.MustCompile(`//\s*Deprecated:(.*)`)

// atDeprecatedPattern ищет @deprecated маркеры.
var atDeprecatedPattern = regexp.MustCompile(`//\s*@deprecated(.*)`)

// legacyCaseConstPattern ищет case constants.Act* в switch-блоке.
var legacyCaseConstPattern = regexp.MustCompile(`^\s*case\s+constants\.(\w+)\s*:`)

// legacyCaseStringPattern ищет case "value": в switch-блоке.
var legacyCaseStringPattern = regexp.MustCompile(`^\s*case\s+"([^"]+)"\s*:`)

// defaultCasePattern ищет default: в switch-блоке (конец обработки).
// Review #30: Паттерн требует, чтобы default: стоял в начале строки (с опциональным whitespace).
// Ограничение: если внутри case-ветки основного switch есть вложенный switch с default:
// на том же уровне отступа, сканирование прервётся раньше. В текущем main.go такого нет.
var defaultCasePattern = regexp.MustCompile(`^\s*default\s*:`)

// noteCommentPattern ищет NOTE-комментарии в main.go.
var noteCommentPattern = regexp.MustCompile(`//\s*NOTE:\s*(.+)`)

// collectDeprecatedAliases собирает deprecated aliases из command registry.
// Не хардкодит список — использует command.ListAllWithAliases().
func collectDeprecatedAliases() []DeprecatedAliasInfo {
	commands := command.ListAllWithAliases()
	var result []DeprecatedAliasInfo

	for _, cmd := range commands {
		if cmd.DeprecatedAlias == "" {
			continue
		}
		result = append(result, DeprecatedAliasInfo{
			DeprecatedName: cmd.DeprecatedAlias,
			NRName:         cmd.Name,
			HandlerPackage: deriveHandlerPackage(cmd.Name),
		})
	}

	return result
}

// deriveHandlerPackage определяет имя пакета handler по NR-имени команды.
// Маппинг статический, т.к. пакеты не меняются часто.
// TODO(H-7): Этот маппинг должен синхронизироваться с реальной структурой пакетов.
// При добавлении нового NR-handler нужно обновить и этот маппинг.
// Review #31: Тест TestDeriveHandlerPackage_CoversAllDeprecatedAliases защищает от рассинхронизации.
func deriveHandlerPackage(nrName string) string {
	packageMap := map[string]string{
		"nr-service-mode-status":  "servicemodestatushandler",
		"nr-service-mode-enable":  "servicemodeenablehandler",
		"nr-service-mode-disable": "servicemodedisablehandler",
		"nr-dbrestore":            "dbrestorehandler",
		"nr-dbupdate":             "dbupdatehandler",
		"nr-create-temp-db":       "createtempdbhandler",
		"nr-store2db":             "store2dbhandler",
		"nr-storebind":            "storebindhandler",
		"nr-create-stores":        "createstoreshandler",
		"nr-convert":              "converthandler",
		"nr-git2store":            "git2storehandler",
		"nr-execute-epf":          "executeepfhandler",
		"nr-sq-scan-branch":       "scanbranch",
		"nr-sq-scan-pr":           "scanpr",
		"nr-sq-report-branch":     "reportbranch",
		"nr-sq-project-update":    "projectupdate",
		"nr-test-merge":           "testmerge",
		"nr-action-menu-build":    "actionmenu",
	}

	if pkg, ok := packageMap[nrName]; ok {
		return pkg
	}
	return "unknown"
}

// scanTodoComments рекурсивно сканирует Go-файлы в rootDir
// и находит TODO(H-7), @deprecated и // Deprecated: маркеры.
// Исключает vendor/, _bmad* директории и *_test.go файлы.
func scanTodoComments(rootDir string) ([]TodoInfo, error) {
	var result []TodoInfo

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Исключаем директории
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || strings.HasPrefix(name, "_bmad") ||
			name == ".git" || name == ".cursor" || name == ".windsurf" ||
			name == ".claude" || name == ".idea" || name == ".vscode" {
				return filepath.SkipDir
			}
			return nil
		}

		// Только .go файлы, исключая тесты
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		todos, scanErr := scanFileForTodos(path, rootDir)
		if scanErr != nil {
			return scanErr
		}
		result = append(result, todos...)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка сканирования директории %s: %w", rootDir, err)
	}

	return result, nil
}

// scanFileForTodos сканирует один файл на наличие deprecated маркеров.
func scanFileForTodos(path, rootDir string) ([]TodoInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла %s: %w", path, err)
	}
	defer file.Close()

	relPath, err := filepath.Rel(rootDir, path)
	if err != nil {
		relPath = path
	}

	var result []TodoInfo
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// TODO(H-7)
		if matches := todoH7Pattern.FindStringSubmatch(line); matches != nil {
			text := strings.TrimSpace(matches[1])
			if text == "" {
				text = strings.TrimSpace(line)
			}
			result = append(result, TodoInfo{
				File: relPath,
				Line: lineNum,
				Text: text,
				Tag:  "H-7",
			})
			continue
		}

		// // Deprecated: ...
		if matches := deprecatedGoPattern.FindStringSubmatch(line); matches != nil {
			result = append(result, TodoInfo{
				File: relPath,
				Line: lineNum,
				Text: strings.TrimSpace(matches[1]),
				Tag:  "Deprecated",
			})
			continue
		}

		// @deprecated
		// Review #33: добавлен continue для консистентности с TODO(H-7) и Deprecated:
		// и предотвращения двойного match при добавлении новых паттернов.
		if matches := atDeprecatedPattern.FindStringSubmatch(line); matches != nil {
			result = append(result, TodoInfo{
				File: relPath,
				Line: lineNum,
				Text: strings.TrimSpace(matches[1]),
				Tag:  "@deprecated",
			})
			continue
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("ошибка чтения файла %s: %w", path, scanErr)
	}

	return result, nil
}

// scanLegacySwitchCases анализирует main.go и находит legacy case-ветки
// в switch-блоке обработки команд.
// rootDir используется для вычисления относительного пути в отчёте.
func scanLegacySwitchCases(mainGoPath, rootDir string, aliases []DeprecatedAliasInfo) ([]LegacyCaseInfo, error) {
	file, err := os.Open(mainGoPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла %s: %w", mainGoPath, err)
	}
	defer file.Close()

	// Вычисляем относительный путь для отчёта
	relPath, relErr := filepath.Rel(rootDir, mainGoPath)
	if relErr != nil {
		relPath = mainGoPath
	}

	// Получаем маппинг deprecated alias → NR name для cross-reference
	deprecatedNames := make(map[string]string)
	for _, info := range aliases {
		deprecatedNames[info.DeprecatedName] = info.NRName
	}

	// Также добавляем маппинг имён констант → имён команд
	constToCmd := buildConstToCmdMap()

	var result []LegacyCaseInfo
	scanner := bufio.NewScanner(file)
	lineNum := 0
	inSwitch := false
	var pendingNote string

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Детектируем начало switch cfg.Command
		if strings.Contains(line, "switch cfg.Command") {
			inSwitch = true
			continue
		}

		// Детектируем конец switch (по default:) — regex исключает ложные совпадения
		// в строковых литералах и вложенных switch-ах
		if inSwitch && defaultCasePattern.MatchString(line) {
			break
		}

		if !inSwitch {
			continue
		}

		// Ищем NOTE-комментарии перед case
		if matches := noteCommentPattern.FindStringSubmatch(line); matches != nil {
			pendingNote = strings.TrimSpace(matches[1])
			continue
		}

		// Review #33: Сбрасываем pendingNote на пустых строках и строках с кодом —
		// предотвращает утечку NOTE-комментария через пустые строки на отдалённую case-ветку.
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			pendingNote = ""
			continue
		}
		if !strings.HasPrefix(trimmed, "//") {
			// Строка с кодом (не case — case обработается ниже и сбросит сам)
			// Проверяем: если это НЕ case-паттерн, сбрасываем pendingNote
			if !legacyCaseConstPattern.MatchString(line) && !legacyCaseStringPattern.MatchString(line) {
				pendingNote = ""
			}
		}

		// Ищем case constants.ActXXX:
		if matches := legacyCaseConstPattern.FindStringSubmatch(line); matches != nil {
			constName := matches[1]
			cmdName := ""
			if name, ok := constToCmd[constName]; ok {
				cmdName = name
			}

			note := ""
			if nrName, ok := deprecatedNames[cmdName]; ok {
				note = fmt.Sprintf("deprecated alias for %s", nrName)
			} else if pendingNote != "" {
				note = pendingNote
			}

			result = append(result, LegacyCaseInfo{
				File:      relPath,
				Line:      lineNum,
				CaseValue: cmdName,
				Note:      note,
			})
			pendingNote = ""
			continue
		}

		// Ищем case "string":
		if matches := legacyCaseStringPattern.FindStringSubmatch(line); matches != nil {
			cmdName := matches[1]
			note := ""
			if nrName, ok := deprecatedNames[cmdName]; ok {
				note = fmt.Sprintf("deprecated alias for %s", nrName)
			} else if pendingNote != "" {
				note = pendingNote
			}

			result = append(result, LegacyCaseInfo{
				File:      relPath,
				Line:      lineNum,
				CaseValue: cmdName,
				Note:      note,
			})
			pendingNote = ""
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("ошибка чтения файла %s: %w", mainGoPath, scanErr)
	}

	return result, nil
}

// buildConstToCmdMap строит маппинг имени константы → значения команды
// для разрешения case constants.ActXXX в имена команд.
// Review #33: кэширование не требуется — функция вызывается один раз за запуск
// (scanLegacySwitchCases вызывается единожды из Execute). Маппинг статический
// и создание map на 18 записей пренебрежимо дёшево.
func buildConstToCmdMap() map[string]string {
	return map[string]string{
		"ActStore2db":        "store2db",
		"ActConvert":         "convert",
		"ActGit2store":       "git2store",
		"ActDbupdate":        "dbupdate",
		"ActDbrestore":       "dbrestore",
		"ActionMenuBuildName": "action-menu-build",
		"ActStoreBind":       "storebind",
		"ActCreateTempDb":    "create-temp-db",
		"ActCreateStores":    "create-stores",
		"ActExecuteEpf":      "execute-epf",
		"ActSQScanBranch":    "sq-scan-branch",
		"ActSQScanPR":        "sq-scan-pr",
		"ActSQProjectUpdate": "sq-project-update",
		"ActSQReportBranch":  "sq-report-branch",
		"ActTestMerge":       "test-merge",
		"ActExtensionPublish": "extension-publish",
		"ActServiceModeEnable":  "service-mode-enable",
		"ActServiceModeDisable": "service-mode-disable",
		"ActServiceModeStatus":  "service-mode-status",
	}
}
