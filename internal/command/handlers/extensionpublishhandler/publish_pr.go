package extensionpublishhandler

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/Kargones/apk-ci/internal/entity/gitea"
)

// GenerateBranchName генерирует имя ветки для обновления расширения.
// Формат: update-{extName}-{version}
// - extName приводится к нижнему регистру
// - пробелы заменяются на дефисы
// - префикс "v" удаляется из версии
//
// Параметры:
//   - extName: имя расширения
//   - version: версия расширения
//
// Возвращает:
//   - string: имя ветки в формате update-{extname}-{version}
func GenerateBranchName(extName, version string) string {
	// Приводим имя к нижнему регистру и заменяем пробелы на дефисы
	normalizedName := strings.ToLower(extName)
	normalizedName = strings.ReplaceAll(normalizedName, " ", "-")

	// Удаляем префикс "v" из версии
	normalizedVersion := strings.TrimPrefix(version, "v")

	return fmt.Sprintf("update-%s-%s", normalizedName, normalizedVersion)
}

// GenerateCommitMessage генерирует сообщение коммита для обновления расширения.
// Формат: chore(ext): update {extName} to {version}
//
// Параметры:
//   - extName: имя расширения
//   - version: версия расширения
//
// Возвращает:
//   - string: сообщение коммита
func GenerateCommitMessage(extName, version string) string {
	return fmt.Sprintf("chore(ext): update %s to %s", extName, version)
}

// BuildExtensionPRBody формирует markdown описание для PR обновления расширения.
// Включает версию, release notes и ссылку на релиз.
// Параметры:
//   - release: информация о релизе (может быть nil)
//   - sourceRepo: полное имя исходного репозитория (org/repo)
//   - extName: имя расширения
//   - releaseURL: URL релиза в исходном репозитории
//
// Возвращает:
//   - string: markdown форматированное описание PR
func BuildExtensionPRBody(release *gitea.Release, sourceRepo, extName, releaseURL string) string {
	var sb strings.Builder

	sb.WriteString("## Extension Update\n\n")
	sb.WriteString(fmt.Sprintf("**Extension:** %s\n", extName))

	// Добавляем версию из release (если есть)
	if release != nil && release.TagName != "" {
		sb.WriteString(fmt.Sprintf("**Version:** %s\n", release.TagName))
	}

	// Добавляем ссылку на источник
	if releaseURL != "" {
		sb.WriteString(fmt.Sprintf("**Source:** [%s](%s)\n", sourceRepo, releaseURL))
	} else if sourceRepo != "" {
		sb.WriteString(fmt.Sprintf("**Source:** %s\n", sourceRepo))
	}

	sb.WriteString("\n### Release Notes\n\n")

	// Добавляем release notes (если есть)
	if release != nil && release.Body != "" {
		// Экранируем специальные символы если необходимо
		sb.WriteString(release.Body)
	} else {
		sb.WriteString("_No release notes provided._")
	}

	sb.WriteString("\n\n---\n")
	sb.WriteString("*This PR was automatically created by apk-ci extension-publish*\n")

	return sb.String()
}

// BuildExtensionPRTitle формирует заголовок PR для обновления расширения.
// Формат: "Update {extName} to {version}"
// Параметры:
//   - extName: имя расширения
//   - version: версия расширения
//
// Возвращает:
//   - string: заголовок PR
func BuildExtensionPRTitle(extName, version string) string {
	return fmt.Sprintf("Update %s to %s", extName, version)
}

// CreateExtensionPR создает Pull Request для обновления расширения.
// Формирует заголовок и описание на основе информации о релизе и синхронизации.
// Параметры:
//   - l: логгер для записи информации о создании PR
//   - api: клиент Gitea API для целевого репозитория
//   - syncResult: результат синхронизации файлов
//   - release: информация о релизе (может быть nil)
//   - extName: имя расширения
//   - sourceRepo: полное имя исходного репозитория
//   - releaseURL: URL релиза в исходном репозитории
//
// Возвращает:
//   - *gitea.PRResponse: информация о созданном PR
//   - error: ошибка создания или nil при успехе
func CreateExtensionPR(l *slog.Logger, api *gitea.API, syncResult *SyncResult, release *gitea.Release, extName, sourceRepo, releaseURL string) (*gitea.PRResponse, error) {
	logger := l

	// Определяем версию для заголовка
	var version string
	if release != nil && release.TagName != "" {
		version = release.TagName
	} else {
		// Если релиз не указан, извлекаем версию из имени ветки
		// Формат ветки: update-{extname}-{version}
		parts := strings.Split(syncResult.NewBranch, "-")
		if len(parts) >= 3 {
			version = parts[len(parts)-1]
		} else {
			version = "unknown"
		}
	}

	// Формируем заголовок и описание PR
	title := BuildExtensionPRTitle(extName, version)
	body := BuildExtensionPRBody(release, sourceRepo, extName, releaseURL)

	// Создаём опции для PR
	opts := gitea.CreatePROptions{
		Title: title,
		Body:  body,
		Head:  syncResult.NewBranch,
		Base:  syncResult.Subscriber.TargetBranch,
	}

	logger.Info("Создание PR для обновления расширения",
		slog.String("extension", extName),
		slog.String("version", version),
		slog.String("head", opts.Head),
		slog.String("base", opts.Base),
	)

	// Создаём PR через API
	pr, err := api.CreatePRWithOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания PR: %w", err)
	}

	logger.Info("PR успешно создан",
		slog.Int64("pr_number", pr.Number),
		slog.String("url", pr.HTMLURL),
	)

	return pr, nil
}
