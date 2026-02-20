package gitea

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Kargones/apk-ci/internal/constants"
)

// GetLatestCommit получает информацию о последнем коммите в ветке.
// Возвращает метаданные самого свежего коммита для отслеживания
// последних изменений в указанной ветке репозитория.
// Параметры:
//   - branch: имя ветки для получения последнего коммита
//
// Возвращает:
//   - *Commit: указатель на структуру с информацией о коммите
//   - error: ошибка получения коммита или nil при успехе
const branchMain = "main"

func (g *API) GetLatestCommit(ctx context.Context, branch string) (*Commit, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/commits?sha=%s&limit=1", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, branch)

	statusCode, body, err := g.sendReq(ctx, urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %w", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении коммитов: статус %d", statusCode)
	}

	var commits []Commit
	err = json.Unmarshal([]byte(body), &commits)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %w", err)
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("коммиты не найдены")
	}

	return &commits[0], nil
}

// GetCommits получает список коммитов в ветке.
// Возвращает массив коммитов для указанной ветки.
// Параметры:
//   - branch: имя ветки для получения коммитов
//   - limit: максимальное количество коммитов для получения (0 - без ограничений)
//
// Возвращает:
//   - []Commit: массив коммитов
//   - error: ошибка получения коммитов или nil при успехе
func (g *API) GetCommits(ctx context.Context, branch string, limit int) ([]Commit, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/commits?sha=%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, branch)
	if limit > 0 {
		urlString = fmt.Sprintf("%s&limit=%d", urlString, limit)
	}

	statusCode, body, err := g.sendReq(ctx, urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %w", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении коммитов: статус %d", statusCode)
	}

	var commits []Commit
	err = json.Unmarshal([]byte(body), &commits)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %w", err)
	}

	return commits, nil
}

// GetFirstCommitOfBranch получает первый коммит ветки.
// Возвращает первый коммит в указанной ветке, который не принадлежит базовой ветке.
// Параметры:
//   - branch: имя ветки для получения первого коммита
//   - _baseBranch: имя базовой ветки для сравнения (не используется)
//
// Возвращает:
//   - *Commit: указатель на первый коммит ветки
//   - error: ошибка получения коммита или nil при успехе
func (g *API) GetFirstCommitOfBranch(ctx context.Context, branch string, _ string) (*Commit, error) {
	// Получаем все коммиты в ветке в обратном порядке (от старых к новым)
	// Это может быть неэффективно для больших историй, но для наших целей подойдет
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/commits?sha=%s&stat=false&verification=false&files=false", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, branch)

	statusCode, body, err := g.sendReq(ctx, urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %w", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении коммитов: статус %d", statusCode)
	}

	var commits []Commit
	err = json.Unmarshal([]byte(body), &commits)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %w", err)
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("коммиты не найдены в ветке %s", branch)
	}

	// Возвращаем последний коммит из списка (он будет первым в истории ветки)
	// Это упрощенный подход. В реальном сценарии может потребоваться более сложная логика
	// для определения "первого" коммита ветки относительно базовой ветки.
	return &commits[len(commits)-1], nil
}

// GetCommitsBetween получает список коммитов между двумя коммитами.
// Возвращает массив коммитов между указанными SHA.
// Параметры:
//   - baseCommitSHA: SHA базового коммита
//   - headCommitSHA: SHA конечного коммита
//
// Возвращает:
//   - []Commit: массив коммитов между указанными коммитами
//   - error: ошибка получения коммитов или nil при успехе
func (g *API) GetCommitsBetween(ctx context.Context, baseCommitSHA, headCommitSHA string) ([]Commit, error) {
	// Gitea API не предоставляет прямого метода для получения коммитов между двумя SHA.
	// Мы можем попробовать использовать логику вида `git log base..head`, но через API это сложно.
	// Вместо этого мы получим коммиты от head и будем идти вниз по истории до base.
	// Это упрощенная реализация.

	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/commits?sha=%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, headCommitSHA)

	statusCode, body, err := g.sendReq(ctx, urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %w", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении коммитов: статус %d", statusCode)
	}

	var commits []Commit
	err = json.Unmarshal([]byte(body), &commits)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %w", err)
	}

	// Фильтруем коммиты до baseCommitSHA
	result := make([]Commit, 0, len(commits))
	foundBase := false
	for _, commit := range commits {
		if commit.SHA == baseCommitSHA {
			foundBase = true
			break // Не включаем base коммит
		}
		result = append(result, commit)
	}

	if !foundBase {
		// Если base коммит не найден, это может означать, что он не в истории head
		// или что история слишком длинная и не полностью загружена.
		// Возвращаем все найденные коммиты.
		// В реальном сценарии здесь может потребоваться более сложная логика.
		return commits, nil
	}

	return result, nil
}

