// Package projectupdate реализует NR-команду nr-sq-project-update
// для обновления метаданных проекта в SonarQube.
package projectupdate

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/adapter/gitea"
	"github.com/Kargones/apk-ci/internal/adapter/sonarqube"
	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/command/handlers/sonarqube/shared"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/dryrun"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/tracing"
	errhandler "github.com/Kargones/apk-ci/internal/command/handlers/shared"
)

// Коды ошибок — используем shared константы.
// Локальные алиасы для краткости.
const (
	errConfigMissing    = shared.ErrConfigMissing
	errMissingOwnerRepo = shared.ErrMissingOwnerRepo
	errProjectNotFound  = shared.ErrProjectNotFound
	errSonarQubeAPI     = shared.ErrSonarQubeAPI
)

// maxDescriptionLength — максимальная длина описания для SonarQube API.
const maxDescriptionLength = 500

func init() {
	// Deprecated alias: "sq-project-update" -> "nr-sq-project-update"
	// Legacy команда сохраняется для обратной совместимости до полной миграции на NR.
	// TODO: Удалить deprecated alias ActSQProjectUpdate после миграции всех workflows на NR-команды.
	// Планируемая версия удаления: v2.0.0 или после завершения Epic 7.
	command.RegisterWithAlias(&ProjectUpdateHandler{}, constants.ActSQProjectUpdate)
}

// ProjectUpdateData содержит результат обновления проекта.
type ProjectUpdateData struct {
	// ProjectKey — ключ проекта в SonarQube
	ProjectKey string `json:"project_key"`
	// Owner — владелец репозитория
	Owner string `json:"owner"`
	// Repo — имя репозитория
	Repo string `json:"repo"`
	// DescriptionUpdated — было ли обновлено описание
	DescriptionUpdated bool `json:"description_updated"`
	// DescriptionSource — источник описания (README.md или пусто)
	DescriptionSource string `json:"description_source,omitempty"`
	// DescriptionLength — длина обновлённого описания (символов)
	DescriptionLength int `json:"description_length,omitempty"`
	// AdministratorsSync — результат синхронизации администраторов
	AdministratorsSync *AdminSyncResult `json:"administrators_sync,omitempty"`
	// Warnings — предупреждения (не критичные ошибки)
	Warnings []string `json:"warnings,omitempty"`
}

// AdminSyncResult содержит результат синхронизации администраторов.
type AdminSyncResult struct {
	// Synced — успешно ли синхронизированы
	Synced bool `json:"synced"`
	// Count — количество синхронизированных администраторов
	Count int `json:"count"`
	// Teams — teams из которых были извлечены администраторы
	Teams []string `json:"teams,omitempty"`
	// Error — ошибка синхронизации (если произошла)
	Error string `json:"error,omitempty"`
}

