# Story 0.5: Создание PR с полной информацией

Status: done

## Story

As a система,
I want создавать PR с информацией о релизе,
so that maintainer понимает что изменилось.

## Story Context

```xml
<story-context>
  <epic id="0" name="Extension Publish" priority="FIRST">
    <goal>Реализовать команду extension-publish для автоматического распространения расширений 1C</goal>
  </epic>

  <story id="0.5" name="Create PR with Info" size="S" risk="Low" priority="P0">
    <dependency type="required" story="0.1">Требует Release структуру для release notes</dependency>
    <dependency type="required" story="0.4">Требует SyncResult для создания PR после синхронизации</dependency>
    <blocks stories="0.6">Завершающий этап процесса публикации</blocks>
  </story>

  <codebase-context>
    <file path="internal/entity/gitea/gitea.go" role="primary">
      <existing-method name="CreatePR" line="553">
        <current-signature>func (g *API) CreatePR(head string) (PR, error)</current-signature>
        <limitation>Фиксированные title и body, не подходит для extension-publish</limitation>
      </existing-method>
    </file>
    <file path="internal/app/extension_publish.go" role="consumer">
      <note>Использует расширенный CreatePR</note>
    </file>
  </codebase-context>

  <business-rules>
    <rule id="BR1">
      PR Title: "Update {extName} to {version}"
      Пример: "Update CommonExt to v1.2.3"
    </rule>
    <rule id="BR2">
      PR Body содержит:
      - Версию расширения
      - Release notes из релиза
      - Ссылку на релиз в источнике
      - Список измененных файлов (опционально)
    </rule>
    <rule id="BR3">
      Автоматический assignee (если настроен в конфигурации)
    </rule>
  </business-rules>
</story-context>
```

## Acceptance Criteria

1. **AC1: Структура CreatePROptions**
   - [x] Создана структура `CreatePROptions` с полями:
     - `Title string` - заголовок PR
     - `Body string` - описание PR (markdown)
     - `Head string` - исходная ветка
     - `Base string` - целевая ветка
     - `Assignees []string` - назначенные пользователи (опционально)
     - `Labels []int64` - ID меток (опционально)

2. **AC2: Метод CreatePRWithOptions()**
   - [x] Сигнатура: `func (g *API) CreatePRWithOptions(opts CreatePROptions) (*PRResponse, error)`
   - [x] URL: `{GiteaURL}/api/v1/repos/{Owner}/{Repo}/pulls`
   - [x] Отправляет JSON с полными параметрами
   - [x] Возвращает созданный PR с номером и URL
   - [x] Обрабатывает ошибку "PR already exists"

3. **AC3: Функция BuildExtensionPRBody()**
   - [x] Сигнатура: `func BuildExtensionPRBody(release *gitea.Release, sourceRepo, extName, releaseURL string) string`
   - [x] Формирует markdown body с:
     - Заголовком "Extension Update"
     - Версией (tag_name)
     - Release notes (body из релиза)
     - Ссылкой на релиз
   - [x] Обрабатывает пустые release notes

4. **AC4: Функция CreateExtensionPR()**
   - [x] Сигнатура: `func CreateExtensionPR(api *gitea.API, syncResult *SyncResult, release *gitea.Release, extName, sourceRepo, releaseURL string) (*gitea.PRResponse, error)`
   - [x] Формирует title по шаблону
   - [x] Вызывает BuildExtensionPRBody() для body
   - [x] Создает PR через CreatePRWithOptions()
   - [x] Логирует созданный PR

5. **AC5: Обработка ошибок**
   - [x] PR уже существует -> вернуть существующий PR
   - [x] Ветка не существует -> информативная ошибка
   - [ ] Конфликты -> создать PR, пометить в body (не реализовано - выходит за scope)

## Tasks / Subtasks

