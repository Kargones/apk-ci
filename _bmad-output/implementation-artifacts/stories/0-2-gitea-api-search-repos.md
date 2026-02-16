# Story 0.2: Расширение Gitea API для поиска репозиториев

Status: done

## Story

As a система,
I want искать репозитории по всей организации,
so that я могу найти все потенциальные подписчики.

## Story Context

```xml
<story-context>
  <epic id="0" name="Extension Publish" priority="FIRST">
    <goal>Реализовать команду extension-publish для автоматического распространения расширений 1C</goal>
  </epic>

  <story id="0.2" name="Gitea API Search Repos" size="S" risk="Low" priority="P0">
    <dependency type="none">Независимая история, может выполняться параллельно с 0.1</dependency>
    <blocks stories="0.3">Необходим для поиска подписанных репозиториев</blocks>
  </story>

  <codebase-context>
    <file path="internal/entity/gitea/gitea.go" role="primary">
      <existing-methods>
        - GetRepositoryContents() - получение содержимого репозитория
        - GetBranches() - получение списка веток
        - sendReq() - базовый HTTP-клиент
      </existing-methods>
      <existing-types>
        - Repo struct {DefaultBranch string}
      </existing-types>
    </file>
  </codebase-context>

  <gitea-api-reference>
    <endpoint method="GET" path="/repos/search">
      <description>Поиск репозиториев</description>
      <query-params>
        - q: string (поисковый запрос)
        - uid: int64 (ID владельца)
        - owner: string (имя владельца/организации) - DEPRECATED, use uid
        - page: int (номер страницы, начиная с 1)
        - limit: int (количество на странице, max 50)
        - sort: string (alpha, created, updated, size, id)
        - order: string (asc, desc)
      </query-params>
      <response>
        - ok: bool
        - data: []Repository
      </response>
    </endpoint>
    <endpoint method="GET" path="/orgs/{org}/repos">
      <description>Получить все репозитории организации (рекомендуется)</description>
      <query-params>
        - page: int
        - limit: int
      </query-params>
    </endpoint>
  </gitea-api-reference>
</story-context>
```

## Acceptance Criteria

1. **AC1: Расширение структуры Repository**
   - [x] Расширена структура `Repo` или создана новая `Repository` с полями:
     - `ID int64` (json: "id")
     - `Name string` (json: "name")
     - `FullName string` (json: "full_name") - формат "owner/repo"
     - `Owner RepositoryOwner` (json: "owner")
     - `DefaultBranch string` (json: "default_branch")
     - `Private bool` (json: "private")
     - `Fork bool` (json: "fork")
   - [x] Создана структура `RepositoryOwner` с полями:
     - `ID int64`
     - `Login string`
     - `Type string` (user/organization)

2. **AC2: Метод SearchOrgRepos(orgName string)**
   - [x] Сигнатура: `func (g *API) SearchOrgRepos(orgName string) ([]Repository, error)`
   - [x] URL: `{GiteaURL}/api/v1/orgs/{orgName}/repos`
   - [x] Автоматическая обработка пагинации (получить ВСЕ репозитории)
   - [x] Лимит на страницу: 50 (максимум Gitea API)
   - [x] Возвращает пустой slice если организация не найдена

3. **AC3: Обработка пагинации**
   - [x] Цикл запросов с увеличением page пока не получен пустой ответ
   - [x] Защита от бесконечного цикла (max 100 страниц = 5000 репозиториев)
   - [x] Задержка между запросами не требуется (локальный Gitea)

4. **AC4: Обработка ошибок**
   - [x] HTTP 404 для несуществующей организации -> пустой slice, nil error
   - [x] Сетевые ошибки -> error с контекстом
   - [x] Формат ошибок соответствует паттерну проекта

## Tasks / Subtasks

