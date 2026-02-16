# Batch Operations API для Gitea

Этот документ описывает новую функциональность для выполнения batch операций с файлами в репозитории Gitea через API.

## Обзор

Функция `SetRepositoryState` позволяет выполнять множественные операции с файлами (создание, обновление, удаление) в одном коммите, что обеспечивает атомарность изменений.

## Структуры данных

### BatchOperation

Представляет одну операцию над файлом:

```go
type BatchOperation struct {
    Operation string `json:"operation"` // "create", "update", "delete"
    Path      string `json:"path"`      // Путь к файлу в репозитории
    Content   string `json:"content,omitempty"` // Содержимое файла (base64)
    SHA       string `json:"sha,omitempty"`     // SHA файла для update/delete
}
```

### BatchCommitRequest

Представляет запрос для batch коммита:

```go
type BatchCommitRequest struct {
    Branch     string           `json:"branch"`
    Author     CommitAuthor     `json:"author"`
    Committer  CommitAuthor     `json:"committer"`
    Message    string           `json:"commit_message"`
    Operations []BatchOperation `json:"operations"`
}
```

### CommitAuthor

Представляет автора коммита:

```go
type CommitAuthor struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

## API методы

### SetRepositoryState

```go
func (g *API) SetRepositoryState(operations []BatchOperation, branch, commitMessage string) error
```

**Параметры:**
- `operations` - массив операций для выполнения
- `branch` - ветка для коммита
- `commitMessage` - сообщение коммита

**Возвращает:**
- `error` - ошибка выполнения или `nil` при успехе

## Примеры использования

### Базовый пример

```go
package main

import (
    "github.com/Kargones/apk-ci/internal/entity/gitea"
    "github.com/Kargones/apk-ci/internal/service"
    "github.com/Kargones/apk-ci/internal/config"
    "encoding/base64"
    "log"
)

func main() {
    // Конфигурация
    cfg := &config.Config{
        GiteaURL:    "https://gitea.example.com",
        Owner:       "myorg",
        Repo:        "myrepo",
        AccessToken: "your_token_here",
        BaseBranch:  "main",
    }

    // Создание сервиса
    factory := &service.GiteaFactory{}
    giteaService, err := factory.CreateGiteaService(cfg)
    if err != nil {
        log.Fatal(err)
    }

    giteaAPI := giteaService.GetAPI()

    // Подготовка операций
    operations := []gitea.BatchOperation{
        {
            Operation: "create",
            Path:      "docs/readme.md",
            Content:   base64.StdEncoding.EncodeToString([]byte("# My Project\n")),
        },
        {
            Operation: "update",
            Path:      "config.json",
            Content:   base64.StdEncoding.EncodeToString([]byte(`{"version": "2.0"}`)),
            SHA:       "existing_file_sha",
        },
        {
            Operation: "delete",
            Path:      "old_config.json",
            SHA:       "old_file_sha",
        },
    }

    // Выполнение операций
    err = giteaAPI.SetRepositoryState(
        operations,
        "main",
        "Обновление конфигурации проекта",
    )
    if err != nil {
        log.Fatal(err)
    }

    log.Println("Операции выполнены успешно")
}
```

### Интеграция с существующим кодом

Для интеграции с существующим кодом, который использует `FileInfo`, можно использовать вспомогательную функцию:

```go
// Создание операций из FileInfo
operations := CreateBatchOperationsFromFileInfos(currentFiles, newFiles)

// Выполнение операций
err := giteaAPI.SetRepositoryState(
    operations,
    "main",
    "Синхронизация файлов",
)
```

## Типы операций

### Create
- Создает новый файл
- Требует: `Operation`, `Path`, `Content`
- SHA не требуется

### Update
- Обновляет существующий файл
- Требует: `Operation`, `Path`, `Content`, `SHA`
- SHA должен соответствовать текущему состоянию файла

### Delete
- Удаляет файл
- Требует: `Operation`, `Path`, `SHA`
- Content не требуется

## Обработка ошибок

Функция возвращает ошибки в следующих случаях:
- Пустой список операций
- Ошибка сериализации JSON
- Ошибка HTTP запроса
- Неуспешный статус ответа (не 200/201)

## Ограничения

1. Содержимое файлов должно быть закодировано в base64
2. Для операций update и delete требуется корректный SHA файла
3. Все операции выполняются в одном коммите (атомарно)
4. Максимальный размер запроса ограничен настройками Gitea сервера

## Соответствие Gitea API

Функция использует Gitea API endpoint:
```
POST /api/v1/repos/{owner}/{repo}/contents
```

Формат запроса соответствует официальной документации Gitea API для batch операций.