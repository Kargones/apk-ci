# План реализации рефакторинга модуля gitea.go

## Обзор

Данный план описывает пошаговую реализацию изменений для устранения зависимости модуля `internal/entity/gitea/gitea.go` от модуля `internal/config`, следуя принципам чистой архитектуры.

## Цели рефакторинга

1. **Устранить прямую зависимость** entity-слоя от конфигурационного слоя
2. **Создать чистый API-слой** для работы с Gitea
3. **Выделить сервисный слой** для бизнес-логики
4. **Обеспечить возможность независимого тестирования** компонентов
5. **Сохранить обратную совместимость** с существующим кодом

## Анализ текущего состояния

### Зависимости от config.Config в gitea.go:

1. **Метод Init()** - использует 7 полей из config:
   - `Command`, `AccessToken`, `GiteaURL`, `Owner`, `Repo`, `BaseBranch`, `NewBranch`

2. **Метод TestMerge()** - принимает `*config.Config` как параметр (не использует)

3. **Функция AnalyzeProject()** - вызывает `cfg.AnalyzeProject(l, "main")`

### Структуры, требующие рефакторинга:

- `GiteaAPI` - основная структура для работы с API
- Функции `TestMerge` и `AnalyzeProject` - содержат бизнес-логику

## Этап 1: Подготовка к рефакторингу

### 1.1 Создание интерфейсов

**Файл:** `internal/entity/gitea/interfaces.go`

```go
package gitea

import (
    "context"
    "log/slog"
)

// GiteaConfig определяет конфигурацию для Gitea API
type GiteaConfig struct {
    GiteaURL    string
    Owner       string
    Repo        string
    AccessToken string
    BaseBranch  string
    NewBranch   string
    Command     string
}

// GiteaAPIInterface определяет методы для работы с Gitea API
type GiteaAPIInterface interface {
    GetIssue(issueNumber int64) (*Issue, error)
    GetFileContent(fileName string) ([]byte, error)
    AddIssueComment(issueNumber int64, commentText string) error
    CloseIssue(issueNumber int64) error
    ConflictPR(prNumber int64) (bool, error)
    ConflictFilesPR(prNumber int64) ([]string, error)
    GetRepositoryContents(branch string) ([]FileInfo, error)
    AnalyzeProjectStructure(branch string) ([]string, error)
    GetLatestCommit(branch string) (*Commit, error)
    GetCommitFiles(commitSHA string) ([]CommitFile, error)
    IsUserInTeam(l *slog.Logger, username string, orgName string, teamName string) (bool, error)
}

// ProjectAnalyzer определяет интерфейс для анализа проектов
type ProjectAnalyzer interface {
    AnalyzeProject(l *slog.Logger, branch string) error
}
```

### 1.2 Создание конструктора для GiteaAPI

**Обновление файла:** `internal/entity/gitea/gitea.go`

```go
// NewGiteaAPI создает новый экземпляр GiteaAPI с переданной конфигурацией
func NewGiteaAPI(config GiteaConfig) *GiteaAPI {
    return &GiteaAPI{
        GiteaURL:    config.GiteaURL,
        Owner:       config.Owner,
        Repo:        config.Repo,
        AccessToken: config.AccessToken,
        BaseBranch:  config.BaseBranch,
        NewBranch:   config.NewBranch,
        Command:     config.Command,
    }
}
```

## Этап 2: Создание сервисного слоя

### 2.1 Создание пакета service

**Файл:** `internal/service/gitea_service.go`

```go
package service

import (
    "context"
    "log/slog"
    
    "github.com/Kargones/apk-ci/internal/config"
    "github.com/Kargones/apk-ci/internal/entity/gitea"
)

// GiteaService содержит бизнес-логику для работы с Gitea
type GiteaService struct {
    api           gitea.GiteaAPIInterface
    config        *config.Config
    projectAnalyzer gitea.ProjectAnalyzer
}

// NewGiteaService создает новый экземпляр GiteaService
func NewGiteaService(api gitea.GiteaAPIInterface, cfg *config.Config, analyzer gitea.ProjectAnalyzer) *GiteaService {
    return &GiteaService{
        api:           api,
        config:        cfg,
        projectAnalyzer: analyzer,
    }
}

// TestMerge выполняет тестирование слияния
func (s *GiteaService) TestMerge(ctx context.Context, l *slog.Logger) error {
    // Перенести логику из gitea.TestMerge
    // Использовать s.api для вызовов API
    // Использовать s.config для доступа к конфигурации
    return nil // TODO: реализовать
}

// AnalyzeProject выполняет анализ проекта
func (s *GiteaService) AnalyzeProject(ctx context.Context, l *slog.Logger) error {
    return s.projectAnalyzer.AnalyzeProject(l, "main")
}
```

### 2.2 Создание адаптера для ProjectAnalyzer

**Файл:** `internal/service/config_analyzer.go`

