# Карта зависимостей и оценка связанности модулей

Этот документ описывает текущие зависимости между ключевыми компонентами проекта и оценивает степень их связанности с точки зрения архитектуры.

## Обзор основных пакетов

- `cmd/apk-ci/main.go`
  - Точка входа. Выполняет `config.MustLoad()` и диспетчеризует команды через `switch cfg.Command` к функциям `internal/app`.

- `internal/app`
  - Оркестрация команд и интеграций.
  - `sonarqube_init.go`: функция `InitSonarQubeServices` конструирует конкретные реализации SonarQube (`entity/sonarqube.NewEntity`, `NewSonarScannerEntity`) и сервисы (`service/sonarqube.*`), возвращает `SQCommandHandler`.
  - `app.go`: команды `SQScanBranch`, `SQScanPR`, `SQReportBranch`, `SQProjectUpdate`, сервисные операции с базами и пр. Использует `config.CreateGiteaAPI` и `InitSonarQubeServices`.

- `internal/config`
  - Загрузка и хранение конфигураций (`MustLoad`, `AppConfig`, `ProjectConfig`, `SecretConfig` и пр.).
  - Содержит фабричный метод `CreateGiteaAPI(cfg *Config) gitea.APIInterface`, который создаёт реализацию `entity/gitea.API`.

- `internal/entity/gitea`
  - Интерфейсы и конкретная реализация клиента Gitea (`APIInterface`, `API`, `NewGiteaAPI`). Интерфейс широкого охвата (PR, коммиты, файлы, команды, ветки, команды пользователей).

- `internal/entity/sonarqube`
  - Интерфейсы и сущности для SonarQube (`APIInterface`, `SonarScannerInterface`, `SQCommandHandlerInterface`, структуры параметров, моделей).

- `internal/service/sonarqube`
  - Бизнес-логика вокруг SonarQube: `Service`, `SonarScannerService`, `BranchScanningService`, `ProjectManagementService`, `ReportingService`, `SQCommandHandler`.
  - Логирование: `LoggingService` (реализует `logging.StructuredLogger`).

- `internal/service` (Gitea)
  - `gitea_service.go`: `GiteaService` оборачивает `gitea.APIInterface` и `gitea.ProjectAnalyzer`.
  - `gitea_factory.go`: фабрика `GiteaFactory` создает `GiteaService` и конфигурацию.

- `internal/logging`
  - Интерфейсы и адаптеры для структурированного логирования: `StructuredLogger`, `SlogAdapter`, утилиты `NewStructuredLogger` и т.д.

## Карта зависимостей (текстовая)

- `cmd/apk-ci` → `internal/config` (MustLoad), `internal/app` (команды).
- `internal/app` → `internal/config` (данные конфигурации, `CreateGiteaAPI`).
- `internal/app` → `internal/service/sonarqube` (через `InitSonarQubeServices` конструирует сервисы).
- `internal/app` → `internal/entity/sonarqube` (внутри `InitSonarQubeServices` вызывает `NewEntity`, `NewSonarScannerEntity`).
- `internal/app` → `internal/entity/gitea` (через `config.CreateGiteaAPI`).
- `internal/service/sonarqube` → `internal/entity/sonarqube` (интерфейсы API и Scanner).
- `internal/service/sonarqube` → `internal/entity/gitea` (узкий набор возможностей, фактически используется часть методов).
- `internal/service` (Gitea) → `internal/entity/gitea` (APIInterface), `internal/config` (Config).
- `internal/logging` используется точечно, но многие сервисы принимают `*slog.Logger` вместо интерфейса `StructuredLogger`.

## Оценка связанности

- Высокая связанность:
  - `internal/app/sonarqube_init.go` жёстко зависит от конкретных реализаций `entity/sonarqube` и формирует сервисы напрямую.
  - `internal/config.CreateGiteaAPI` жёстко зависит от `entity/gitea`. Конфигурационный слой становится фабрикой клиентов.
  - Сервисы `service/sonarqube` и `SQCommandHandler` повсеместно принимают `*slog.Logger`, что привязывает бизнес-логику к конкретной библиотеке логирования.

- Средняя связанность:
  - `internal/app/app.go` в командах одновременно отвечает за принятие решения (бизнес-условия), подготовку временных директорий, работу с `git` и вызовы сервисов.
  - `service/gitea_factory.go` и `config.CreateGiteaAPI` дублируют ответственность создания клиента Gitea.

- Низкая связанность:
  - `internal/service/sonarqube` относительно хорошо изолирует бизнес-операции через интерфейсы entity-слоя.

## Ключевые точки сопряжения

- Создание зависимостей:
  - Gitea API создаётся в `config.CreateGiteaAPI` и также через `service.NewGiteaFactory().CreateGiteaService` (неоднородность паттерна).
  - SonarQube сервисы создаются в `internal/app/InitSonarQubeServices` (прямые вызовы конструкторов сущностей и сервисов).

- Логирование:
  - Есть интерфейс `logging.StructuredLogger`, но большинство компонентов используют `*slog.Logger` напрямую, обходя адаптер.

## Выводы

- Внешние клиенты (Gitea, SonarQube) создаются ближе к уровню `app`/`config`, что повышает связанность верхнего уровня с деталями реализации и нарушает DIP.
- Широкие интерфейсы `gitea.APIInterface` и `sonarqube.APIInterface` затрудняют изоляцию зависимостей для конкретных кейсов (ISP).
- Логирование реализовано как интерфейс, но фактически используется конкретный тип (`*slog.Logger`), что препятствует подмене (LSP/DIP).

## Риски

- Добавление новых команд требует изменения `cmd/main.go` (нарушение OCP).
- Расширение Gitea/SonarQube возможностей приводит к изменению зависимых модулей из-за широких интерфейсов и прямых конструкторов.