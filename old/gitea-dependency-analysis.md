# Анализ зависимостей модуля gitea.go от config

## Обзор проблемы

Модуль `internal/entity/gitea/gitea.go` имеет прямую зависимость от модуля `internal/config`, что нарушает принципы чистой архитектуры. Entity-слой не должен зависеть от конфигурационного слоя.

## Текущие зависимости

### 1. Импорт модуля config
```go
import "github.com/Kargones/apk-ci/internal/config"
```

### 2. Использование в методах

#### Метод Init
```go
func (g *GiteaAPI) Init(cfg *config.Config) {
    g.Command = cfg.Command
    g.AccessToken = cfg.AccessToken
    g.GiteaURL = cfg.GiteaURL
    g.Owner = cfg.Owner
    g.Repo = cfg.Repo
    g.BaseBranch = cfg.BaseBranch
    g.NewBranch = cfg.NewBranch
}
```

#### Метод TestMerge
```go
func (g *GiteaAPI) TestMerge(_ *context.Context, l *slog.Logger, _ *config.Config) error
```

#### Функция AnalyzeProject
```go
func AnalyzeProject(_ *context.Context, l *slog.Logger, cfg *config.Config) error {
    return cfg.AnalyzeProject(l, "main")
}
```


### Метод решения: Разделение на отдельные сервисы

#### Описание
Выделить функции, зависящие от config, в отдельный сервисный слой.

#### Реализация
```go
// В gitea.go остаются только чистые API методы
type GiteaAPI struct {
    // только поля для API
}

// Новый сервис в отдельном пакете
type GiteaService struct {
    api    *GiteaAPI
    config *config.Config
}

func (s *GiteaService) AnalyzeProject(ctx context.Context, l *slog.Logger) error {
    return s.config.AnalyzeProject(l, "main")
}
```

#### Преимущества
- Четкое разделение ответственности
- Соответствие принципам чистой архитектуры
- Возможность независимого тестирования
- Переиспользование GiteaAPI в разных контекстах

#### Недостатки
- Необходимость рефакторинга существующего кода
- Увеличение количества файлов и пакетов
- Потенциальное усложнение архитектуры