// GetCommitFiles получает список файлов, измененных в коммите.
// Возвращает информацию о всех файлах, которые были добавлены,
// изменены или удалены в указанном коммите.
// Параметры:
//   - commitSHA: SHA хеш коммита для анализа
//
// Возвращает:
//   - []CommitFile: список файлов с информацией об изменениях
//   - error: ошибка получения файлов или nil при успехе
func (g *API) GetCommitFiles(ctx context.Context, commitSHA string) ([]CommitFile, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/git/commits/%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, commitSHA)

	statusCode, body, err := g.sendReq(ctx, urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %w", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении информации о коммите: статус %d", statusCode)
	}

	var commitData struct {
		Files []CommitFile `json:"files"`
	}
	err = json.Unmarshal([]byte(body), &commitData)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %w", err)
	}

	return commitData.Files, nil
}

// GetBranchCommitRange получает первый и последний коммит в ветке.
// Для веток кроме main получает первый и последний коммит согласно логике ветвления.
// Для ветки main:
//   - для первого коммита: если есть коммит с тегом "sq-start", то берет его, иначе первый коммит
//   - для последнего коммита: берет последний коммит
//
// Коммиты располагаются в строгом порядке от старого к новому.
//
// Параметры:
//   - branch: имя ветки для анализа
//
// Возвращает:
//   - *BranchCommitRange: структура с первым и последним коммитом
//   - error: ошибка получения коммитов или nil при успехе
func (g *API) GetBranchCommitRange(ctx context.Context, branch string) (*BranchCommitRange, error) {
	if branch == branchMain || branch == "master" {
		return g.getMainBranchCommitRange(ctx, branch)
	}
	return g.getFeatureBranchCommitRange(ctx, branch)
}

// getMainBranchCommitRange получает диапазон коммитов для главной ветки.
// Для первого коммита ищет коммит с тегом "sq-start", если не найден - берет первый коммит.
// Для последнего коммита берет последний коммит в ветке.
func (g *API) getMainBranchCommitRange(ctx context.Context, branch string) (*BranchCommitRange, error) {
	// Получаем последний коммит
	lastCommit, err := g.GetLatestCommit(ctx, branch)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения последнего коммита: %w", err)
	}

	// Ищем коммит с тегом "sq-start"
	firstCommit, err := g.findCommitWithTag(ctx, branch, "sq-start")
	if err != nil {
		// Если коммит с тегом не найден, берем первый коммит в истории
		firstCommit, err = g.getFirstCommitInHistory(ctx, branch)
		if err != nil {
			return nil, fmt.Errorf("ошибка получения первого коммита: %w", err)
		}
	}

	return &BranchCommitRange{
		FirstCommit: firstCommit,
		LastCommit:  lastCommit,
	}, nil
}

// getFeatureBranchCommitRange получает диапазон коммитов для feature ветки.
// Использует compare API для определения первого коммита (merge base) и последнего коммита.
func (g *API) getFeatureBranchCommitRange(ctx context.Context, branch string) (*BranchCommitRange, error) {
	// Получаем последний коммит ветки
	lastCommit, err := g.GetLatestCommit(ctx, branch)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения последнего коммита: %w", err)
	}

	// Сравниваем с базовой веткой для получения merge base
	baseBranch := g.BaseBranch
	if baseBranch == "" {
		baseBranch = branchMain
	}

	compareResult, err := g.compareBranches(ctx, baseBranch, branch)
	if err != nil {
		return nil, fmt.Errorf("ошибка сравнения веток: %w", err)
	}

	// Если общий предок не найден, используем первый коммит базовой ветки
	var firstCommit *Commit
	if compareResult.MergeBaseCommit == nil {
		// Fallback: используем первый коммит базовой ветки
		firstCommit, err = g.getFirstCommitInHistory(ctx, baseBranch)
		if err != nil {
			return nil, fmt.Errorf("не удалось получить первый коммит базовой ветки %s: %w", baseBranch, err)
		}
	} else {
		firstCommit = compareResult.MergeBaseCommit
	}

	return &BranchCommitRange{
		FirstCommit: firstCommit,
		LastCommit:  lastCommit,
	}, nil
}

// findCommitWithTag ищет коммит с указанным тегом в ветке.
// Возвращает коммит, если тег найден, иначе возвращает ошибку.
func (g *API) findCommitWithTag(ctx context.Context, _, tag string) (*Commit, error) {
	// Получаем список тегов
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/tags", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo)

	statusCode, body, err := g.sendReq(ctx, urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса тегов: %w", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении тегов: статус %d", statusCode)
	}

	var tags []struct {
		Name   string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}

	err = json.Unmarshal([]byte(body), &tags)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON тегов: %w", err)
	}

	// Ищем тег "sq-start"
	for _, t := range tags {
		if t.Name == tag {
			// Получаем полную информацию о коммите
			return g.getCommitBySHA(ctx, t.Commit.SHA)
		}
	}

	return nil, fmt.Errorf("тег %s не найден", tag)
}

