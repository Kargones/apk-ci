# Story 0.3: Поиск подписанных репозиториев

Status: done

## Story

As a система,
I want найти все репозитории, подписанные на расширение,
so that я знаю куда публиковать обновление.

## Story Context

```xml
<story-context>
  <epic id="0" name="Extension Publish" priority="FIRST">
    <goal>Реализовать команду extension-publish для автоматического распространения расширений 1C</goal>
    <subscription-mechanism>
      Подписка через ветку с именем: {Org}_{Repo}_{ExtDir}
      Пример: MyOrg_MyProject_cfe/myext
      Ветка в репозитории-ИСТОЧНИКЕ указывает на репозиторий-ПОДПИСЧИК
    </subscription-mechanism>
  </epic>

  <story id="0.3" name="Find Subscribed Repos" size="M" risk="Medium" priority="P0">
    <dependency type="required" story="0.2">Требует SearchOrgRepos() для получения списка репозиториев</dependency>
    <blocks stories="0.4, 0.6">Ключевая логика определения целей публикации</blocks>
  </story>

  <codebase-context>
    <file path="internal/app/extension_publish.go" role="primary" status="new">
      <note>Новый файл для логики extension-publish</note>
    </file>
    <file path="internal/entity/gitea/gitea.go" role="dependency">
      <methods-used>
        - SearchOrgRepos() - из Story 0.2
        - GetBranches() - существующий метод
      </methods-used>
    </file>
    <file path="internal/config/config.go" role="dependency">
      <note>Конфигурация с GiteaURL, Owner, AccessToken</note>
    </file>
  </codebase-context>

  <business-rules>
    <rule id="BR1">
      Паттерн ветки-подписки: {Org}_{Repo}_{ExtDir}
      - Org: имя организации целевого репозитория
      - Repo: имя целевого репозитория
      - ExtDir: путь к каталогу расширения (/ заменяется на _)
      Пример: APKHolding_ERP_cfe_CommonExt -> org=APKHolding, repo=ERP, dir=cfe/CommonExt
    </rule>
    <rule id="BR2">
      Ветка-подписка создается в репозитории ИСТОЧНИКЕ расширения
      и указывает на репозиторий-ПОДПИСЧИК
    </rule>
    <rule id="BR3">
      Целевой каталог в подписчике соответствует ExtDir из имени ветки
    </rule>
  </business-rules>
</story-context>
```

## Acceptance Criteria

1. **AC1: Структура SubscribedRepo**
   - [x] Создана структура `SubscribedRepo` с полями:
     - `Organization string` - организация целевого репозитория
     - `Repository string` - имя целевого репозитория
     - `TargetBranch string` - ветка по умолчанию целевого репозитория
     - `TargetDirectory string` - целевой каталог для расширения
     - `SubscriptionBranch string` - имя ветки-подписки (для отладки)

2. **AC2: Функция ParseSubscriptionBranch(branchName string)**
   - [x] Сигнатура: `func ParseSubscriptionBranch(branchName string) (*SubscribedRepo, error)`
   - [x] Парсит имя ветки по паттерну `{Org}_{Repo}_{ExtDir}`
   - [x] ExtDir может содержать `_` которые заменяются на `/`
   - [x] Возвращает nil, error для некорректных имен веток
   - [x] Минимум 3 части разделенных `_`

3. **AC3: Функция FindSubscribedRepos()**
   - [x] Сигнатура: `func FindSubscribedRepos(api *gitea.API, sourceOrg, sourceRepo string) ([]SubscribedRepo, error)`
   - [x] Получает список веток исходного репозитория через `GetBranches()`
   - [x] Фильтрует ветки по паттерну подписки
   - [x] Для каждой ветки-подписки проверяет существование целевого репозитория
   - [x] Возвращает список валидных подписчиков

4. **AC4: Валидация целевых репозиториев**
   - [x] Для каждого найденного подписчика проверяется:
     - Репозиторий существует
     - Репозиторий доступен (не private без доступа)
   - [x] Недоступные репозитории пропускаются с warning-логом

5. **AC5: Обработка ошибок**
   - [x] Ошибки парсинга веток логируются, но не прерывают процесс
   - [x] Сетевые ошибки при проверке репозиториев прерывают процесс
   - [x] Возвращается информативная ошибка с контекстом

## Tasks / Subtasks

