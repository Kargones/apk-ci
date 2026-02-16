# Анализ структуры исходного кода

## Дерево проекта

```
benadis-runner/
├── cmd/
│   └── benadis-runner/
│       ├── main.go                    # Точка входа, маршрутизатор команд
│       ├── main_test.go               # Тесты main
│       ├── create_temp_db_test.go     # Тесты создания временной БД
│       └── yaml_integration_test.go   # Интеграционные тесты YAML
│
├── internal/                          # Внутренняя бизнес-логика
│   ├── app/                           # Слой приложения (оркестрация)
│   │   ├── app.go                     # Основные функции команд
│   │   ├── app_test.go
│   │   ├── action_menu_build.go       # Построение меню действий
│   │   ├── sonarqube_init.go          # Инициализация SonarQube
│   │   └── execute_epf_test.go
│   │
│   ├── config/                        # Система конфигурации
│   │   ├── config.go                  # Загрузка и валидация
│   │   ├── config_test.go
│   │   ├── sonarqube.go               # Конфиг SonarQube
│   │   └── sonarqube_test.go
│   │
│   ├── constants/                     # Константы приложения
│   │   ├── constants.go               # Все команды и константы
│   │   └── constants_test.go
│   │
│   ├── service/                       # Сервисный слой
│   │   ├── gitea_service.go           # Сервис Gitea
│   │   ├── gitea_factory.go           # Фабрика Gitea
│   │   ├── config_analyzer.go         # Анализатор конфигурации
│   │   └── *_test.go
│   │
│   ├── entity/                        # Доменные сущности
│   │   ├── one/                       # 1C:Enterprise сущности
│   │   │   ├── convert/               # Конвертация форматов
│   │   │   │   ├── convert.go         # Основной workflow
│   │   │   │   └── convert_test.go
│   │   │   ├── designer/              # Операции Designer
│   │   │   │   ├── designer.go        # Создание БД, выгрузка
│   │   │   │   └── designer_test.go
│   │   │   ├── edt/                   # EDT интеграция
│   │   │   │   ├── edt.go
│   │   │   │   └── edt_test.go
│   │   │   ├── enterprise/            # Режим Enterprise
│   │   │   │   ├── enterprise.go      # Выполнение EPF
│   │   │   │   └── enterprise_test.go
│   │   │   └── store/                 # Хранилище конфигурации
│   │   │       ├── store.go
│   │   │       └── store_test.go
│   │   │
│   │   ├── dbrestore/                 # Восстановление MSSQL
│   │   │   ├── dbrestore.go
│   │   │   └── dbrestore_test.go
│   │   │
│   │   ├── gitea/                     # Gitea API клиент
│   │   │   ├── gitea.go               # Основной клиент
│   │   │   ├── interfaces.go          # Интерфейсы
│   │   │   └── *_test.go              # Множество тестов
│   │   │
│   │   ├── sonarqube/                 # SonarQube интеграция
│   │   │   ├── sonarqube.go           # Основной модуль
│   │   │   ├── scanner.go             # Сканер
│   │   │   ├── branch_scanner.go      # Сканирование веток
│   │   │   ├── service.go             # Сервис
│   │   │   ├── validator.go           # Валидация
│   │   │   ├── resource_manager.go    # Управление ресурсами
│   │   │   ├── interfaces.go
│   │   │   ├── errors.go
│   │   │   └── *_test.go
│   │   │
│   │   └── filer/                     # Файловые операции
│   │       ├── disk_fs.go             # Дисковая ФС
│   │       ├── memory_fs.go           # In-memory ФС
│   │       ├── memory_file.go         # Memory файл
│   │       ├── interfaces.go          # Интерфейсы
│   │       ├── factory.go             # Фабрика
│   │       ├── temp_manager.go        # Менеджер временных файлов
│   │       ├── utils.go               # Утилиты
│   │       ├── errors.go              # Ошибки
│   │       ├── options.go             # Опции
│   │       ├── types.go               # Типы
│   │       └── *_test.go              # Много тестов (100% coverage)
│   │
│   ├── git/                           # Git операции
│   │   ├── git.go                     # Клонирование, switch, и т.д.
│   │   ├── git_test.go
│   │   └── git_timeout_test.go
│   │
│   ├── rac/                           # RAC (1C консоль администрирования)
│   │   ├── rac.go                     # RAC клиент
│   │   ├── service_mode.go            # Сервисный режим
│   │   └── rac_test.go
│   │
│   ├── logging/                       # Логирование
│   │   ├── adapter.go                 # Адаптер логгера
│   │   ├── utils.go
│   │   └── logging_test.go
│   │
│   └── servicemode/                   # Управление сервисным режимом
│       └── (файлы модуля)
│
├── vendor/                            # Вендоринг зависимостей
│
├── docs/                              # Документация
│   ├── architecture/                  # ADR, SOLID, dependency map
│   ├── guides/                        # Руководства
│   ├── quality/                       # Стратегия тестирования
│   ├── checklists/                    # Чеклисты
│   └── Инструкции/                    # DevOps инструкции
│
├── .wiki/                             # Wiki документация (RU)
├── .qoder/repowiki/en/                # Wiki документация (EN)
├── old/                               # Устаревшая документация
│
├── config/                            # Конфигурационные файлы
├── scripts/                           # Скрипты сборки
├── test/                              # Тестовые данные
│
├── Makefile                           # Сборка и автоматизация
├── go.mod                             # Go модуль
├── go.sum                             # Checksums зависимостей
├── .golangci.yml                      # Конфигурация линтера
├── Taskfile.yml                       # Task runner
│
├── README.md                          # Главный README
├── CLAUDE.md                          # Инструкции для Claude
├── GEMINI.md                          # Инструкции для Gemini
└── QWEN.md                            # Инструкции для Qwen
```

