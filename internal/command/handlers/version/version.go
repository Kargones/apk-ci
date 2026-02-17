// Package version реализует NR-команду nr-version для вывода информации о версии приложения.
// Это первая NR-команда, демонстрирующая работу архитектуры:
// Registry → Handler → OutputWriter → Logger + TraceID.
package version

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
)

func RegisterCmd() error {
	return command.Register(&VersionHandler{})
}

// VersionData содержит информацию о версии приложения.
type VersionData struct {
	// Version — полная версия приложения.
	Version string `json:"version"`

	// GoVersion — версия Go, использованная при сборке.
	GoVersion string `json:"go_version"`

	// Commit — хеш коммита на момент сборки.
	Commit string `json:"commit"`

	// RollbackMapping содержит маппинг NR-команд на их deprecated-алиасы.
	// Story 7.4 AC4: rollback-маппинг в JSON выводе.
	RollbackMapping []RollbackEntry `json:"rollback_mapping"`
}

// RollbackEntry описывает маппинг NR-команды на legacy-алиас для rollback.
type RollbackEntry struct {
	// NRCommand — имя NR-команды (например, "nr-service-mode-status").
	NRCommand string `json:"nr_command"`
	// LegacyAlias — deprecated-алиас (например, "service-mode-status").
	// Пустая строка если rollback недоступен.
	LegacyAlias string `json:"legacy_alias"`
}

// writeText выводит информацию о версии в человекочитаемом формате.
func (d *VersionData) writeText(w io.Writer) error {
	_, err := fmt.Fprintf(w, "apk-ci version %s\n  Go:     %s\n  Commit: %s\n",
		d.Version, d.GoVersion, d.Commit)
	if err != nil {
		return err
	}

	// Story 7.4 AC4: секция Rollback Mapping
	if len(d.RollbackMapping) > 0 {
		_, err = fmt.Fprintln(w, "\nRollback Mapping:")
		if err != nil {
			return err
		}
		for _, entry := range d.RollbackMapping {
			alias := entry.LegacyAlias
			if alias == "" {
				alias = "(нет)"
			}
			_, err = fmt.Fprintf(w, "  %-30s → %s\n", entry.NRCommand, alias)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// buildVersionData создаёт VersionData с fallback значениями.
// Если version пустой — используется "dev", если commit пустой — "unknown".
func buildVersionData(version, commit string) *VersionData {
	if version == "" {
		version = "dev"
	}
	if commit == "" {
		commit = "unknown"
	}
	return &VersionData{
		Version:         version,
		GoVersion:       runtime.Version(),
		Commit:          commit,
		RollbackMapping: buildRollbackMapping(),
	}
}

// buildRollbackMapping строит маппинг NR-команд на deprecated-алиасы
// на основе данных из реестра команд. Включает только команды с префиксом "nr-",
// т.к. rollback-маппинг релевантен только для NR-архитектуры.
// Story 7.4 AC4: таблица всех NR-команд с их deprecated-алиасами.
func buildRollbackMapping() []RollbackEntry {
	commands := command.ListAllWithAliases()
	entries := make([]RollbackEntry, 0, len(commands))
	for _, cmd := range commands {
		if !strings.HasPrefix(cmd.Name, "nr-") {
			continue
		}
		entries = append(entries, RollbackEntry{
			NRCommand:   cmd.Name,
			LegacyAlias: cmd.DeprecatedAlias,
		})
	}
	return entries
}

// VersionHandler обрабатывает команду nr-version.
type VersionHandler struct{}

// Name возвращает имя команды.
func (h *VersionHandler) Name() string {
	return constants.ActNRVersion
}

// Description возвращает описание команды для вывода в help.
func (h *VersionHandler) Description() string {
	return "Вывод информации о версии приложения"
}

// Execute выполняет команду nr-version: собирает данные о версии и выводит результат.
func (h *VersionHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	versionData := buildVersionData(constants.Version, constants.PreCommitHash)

	// Получаем trace ID из контекста или генерируем новый
	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Story 7.3 AC-8: plan-only для команд без поддержки плана
	// Review #36: !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRVersion)
	}

	// Текстовый формат — специализированный вывод без metadata/trace_id.
	// Используется writeText напрямую (не через output.Writer), т.к. текстовый вывод
	// версии имеет компактный формат, отличный от стандартного Result.
	// Metadata (trace_id, duration_ms) доступна только в JSON формате.
	if format != output.FormatJSON {
		return versionData.writeText(os.Stdout)
	}

	// JSON формат — стандартный Result
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRVersion,
		Data:    versionData,
		Metadata: &output.Metadata{
			DurationMs: time.Since(start).Milliseconds(),
			TraceID:    traceID,
			APIVersion: constants.APIVersion,
		},
	}

	writer := output.NewWriter(format)
	return writer.Write(os.Stdout, result)
}
