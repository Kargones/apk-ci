# Нарушения принципов SOLID: анализ текущей архитектуры

Документ фиксирует примеры нарушений SRP, OCP, LSP, ISP, DIP в кодовой базе и даёт краткие рекомендации.

## SRP (Single Responsibility Principle)

- `internal/app/app.go`
  - Примеры: функции `SQScanBranch`, `SQScanPR`, `SQReportBranch` выполняют одновременно:
    - принятие бизнес-решений (нужно ли сканировать ветку: `shouldRunScanBranch`, `hasRelevantChangesInCommit`),
    - инфраструктурные операции (создание временных директорий, клонирование репозитория),
    - оркестрацию сервисов (инициализация и вызов `InitSonarQubeServices`, обработка ошибок).
  - Рекомендация: выделить подготовку окружения (clone/temp dirs) в отдельные инфраструктурные сервисы; бизнес-решение — в сервис уровня домена; orchestration — в отдельный командный обработчик.

- `internal/config/config.go`
  - Пример: `CreateGiteaAPI(cfg *Config)` — конфигурационный модуль создаёт конкретную реализацию клиента Gitea. Это смешение обязанностей: загрузка конфигурации vs. фабрика внешних клиентов.
  - Рекомендация: вынести фабрики клиентов в отдельный `internal/factory` или `internal/di` слой.

## OCP (Open-Closed Principle)

- `cmd/apk-ci/main.go`
  - Диспетчеризация команд через `switch cfg.Command` требует правок при добавлении новой команды.
  - Рекомендация: зарегистрировать команды через карту или реестр обработчиков (плагинообразный подход), чтобы расширение происходило без изменений существующего кода.

- `internal/app/sonarqube_init.go`
  - Жёсткое конструирование зависимостей конкретными типами (`NewEntity`, `NewSonarScannerEntity`, сервисы). Добавление новой стратегии (например, ретраи, кеши, другой транспорт) требует изменения функции.
  - Рекомендация: использовать абстракции и провайдеры (factory/DI), принимать интерфейсы и конфигурируемые билдеры.

## LSP (Liskov Substitution Principle)

- Логирование
  - Несмотря на наличие `logging.StructuredLogger`, большинство компонентов принимают `*slog.Logger`. Это усложняет подмену на любой другой совместимый логгер и ограничивает соблюдение LSP.
  - Рекомендация: принимать `logging.StructuredLogger` в конструкторах и полях сервисов; адаптировать `slog` через `SlogAdapter`.

- Gitea API
  - В тестах используется `gitea.API` как заглушка, но интерфейс API очень широк, что затрудняет корректные подстановки (моки должны реализовывать весь интерфейс).
  - Рекомендация: сегрегировать интерфейс на узкие роли (чтение PR, коммиты, файлы и т.д.), использовать композицию.

## ISP (Interface Segregation Principle)

- `internal/entity/gitea/interfaces.go` — `APIInterface` включает множество разнотипных операций (Issues, Files, Commit History, PR, Batch, Teams, Branches).
  - Рекомендация: разделить на специализированные интерфейсы: `PRReader`, `BranchReader`, `CommitHistory`, `RepoFiles`, `TeamDirectory`, `RepoBatchWriter` и т.д. Сервисы должны зависеть от минимально необходимого набора.

- `internal/entity/sonarqube/interfaces.go` — `APIInterface` агрегирует большой набор операций (Projects, Analyses, Issues, Metrics, Profiles, Gates, Rules).
  - Рекомендация: выделить интерфейсы по областям: `ProjectsAPI`, `AnalysesAPI`, `IssuesAPI`, `MetricsAPI`, `QualityProfilesAPI`, `QualityGatesAPI`, `RulesAPI`.

## DIP (Dependency Inversion Principle)

- Конструирование зависимостей на верхнем уровне:
  - `internal/app/sonarqube_init.go` создаёт конкретные реализации SonarQube Entity и Scanner Entity. Верхний уровень зависит от деталей реализации, а не от абстракций.
  - `internal/config.CreateGiteaAPI` создаёт конкретный `gitea.API`, что привязывает слой конфигурации к деталям.
  - Рекомендация: перейти к DI/Factory подходу, где верхний слой получает зависимости через интерфейсы, а создание конкретных реализаций — в отдельном провайдере.

- Логирование
  - Сервисы используют `*slog.Logger`. Это прямое зависимое от `slog`. Следует принимать `logging.StructuredLogger`.

## Примеры из кода (ссылки)

- `internal/app/sonarqube_init.go` — конструирование зависимостей SonarQube, нарушение DIP/OCP.
- `internal/config/config.go:349-364` — `CreateGiteaAPI`, нарушение SRP/DIP.
- `internal/app/app.go` — `SQScanBranch` (бизнес-логика + инфраструктура), нарушение SRP.
- `internal/service/sonarqube/command_handler.go` — поле `logger *slog.Logger`, ограничивает соблюдение LSP/DIP.

## Краткие рекомендации

1. Ввести уровень DI/провайдеров с интерфейсами и минимальными фабриками.
2. Сегрегировать широкие интерфейсы Gitea и SonarQube на роли.
3. Перевести сервисы на `logging.StructuredLogger` и убрать прямую зависимость от `*slog.Logger`.
4. Заменить `switch` в `cmd/main` на реестр команд.
5. Разделить функции `app.go` на отдельные слои: окружение, бизнес-решение, оркестрация.