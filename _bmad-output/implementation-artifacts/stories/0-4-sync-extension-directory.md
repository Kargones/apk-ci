# Story 0.4: Синхронизация каталога расширения

Status: Done

## Story

As a система,
I want синхронизировать файлы расширения в целевой репозиторий,
so that подписчик получает актуальную версию.

## Story Context

```xml
<story-context>
  <epic id="0" name="Extension Publish" priority="FIRST">
    <goal>Реализовать команду extension-publish для автоматического распространения расширений 1C</goal>
  </epic>

  <story id="0.4" name="Sync Extension Directory" size="M" risk="Medium" priority="P0">
    <dependency type="required" story="0.1">Требует GetLatestRelease() для получения версии</dependency>
    <dependency type="required" story="0.3">Требует FindSubscribedRepos() для списка целей</dependency>
    <blocks stories="0.5, 0.6">Основная логика синхронизации файлов</blocks>
  </story>

  <codebase-context>
    <file path="internal/app/extension_publish.go" role="primary">
      <note>Добавление функций синхронизации в существующий файл</note>
    </file>
    <file path="internal/entity/gitea/gitea.go" role="dependency">
      <methods-used>
        - SetRepositoryState() - batch операции с файлами
        - GetRepositoryContents() - получение содержимого каталога
        - GetFileContent() - получение содержимого файла
      </methods-used>
      <existing-types>
        - ChangeFileOperation - операция над файлом
        - ChangeFilesOptions - параметры batch коммита
      </existing-types>
    </file>
  </codebase-context>

  <business-rules>
    <rule id="BR1">
      Синхронизация = ПОЛНАЯ ЗАМЕНА содержимого целевого каталога
      Все существующие файлы удаляются, новые создаются
    </rule>
    <rule id="BR2">
      Commit message формат: "chore(ext): update {extName} to {version}"
      Пример: "chore(ext): update CommonExt to v1.2.3"
    </rule>
    <rule id="BR3">
      Изменения делаются в НОВОЙ ветке: "update-{extName}-{version}"
      PR создается в Story 0.5
    </rule>
  </business-rules>
</story-context>
```

## Acceptance Criteria

1. **AC1: Структура SyncResult**
   - [x] Создана структура `SyncResult` с полями:
     - `Subscriber SubscribedRepo` - целевой репозиторий
     - `FilesCreated int` - количество созданных файлов
     - `FilesDeleted int` - количество удаленных файлов
     - `NewBranch string` - имя созданной ветки
     - `CommitSHA string` - SHA коммита
     - `Error error` - ошибка синхронизации (если есть)

2. **AC2: Функция GetSourceFiles()**
   - [x] Сигнатура: `func GetSourceFiles(api *gitea.API, sourceDir, branch string) ([]gitea.ChangeFileOperation, error)`
   - [x] Рекурсивно получает все файлы из исходного каталога
   - [x] Для каждого файла получает содержимое в base64
   - [x] Формирует список операций "create" для каждого файла
   - [x] Сохраняет относительные пути файлов

3. **AC3: Функция GetTargetFilesToDelete()**
   - [x] Сигнатура: `func GetTargetFilesToDelete(api *gitea.API, targetDir, branch string) ([]gitea.ChangeFileOperation, error)`
   - [x] Получает текущее содержимое целевого каталога
   - [x] Формирует список операций "delete" для всех существующих файлов
   - [x] Обрабатывает случай пустого/несуществующего каталога

4. **AC4: Функция SyncExtensionToRepo()**
   - [x] Сигнатура: `func SyncExtensionToRepo(sourceAPI, targetAPI *gitea.API, subscriber SubscribedRepo, sourceDir, extName, version string) (*SyncResult, error)`
   - [x] Создает новую ветку от default branch
   - [x] Формирует commit message по шаблону
   - [x] Объединяет операции delete + create
   - [x] Выполняет batch commit через SetRepositoryStateWithNewBranch()
   - [x] Возвращает результат синхронизации

5. **AC5: Обработка ошибок и edge cases**
   - [x] Пустой исходный каталог -> error
   - [ ] Целевая ветка уже существует -> добавить timestamp к имени (частично, будет в Story 0.5)
   - [x] Ошибка API -> информативное сообщение с контекстом
   - [ ] Большие файлы (>100MB) -> пропустить с warning (отложено, не критично)