// getFirstCommitInHistory получает самый первый коммит в истории ветки.
func (g *API) getFirstCommitInHistory(ctx context.Context, branch string) (*Commit, error) {
	// Получаем все коммиты в ветке
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/commits?sha=%s&stat=false&verification=false&files=false", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, branch)

	statusCode, body, err := g.sendReq(ctx, urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %w", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении коммитов: статус %d", statusCode)
	}

	var commits []Commit
	err = json.Unmarshal([]byte(body), &commits)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON: %w", err)
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("коммиты не найдены в ветке %s", branch)
	}

	// Возвращаем последний коммит из списка (самый старый в истории)
	return &commits[len(commits)-1], nil
}

// compareBranches сравнивает две ветки и возвращает результат сравнения.
// Использует Gitea API для получения общего предка и списка коммитов.
// Гарантирует, что MergeBaseCommit будет найден и возвращен.
func (g *API) compareBranches(ctx context.Context, base, head string) (*CompareResult, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/compare/%s...%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, base, head)

	statusCode, body, err := g.sendReq(ctx, urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса сравнения: %w", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при сравнении веток: статус %d", statusCode)
	}

	var result CompareResult
	err = json.Unmarshal([]byte(body), &result)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON результата сравнения: %w", err)
	}

	// Если общий предок не найден через API сравнения, попробуем найти его вручную
	if result.MergeBaseCommit == nil {
		mergeBase, err := g.findMergeBase(ctx, base, head)
		if err != nil {
			return nil, fmt.Errorf("не удалось найти общего предка веток %s и %s: %w", base, head, err)
		}
		result.MergeBaseCommit = mergeBase
	}

	return &result, nil
}

// findMergeBase находит коммит в базовой ветке, предшествующий первому коммиту ветки head.
// Определяет состояние базовой ветки до внесения первого изменения в head ветку.
// Возвращает коммит из base ветки, который был актуален до создания head ветки.
func (g *API) findMergeBase(ctx context.Context, base, head string) (*Commit, error) {
	// Получаем историю коммитов целевой ветки (head)
	headCommits, err := g.getAllCommits(ctx, head)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить коммиты ветки head %s: %w", head, err)
	}

	if len(headCommits) == 0 {
		return nil, fmt.Errorf("ветка head %s не содержит коммитов", head)
	}

	// Находим самый первый (самый старый) коммит в head ветке
	// Коммиты в массиве упорядочены от новых к старым, поэтому берем последний элемент
	firstHeadCommit := &headCommits[len(headCommits)-1]

	// Получаем историю коммитов базовой ветки
	baseCommits, err := g.getAllCommits(ctx, base)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить коммиты базовой ветки %s: %w", base, err)
	}

	if len(baseCommits) == 0 {
		return nil, fmt.Errorf("базовая ветка %s не содержит коммитов", base)
	}

	// Ищем позицию первого коммита head ветки в базовой ветке
	var baseCommitIndex = -1
	for i, baseCommit := range baseCommits {
		if baseCommit.SHA == firstHeadCommit.SHA {
			baseCommitIndex = i
			break
		}
	}

	// Если первый коммит head ветки не найден в базовой ветке,
	// возвращаем последний коммит базовой ветки
	if baseCommitIndex == -1 {
		return &baseCommits[0], nil // Первый элемент = самый новый коммит
	}

	// Если первый коммит head ветки - это самый первый коммит в базовой ветке,
	// то предыдущего коммита нет
	if baseCommitIndex == len(baseCommits)-1 {
		return nil, fmt.Errorf("первый коммит ветки head %s является первым коммитом в базовой ветке %s", head, base)
	}

	// Возвращаем предыдущий коммит в базовой ветке
	return &baseCommits[baseCommitIndex+1], nil
}

// getAllCommits получает все коммиты ветки.
func (g *API) getAllCommits(ctx context.Context, branch string) ([]Commit, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/commits?sha=%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, branch)

	statusCode, body, err := g.sendReq(ctx, urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса коммитов: %w", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении коммитов ветки %s: статус %d", branch, statusCode)
	}

	var commits []Commit
	err = json.Unmarshal([]byte(body), &commits)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON коммитов: %w", err)
	}

	return commits, nil
}

// getCommitBySHA получает информацию о коммите по его SHA.
func (g *API) getCommitBySHA(ctx context.Context, sha string) (*Commit, error) {
	urlString := fmt.Sprintf("%s/api/%s/repos/%s/%s/commits?sha=%s", g.GiteaURL, constants.APIVersion, g.Owner, g.Repo, sha)

	statusCode, body, err := g.sendReq(ctx, urlString, "", "GET")
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса коммита: %w", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка при получении коммита: статус %d", statusCode)
	}

	var commits []Commit
	err = json.Unmarshal([]byte(body), &commits)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе JSON коммита: %w", err)
	}

	if len(commits) == 0 {
		return nil, fmt.Errorf("коммит с SHA %s не найден", sha)
	}

	// Ищем коммит с точным SHA в массиве
	for _, commit := range commits {
		if commit.SHA == sha {
			return &commit, nil
		}
	}

	return nil, fmt.Errorf("коммит с SHA %s не найден", sha)
}