- [x] **Task 1: Создание/расширение структур** (AC: #1)
  - [x] 1.1 Создать структуру `Repository` (не путать с существующей `Repo`)
  - [x] 1.2 Создать структуру `RepositoryOwner`
  - [x] 1.3 Добавить документирующие комментарии

- [x] **Task 2: Реализация SearchOrgRepos()** (AC: #2, #3, #4)
  - [x] 2.1 Создать метод с базовым запросом
  - [x] 2.2 Реализовать цикл пагинации
  - [x] 2.3 Добавить защиту от бесконечного цикла
  - [x] 2.4 Обработать edge cases (пустая организация, 404)

- [x] **Task 3: Обновление интерфейса** (AC: #1-2)
  - [x] 3.1 Добавить метод в интерфейс `GiteaAPI` в `interfaces.go`

## Dev Notes

### Архитектурные паттерны проекта

**Пагинация:**
```go
const maxPages = 100
const pageLimit = 50

var allRepos []Repository
for page := 1; page <= maxPages; page++ {
    urlString := fmt.Sprintf("%s/api/%s/orgs/%s/repos?page=%d&limit=%d",
        g.GiteaURL, constants.APIVersion, orgName, page, pageLimit)

    statusCode, body, err := g.sendReq(urlString, "", "GET")
    if err != nil {
        return nil, fmt.Errorf("ошибка при запросе репозиториев: %v", err)
    }

    if statusCode == http.StatusNotFound {
        return []Repository{}, nil // Организация не найдена
    }

    if statusCode != http.StatusOK {
        return nil, fmt.Errorf("ошибка при получении репозиториев: статус %d", statusCode)
    }

    var repos []Repository
    if err := json.Unmarshal([]byte(body), &repos); err != nil {
        return nil, fmt.Errorf("ошибка при разборе JSON: %v", err)
    }

    if len(repos) == 0 {
        break // Достигли конца
    }

    allRepos = append(allRepos, repos...)
}
return allRepos, nil
```

### Почему orgs/{org}/repos вместо repos/search

1. `repos/search` требует дополнительных параметров и менее предсказуем
2. `/orgs/{org}/repos` - прямой endpoint для организаций
3. Меньше сложности с фильтрацией результатов

### Структура файла gitea.go

- Новые структуры добавить после строки 97 (после `ProjectAnalysis`)
- Метод добавить в раздел методов API (~после строки 840)

### Пример структур

```go
// Repository представляет полную информацию о репозитории в Gitea.
type Repository struct {
    ID            int64           `json:"id"`
    Name          string          `json:"name"`
    FullName      string          `json:"full_name"`
    Owner         RepositoryOwner `json:"owner"`
    DefaultBranch string          `json:"default_branch"`
    Private       bool            `json:"private"`
    Fork          bool            `json:"fork"`
}

// RepositoryOwner представляет владельца репозитория.
type RepositoryOwner struct {
    ID    int64  `json:"id"`
    Login string `json:"login"`
    Type  string `json:"type"` // "User" или "Organization"
}
```

### Project Structure Notes

- Размещение: `internal/entity/gitea/gitea.go`
- Существующая структура `Repo` (строка 27-30) минимальна, нужна новая `Repository`
- Тесты: новый файл `gitea_search_test.go` или добавить в существующие

### References

- [Source: internal/entity/gitea/gitea.go#GetBranches] - пример работы со списками
- [Source: internal/entity/gitea/gitea.go#GetRepositoryContents] - пример запросов к репо
- [Gitea API Docs: /orgs/{org}/repos](https://docs.gitea.com/api/1.20/#tag/organization/operation/orgListRepos)

## Definition of Done

- [x] Структуры `Repository` и `RepositoryOwner` созданы с документацией
- [x] Метод `SearchOrgRepos()` реализован с полной пагинацией
- [x] Пагинация корректно обрабатывает большое количество репозиториев
- [x] Edge cases обработаны (пустая организация, 404)
- [x] Unit-тесты с мокированием пагинации
- [x] Интерфейс в `interfaces.go` обновлен
- [x] Код проходит `go vet` (golangci-lint не установлен)
- [x] Код проходит `go test` (все unit-тесты)

## Dev Agent Record

### Context Reference

- Epic: bdocs/epics/epic-0-extension-publish.md
- Story Context: Embedded XML above

### Agent Model Used

Claude Opus 4.5

### Completion Notes List

1. **Структуры Repository и RepositoryOwner** — добавлены в `gitea.go` после `ProjectAnalysis` (строки 100-118) с полной документацией на русском языке
2. **Метод SearchOrgRepos()** — реализован в `gitea.go` (строки 1820-1871) с автоматической пагинацией, защитой от бесконечного цикла (max 100 страниц) и корректной обработкой 404
3. **Интерфейс APIInterface** — обновлён в `interfaces.go`, добавлен метод `SearchOrgRepos(orgName string) ([]Repository, error)`
4. **Unit-тесты** — создан новый файл `gitea_search_test.go` с 6 тестами: структура Repository, успешный поиск, 404, пагинация, защита от бесконечного цикла, сетевые ошибки
5. **Mock-объекты обновлены** — добавлены методы `GetLatestRelease`, `GetReleaseByTag`, `SearchOrgRepos` в mock-объекты в `gitea_service_test.go` и `sonarqube_init_test.go`

### File List

**Новые файлы:**
- `internal/entity/gitea/gitea_search_test.go` — unit-тесты для SearchOrgRepos

**Изменённые файлы:**
- `internal/entity/gitea/gitea.go` — структуры Repository, RepositoryOwner и метод SearchOrgRepos
- `internal/entity/gitea/interfaces.go` — метод SearchOrgRepos в APIInterface
- `internal/service/gitea_service_test.go` — mock-методы для интерфейса
- `internal/app/sonarqube_init_test.go` — mock-методы для интерфейса
- `bdocs/sprint-artifacts/sprint-status.yaml` — статус истории in-progress → review

### Senior Developer Review (AI)

**Reviewer:** Amelia (Dev Agent) | **Date:** 2025-12-08 | **Outcome:** ✅ APPROVED

**Findings Fixed:**
- M1: Добавлен тест `TestSearchOrgRepos_InvalidJSON` для обработки невалидного JSON
- M2: Добавлен тест `TestSearchOrgRepos_ServerError` (tabular test для HTTP 500, 403, 502, 503)
- M3: Константы `maxPages` и `pageLimit` вынесены на уровень пакета как `SearchOrgReposMaxPages` и `SearchOrgReposPageLimit` с документацией

**Remaining (LOW):**
- L1: Несогласованность стиля mock-объектов (nil vs empty slice) — косметическое
- L2: Тест NetworkError медленный (3.7s) — не критично

**Validation Summary:**
- Все AC реализованы: ✅
- Все Tasks [x] завершены: ✅
- Unit-тесты: 8 тестов PASS (было 6, добавлено 2)
- go vet: ✅ без ошибок

### Change Log

- **2025-12-08**: Code Review — исправлены 3 MEDIUM issues (добавлены тесты, вынесены константы с документацией)
- **2025-12-08**: Реализована история 0-2-gitea-api-search-repos — добавлены структуры Repository/RepositoryOwner и метод SearchOrgRepos с полной пагинацией и unit-тестами