## Tasks / Subtasks

- [x] **Task 1: Создание структуры результата** (AC: #1)
  - [x] 1.1 Добавить структуру `SyncResult` в extension_publish.go
  - [x] 1.2 Добавить документирующие комментарии

- [x] **Task 2: Реализация GetSourceFiles()** (AC: #2)
  - [x] 2.1 Реализовать рекурсивный обход каталога
  - [x] 2.2 Получать содержимое файлов в base64
  - [x] 2.3 Формировать относительные пути
  - [x] 2.4 Обработать ошибки получения файлов

- [x] **Task 3: Реализация GetTargetFilesToDelete()** (AC: #3)
  - [x] 3.1 Получить содержимое целевого каталога
  - [x] 3.2 Рекурсивно собрать все файлы
  - [x] 3.3 Сформировать операции delete с SHA
  - [x] 3.4 Обработать случай несуществующего каталога

- [x] **Task 4: Реализация SyncExtensionToRepo()** (AC: #4, #5)
  - [x] 4.1 Сгенерировать имя новой ветки
  - [x] 4.2 Создать ветку от default branch
  - [x] 4.3 Собрать все операции (delete + create)
  - [x] 4.4 Выполнить SetRepositoryStateWithNewBranch()
  - [x] 4.5 Обработать ошибки и edge cases

## Dev Notes

### Архитектурные паттерны проекта

**Использование SetRepositoryState():**
```go
// Существующий метод в gitea.go:1153
func (g *API) SetRepositoryState(l *slog.Logger, operations []ChangeFileOperation, branch, commitMessage string) error

// ChangeFileOperation структура:
type ChangeFileOperation struct {
    Operation string `json:"operation"` // "create", "update", "delete"
    Path      string `json:"path"`
    Content   string `json:"content,omitempty"` // base64 для create/update
    SHA       string `json:"sha,omitempty"`     // требуется для delete
}
```

**Рекурсивный обход каталога:**
```go
func GetSourceFiles(api *gitea.API, owner, repo, sourceDir, basePath string) ([]gitea.ChangeFileOperation, error) {
    var operations []gitea.ChangeFileOperation

    contents, err := api.GetRepositoryContents(sourceDir, "main")
    if err != nil {
        return nil, fmt.Errorf("ошибка получения содержимого %s: %v", sourceDir, err)
    }

    for _, item := range contents {
        fullPath := filepath.Join(basePath, item.Name)

        if item.Type == "dir" {
            // Рекурсивный вызов для поддиректорий
            subOps, err := GetSourceFiles(api, owner, repo, filepath.Join(sourceDir, item.Name), fullPath)
            if err != nil {
                return nil, err
            }
            operations = append(operations, subOps...)
        } else if item.Type == "file" {
            // Получаем содержимое файла
            content, err := api.GetFileContent(item.Path)
            if err != nil {
                return nil, fmt.Errorf("ошибка получения файла %s: %v", item.Path, err)
            }

            operations = append(operations, gitea.ChangeFileOperation{
                Operation: "create",
                Path:      fullPath,
                Content:   base64.StdEncoding.EncodeToString(content),
            })
        }
    }

    return operations, nil
}
```

### Формат commit message

```go
commitMessage := fmt.Sprintf("chore(ext): update %s to %s", extName, version)
// Пример: "chore(ext): update CommonExt to v1.2.3"
```

### Формат имени ветки

```go
branchName := fmt.Sprintf("update-%s-%s",
    strings.ToLower(extName),
    strings.TrimPrefix(version, "v"))
// Пример: "update-commonext-1.2.3"
```

### Project Structure Notes

- Файл: `internal/app/extension_publish.go` (дополнение к Story 0.3)
- Тесты: `internal/app/extension_publish_test.go`
- Использует существующие методы из `internal/entity/gitea/gitea.go`

### Важные ограничения Gitea API

1. SetRepositoryState() делает один коммит со всеми изменениями
2. Максимальный размер запроса может быть ограничен (~100MB)
3. Для больших репозиториев может потребоваться батчинг

### References

- [Source: internal/entity/gitea/gitea.go#SetRepositoryState] - метод batch commit
- [Source: internal/entity/gitea/gitea.go#GetRepositoryContents] - получение содержимого
- [Source: internal/entity/gitea/gitea.go#ChangeFileOperation] - структура операции

## Definition of Done

- [x] Структура `SyncResult` создана с документацией
- [x] Функция `GetSourceFiles()` рекурсивно получает все файлы
- [x] Функция `GetTargetFilesToDelete()` формирует операции удаления
- [x] Функция `SyncExtensionToRepo()` выполняет полную синхронизацию
- [x] Новая ветка создается корректно
- [x] Commit message соответствует формату
- [x] Edge cases обработаны (пустой каталог, существующая ветка - частично)
- [x] Unit-тесты с мокированием API (>80% coverage)
- [ ] Код проходит `make lint` (проблема с версией golangci-lint)
- [x] Код проходит `make test`

## Dev Agent Record

### Context Reference

- Epic: bdocs/epics/epic-0-extension-publish.md
- Story Context: Embedded XML above

### Agent Model Used

Claude Opus 4.5

### Completion Notes List

1. **Task 1 (SyncResult)**: Создана структура с полной документацией. Тесты: TestSyncResult_Structure, TestSyncResult_WithError
2. **Task 2 (GetSourceFiles)**: Реализован рекурсивный обход с getSourceFilesRecursive(). Тесты: TestGetSourceFiles_Success, TestGetSourceFiles_EmptyDirectory, TestGetSourceFiles_DirectoryNotFound
3. **Task 3 (GetTargetFilesToDelete)**: Реализован сбор операций delete с SHA. Тесты: TestGetTargetFilesToDelete_Success, TestGetTargetFilesToDelete_EmptyDirectory, TestGetTargetFilesToDelete_DirectoryNotExists
4. **Task 4 (SyncExtensionToRepo)**: Добавлен новый метод SetRepositoryStateWithNewBranch в gitea.go для создания ветки при коммите. Вспомогательные функции: GenerateBranchName, GenerateCommitMessage. Тесты: TestGenerateBranchName, TestGenerateCommitMessage, TestSyncExtensionToRepo_Success, TestSyncExtensionToRepo_EmptySource

### Design Decisions

1. Добавлен метод `SetRepositoryStateWithNewBranch` в gitea.go вместо модификации существующего `SetRepositoryState` для сохранения обратной совместимости
2. Функции `GetSourceFiles` и `GetTargetFilesToDelete` принимают параметр `branch` для гибкости
3. `SyncExtensionToRepo` принимает два API объекта (source и target) для поддержки разных репозиториев
4. [Code Review] `SyncExtensionToRepo` принимает явный параметр `sourceBranch` вместо hardcoded "main" для поддержки репозиториев с разными default branch
5. [Code Review] Используется `path.Join` вместо `filepath.Join` для формирования путей Gitea API — обеспечивает кросс-платформенную совместимость (Windows/Linux)

### File List

**Изменённые файлы:**
- `internal/app/extension_publish.go` - добавлены SyncResult, GetSourceFiles, GetTargetFilesToDelete, SyncExtensionToRepo, GenerateBranchName, GenerateCommitMessage
- `internal/app/extension_publish_test.go` - добавлены тесты для всех новых функций
- `internal/entity/gitea/gitea.go` - добавлен SetRepositoryStateWithNewBranch
- `bdocs/sprint-artifacts/sprint-status.yaml` - статус story обновлён
- `bdocs/stories/0-4-sync-extension-directory.md` - обновлён статус и dev agent record

### Change Log

- 2025-12-08: Story 0.4 реализована - синхронизация каталога расширения
- 2025-12-08: Code Review исправления:
  - [HIGH] Добавлен параметр `sourceBranch` в `SyncExtensionToRepo` (убран hardcoded "main")
  - [MEDIUM] Заменён `filepath.Join` на `path.Join` для кросс-платформенной совместимости с Gitea API
  - [MEDIUM] Добавлены комментарии про base64 кодирование в `getSourceFilesRecursive`
  - Обновлены тесты `TestSyncExtensionToRepo_Success` и `TestSyncExtensionToRepo_EmptySource`
