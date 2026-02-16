# Пакет di — Dependency Injection через Wire

Пакет `di` реализует Dependency Injection с использованием [Google Wire](https://github.com/google/wire) для compile-time генерации кода инъекции зависимостей.

## Обзор

Wire генерирует код для инициализации зависимостей на этапе компиляции, что обеспечивает:
- **Нулевой runtime overhead** — нет reflection или контейнеров в runtime
- **Type safety** — все ошибки обнаруживаются на этапе компиляции
- **Простота отладки** — сгенерированный код читаемый и отлаживаемый

## Структура пакета

```
internal/di/
├── app.go              # Определение App struct с зависимостями
├── interfaces.go       # Документация по интерфейсам (определены в своих пакетах)
├── providers.go        # Provider функции для создания зависимостей
├── providers_test.go   # Unit тесты для providers
├── wire.go             # Wire definitions (//go:build wireinject)
├── wire_gen.go         # Сгенерированный Wire код (НЕ РЕДАКТИРОВАТЬ)
├── integration_test.go # Интеграционные тесты
└── README.md           # Эта документация
```

## Использование

### Базовый пример

```go
package main

import (
    "log"

    "github.com/Kargones/apk-ci/internal/config"
    "github.com/Kargones/apk-ci/internal/di"
)

func main() {
    // 1. Загружаем конфигурацию (MustLoad паникует при ошибке)
    cfg := config.MustLoad()

    // 2. Инициализируем App через Wire DI
    app, err := di.InitializeApp(cfg)
    if err != nil {
        log.Fatal(err)
    }

    // 3. Используем зависимости
    app.Logger.With("trace_id", app.TraceID).Info("Приложение запущено")
}
```

### App struct

```go
type App struct {
    Config       *config.Config    // Конфигурация приложения
    Logger       logging.Logger    // Структурированный логгер
    OutputWriter output.Writer     // Форматировщик вывода (JSON/Text)
    TraceID      string            // Уникальный ID для корреляции логов
}
```

## Добавление новых зависимостей

### Шаг 1: Добавить поле в App struct

```go
// app.go
type App struct {
    // ... существующие поля

    // Новая зависимость
    MyService MyServiceInterface
}
```

### Шаг 2: Создать provider функцию

```go
// providers.go

// ProvideMyService создаёт экземпляр MyService.
// Документируйте зависимости и поведение.
func ProvideMyService(cfg *config.Config, logger logging.Logger) MyServiceInterface {
    return myservice.New(cfg, logger)
}
```

### Шаг 3: Добавить provider в ProviderSet

```go
// wire.go
var ProviderSet = wire.NewSet(
    ProvideLogger,
    ProvideOutputWriter,
    ProvideTraceID,
    ProvideMyService,  // <-- Добавить сюда
    wire.Struct(new(App), "*"),
)
```

### Шаг 4: Перегенерировать wire_gen.go

```bash
# Вариант 1: через make
make generate-wire

# Вариант 2: напрямую
go generate ./internal/di/...

# Вариант 3: через wire CLI
wire gen ./internal/di/...
```

### Шаг 5: Добавить тесты

```go
// providers_test.go
func TestProvideMyService_ReturnsNonNil(t *testing.T) {
    cfg := &config.Config{}
    logger := ProvideLogger(cfg)

    svc := ProvideMyService(cfg, logger)

    assert.NotNil(t, svc)
}
```

## Граф зависимостей

```
                    ┌─────────────┐
                    │   Config    │
                    │  (external) │
                    └──────┬──────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
           ▼               ▼               ▼
    ┌──────────┐    ┌──────────┐    ┌──────────┐
    │  Logger  │    │ Output   │    │  Trace   │
    │ (slog)   │    │ Writer   │    │   ID     │
    └──────────┘    └──────────┘    └──────────┘
           │               │               │
           └───────────────┼───────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │     App     │
                    │  (injected) │
                    └─────────────┘
```

## Конфигурация

### Logger

Logger настраивается через `LoggingConfig` в `config.Config`:

```yaml
# app.yaml
logging:
  level: info      # debug, info, warn, error
  format: text     # text, json
```

### OutputWriter

OutputWriter определяется переменной окружения:

```bash
export BR_OUTPUT_FORMAT=json  # json или text
```

### TraceID

TraceID генерируется автоматически при инициализации App.
Формат: 32-символьный hex string (например: `a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6`).

## CI/CD интеграция

### Makefile targets

```bash
# Генерация wire_gen.go
make generate-wire

# Проверка актуальности wire_gen.go
make check-wire
```

### CI pipeline

```yaml
steps:
  - name: Generate Wire
    run: make generate-wire

  - name: Verify Wire Generated
    run: git diff --exit-code internal/di/wire_gen.go
```

## Тестирование

```bash
# Unit тесты
go test ./internal/di/... -v

# С покрытием
go test ./internal/di/... -coverprofile=coverage.out
```

## Важные замечания

1. **wire_gen.go** — сгенерированный файл, НЕ редактируйте вручную
2. **wire.go** имеет build tag `//go:build wireinject` — компилируется только для Wire
3. Ошибки зависимостей (nil, циклы) обнаруживаются на этапе компиляции
4. Интерфейсы определены в своих пакетах (`logging.Logger`, `output.Writer`)

## Связанные пакеты

- `internal/pkg/logging` — Logger interface и SlogAdapter
- `internal/pkg/output` — OutputWriter interface и реализации
- `internal/pkg/tracing` — TraceID генерация
- `internal/config` — Config struct