```go
package service

import (
    "log/slog"
    
    "github.com/Kargones/apk-ci/internal/config"
)

// ConfigAnalyzer адаптер для config.Config, реализующий ProjectAnalyzer
type ConfigAnalyzer struct {
    config *config.Config
}

// NewConfigAnalyzer создает новый адаптер
func NewConfigAnalyzer(cfg *config.Config) *ConfigAnalyzer {
    return &ConfigAnalyzer{config: cfg}
}

// AnalyzeProject реализует интерфейс ProjectAnalyzer
func (ca *ConfigAnalyzer) AnalyzeProject(l *slog.Logger, branch string) error {
    return ca.config.AnalyzeProject(l, branch)
}
```

## Этап 3: Рефакторинг существующего кода

### 3.1 Обновление gitea.go

**Изменения в файле:** `internal/entity/gitea/gitea.go`

1. **Удалить импорт config:**
```go
// Удалить эту строку:
// import "github.com/Kargones/apk-ci/internal/config"
```

2. **Заменить метод Init:**
```go
// Заменить существующий метод Init на:
func (g *GiteaAPI) UpdateConfig(config GiteaConfig) {
    g.Command = config.Command
    g.AccessToken = config.AccessToken
    g.GiteaURL = config.GiteaURL
    g.Owner = config.Owner
    g.Repo = config.Repo
    g.BaseBranch = config.BaseBranch
    g.NewBranch = config.NewBranch
}

// Добавить метод для обратной совместимости (deprecated)
func (g *GiteaAPI) Init(cfg interface{}) {
    // Временная реализация для обратной совместимости
    // Будет удалена в следующих версиях
}
```

3. **Переместить функции в сервисный слой:**
```go
// Удалить эти функции из gitea.go:
// - func (g *GiteaAPI) TestMerge
// - func AnalyzeProject
```

### 3.2 Создание фабрики для конфигурации

**Файл:** `internal/service/gitea_factory.go`

```go
package service

import (
    "github.com/Kargones/apk-ci/internal/config"
    "github.com/Kargones/apk-ci/internal/entity/gitea"
)

// GiteaFactory создает компоненты для работы с Gitea
type GiteaFactory struct{}

// NewGiteaFactory создает новую фабрику
func NewGiteaFactory() *GiteaFactory {
    return &GiteaFactory{}
}

// CreateGiteaConfig создает конфигурацию Gitea из config.Config
func (f *GiteaFactory) CreateGiteaConfig(cfg *config.Config) gitea.GiteaConfig {
    return gitea.GiteaConfig{
        GiteaURL:    cfg.GiteaURL,
        Owner:       cfg.Owner,
        Repo:        cfg.Repo,
        AccessToken: cfg.AccessToken,
        BaseBranch:  cfg.BaseBranch,
        NewBranch:   cfg.NewBranch,
        Command:     cfg.Command,
    }
}

// CreateGiteaService создает полностью настроенный GiteaService
func (f *GiteaFactory) CreateGiteaService(cfg *config.Config) *GiteaService {
    giteaConfig := f.CreateGiteaConfig(cfg)
    api := gitea.NewGiteaAPI(giteaConfig)
    analyzer := NewConfigAnalyzer(cfg)
    
    return NewGiteaService(api, cfg, analyzer)
}
```

## Этап 4: Обновление клиентского кода

### 4.1 Обновление app.go

**Файл:** `internal/app/app.go`

```go
// Заменить прямое использование gitea.GiteaAPI на GiteaService

import (
    "github.com/Kargones/apk-ci/internal/service"
)

// В функции, где используется gitea:
func someFunction(cfg *config.Config) {
    // Старый код:
    // giteaAPI := &gitea.GiteaAPI{}
    // giteaAPI.Init(cfg)
    
    // Новый код:
    factory := service.NewGiteaFactory()
    giteaService := factory.CreateGiteaService(cfg)
    
    // Использовать giteaService вместо прямых вызовов API
}
```

### 4.2 Обновление action_menu_build.go

**Файл:** `internal/app/action_menu_build.go`

```go
// Найти и заменить вызовы:
// gitea.TestMerge(ctx, l, cfg) -> giteaService.TestMerge(ctx, l)
// gitea.AnalyzeProject(ctx, l, cfg) -> giteaService.AnalyzeProject(ctx, l)
```

## Этап 5: Тестирование

### 5.1 Создание unit-тестов

**Файл:** `internal/entity/gitea/gitea_test.go`

```go
package gitea_test

import (
    "testing"
    
    "github.com/Kargones/apk-ci/internal/entity/gitea"
)

func TestNewGiteaAPI(t *testing.T) {
    config := gitea.GiteaConfig{
        GiteaURL:    "https://gitea.example.com",
        Owner:       "testowner",
        Repo:        "testrepo",
        AccessToken: "testtoken",
        BaseBranch:  "main",
        NewBranch:   "feature",
        Command:     "test",
    }
    
    api := gitea.NewGiteaAPI(config)
    
    if api.GiteaURL != config.GiteaURL {
        t.Errorf("Expected GiteaURL %s, got %s", config.GiteaURL, api.GiteaURL)
    }
    // Добавить остальные проверки
}
```