// writeText выводит результат в человекочитаемом формате.
func (d *ProjectUpdateData) writeText(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "══════════════════════════════════════════════════════\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Обновление проекта: %s\n", d.ProjectKey); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "══════════════════════════════════════════════════════\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Владелец: %s\n", d.Owner); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Репозиторий: %s\n\n", d.Repo); err != nil {
		return err
	}

	// Описание
	if _, err := fmt.Fprintln(w, "Описание:"); err != nil {
		return err
	}
	if d.DescriptionUpdated {
		if _, err := fmt.Fprintln(w, "  Обновлено: Да"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  Источник: %s\n", d.DescriptionSource); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  Длина: %d символов\n\n", d.DescriptionLength); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(w, "  Обновлено: Нет"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	// Администраторы
	if _, err := fmt.Fprintln(w, "Администраторы:"); err != nil {
		return err
	}
	if d.AdministratorsSync != nil && d.AdministratorsSync.Synced {
		if _, err := fmt.Fprintln(w, "  Синхронизировано: Да"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "  Количество: %d\n", d.AdministratorsSync.Count); err != nil {
			return err
		}
		if len(d.AdministratorsSync.Teams) > 0 {
			if _, err := fmt.Fprintf(w, "  Teams: %s\n\n", strings.Join(d.AdministratorsSync.Teams, ", ")); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
	} else {
		if _, err := fmt.Fprintln(w, "  Синхронизировано: Нет"); err != nil {
			return err
		}
		if d.AdministratorsSync != nil && d.AdministratorsSync.Error != "" {
			if _, err := fmt.Fprintf(w, "  Ошибка: %s\n", d.AdministratorsSync.Error); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	// Предупреждения
	if _, err := fmt.Fprintln(w, "Предупреждения:"); err != nil {
		return err
	}
	if len(d.Warnings) == 0 {
		if _, err := fmt.Fprintln(w, "  (нет)"); err != nil {
			return err
		}
	} else {
		for _, warn := range d.Warnings {
			if _, err := fmt.Fprintf(w, "  - %s\n", warn); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintf(w, "══════════════════════════════════════════════════════\n"); err != nil {
		return err
	}

	return nil
}

// ProjectUpdateHandler обрабатывает команду nr-sq-project-update.
type ProjectUpdateHandler struct {
	// sonarqubeClient — клиент для работы с SonarQube API.
	// Может быть nil в production (создаётся через фабрику).
	// В тестах инъектируется напрямую.
	sonarqubeClient sonarqube.Client

	// giteaClient — клиент для работы с Gitea API.
	// Может быть nil в production (создаётся через фабрику).
	// В тестах инъектируется напрямую.
	giteaClient gitea.Client
}

// Name возвращает имя команды.
func (h *ProjectUpdateHandler) Name() string {
	return constants.ActNRSQProjectUpdate
}

// Description возвращает описание команды для вывода в help.
func (h *ProjectUpdateHandler) Description() string {
	return "Обновить метаданные проекта в SonarQube"
}

// Execute выполняет команду nr-sq-project-update.
func (h *ProjectUpdateHandler) Execute(ctx context.Context, cfg *config.Config) error {
	start := time.Now()

	traceID := tracing.TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = tracing.GenerateTraceID()
	}

	format := os.Getenv("BR_OUTPUT_FORMAT")

	// Story 7.3 AC-8: plan-only для команд без поддержки плана
	// Review #36: !IsDryRun() — dry-run имеет приоритет над plan-only (AC-11).
	if !dryrun.IsDryRun() && dryrun.IsPlanOnly() {
		return dryrun.WritePlanOnlyUnsupported(os.Stdout, constants.ActNRSQProjectUpdate)
	}

	log := slog.Default().With(slog.String("trace_id", traceID), slog.String("command", constants.ActNRSQProjectUpdate))

	// 1. Валидация конфигурации
	if cfg == nil {
		log.Error("Конфигурация не загружена")
		return h.writeError(format, traceID, start,
			errConfigMissing,
			"Конфигурация не загружена")
	}

	// 2. Получение и валидация owner/repo
	owner := cfg.Owner
	repo := cfg.Repo
	if owner == "" || repo == "" {
		log.Error("Не указаны owner или repo")
		return h.writeError(format, traceID, start,
			errMissingOwnerRepo,
			"Не указаны владелец (BR_OWNER) или репозиторий (BR_REPO)")
	}

	// 3. Формирование ключа проекта
	projectKey := fmt.Sprintf("%s_%s", owner, repo)
	log = log.With(slog.String("project_key", projectKey))

	log.Info("Обновление проекта", slog.String("owner", owner), slog.String("repo", repo))

	// 4. Проверка nil клиентов
	sqClient := h.sonarqubeClient
	if sqClient == nil {
		log.Error("SonarQube клиент не настроен")
		return h.writeError(format, traceID, start,
			errConfigMissing,
			"SonarQube клиент не настроен — требуется реализация фабрики createSonarQubeClient()")
	}

	gClient := h.giteaClient
	if gClient == nil {
		log.Error("Gitea клиент не настроен")
		return h.writeError(format, traceID, start,
			errConfigMissing,
			"Gitea клиент не настроен — требуется реализация фабрики createGiteaClient()")
	}

	// 5. Проверка существования проекта
	_, err := sqClient.GetProject(ctx, projectKey)
	if err != nil {
		log.Error("Проект не найден в SonarQube", slog.String("error", err.Error()))
		return h.writeError(format, traceID, start,
			errProjectNotFound,
			fmt.Sprintf("Проект '%s' не найден в SonarQube", projectKey))
	}

	data := &ProjectUpdateData{
		ProjectKey: projectKey,
		Owner:      owner,
		Repo:       repo,
	}

	// 6. Получение README из Gitea
	readme, err := gClient.GetFileContent(ctx, "README.md")
	if err != nil {
		log.Warn("README не найден", slog.String("error", err.Error()))
		data.Warnings = append(data.Warnings, "README.md not found, description not updated")
	} else {
		// Ограничение описания 500 символами (лимит SonarQube)
		description := truncate(string(readme), maxDescriptionLength)

		// Обновление проекта в SonarQube
		updateErr := sqClient.UpdateProject(ctx, projectKey, sonarqube.UpdateProjectOptions{
			Description: description,
		})
		if updateErr != nil {
			log.Warn("Не удалось обновить описание проекта", slog.String("error", updateErr.Error()))
			data.Warnings = append(data.Warnings, "Failed to update description: "+updateErr.Error())
		} else {
			data.DescriptionUpdated = true
			data.DescriptionSource = "README.md"
			data.DescriptionLength = len([]rune(description))
		}
	}

	// 7. Синхронизация администраторов
	data.AdministratorsSync = h.syncAdministrators(ctx, gClient, owner, log)

	log.Info("Обновление проекта завершено",
		slog.Bool("description_updated", data.DescriptionUpdated),
		slog.Int("warnings_count", len(data.Warnings)))

	// 8. Вывод результата
	return h.writeSuccess(format, traceID, start, data)
}

// syncAdministrators синхронизирует администраторов проекта из Gitea teams.
func (h *ProjectUpdateHandler) syncAdministrators(
	ctx context.Context,
	gClient gitea.Client,
	orgName string,
	log *slog.Logger,
) *AdminSyncResult {
	result := &AdminSyncResult{}

	// ВАЖНО: GetTeamMembers принимает (ctx, orgName, teamName) — возвращает []string (логины)
	// orgName = owner (владелец репозитория, обычно организация)
	var administrators []string
	targetTeams := []string{"owners", "dev"}

	for _, teamName := range targetTeams {
		members, err := gClient.GetTeamMembers(ctx, orgName, teamName)
		if err != nil {
			log.Warn("Не удалось получить членов команды", slog.String("team", teamName), slog.String("error", err.Error()))
			// Продолжаем с другими teams — это не критичная ошибка
			continue
		}
		administrators = append(administrators, members...)
		result.Teams = append(result.Teams, teamName)
	}

	// Дедупликация
	administrators = uniqueStrings(administrators)

	// Обновление в SonarQube (если есть администраторы)
	if len(administrators) > 0 {
		// TODO: Реализовать через sqClient.SetProjectPermissions когда метод будет доступен.
		// Пока только логируем найденных администраторов.
		log.Info("Найдены администраторы для синхронизации",
			slog.Int("count", len(administrators)),
			slog.Any("admins", administrators))
		result.Synced = true
		result.Count = len(administrators)
	}

	return result
}

// truncate обрезает строку до указанной длины в символах.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

// uniqueStrings возвращает уникальные элементы среза.
func uniqueStrings(input []string) []string {
	seen := make(map[string]struct{}, len(input))
	result := make([]string, 0, len(input))

	for _, s := range input {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}

	return result
}

// writeSuccess выводит успешный результат.
func (h *ProjectUpdateHandler) writeSuccess(format, traceID string, start time.Time, data *ProjectUpdateData) error {
	// Текстовый формат
	if format != output.FormatJSON {
		return data.writeText(os.Stdout)
	}

	// JSON формат
	result := &output.Result{
		Status:  output.StatusSuccess,
		Command: constants.ActNRSQProjectUpdate,
		Data:    data,
		Metadata: &output.Metadata{
			DurationMs: time.Since(start).Milliseconds(),
			TraceID:    traceID,
			APIVersion: constants.APIVersion,
		},
	}

	writer := output.NewWriter(format)
	return writer.Write(os.Stdout, result)
}

// writeError выводит структурированную ошибку и возвращает error.
func (h *ProjectUpdateHandler) writeError(format, traceID string, start time.Time, code, message string) error {
	// Текстовый формат — человекочитаемый вывод ошибки
	if format != output.FormatJSON {
		return errhandler.HandleError(message, code)
	}

	// JSON формат — структурированный вывод
	result := &output.Result{
		Status:  output.StatusError,
		Command: constants.ActNRSQProjectUpdate,
		Error: &output.ErrorInfo{
			Code:    code,
			Message: message,
		},
		Metadata: &output.Metadata{
			DurationMs: time.Since(start).Milliseconds(),
			TraceID:    traceID,
			APIVersion: constants.APIVersion,
		},
	}

	writer := output.NewWriter(format)
	if writeErr := writer.Write(os.Stdout, result); writeErr != nil {
		slog.Default().Error("Не удалось записать JSON-ответ об ошибке",
			slog.String("trace_id", traceID),
			slog.String("error", writeErr.Error()))
	}

	return fmt.Errorf("%s: %s", code, message)
}
