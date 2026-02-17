package migratehandler

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Replacement описывает одну замену в файле.
type Replacement struct {
	// Line — номер строки (1-based).
	Line int
	// OldCommand — старое имя команды (legacy).
	OldCommand string
	// NewCommand — новое имя команды (NR).
	NewCommand string
}

// brCommandPatterns — регулярные выражения для поиска BR_COMMAND с legacy-командами.
// Формат 1: BR_COMMAND: value (YAML env-блок, с опциональными кавычками)
// Формат 2: BR_COMMAND=value (inline в run)
// Review #32: regex намеренно case-sensitive ([a-z]) — все валидные имена команд
// в constants.go строго lowercase. Uppercase не является валидным значением BR_COMMAND.
// Compiled regex patterns are effectively constant (initialized once, never reassigned).
var brCommandEnvPattern = regexp.MustCompile(
	`^(\s*BR_COMMAND\s*:\s*)("?)([a-z][a-z0-9-]*)("?.*)$`,
)

var brCommandInlinePattern = regexp.MustCompile(
	`(BR_COMMAND=)(["']?)([a-z][a-z0-9-]*)(["']?)`,
)

var brCommandSingleQuotePattern = regexp.MustCompile(
	`^(\s*BR_COMMAND\s*:\s*')([a-z][a-z0-9-]*)('.*)$`,
)

// scanDirectory рекурсивно ищет .yml и .yaml файлы в указанной директории.
func scanDirectory(path string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		if ext == ".yml" || ext == ".yaml" {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка сканирования директории %s: %w", path, err)
	}

	return files, nil
}

// scanFile сканирует файл и находит все замены legacy BR_COMMAND на NR.
// Quick-skip: если файл не содержит "BR_COMMAND" — возвращает пустой slice.
func scanFile(path string, legacyToNR map[string]string) ([]Replacement, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла %s: %w", path, err)
	}

	// AC7: Quick-skip — если файл не содержит BR_COMMAND, пропускаем
	if !strings.Contains(string(content), "BR_COMMAND") {
		return nil, nil
	}

	var replacements []Replacement
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Пропускаем строки-комментарии YAML
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Формат 1: BR_COMMAND: value (с двойными кавычками или без)
		if matches := brCommandEnvPattern.FindStringSubmatch(line); matches != nil {
			// Review #31: Проверяем симметричность кавычек — если есть открывающая ",
			// group 4 должна начинаться с ". Асимметрия = невалидный YAML, пропускаем.
			openQuote := matches[2]
			rest := matches[4]
			if openQuote == "\"" && (len(rest) == 0 || rest[0] != '"') {
				continue
			}
			cmdName := matches[3]
			if newName, ok := legacyToNR[cmdName]; ok {
				replacements = append(replacements, Replacement{
					Line:       lineNum,
					OldCommand: cmdName,
					NewCommand: newName,
				})
			}
			continue
		}

		// Формат 1b: BR_COMMAND: 'value' (с одинарными кавычками)
		if matches := brCommandSingleQuotePattern.FindStringSubmatch(line); matches != nil {
			cmdName := matches[2]
			if newName, ok := legacyToNR[cmdName]; ok {
				replacements = append(replacements, Replacement{
					Line:       lineNum,
					OldCommand: cmdName,
					NewCommand: newName,
				})
			}
			continue
		}

		// Формат 2: BR_COMMAND=value (inline в run-блоках, с опциональными кавычками)
		// TODO(#58): FindStringSubmatch находит только первое вхождение.
		// Если на одной строке несколько BR_COMMAND= (крайне маловероятно в Gitea Actions),
		// только первое будет обработано. Для полного покрытия нужен FindAllStringSubmatch
		// + поддержка множественных Replacement на строку в applyReplacements.
		if matches := brCommandInlinePattern.FindStringSubmatch(line); matches != nil {
			// Review #30: Проверяем симметричность кавычек — open quote (group 2)
			// должна совпадать с close quote (group 4). Асимметрия вроде BR_COMMAND="value'
			// является невалидным YAML/shell и пропускается.
			if matches[2] != matches[4] {
				continue
			}
			cmdName := matches[3]
			if newName, ok := legacyToNR[cmdName]; ok {
				replacements = append(replacements, Replacement{
					Line:       lineNum,
					OldCommand: cmdName,
					NewCommand: newName,
				})
			}
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("ошибка сканирования файла %s: %w", path, scanErr)
	}

	return replacements, nil
}

// backupFile создаёт копию файла с суффиксом .bak.
// Сохраняет permissions и modification time оригинала.
func backupFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла для backup %s: %w", path, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("ошибка получения информации о файле %s: %w", path, err)
	}

	backupPath := path + ".bak"
	if err := os.WriteFile(backupPath, content, info.Mode()); err != nil {
		return fmt.Errorf("ошибка создания backup %s: %w", backupPath, err)
	}

	// Сохраняем modification time оригинала в backup для трассируемости
	if err := os.Chtimes(backupPath, info.ModTime(), info.ModTime()); err != nil {
		return fmt.Errorf("ошибка установки времени модификации backup %s: %w", backupPath, err)
	}

	return nil
}

// applyReplacements применяет замены к файлу атомарно (write to temp → rename).
func applyReplacements(path string, replacements []Replacement) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла %s: %w", path, err)
	}

	lines := strings.Split(string(content), "\n")

	// Индексируем замены по номеру строки для быстрого поиска
	replByLine := make(map[int]Replacement, len(replacements))
	for _, r := range replacements {
		replByLine[r.Line] = r
	}

	// Применяем замены
	for i, line := range lines {
		lineNum := i + 1
		r, ok := replByLine[lineNum]
		if !ok {
			continue
		}

		// Заменяем old command на new command в строке
		// Формат 1: BR_COMMAND: value
		if matches := brCommandEnvPattern.FindStringSubmatch(line); matches != nil {
			lines[i] = matches[1] + matches[2] + r.NewCommand + matches[4]
			continue
		}

		// Формат 1b: BR_COMMAND: 'value'
		if matches := brCommandSingleQuotePattern.FindStringSubmatch(line); matches != nil {
			lines[i] = matches[1] + r.NewCommand + matches[3]
			continue
		}

		// Формат 2: BR_COMMAND=value (inline, с опциональными кавычками)
		lines[i] = brCommandInlinePattern.ReplaceAllStringFunc(line, func(match string) string {
			sub := brCommandInlinePattern.FindStringSubmatch(match)
			if sub[3] == r.OldCommand {
				return sub[1] + sub[2] + r.NewCommand + sub[4]
			}
			return match
		})
	}

	newContent := strings.Join(lines, "\n")

	// Атомарная запись: пишем во временный файл, затем rename
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("ошибка получения информации о файле %s: %w", path, err)
	}

	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".migrate-*.tmp")
	if err != nil {
		return fmt.Errorf("ошибка создания временного файла: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, writeErr := tmpFile.WriteString(newContent); writeErr != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("ошибка записи временного файла: %w", writeErr)
	}

	if closeErr := tmpFile.Close(); closeErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("ошибка закрытия временного файла: %w", closeErr)
	}

	if chmodErr := os.Chmod(tmpPath, info.Mode()); chmodErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("ошибка установки прав на временный файл: %w", chmodErr)
	}

	if renameErr := os.Rename(tmpPath, path); renameErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("ошибка переименования временного файла: %w", renameErr)
	}

	return nil
}