## Критические директории

### `cmd/benadis-runner/`
Точка входа приложения. `main.go` содержит switch-statement для маршрутизации 17 команд через `BR_COMMAND`.

### `internal/app/`
Слой оркестрации. Содержит высокоуровневые функции для каждой команды, координирует работу между сервисами и сущностями.

### `internal/config/`
Многоуровневая система конфигурации. Загружает данные из Gitea API во время выполнения, поддерживает переменные окружения с разными префиксами.

### `internal/entity/one/`
Сущности для работы с 1C:Enterprise:
- **convert** - конвертация EDT ↔ XML
- **designer** - операции Designer (создание БД, выгрузка)
- **edt** - интеграция с EDT CLI (ring)
- **enterprise** - режим Enterprise (выполнение EPF)
- **store** - управление хранилищем конфигурации

### `internal/entity/sonarqube/`
Полноценная интеграция с SonarQube: сканирование веток и PR, отчёты, валидация, управление ресурсами.

### `internal/entity/filer/`
Абстракция файловой системы с реализациями для диска и памяти. Высокое покрытие тестами.

### `internal/rac/`
Клиент для RAC (консоль администрирования 1C). Управление сервисным режимом, сессиями, кластером.

## Паттерны кода

### Функции оркестрации
Каждая команда имеет соответствующую функцию в `internal/app/app.go`:
```go
func ServiceModeEnable(ctx *context.Context, l *slog.Logger, cfg *config.Config, infobaseName string, terminateSessions bool) error
```

### Обработка ошибок
Структурированное логирование с slog:
```go
l.Error("Ошибка включения сервисного режима",
    slog.String("infobase", infobaseName),
    slog.String("error", err.Error()),
)
```

### Тестирование
Табличные тесты, моки через интерфейсы, sqlmock для БД:
```go
func TestServiceModeEnable(t *testing.T) {
    tests := []struct {
        name    string
        // ...
    }{
        // test cases
    }
}
```