- [x] **Task 1: Создание файла и структур** (AC: #1)
  - [x] 1.1 Создать файл `internal/app/extension_publish.go`
  - [x] 1.2 Добавить структуру `SubscribedRepo`
  - [x] 1.3 Добавить документирующие комментарии

- [x] **Task 2: Реализация ParseSubscriptionBranch()** (AC: #2)
  - [x] 2.1 Реализовать парсинг имени ветки
  - [x] 2.2 Обработать edge cases (мало частей, пустые части)
  - [x] 2.3 Реализовать замену `_` на `/` в ExtDir
  - [x] 2.4 Написать unit-тесты для парсера

- [x] **Task 3: Реализация FindSubscribedRepos()** (AC: #3, #4, #5)
  - [x] 3.1 Получить список веток репозитория
  - [x] 3.2 Фильтровать ветки по паттерну подписки
  - [x] 3.3 Парсить каждую ветку-подписку
  - [x] 3.4 Валидировать существование целевых репозиториев
  - [x] 3.5 Собрать результаты и обработать ошибки

## Dev Notes

### Архитектурные паттерны проекта

**Структура app-слоя:**
```go
// internal/app/extension_publish.go
package app

import (
    "fmt"
    "log/slog"
    "strings"

    "github.com/Kargones/apk-ci/internal/entity/gitea"
)
```

**Паттерн парсинга ветки:**
```go
// ParseSubscriptionBranch парсит имя ветки-подписки.
// Формат: {Org}_{Repo}_{ExtDir}
// ExtDir может содержать вложенные пути, где / заменен на _
// Пример: "APKHolding_ERP_cfe_CommonExt" -> Org=APKHolding, Repo=ERP, Dir=cfe/CommonExt
func ParseSubscriptionBranch(branchName string) (*SubscribedRepo, error) {
    parts := strings.SplitN(branchName, "_", 3)
    if len(parts) < 3 {
        return nil, fmt.Errorf("некорректный формат ветки подписки: %s (ожидается {Org}_{Repo}_{ExtDir})", branchName)
    }

    org := parts[0]
    repo := parts[1]
    extDir := strings.ReplaceAll(parts[2], "_", "/")

    if org == "" || repo == "" || extDir == "" {
        return nil, fmt.Errorf("пустые компоненты в ветке подписки: %s", branchName)
    }

    return &SubscribedRepo{
        Organization:       org,
        Repository:         repo,
        TargetDirectory:    extDir,
        SubscriptionBranch: branchName,
    }, nil
}
```

### Логика определения веток-подписок

Ветка считается подпиской, если:
1. Содержит минимум 2 символа `_`
2. Не является служебной веткой (main, master, develop, xml, edt)
3. Первые две части (до второго `_`) не пустые

### Исключаемые ветки

```go
var excludedBranches = map[string]bool{
    "main":    true,
    "master":  true,
    "develop": true,
    "xml":     true,
    "edt":     true,
}
```

### Пример структуры

```go
// SubscribedRepo представляет репозиторий, подписанный на обновления расширения.
type SubscribedRepo struct {
    Organization       string // Организация целевого репозитория
    Repository         string // Имя целевого репозитория
    TargetBranch       string // Ветка по умолчанию (main/master)
    TargetDirectory    string // Каталог для размещения расширения
    SubscriptionBranch string // Имя ветки-подписки в источнике
}
```

### Project Structure Notes

- Новый файл: `internal/app/extension_publish.go`
- Тесты: `internal/app/extension_publish_test.go`
- Зависимости: `internal/entity/gitea/gitea.go`

### References

- [Source: internal/entity/gitea/gitea.go#GetBranches] - метод получения веток
- [Source: bdocs/epics/epic-0-extension-publish.md#Story-0.3] - бизнес-требования
- [Source: internal/app/app.go] - примеры структуры app-слоя

## Definition of Done

- [x] Файл `internal/app/extension_publish.go` создан
- [x] Структура `SubscribedRepo` реализована с документацией
- [x] Функция `ParseSubscriptionBranch()` корректно парсит все форматы веток
- [x] Функция `FindSubscribedRepos()` находит и валидирует подписчиков
- [x] Edge cases обработаны (некорректные ветки, несуществующие репозитории)
- [x] Unit-тесты для ParseSubscriptionBranch (>90% coverage)
- [x] Unit-тесты для FindSubscribedRepos с мокированием API
- [x] Код проходит `go vet` (golangci-lint требует v2, конфиг для v1)
- [x] Код проходит `make test` (internal/app)

## Dev Agent Record

### Context Reference

- Epic: bdocs/epics/epic-0-extension-publish.md
- Story Context: Embedded XML above

### Agent Model Used

Claude Opus 4.5

### Completion Notes List

- Создан файл `internal/app/extension_publish.go` с полной реализацией
- Структура `SubscribedRepo` содержит все необходимые поля с документацией
- `ParseSubscriptionBranch()` реализует парсинг по формату `{Org}_{Repo}_{ExtDir}` с заменой `_` на `/`
- `IsSubscriptionBranch()` — вспомогательная функция для фильтрации служебных веток (использует ParseSubscriptionBranch)
- `FindSubscribedRepos()` получает ветки, фильтрует подписки, валидирует через `SearchOrgRepos()`
- Исключаемые ветки: main, master, develop, xml, edt
- Сетевые ошибки прерывают процесс, ошибки парсинга — логируются с warning
- 10 unit-тестов для парсера (valid/invalid branches, edge cases с двойным подчёркиванием)
- 5 интеграционных тестов для `FindSubscribedRepos` с httptest mock-сервером
- Все 15 тестов проходят успешно
- `go vet` прошёл без замечаний

**Code Review Fixes (2025-12-08):**
- Исправлен баг: sourceOrg не использовался в FindSubscribedRepos — теперь используется api.Owner
- Добавлено кэширование SearchOrgRepos для избежания N+1 запросов
- Рефакторинг: IsSubscriptionBranch теперь использует ParseSubscriptionBranch (единый источник истины)
- Добавлена валидация: extDir не может начинаться со `/` (защита от двойного `_`)
- Унифицированы ключи логов (английский формат)

### File List

**Новые файлы:**
- `internal/app/extension_publish.go` — основная реализация
- `internal/app/extension_publish_test.go` — unit и интеграционные тесты
- `internal/entity/gitea/gitea_search_test.go` — тесты для SearchOrgRepos

**Изменённые файлы:**
- `internal/entity/gitea/gitea.go` — добавлены константы пагинации
- `bdocs/sprint-artifacts/sprint-status.yaml` — статус истории
- `bdocs/stories/0-3-find-subscribed-repos.md` — обновление статуса
- `bdocs/stories/0-2-gitea-api-search-repos.md` — обновление статуса

## Change Log

- 2025-12-08: Реализована Story 0.3 — поиск подписанных репозиториев через ветки-подписки
- 2025-12-08: Code Review fixes — исправлен баг sourceOrg, добавлено кэширование, рефакторинг IsSubscriptionBranch