- [x] **Task 1: Создание структуры CreatePROptions** (AC: #1)
  - [x] 1.1 Добавить структуру в `gitea.go`
  - [x] 1.2 Добавить документирующие комментарии

- [x] **Task 2: Реализация CreatePRWithOptions()** (AC: #2)
  - [x] 2.1 Реализовать метод в `gitea.go`
  - [x] 2.2 Сериализовать options в JSON
  - [x] 2.3 Обработать ответ API
  - [x] 2.4 Обработать случай существующего PR

- [x] **Task 3: Реализация BuildExtensionPRBody()** (AC: #3)
  - [x] 3.1 Создать функцию в `extension_publish.go`
  - [x] 3.2 Сформировать markdown шаблон
  - [x] 3.3 Обработать пустые release notes

- [x] **Task 4: Реализация CreateExtensionPR()** (AC: #4, #5)
  - [x] 4.1 Сформировать title
  - [x] 4.2 Вызвать BuildExtensionPRBody()
  - [x] 4.3 Создать PR через API
  - [x] 4.4 Обработать ошибки

## Dev Notes

### Архитектурные паттерны проекта

**Существующий метод CreatePR():**
```go
// gitea.go:553 - текущая реализация
func (g *API) CreatePR(head string) (PR, error) {
    reqBody := fmt.Sprintf(`{
        "base": "%s",
        "body": "Test conflict",
        "head": "%s",
        "title": "Test merge %s to %s"
    }`, g.NewBranch, head, head, g.NewBranch)
    // ...
}
```

**Новая структура CreatePROptions:**
```go
// CreatePROptions содержит параметры для создания Pull Request.
type CreatePROptions struct {
    Title     string   `json:"title"`
    Body      string   `json:"body"`
    Head      string   `json:"head"`
    Base      string   `json:"base"`
    Assignees []string `json:"assignees,omitempty"`
    Labels    []int64  `json:"labels,omitempty"` // ID меток
}
```

**Шаблон PR Body:**
```markdown
## Extension Update

**Extension:** {extName}
**Version:** {version}
**Source:** [{sourceOrg}/{sourceRepo}]({releaseURL})

### Release Notes

{release.Body}

---
*This PR was automatically created by benadis-runner extension-publish*
```

### Gitea API для создания PR

```json
POST /api/v1/repos/{owner}/{repo}/pulls
{
    "title": "string",
    "body": "string",
    "head": "string",       // исходная ветка
    "base": "string",       // целевая ветка
    "assignees": ["user1"], // опционально
    "labels": [1, 2]        // ID меток, опционально
}
```

**Ответ:**
```json
{
    "id": 123,
    "number": 45,
    "url": "https://...",
    "html_url": "https://...",
    "state": "open",
    "title": "...",
    "body": "..."
}
```

### Project Structure Notes

- Структура `CreatePROptions`: `internal/entity/gitea/gitea.go`
- Метод `CreatePRWithOptions()`: `internal/entity/gitea/gitea.go`
- Функции `BuildExtensionPRBody()`, `CreateExtensionPR()`: `internal/app/extension_publish.go`

### Обработка существующего PR

Gitea возвращает HTTP 409 Conflict если PR уже существует:
```go
if statusCode == http.StatusConflict {
    // Попробовать найти существующий PR
    // или вернуть информативную ошибку
}
```

### References

- [Source: internal/entity/gitea/gitea.go#CreatePR] - существующий метод
- [Source: internal/entity/gitea/gitea.go#PRData] - структура ответа
- [Gitea API: Create Pull Request](https://docs.gitea.com/api/1.20/#tag/repository/operation/repoCreatePullRequest)

## Definition of Done

- [x] Структура `CreatePROptions` создана в gitea.go
- [x] Метод `CreatePRWithOptions()` реализован и тестирован
- [x] Функция `BuildExtensionPRBody()` формирует корректный markdown
- [x] Функция `CreateExtensionPR()` создает PR с полной информацией
- [x] Обработаны edge cases (существующий PR, пустые release notes)
- [x] Unit-тесты для всех функций
- [x] Интерфейс в `interfaces.go` обновлен
- [x] Код проходит `make test`

## Dev Agent Record

### Context Reference

- Epic: bdocs/epics/epic-0-extension-publish.md
- Story Context: Embedded XML above

### Agent Model Used

Claude Opus 4.5

### Completion Notes List

1. Реализована структура `CreatePROptions` в gitea.go:220 с полями Title, Body, Head, Base, Assignees, Labels
2. Реализована структура `PRResponse` в gitea.go:242 для ответа API
3. Реализован метод `CreatePRWithOptions()` в gitea.go:687 с обработкой HTTP 409 Conflict
4. Добавлен приватный метод `findExistingPR()` в gitea.go:728 для поиска существующего PR при конфликте
5. Реализована функция `BuildExtensionPRBody()` в extension_publish.go:454 с markdown шаблоном
6. Реализована вспомогательная функция `BuildExtensionPRTitle()` в extension_publish.go:496
7. Реализована функция `CreateExtensionPR()` в extension_publish.go:513 с логированием
8. Обновлен интерфейс `APIInterface` в interfaces.go:46
9. Обновлен MockGiteaAPI в sonarqube_init_test.go:100

### Code Review Fixes (2025-12-08)

10. Добавлен метод `CreatePRWithOptions()` в MockAPI в gitea_service_test.go:151 (исправлена ошибка компиляции)
11. Добавлены тесты для edge cases: `TestBuildExtensionPRTitle_EmptyExtName`, `TestBuildExtensionPRBody_EmptyExtName`

### File List

**Modified files:**
- internal/entity/gitea/gitea.go - добавлены CreatePROptions, PRResponse, CreatePRWithOptions(), findExistingPR()
- internal/entity/gitea/interfaces.go - добавлен метод CreatePRWithOptions в интерфейс
- internal/entity/gitea/gitea_test.go - добавлены тесты для CreatePRWithOptions
- internal/app/extension_publish.go - добавлены BuildExtensionPRBody(), BuildExtensionPRTitle(), CreateExtensionPR()
- internal/app/extension_publish_test.go - добавлены тесты для новых функций, включая edge cases
- internal/app/sonarqube_init_test.go - обновлен MockGiteaAPI
- internal/service/gitea_service_test.go - обновлен MockAPI (добавлен CreatePRWithOptions)
- bdocs/stories/0-5-create-pr-with-info.md - обновлен статус
- bdocs/sprint-artifacts/sprint-status.yaml - обновлен статус
