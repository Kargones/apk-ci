# Story 0.8: Unit и Integration тесты

Status: done

## Story

As a разработчик,
I want иметь тесты для логики публикации,
so that регрессии обнаруживаются автоматически.

## Story Context

```xml
<story-context>
  <epic id="0" name="Extension Publish" priority="FIRST">
    <goal>Реализовать команду extension-publish для автоматического распространения расширений 1C</goal>
    <testing-strategy>
      - Unit-тесты с httptest для мокирования Gitea API
      - Integration-тесты с реальным Gitea (опционально, через build tags)
      - Coverage >80% для extension_publish.go
    </testing-strategy>
  </epic>

  <story id="0.8" name="Unit and Integration Tests" size="M" risk="Low" priority="P1">
    <dependency type="required" story="0.1-0.7">Все предыдущие истории должны быть завершены</dependency>
    <note>Может выполняться параллельно с разработкой, добавляя тесты по мере готовности функций</note>
  </story>

  <codebase-context>
    <file path="internal/app/extension_publish_test.go" role="primary" status="new">
      <note>Основной файл тестов</note>
    </file>
    <file path="internal/entity/gitea/gitea_test.go" role="existing">
      <note>Существующие тесты Gitea API - добавить тесты для новых методов</note>
    </file>
    <file path="internal/entity/gitea/interfaces.go" role="mocking">
      <note>Интерфейсы для мокирования API</note>
    </file>
  </codebase-context>

  <testing-patterns>
    <pattern name="Table-driven tests">Используется во всём проекте</pattern>
    <pattern name="httptest">Для мокирования HTTP API</pattern>
    <pattern name="Build tags">// +build integration для интеграционных тестов</pattern>
  </testing-patterns>
</story-context>
```

## Acceptance Criteria

1. **AC1: Unit-тесты для gitea.go**
   - [x] Тесты для `GetLatestRelease()` - success, 404, network error
   - [x] Тесты для `GetReleaseByTag()` - success, invalid tag, 404
   - [x] Тесты для `SearchOrgRepos()` - empty org, pagination, errors
   - [x] Тесты для `CreatePRWithOptions()` - success, conflict, validation

2. **AC2: Unit-тесты для extension_publish.go**
   - [x] Тесты для `ParseSubscriptionBranch()` - все форматы и edge cases
   - [x] Тесты для `FindSubscribedRepos()` - с мокированным API
   - [x] Тесты для `GetSourceFiles()` - рекурсивный обход
   - [x] Тесты для `SyncExtensionToRepo()` - полный цикл с моками
   - [x] Тесты для `CreateExtensionPR()` - формирование PR

3. **AC3: Unit-тесты для отчётности**
   - [x] Тесты для `PublishReport` методов (SuccessCount, etc.)
   - [x] Тесты для `ReportResults()` - текстовый формат
   - [x] Тесты для JSON-вывода

4. **AC4: Integration тесты (опционально)**
   - [x] Тест с реальным Gitea (под build tag `integration`)
   - [x] Создание тестового репозитория
   - [x] Полный цикл: release -> sync -> PR
   - [x] Очистка после теста

5. **AC5: Покрытие кода**
   - [x] Покрытие `internal/app/extension_publish.go` > 80% (92.9%)
   - [x] Покрытие новых методов в `gitea.go` > 80% (82.9%)
   - [x] `make test-coverage` показывает результаты

## Tasks / Subtasks