**Файл:** `internal/service/gitea_service_test.go`

```go
package service_test

import (
    "context"
    "log/slog"
    "testing"
    
    "github.com/Kargones/apk-ci/internal/service"
)

// Создать mock для GiteaAPIInterface
type mockGiteaAPI struct{}

// Реализовать методы интерфейса

// Создать mock для ProjectAnalyzer
type mockProjectAnalyzer struct{}

func (m *mockProjectAnalyzer) AnalyzeProject(l *slog.Logger, branch string) error {
    return nil
}

func TestGiteaService_AnalyzeProject(t *testing.T) {
    api := &mockGiteaAPI{}
    analyzer := &mockProjectAnalyzer{}
    service := service.NewGiteaService(api, nil, analyzer)
    
    ctx := context.Background()
    l := slog.Default()
    
    err := service.AnalyzeProject(ctx, l)
    if err != nil {
        t.Errorf("Expected no error, got %v", err)
    }
}
```

### 5.2 Интеграционные тесты

**Файл:** `internal/service/integration_test.go`

```go
package service_test

import (
    "testing"
    
    "github.com/Kargones/apk-ci/internal/config"
    "github.com/Kargones/apk-ci/internal/service"
)

func TestGiteaFactory_CreateGiteaService(t *testing.T) {
    // Создать тестовую конфигурацию
    cfg := &config.Config{
        GiteaURL:    "https://gitea.example.com",
        Owner:       "testowner",
        Repo:        "testrepo",
        AccessToken: "testtoken",
        BaseBranch:  "main",
        NewBranch:   "feature",
        Command:     "test",
    }
    
    factory := service.NewGiteaFactory()
    giteaService := factory.CreateGiteaService(cfg)
    
    if giteaService == nil {
        t.Error("Expected GiteaService to be created")
    }
}
```

## Этап 6: Миграция и развертывание

### 6.1 Поэтапная миграция

1. **Фаза 1: Подготовка**
   - Создать новые интерфейсы и структуры
   - Добавить конструкторы и фабрики
   - Сохранить старые методы для обратной совместимости

2. **Фаза 2: Миграция клиентского кода**
   - Обновить app.go для использования нового API
   - Обновить action_menu_build.go
   - Запустить тесты для проверки совместимости

3. **Фаза 3: Очистка**
   - Удалить deprecated методы
   - Удалить импорт config из gitea.go
   - Переместить функции в сервисный слой

### 6.2 Проверка качества кода

```bash
# Запуск линтера
golangci-lint run

# Запуск тестов
go test ./...

# Проверка покрытия
go test -cover ./...

# Проверка зависимостей
go mod tidy
go mod verify
```

### 6.3 Документация изменений

**Файл:** `docs/migration-guide.md`

```markdown
# Руководство по миграции Gitea API

## Изменения в API

### Было:
```go
giteaAPI := &gitea.GiteaAPI{}
giteaAPI.Init(cfg)
gitea.TestMerge(ctx, l, cfg)
```

### Стало:
```go
factory := service.NewGiteaFactory()
giteaService := factory.CreateGiteaService(cfg)
giteaService.TestMerge(ctx, l)
```

## Преимущества новой архитектуры

1. Четкое разделение ответственности
2. Возможность независимого тестирования
3. Соответствие принципам чистой архитектуры
4. Упрощение mock-тестирования
```

## Этап 7: Мониторинг и поддержка

### 7.1 Метрики качества

- **Покрытие тестами:** > 80%
- **Цикломатическая сложность:** < 10
- **Количество зависимостей:** минимизировано
- **Время сборки:** не увеличено

### 7.2 План поддержки

1. **Первые 2 недели:** Мониторинг ошибок и производительности
2. **1 месяц:** Сбор обратной связи от разработчиков
3. **3 месяца:** Оптимизация на основе реального использования

## Заключение

Данный план обеспечивает поэтапную миграцию от монолитной архитектуры к чистой архитектуре с четким разделением ответственности. Каждый этап может быть выполнен независимо, что минимизирует риски и обеспечивает возможность отката изменений.

### Ожидаемые результаты:

1. ✅ Устранена зависимость entity-слоя от config
2. ✅ Создан чистый API-слой для Gitea
3. ✅ Выделен сервисный слой для бизнес-логики
4. ✅ Обеспечена возможность независимого тестирования
5. ✅ Сохранена обратная совместимость

### Временные затраты:

- **Этап 1-2:** 2-3 дня
- **Этап 3-4:** 3-4 дня
- **Этап 5:** 2-3 дня
- **Этап 6-7:** 1-2 дня

**Общее время:** 8-12 рабочих дней