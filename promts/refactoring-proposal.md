# Предложение по рефакторингу benadis-runner

**Дата:** 2025-11-26
**Автор:** XoR
**Версия:** 1.0

---

## 1. Описание проблемы

### 1.1. Текущее состояние

benadis-runner — CLI-инструмент автоматизации для 1C:Enterprise, содержащий 17 команд для управления сервисным режимом, операций с базами данных, синхронизации конфигураций и интеграций с SonarQube/Gitea.

### 1.2. Выявленные архитектурные проблемы

#### Нарушения принципов SOLID

| Принцип | Нарушение | Последствия |
|---------|-----------|-------------|
| **Single Responsibility (SRP)** | `main.go` содержит switch на 17 команд, `app.go` смешивает оркестрацию с бизнес-логикой | Сложность поддержки, высокая связанность |
| **Open/Closed (OCP)** | Добавление новой команды требует изменения main.go | Риск регрессий, невозможность расширения без модификации |
| **Liskov Substitution (LSP)** | Отсутствие единого интерфейса для команд | Невозможность полиморфной обработки |
| **Interface Segregation (ISP)** | Широкие интерфейсы (например, GiteaClient) вынуждают реализовывать неиспользуемые методы | Избыточные зависимости, сложность мокирования |
| **Dependency Inversion (DIP)** | Прямое создание зависимостей внутри функций вместо injection | Сложность тестирования, жёсткая связанность |

#### Технический долг

1. **Дублирование кода**: Одинаковые паттерны выполнения команд в разных частях `app.go`
2. **Отсутствие структурированного вывода**: Результаты команд смешиваются с логами
3. **Нет единой обработки ошибок**: Разные команды обрабатывают ошибки по-разному
4. **Слабая тестируемость**: Зависимости от внешних систем (1C, MSSQL, Gitea) создаются внутри функций

#### Операционные проблемы

1. **Диагностика проблем затруднена**: Нет trace_id для корреляции логов
2. **Нет прогресса долгих операций**: Пользователи не знают сколько ждать
3. **Отсутствие алертинга**: О проблемах узнают только после анализа логов

### 1.3. Количественные метрики проблем

| Метрика | Текущее значение | Проблема |
|---------|------------------|----------|
| Размер switch в main.go | 17 case-блоков | Нарушение OCP |
| Функций в app.go | ~30 | Нарушение SRP |
| Покрытие тестами (оценка) | < 50% | Низкая уверенность в изменениях |
| Cyclomatic complexity | > 10 для ключевых функций | Сложность понимания и поддержки |

---

## 2. Описание выбранного метода решения

### 2.1. Общий подход: Поэтапный рефакторинг с NR-миграцией

**Стратегия:** Параллельная разработка новой архитектуры с постепенной миграцией существующего функционала через механизм NR-команд (New Refactored).

**Ключевые архитектурные решения:**

#### 2.1.1. Dependency Injection через Wire

**Проблема:** Прямое создание зависимостей нарушает DIP и затрудняет тестирование.

**Решение:** Google Wire для compile-time DI.

```go
// Вместо:
func ServiceModeEnable(...) error {
    client := rac.NewClient(config) // Прямое создание
    return client.Enable()
}

// Станет:
func NewServiceModeHandler(client RACClient) *ServiceModeHandler {
    return &ServiceModeHandler{client: client}
}
```

**Обоснование выбора Wire:**
- Compile-time — ошибки обнаруживаются при сборке, а не в runtime
- Нет reflection overhead
- Идиоматичен для Go-проектов

#### 2.1.2. Command Registry с self-registration

**Проблема:** Switch в main.go нарушает OCP.

**Решение:** Registry pattern с регистрацией через init().

```go
// internal/command/registry.go
var registry = make(map[string]Handler)

func Register(h Handler) {
    registry[h.Name()] = h
}

// internal/command/handlers/servicemode/status.go
func init() {
    command.Register(&StatusHandler{})
}
```