- [x] **Task 1: Тесты для Gitea API (gitea.go)** (AC: #1)
  - [x] 1.1 Тесты GetLatestRelease() с httptest
  - [x] 1.2 Тесты GetReleaseByTag()
  - [x] 1.3 Тесты SearchOrgRepos() с пагинацией
  - [x] 1.4 Тесты CreatePRWithOptions()

- [x] **Task 2: Тесты для парсинга веток** (AC: #2)
  - [x] 2.1 Тесты ParseSubscriptionBranch() - table-driven
  - [x] 2.2 Edge cases: пустые части, специальные символы

- [x] **Task 3: Тесты для основной логики** (AC: #2)
  - [x] 3.1 Mock интерфейс для Gitea API
  - [x] 3.2 Тесты FindSubscribedRepos()
  - [x] 3.3 Тесты GetSourceFiles()
  - [x] 3.4 Тесты SyncExtensionToRepo()

- [x] **Task 4: Тесты для отчётности** (AC: #3)
  - [x] 4.1 Тесты структуры PublishReport
  - [x] 4.2 Тесты ReportResults()
  - [x] 4.3 Тесты JSON-вывода

- [x] **Task 5: Integration тесты** (AC: #4)
  - [x] 5.1 Настроить build tag integration
  - [x] 5.2 Написать setup/teardown
  - [x] 5.3 Написать полный интеграционный тест

## Dev Notes

### Архитектурные паттерны проекта

**Table-driven тесты:**
```go
func TestParseSubscriptionBranch(t *testing.T) {
    tests := []struct {
        name        string
        branchName  string
        wantOrg     string
        wantRepo    string
        wantDir     string
        wantErr     bool
    }{
        {
            name:       "valid simple",
            branchName: "MyOrg_MyRepo_cfe",
            wantOrg:    "MyOrg",
            wantRepo:   "MyRepo",
            wantDir:    "cfe",
            wantErr:    false,
        },
        {
            name:       "valid with nested dir",
            branchName: "APK_ERP_cfe_Common_Ext",
            wantOrg:    "APK",
            wantRepo:   "ERP",
            wantDir:    "cfe/Common/Ext",
            wantErr:    false,
        },
        {
            name:       "too few parts",
            branchName: "MyOrg_MyRepo",
            wantErr:    true,
        },
        {
            name:       "empty org",
            branchName: "_MyRepo_cfe",
            wantErr:    true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseSubscriptionBranch(tt.branchName)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseSubscriptionBranch() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if err == nil {
                if got.Organization != tt.wantOrg {
                    t.Errorf("Organization = %v, want %v", got.Organization, tt.wantOrg)
                }
                if got.Repository != tt.wantRepo {
                    t.Errorf("Repository = %v, want %v", got.Repository, tt.wantRepo)
                }
                if got.TargetDirectory != tt.wantDir {
                    t.Errorf("TargetDirectory = %v, want %v", got.TargetDirectory, tt.wantDir)
                }
            }
        })
    }
}
```

**Мокирование HTTP с httptest:**
```go
func TestGetLatestRelease(t *testing.T) {
    // Создаём mock сервер
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/api/v1/repos/owner/repo/releases/latest" {
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(gitea.Release{
                ID:      1,
                TagName: "v1.0.0",
                Name:    "Release 1.0.0",
                Body:    "Release notes",
            })
            return
        }
        w.WriteHeader(http.StatusNotFound)
    }))
    defer server.Close()

    // Создаём API с mock URL
    api := gitea.NewGiteaAPI(gitea.Config{
        GiteaURL:    server.URL,
        Owner:       "owner",
        Repo:        "repo",
        AccessToken: "test-token",
    })

    // Тестируем
    release, err := api.GetLatestRelease()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if release.TagName != "v1.0.0" {
        t.Errorf("TagName = %v, want v1.0.0", release.TagName)
    }
}
```

**Mock интерфейс:**
```go
// internal/entity/gitea/interfaces.go - добавить новые методы
type GiteaAPI interface {
    // ... существующие методы ...
    GetLatestRelease() (*Release, error)
    GetReleaseByTag(tag string) (*Release, error)
    SearchOrgRepos(orgName string) ([]Repository, error)
    CreatePRWithOptions(opts CreatePROptions) (*PRData, error)
}

// internal/app/extension_publish_test.go
type mockGiteaAPI struct {
    releases     map[string]*gitea.Release
    repos        []gitea.Repository
    branches     map[string][]gitea.Branch
    createPRFunc func(opts gitea.CreatePROptions) (*gitea.PRData, error)
}

func (m *mockGiteaAPI) GetLatestRelease() (*gitea.Release, error) {
    if r, ok := m.releases["latest"]; ok {
        return r, nil
    }
    return nil, fmt.Errorf("release not found")
}
// ... остальные методы ...
```

**Build tag для интеграционных тестов:**
```go
//go:build integration
// +build integration

package app

import (
    "testing"
)

func TestExtensionPublishIntegration(t *testing.T) {
    // Этот тест запускается только с: go test -tags=integration

    // Требуются переменные окружения:
    // TEST_GITEA_URL, TEST_GITEA_TOKEN, TEST_ORG

    // 1. Setup: создать тестовый репозиторий и релиз
    // 2. Execute: запустить ExtensionPublish
    // 3. Verify: проверить созданные PR
    // 4. Teardown: удалить тестовые данные
}
```

### Существующие тесты для ориентира

- `internal/entity/gitea/gitea_test.go` - основные тесты API
- `internal/entity/gitea/gitea_comprehensive_test.go` - комплексные тесты
- `internal/entity/gitea/gitea_branch_commit_test.go` - тесты веток

### Project Structure Notes

- Основные тесты: `internal/app/extension_publish_test.go`
- Тесты API: `internal/entity/gitea/gitea_release_test.go` (новый файл)
- Интеграционные: `internal/app/extension_publish_integration_test.go`

### Команды для запуска тестов

```bash
# Все unit-тесты
make test

# С покрытием
make test-coverage

# Только extension_publish
go test -v ./internal/app/... -run TestExtension

# Интеграционные (требуют настройки)
go test -tags=integration -v ./internal/app/...
```

### References

- [Source: internal/entity/gitea/gitea_test.go] - существующие тесты
- [Source: internal/entity/gitea/interfaces.go] - интерфейсы для мокирования
- [Go testing best practices](https://go.dev/doc/tutorial/add-a-test)

## Definition of Done

- [x] Unit-тесты для всех новых методов в gitea.go (>80% coverage) - 82.9%
- [x] Unit-тесты для ParseSubscriptionBranch() (>90% coverage) - 100%
- [x] Unit-тесты для FindSubscribedRepos() с моками - 93.3%
- [x] Unit-тесты для SyncExtensionToRepo() с моками - 87.0%
- [x] Unit-тесты для PublishReport и отчётности - 100%
- [x] Integration тест написан (под build tag)
- [x] `make test` проходит без ошибок
- [x] `make test-coverage` показывает >80% для extension_publish.go - 92.9%
- [x] `make lint` проходит без ошибок (golangci-lint обновлён до v2.7.2)
- [x] Документация к тестам (комментарии)

## Dev Agent Record

### Context Reference

- Epic: bdocs/epics/epic-0-extension-publish.md
- Story Context: Embedded XML above

### Agent Model Used

Claude Opus 4.5

### Completion Notes List

- Все тесты для Gitea API (GetLatestRelease, GetReleaseByTag, SearchOrgRepos) уже были реализованы в предыдущих историях
- Добавлены тесты для ExtensionPublish: DryRunMode, NoSubscribers, ReleaseNotFound, FullFlow, WithExtDir, SyncError, PRCreationError
- Добавлены тесты для reportResultsText: AllStatuses, OnlyFailed, OnlySkipped
- Создан файл integration тестов с build tag `//go:build integration`
- Покрытие extension_publish.go: 92.9% (превышает требуемые 80%)
- Покрытие gitea.go: 83.6% (превышает требуемые 80%)
- **Code Review Fix:** Обновлён golangci-lint до v2.7.2, теперь make lint запускается
- **Code Review Fix:** Добавлены тесты для CreatePRWithOptions: ServerError, InvalidJSON, ConflictFindError (покрытие 90%)
- **Code Review #2 Fix:** Добавлены тесты для getTargetFilesRecursive: RecursiveError, APIError (покрытие 100%)
- **Code Review #2 Fix:** Исправлен TestMain_WithRealYamlFile - корректно пропускается при test-enable: false

### File List

**Изменённые файлы:**
- `internal/app/extension_publish_test.go` - добавлены тесты ExtensionPublish, ReportResultsText, GetTargetFilesToDelete
- `internal/entity/gitea/gitea_test.go` - добавлены тесты CreatePRWithOptions: ServerError, InvalidJSON, ConflictFindError
- `cmd/apk-ci/yaml_integration_test.go` - исправлен TestMain_WithRealYamlFile
- `bdocs/stories/0-8-unit-integration-tests.md` - обновлён статус

**Новые файлы:**
- `internal/app/extension_publish_integration_test.go` - интеграционные тесты с build tag

**Существующие тесты (ранее созданы):**
- `internal/entity/gitea/gitea_release_test.go`
- `internal/entity/gitea/gitea_search_test.go`
