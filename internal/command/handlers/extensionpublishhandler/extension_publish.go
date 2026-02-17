// Package extensionpublishhandler содержит логику для публикации расширений 1C.
// Реализует механизм поиска подписанных репозиториев через файл project.yaml.
package extensionpublishhandler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/gitea"
)

// ExtensionPublish выполняет публикацию расширения 1C в репозитории подписчиков.
// Команда выполняет полный цикл:
// 1. Получает информацию о релизе из исходного репозитория
// 2. Находит все репозитории, подписанные на обновления
// 3. Для каждого подписчика синхронизирует файлы и создает Pull Request
//
// Переменные окружения:
//   - GITHUB_REPOSITORY: полное имя исходного репозитория (owner/repo)
//   - GITHUB_REF_NAME: тег релиза (например, v1.2.3)
//   - BR_EXT_DIR: каталог с расширением в исходном репозитории (опционально)
//   - BR_DRY_RUN: если "true", выполняется в режиме без изменений
//
// Параметры:
//   - ctx: контекст выполнения
//   - l: логгер
//   - cfg: конфигурация приложения
//
// Возвращает:
//   - error: агрегированная ошибка или nil при успехе
func ExtensionPublish(_ context.Context, l *slog.Logger, cfg *config.Config) error {
	// 1. Получение параметров из конфигурации
	releaseTag := cfg.ReleaseTag
	extensions := cfg.AddArray // Список расширений для публикации
	dryRun := cfg.DryRun

	// 2. Валидация обязательных параметров
	// Проверяем Owner и Repo из конфигурации (заполняются из GITHUB_REPOSITORY)
	if cfg.Owner == "" && cfg.Repo == "" {
		return fmt.Errorf("переменная окружения GITHUB_REPOSITORY не установлена")
	}
	if cfg.Owner == "" || cfg.Repo == "" {
		return fmt.Errorf("некорректный формат GITHUB_REPOSITORY: %s/%s (ожидается owner/repo)", cfg.Owner, cfg.Repo)
	}

	if releaseTag == "" {
		return fmt.Errorf("переменная окружения GITHUB_REF_NAME не установлена")
	}

	// Валидация конфигурации
	if cfg.GiteaURL == "" {
		return fmt.Errorf("GiteaURL не настроен в конфигурации")
	}
	if cfg.AccessToken == "" {
		return fmt.Errorf("AccessToken не настроен в конфигурации")
	}

	sourceRepo := fmt.Sprintf("%s/%s", cfg.Owner, cfg.Repo)
	owner := cfg.Owner
	repo := cfg.Repo

	l.Info("Запуск публикации расширения",
		slog.String("repository", sourceRepo),
		slog.String("release_tag", releaseTag),
		slog.Any("extensions", extensions),
		slog.Bool("dry_run", dryRun),
	)

	// 3. Инициализация Gitea API для исходного репозитория
	// Используем releaseTag для получения файлов из тега релиза
	sourceAPI := gitea.NewGiteaAPI(gitea.Config{
		GiteaURL:    cfg.GiteaURL,
		Owner:       owner,
		Repo:        repo,
		AccessToken: cfg.AccessToken,
		BaseBranch:  "main",
	})

	// 4. Получение информации о релизе
	release, err := sourceAPI.GetReleaseByTag(releaseTag)
	if err != nil {
		return fmt.Errorf("ошибка получения релиза %s: %w", releaseTag, err)
	}

	l.Info("Найден релиз",
		slog.String("tag", release.TagName),
		slog.String("name", release.Name),
	)

	// 5. Поиск подписчиков
	// Расширения уже загружены из конфигурации в cfg.AddArray
	subscribers, err := FindSubscribedRepos(l, sourceAPI, repo, extensions)
	if err != nil {
		return fmt.Errorf("ошибка поиска подписчиков: %w", err)
	}

	if len(subscribers) == 0 {
		l.Info("Подписчики не найдены, завершение")
		return nil
	}

	l.Info("Найдено подписчиков",
		slog.Int("count", len(subscribers)),
	)

	// 6. Инициализация отчёта
	// Имя расширения будет определяться для каждого подписчика индивидуально
	report := &PublishReport{
		ExtensionName: repo, // По умолчанию используем имя репозитория
		Version:       release.TagName,
		SourceRepo:    sourceRepo,
		StartTime:     time.Now(),
	}

	// 8. Обработка каждого подписчика (continue on error)
	for _, sub := range subscribers {
		startTime := time.Now()

		l.Info("Обработка подписчика",
			slog.String("organization", sub.Organization),
			slog.String("repository", sub.Repository),
			slog.String("target_dir", sub.TargetDirectory),
		)

		// Dry-run режим — пропускаем без изменений
		if dryRun {
			l.Info("DRY-RUN: будет синхронизирован",
				slog.String("target", fmt.Sprintf("%s/%s", sub.Organization, sub.Repository)),
				slog.String("dir", sub.TargetDirectory),
			)
			report.Results = append(report.Results, PublishResult{
				Subscriber:   sub,
				Status:       StatusSkipped,
				ErrorMessage: "dry-run mode",
				DurationMs:   time.Since(startTime).Milliseconds(),
			})
			continue
		}

		// Создаём API для целевого репозитория
		targetAPI := gitea.NewGiteaAPI(gitea.Config{
			GiteaURL:    cfg.GiteaURL,
			Owner:       sub.Organization,
			Repo:        sub.Repository,
			AccessToken: cfg.AccessToken,
			BaseBranch:  sub.TargetBranch,
		})
		// Анализируем проект
		analysis, err := targetAPI.AnalyzeProject("main")
		if err != nil {
			l.Error("Ошибка анализа проекта",
				slog.String("error", err.Error()),
			)
			return err
		}
		var targetProjectName string
		// Заполняем поля конфигурации
		if len(analysis) == 0 {
			l.Error("Проект не найден или не соответствует критериям",
				slog.String("organization", sub.Organization),
				slog.String("repository", sub.Repository),
				slog.String("target_branch", sub.TargetBranch),
				slog.String("target_directory", sub.TargetDirectory),
			)

			continue
		} else {
			targetProjectName = analysis[0]
			l.Debug("Результат анализа проекта",
				slog.String("project_name", targetProjectName),
				slog.Any("extensions", cfg.AddArray),
			)
		}

		// Определяем исходный каталог и имя расширения из TargetDirectory подписчика
		sourceDir := cfg.ProjectName + "." + sub.TargetDirectory
		targetDir := targetProjectName + "." + sub.TargetDirectory

		// Синхронизируем файлы
		// extName используется для формирования имени ветки и commit message
		extName := sub.TargetDirectory
		syncResult, err := SyncExtensionToRepo(
			l,
			sourceAPI,
			targetAPI,
			sub,
			sourceDir,
			sourceAPI.BaseBranch,
			targetDir,
			extName,
			release.TagName,
		)
		if err != nil {
			l.Error("Ошибка синхронизации",
				slog.String("target", fmt.Sprintf("%s/%s", sub.Organization, sub.Repository)),
				slog.String("error", err.Error()),
			)
			report.Results = append(report.Results, PublishResult{
				Subscriber:   sub,
				Status:       StatusFailed,
				Error:        err,
				ErrorMessage: err.Error(),
				DurationMs:   time.Since(startTime).Milliseconds(),
			})
			continue // Continue on error — не прерываем цикл
		}

		if syncResult.Error != nil {
			l.Error("Ошибка синхронизации (результат)",
				slog.String("target", fmt.Sprintf("%s/%s", sub.Organization, sub.Repository)),
				slog.String("error", syncResult.Error.Error()),
			)
			report.Results = append(report.Results, PublishResult{
				Subscriber:   sub,
				Status:       StatusFailed,
				SyncResult:   syncResult,
				Error:        syncResult.Error,
				ErrorMessage: syncResult.Error.Error(),
				DurationMs:   time.Since(startTime).Milliseconds(),
			})
			continue // Continue on error
		}

		l.Info("Файлы синхронизированы",
			slog.String("target", fmt.Sprintf("%s/%s", sub.Organization, sub.Repository)),
			slog.String("branch", syncResult.NewBranch),
			slog.Int("files_created", syncResult.FilesCreated),
			slog.Int("files_deleted", syncResult.FilesDeleted),
		)

		// Формируем URL релиза
		releaseURL := fmt.Sprintf("%s/%s/releases/tag/%s", cfg.GiteaURL, sourceRepo, releaseTag)

		// Создаём Pull Request
		pr, err := CreateExtensionPR(l, targetAPI, syncResult, release, extName, sourceRepo, releaseURL)
		if err != nil {
			l.Error("Ошибка создания PR",
				slog.String("target", fmt.Sprintf("%s/%s", sub.Organization, sub.Repository)),
				slog.String("error", err.Error()),
			)
			report.Results = append(report.Results, PublishResult{
				Subscriber:   sub,
				Status:       StatusFailed,
				SyncResult:   syncResult,
				Error:        err,
				ErrorMessage: err.Error(),
				DurationMs:   time.Since(startTime).Milliseconds(),
			})
			continue // Continue on error
		}

		l.Info("PR успешно создан",
			slog.String("target", fmt.Sprintf("%s/%s", sub.Organization, sub.Repository)),
			slog.Int64("pr_number", pr.Number),
			slog.String("url", pr.HTMLURL),
		)

		report.Results = append(report.Results, PublishResult{
			Subscriber: sub,
			Status:     StatusSuccess,
			SyncResult: syncResult,
			PRNumber:   int(pr.Number),
			PRURL:      pr.HTMLURL,
			DurationMs: time.Since(startTime).Milliseconds(),
		})
	}

	// 9. Финализация отчёта
	report.EndTime = time.Now()

	// 10. Вывод отчёта
	if err := ReportResults(report, l); err != nil {
		return fmt.Errorf("ошибка вывода отчёта: %w", err)
	}

	// 11. Return error если есть хотя бы одна ошибка (exit code = 1)
	if report.HasErrors() {
		return fmt.Errorf("публикация завершена с %d ошибками из %d",
			report.FailedCount(), len(report.Results))
	}

	return nil
}