**Преимущества:**
- Новые команды добавляются без изменения main.go
- Автоматическая генерация help из registry
- Поддержка deprecated-алиасов для миграции

#### 2.1.3. Role-based Interface Segregation

**Проблема:** Широкие интерфейсы требуют реализации неиспользуемых методов.

**Решение:** Разделение на role-based интерфейсы.

```go
// Вместо:
type GiteaClient interface {
    GetPR()
    CreatePR()
    GetCommits()
    GetFile()
    // ...ещё 10 методов
}

// Станет:
type PRReader interface { GetPR() }
type CommitReader interface { GetCommits() }
type FileReader interface { GetFile() }
```

#### 2.1.4. Strategy Pattern для сменных реализаций

**Проблема:** Жёсткая привязка к конкретным инструментам (1cv8).

**Решение:** Strategy + Factory для выбора реализации через конфигурацию.

```go
// Выбор через config:
// implementations:
//   config_export: "ibcmd"  # или "1cv8" или "native"

type ConfigExporter interface {
    Export(ctx context.Context, opts ExportOptions) error
}

func (f *Factory) NewConfigExporter() ConfigExporter {
    switch f.config.Implementations.ConfigExport {
    case "ibcmd":
        return ibcmd.NewExporter()
    default:
        return onecv8.NewExporter()
    }
}
```

#### 2.1.5. Structured Output + Errors

**Проблема:** Результаты команд не структурированы, сложно интегрировать с автоматизацией.

**Решение:** OutputWriter интерфейс с поддержкой text/JSON/YAML.

```json
{
  "status": "success",
  "command": "service-mode-status",
  "data": { "enabled": false, "sessions_count": 5 },
  "metadata": { "duration_ms": 150, "trace_id": "abc123" }
}
```

### 2.2. Обоснование выбора подхода

| Альтернатива | Причина отклонения |
|--------------|-------------------|
| Полный rewrite | Высокий риск, длительный период без production value |
| Микросервисы | Overkill для CLI-инструмента, усложнение deployment |
| Runtime DI (dig/fx) | Ошибки в runtime, reflection overhead |
| Без миграции (только новые команды) | Накопление технического долга в legacy коде |

---

## 3. Краткое описание этапов рефакторинга

### Фаза 1: Архитектурный фундамент (Epic 1)

**Цель:** Создать инфраструктуру для новых команд.

**Результаты:**
- Command Registry с self-registration
- Wire DI setup с базовыми providers
- OutputWriter (text/JSON)
- Logger interface + slog adapter с trace_id
- Первая NR-команда (nr-version) в production

**Критерий завершения:** nr-version работает в production pipeline.

### Фаза 2: Proof of Concept (Epic 2)

**Цель:** Валидация архитектуры на реальной команде.

**Результаты:**
- nr-service-mode-status с информацией о сессиях
- nr-service-mode-enable с force disconnect
- nr-service-mode-disable
- Idempotent state-aware execution

**Критерий завершения:** service-mode команды используются в 3+ production pipelines.

### Фаза 3: Миграция критичных команд (Epics 3-5)

**Цель:** Перенести ключевой функционал на новую архитектуру.

**Epic 3 — Database Operations:**
- nr-dbrestore с auto-timeout и progress bar
- nr-dbupdate
- nr-create-temp-db
- Dry-run режим

**Epic 4 — Configuration Sync:**
- 1C Operations Factory (strategy pattern)
- nr-store2db, nr-storebind, nr-create-stores
- nr-convert, nr-git2store, nr-execute-epf

**Epic 5 — Quality & Integration:**
- SonarQube и Gitea adapters
- nr-sq-scan-branch, nr-sq-scan-pr, nr-sq-report-branch
- nr-test-merge, nr-action-menu-build
- Command summary для всех команд

**Критерий завершения:** Все 17 NR-команд проходят тесты.

### Фаза 4: Advanced Observability (Epic 6)

**Цель:** Полноценная диагностика без доступа к production.

