# Документация проекта apk-ci

> **Дата генерации:** 2025-11-24 (обновлено: Exhaustive Rescan)
> **Режим сканирования:** Exhaustive Scan (full_rescan)
> **Версия workflow:** 1.2.0
> **Миграция в BMAD v6:** 2025-12-24

## Обзор проекта

| Параметр | Значение |
|----------|----------|
| **Тип** | Монолит (CLI инструмент) |
| **Язык** | Go 1.25.1 |
| **Архитектура** | Clean Architecture + Command Pattern |

### Краткое описание

`apk-ci` — инструмент автоматизации для систем 1C:Enterprise. Оркестрирует CI/CD процессы, управляет сервисным режимом, операциями с базами данных MSSQL, синхронизацией Git ↔ хранилище 1C, и интегрируется с SonarQube.

### Технологический стек

- **Язык**: Go 1.25.1
- **База данных**: MSSQL (go-mssqldb)
- **Конфигурация**: cleanenv
- **Тестирование**: testify, sqlmock
- **Сборка**: Make, golangci-lint v2

## Артефакты планирования (текущий каталог)

### Основные документы

- [Обзор проекта](./project-overview.md) — общая информация, команды, интеграции
- [Архитектура](./architecture.md) — слои, паттерны, диаграммы
- [PRD](./prd.md) — требования к продукту
- [Анализ структуры кода](./source-tree-analysis.md) — дерево проекта, критические директории
- [Эпики и истории](./epics.md) — декомпозиция задач на эпики и user stories

### Эпики (в работе)

- [Epic 0: Extension Publish](./epics/epic-0-extension-publish.md) — автопубликация расширений
- [Epic 1: Foundation](./epics/epic-1-foundation.md) — архитектурный фундамент
- [Epic 2: Service Mode](./epics/epic-2-service-mode.md) — управление сервисным режимом
- [Epic 3: DB Operations](./epics/epic-3-db-operations.md) — операции с БД
- [Epic 4: Config Sync](./epics/epic-4-config-sync.md) — синхронизация конфигураций
- [Epic 5: Quality Integration](./epics/epic-5-quality-integration.md) — интеграция качества
- [Epic 6: Observability](./epics/epic-6-observability.md) — наблюдаемость
- [Epic 7: Finalization](./epics/epic-7-finalization.md) — финализация

### Исследования

- [Брейншторминг](./brainstorming-session-results-2025-11-24.md) — результаты мозгового штурма
- [Техническое исследование](./research-technical-2025-11-25.md) — технический анализ

### Проверка готовности

- [Implementation Readiness Report](./implementation-readiness-report-2025-12-08.md) — отчёт о готовности к реализации

### Метаданные

- [Отчёт о сканировании](./project-scan-report.json) — состояние workflow, прогресс

## Артефакты реализации

- [Sprint Status](../implementation-artifacts/sprint-artifacts/sprint-status.yaml) — статус спринта
- [Stories](../implementation-artifacts/stories/) — user stories

## Существующая документация

### Wiki (Русский)

- [Главная](../../.wiki/README.md)
- [Введение](../../.wiki/Введение.md)
- [Справочник команд](../../.wiki/Справочник-команд/README.md)
- [Руководство по конфигурации](../../.wiki/Руководство-по-конфигурации/README.md)
- [Основные модули](../../.wiki/Основные-модули/README.md)
- [Устранение неполадок](../../.wiki/Устранение-неполадок.md)

### Wiki (English)

- [Tool Overview](../../.qoder/repowiki/en/content/Tool%20Overview%20%26%20Core%20Value.md)
- [Installation Guide](../../.qoder/repowiki/en/content/Installation%20Guide.md)
- [Command Reference](../../.qoder/repowiki/en/content/Command%20Reference/Command%20Reference.md)
- [Configuration Management](../../.qoder/repowiki/en/content/Configuration%20Management/Configuration%20Management.md)
- [Advanced Integrations](../../.qoder/repowiki/en/content/Advanced%20Integrations/Advanced%20Integrations.md)
- [Development Guide](../../.qoder/repowiki/en/content/Development%20Guide.md)
- [Troubleshooting](../../.qoder/repowiki/en/content/Troubleshooting.md)

### Архитектура и качество

- [ADR: DI и разделение интерфейсов](../../docs/architecture/adr/0001-di-and-interface-segregation.md)
- [SOLID нарушения](../../docs/architecture/solid-violations.md)
- [Карта зависимостей](../../docs/architecture/dependency-map.md)
- [SOLID чеклист](../../docs/checklists/solid-compliance.md)
- [Стратегия тестирования](../../docs/quality/testing-strategy.md)
- [Руководство по разработке](../../docs/guides/development-guide.md)

### AI инструкции

- [CLAUDE.md](../../CLAUDE.md) — инструкции для Claude Code
- [GEMINI.md](../../GEMINI.md) — инструкции для Gemini
- [QWEN.md](../../QWEN.md) — инструкции для Qwen

### DevOps инструкции

- [Обновление версии платформы 1С/EDT](../../docs/Инструкции/Инструкция%20по%20обновлению%20версии%20платформы%201С%20или%20EDT%20в%20DevOps.md)
- [Инструкция релиз-менеджера GitOps](../../docs/Инструкции/Инструкция%20релиз-менеджера%20gitops.md)

## Быстрый старт

### Сборка

```bash
make build
```

### Запуск команды

```bash
BR_COMMAND=service-mode-status BR_INFOBASE_NAME=MyInfobase ./build/apk-ci
```

### Тестирование

```bash
make test
```

### Проверка качества

```bash
make check
```

## Доступные команды (18)

| Команда | Описание |
|---------|----------|
| `extension-publish` | Автопубликация расширений в подписанные репозитории |
| `service-mode-enable` | Включить сервисный режим |
| `service-mode-disable` | Отключить сервисный режим |
| `service-mode-status` | Статус сервисного режима |
| `dbrestore` | Восстановить MSSQL из бэкапа |
| `dbupdate` | Обновить структуру БД |
| `create-temp-db` | Создать временную БД |
| `store2db` | Загрузить из хранилища в БД |
| `storebind` | Привязать хранилище к БД |
| `git2store` | Синхронизация Git → хранилище |
| `create-stores` | Создать хранилища |
| `convert` | Конвертация EDT ↔ XML |
| `execute-epf` | Выполнить внешнюю обработку |
| `action-menu-build` | Построить меню действий |
| `sq-scan-branch` | Сканировать ветку SonarQube |
| `sq-scan-pr` | Сканировать PR SonarQube |
| `sq-project-update` | Обновить проект SonarQube |
| `sq-report-branch` | Отчёт по ветке SonarQube |
| `test-merge` | Проверить конфликты слияния |

## Следующие шаги

Для работы со спринтом:

```
/bmad:bmm:workflows:dev-story
```

---

*Документация сгенерирована с помощью BMAD Document Project Workflow v1.2.0*
*Мигрирована в BMAD v6 формат: 2025-12-24*
