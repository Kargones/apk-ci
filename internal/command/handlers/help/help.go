// Package help реализует NR-команду help для вывода списка всех доступных команд.
// Команды группируются на NR-команды (зарегистрированные в Registry) и legacy-команды.
package help

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
)

func init() {
	command.Register(&Handler{})
}

// Data содержит информацию обо всех доступных командах.
type Data struct {
	// NRCommands — команды нового формата (зарегистрированные в Registry).
	NRCommands []CommandInfo `json:"nr_commands"`
	// LegacyCommands — legacy-команды (обрабатываются через switch в main.go).
	LegacyCommands []CommandInfo `json:"legacy_commands"`
}

// CommandInfo описывает одну команду.
type CommandInfo struct {
	// Name — имя команды.
	Name string `json:"name"`
	// Description — описание команды.
	Description string `json:"description"`
	// Deprecated — true если команда deprecated.
	Deprecated bool `json:"deprecated,omitempty"`
	// NewName — новое имя команды (если deprecated).
	NewName string `json:"new_name,omitempty"`
}

// legacyCommands — описания legacy-команд, не зарегистрированных в Registry.
// Временное решение: legacy-команды будут мигрированы в NR-формат.
var legacyCommands = map[string]string{
	constants.ActConvert:            "Конвертация форматов данных",
	constants.ActGit2store:          "Синхронизация Git → хранилище 1C",
	constants.ActDbrestore:          "Восстановление базы данных",
	constants.ActServiceModeEnable:  "Включение сервисного режима",
	constants.ActServiceModeDisable: "Отключение сервисного режима",
	constants.ActServiceModeStatus:  "Статус сервисного режима",
	constants.ActStore2db:           "Загрузка конфигурации из хранилища",
	constants.ActStoreBind:          "Привязка хранилища к базе",
	constants.ActDbupdate:           "Обновление базы данных",
	constants.ActionMenuBuildName:   "Построение меню действий",
	constants.ActCreateTempDb:       "Создание временной базы данных",
	constants.ActCreateStores:       "Создание хранилищ конфигурации",
	constants.ActExecuteEpf:         "Выполнение внешней обработки",
	constants.ActSQScanBranch:       "Сканирование ветки SonarQube",
	constants.ActSQScanPR:           "Сканирование PR SonarQube",
	constants.ActSQProjectUpdate:    "Обновление проекта SonarQube",
	constants.ActSQReportBranch:     "Отчёт по ветке SonarQube",
	constants.ActTestMerge:          "Проверка конфликтов слияния",
	constants.ActExtensionPublish:   "Публикация расширения 1C",
}

// Handler обрабатывает команду help.
type Handler struct{}

// Name возвращает имя команды.
func (h *Handler) Name() string {
	return constants.ActHelp
}

// Description возвращает описание команды для вывода в help.
func (h *Handler) Description() string {
	return "Вывод списка доступных команд"
}

// Execute выполняет команду help: собирает список команд и выводит результат.
func (h *Handler) Execute(ctx context.Context, _ *config.Config) error {
	// Story 7.3 AC-8: plan-only для команд без поддержки плана (early return)
	// Review #36: !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActHelp)
	}

	start := time.Now()

	helpData := buildData()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Текстовый формат — специализированный вывод без metadata (trace_id, duration_ms).
	// Metadata доступна только в JSON формате (аналогично nr-version).
	if format != output.FormatJSON {
		return helpData.writeText(os.Stdout)
	}

	// JSON формат — стандартный Result.
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActHelp,
		Data:    helpData,
		Metadata: &output.Metadata{
			DurationMs: time.Since(start).Milliseconds(),
			TraceID:    traceID,
			APIVersion: constants.APIVersion,
		},
	}

	writer := output.NewWriter(format)
	return writer.Write(os.Stdout, result)
}

// buildData собирает информацию обо всех доступных командах.
func buildData() *Data {
	data := &Data{}

	// NR-команды из Registry
	allHandlers := command.All()
	for name, handler := range allHandlers {
		info := CommandInfo{
			Name:        name,
			Description: handler.Description(),
		}

		// Проверяем deprecated статус через опциональный interface
		if dep, ok := handler.(command.Deprecatable); ok && dep.IsDeprecated() {
			info.Deprecated = true
			info.NewName = dep.NewName()
		}

		data.NRCommands = append(data.NRCommands, info)
	}
	sort.Slice(data.NRCommands, func(i, j int) bool {
		return data.NRCommands[i].Name < data.NRCommands[j].Name
	})

	// Legacy-команды
	for name, desc := range legacyCommands {
		data.LegacyCommands = append(data.LegacyCommands, CommandInfo{
			Name:        name,
			Description: desc,
		})
	}
	sort.Slice(data.LegacyCommands, func(i, j int) bool {
		return data.LegacyCommands[i].Name < data.LegacyCommands[j].Name
	})

	return data
}

// writeText выводит информацию о командах в человекочитаемом формате.
func (d *Data) writeText(w io.Writer) error {
	var sb strings.Builder

	sb.WriteString("apk-ci — инструмент автоматизации 1C:Enterprise\n")
	sb.WriteString("\nNR-команды:\n")

	// Определяем максимальную длину имени для выравнивания
	maxLen := 0
	for _, cmd := range d.NRCommands {
		if len(cmd.Name) > maxLen {
			maxLen = len(cmd.Name)
		}
	}
	for _, cmd := range d.LegacyCommands {
		if len(cmd.Name) > maxLen {
			maxLen = len(cmd.Name)
		}
	}

	for _, cmd := range d.NRCommands {
		desc := cmd.Description
		if cmd.Deprecated {
			desc = fmt.Sprintf("[deprecated → %s] %s", cmd.NewName, desc)
		}
		fmt.Fprintf(&sb, "  %-*s  %s\n", maxLen, cmd.Name, desc)
	}

	sb.WriteString("\nLegacy-команды:\n")
	for _, cmd := range d.LegacyCommands {
		fmt.Fprintf(&sb, "  %-*s  %s\n", maxLen, cmd.Name, cmd.Description)
	}

	sb.WriteString("\nОпции:\n")
	sb.WriteString("  BR_OUTPUT_FORMAT=json    Машиночитаемый вывод\n")
	sb.WriteString("  BR_DRY_RUN=true          Dry-run: план + выполнение пропускается\n")
	sb.WriteString("  BR_PLAN_ONLY=true        Только план операций без выполнения\n")
	sb.WriteString("  BR_VERBOSE=true          Подробный вывод с планом операций\n")
	sb.WriteString("  BR_SHADOW_RUN=true       Shadow-run: сравнение NR и legacy версий\n")

	_, err := fmt.Fprint(w, sb.String())
	return err
}