**Результаты:**
- Log file rotation
- Email/Telegram/Webhook алертинг
- Prometheus метрики
- OpenTelemetry трассировка
- Delve debugging support

**Критерий завершения:** Проблемы диагностируются по логам/трейсам без доступа к серверу.

### Фаза 5: Финализация (Epic 7)

**Цель:** Завершение миграции и cleanup.

**Результаты:**
- Shadow-run режим для сравнения legacy и NR
- Smoke-тесты на реальных конфигурациях
- Migration script для пайплайнов
- Удаление deprecated кода
- ADR management process

**Критерий завершения:** Deprecated код удалён, все pipelines на NR-командах.

---

## 4. Выгоды от рефакторинга

### 4.1. Бизнес-выгоды

| Выгода | Описание | Измеримый эффект |
|--------|----------|------------------|
| **Снижение Time-to-Market** | Новый функционал через создание модуля без изменения ядра | -50% времени на добавление команды |
| **Уменьшение рисков** | Тесты + shadow-run + canary deployment | -70% production incidents |
| **Улучшение диагностики** | Структурированные логи с trace_id | -80% времени на поиск причины |
| **Интеграционная гибкость** | JSON/YAML output | Интеграция с любыми системами автоматизации |

### 4.2. Технические выгоды

| Аспект | До рефакторинга | После рефакторинга |
|--------|-----------------|-------------------|
| **Добавление команды** | Изменение main.go + app.go | Создание 1 файла handler |
| **Тестирование** | Mock всего приложения | Mock отдельных интерфейсов |
| **Смена реализации** | Изменение кода | Изменение конфигурации |
| **Покрытие тестами** | < 50% | Цель: 80%+ |
| **Cyclomatic complexity** | > 10 | Цель: < 10 |

### 4.3. Операционные выгоды

1. **Progress bar** для долгих операций (dbrestore, git2store)
2. **Dry-run режим** для безопасной проверки команд
3. **Session info** в service-mode-status
4. **Summary** с ключевыми метриками после каждой команды
5. **Алертинг** о проблемах в Telegram/email

### 4.4. Платформенный потенциал

После рефакторинга benadis-runner становится **платформой**, а не просто набором утилит:

- **AI-анализ кода** — добавить подсистему AI без изменения ядра
- **Анализ пользовательских логов** — новый модуль использует существующий доступ к базам
- **Web Interface** — команды уже возвращают структурированный JSON
- **Plugin-система** — архитектура готова к динамической загрузке модулей

---

## 5. Риски и митигация

### 5.1. Критические риски

| Риск | Вероятность | Влияние | Митигация |
|------|-------------|---------|-----------|
| **NR ломает production** | Средняя | Высокое | Shadow-run + canary deployment + мгновенный rollback |
| **Затягивание рефакторинга** | Средняя | Высокое | Gate: 3 NR-команды в production до массовой миграции |
| **Регрессии в DB operations** | Средняя | Критическое | IsProduction проверка, НИКОГДА restore В production |

### 5.2. Мониторинг успешности

- Количество NR-команд в production (цель: все 17)
- Test coverage (цель: 80%+)
- Количество incidents из-за NR-команд (цель: 0)
- Время добавления новой команды (цель: < 4 часов)

---

## 6. Заключение

Рефакторинг benadis-runner — это трансформация из набора утилит в расширяемую платформу для 1C DevOps. Поэтапный подход с NR-миграцией минимизирует риски и обеспечивает непрерывную работу существующего функционала.

**Ключевые архитектурные решения:**
1. Wire для compile-time DI
2. Command Registry для OCP-совместимости
3. Role-based interfaces для ISP
4. Strategy pattern для сменных реализаций
5. Structured output для интеграции

**Ожидаемые результаты:**
- 50% сокращение времени на добавление функционала
- 80%+ покрытие тестами
- Диагностика проблем без доступа к production
- Платформа, готовая к расширению (AI, Web UI, Plugins)

---

_Документ подготовлен на основе анализа PRD, Architecture, Epics и технических исследований проекта._
