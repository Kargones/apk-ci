# API документация

Данный документ содержит подробную документацию по всем экспортируемым функциям, структурам и интерфейсам benadis-runner на основе Go Doc комментариев.

## Обзор API

benadis-runner предоставляет программный интерфейс для интеграции с различными компонентами системы 1С:Предприятие. API разделен на несколько модулей, каждый из которых отвечает за определенную область функциональности.

### Основные модули API

- **App Layer** (`internal/app`) — высокоуровневые функции приложения
- **Config Management** (`internal/config`) — управление конфигурацией
- **1C Integration** (`internal/entity/one`) — интеграция с платформой 1С
- **RAC Operations** (`internal/rac`) — операции с консолью удаленного администрирования
- **Gitea Integration** (`internal/entity/gitea`) — интеграция с Gitea API
- **Service Mode** (`internal/servicemode`) — управление сервисным режимом
- **Utilities** (`internal/util`) — вспомогательные утилиты

## App Layer API

### Функции высокого уровня

#### Git2Store
```go
func Git2Store(ctx *context.Context, l *slog.Logger, cfg *config.Config) error
```
Выполняет синхронизацию изменений из Git-репозитория в хранилище конфигураций 1С.

**Параметры:**
- `ctx` — контекст выполнения операции
- `l` — логгер для записи сообщений
- `cfg` — конфигурация приложения

**Возвращает:**
- `error` — ошибка выполнения или nil при успехе

**Процесс выполнения:**
1. Клонирование Git-репозитория
2. Создание временной информационной базы
3. Загрузка конфигурации из файлов
4. Привязка к хранилищу конфигураций
5. Объединение изменений и сохранение в хранилище

#### Convert
```go
func Convert(ctx *context.Context, l *slog.Logger, cfg *config.Config) error
```
Выполняет конвертацию конфигураций между различными форматами.

**Параметры:**
- `ctx` — контекст выполнения операции
- `l` — логгер для записи сообщений и ошибок
- `cfg` — конфигурация приложения с настройками проекта

**Возвращает:**
- `error` — ошибка конвертации или nil при успехе

#### Store2DbWithConfig
```go
func Store2DbWithConfig(ctx *context.Context, l *slog.Logger, cfg *config.Config) error
```
Выполняет загрузку конфигурации из хранилища 1С в информационную базу.

#### DbUpdateWithConfig
```go
func DbUpdateWithConfig(ctx *context.Context, l *slog.Logger, cfg *config.Config) error
```
Выполняет обновление структуры базы данных 1C с использованием указанного файла конфигурации. Расширенная версия DbUpdate, позволяющая использовать пользовательский файл конфигурации для более гибкого управления процессом обновления структуры базы данных.

**Особенности:**
- Инициализирует сетевой ключ защиты HASP
- Выполняет обновление дважды для корректного обновления расширений

#### DbRestoreWithConfig
```go
func DbRestoreWithConfig(ctx *context.Context, l *slog.Logger, cfg *config.Config, dbname string) error
```
Выполняет восстановление информационной базы из резервной копии.

**Параметры:**
- `dbname` — имя восстанавливаемой базы данных

### Функции сервисного режима

#### ServiceModeEnable
```go
func ServiceModeEnable(ctx *context.Context, l *slog.Logger, cfg *config.Config, infobaseName string, terminateSessions bool) error
```
Включает сервисный режим для указанной информационной базы.

**Параметры:**
- `infobaseName` — имя информационной базы
- `terminateSessions` — флаг принудительного завершения активных сессий

#### ServiceModeDisable
```go
func ServiceModeDisable(ctx *context.Context, l *slog.Logger, cfg *config.Config, infobaseName string) error
```
Отключает сервисный режим для указанной информационной базы.

#### ServiceModeStatus
```go
func ServiceModeStatus(ctx *context.Context, l *slog.Logger, cfg *config.Config, infobaseName string) error
```
Получает текущий статус сервисного режима для указанной информационной базы.

### Вспомогательные функции

#### NetHaspInit
```go
func NetHaspInit(ctx *context.Context, l *slog.Logger)
```
Выполняет инициализацию сетевого ключа защиты HASP для 1C:Enterprise. Настраивает подключение к серверу лицензий HASP и проверяет доступность необходимых лицензий для работы с базой данных 1C.

#### CreateTempDbWrapper
```go
func CreateTempDbWrapper(ctx *context.Context, l *slog.Logger, cfg *config.Config) (string, error)
```
Создает временную информационную базу для тестирования.

**Возвращает:**
- `string` — строка подключения к созданной базе
- `error` — ошибка создания

#### CreateStoresWrapper
```go
func CreateStoresWrapper(ctx *context.Context, l *slog.Logger, cfg *config.Config) error
```
Создает хранилища конфигураций для проекта.

#### StoreBind
```go
func StoreBind(ctx *context.Context, l *slog.Logger, cfg *config.Config) error
```
Выполняет привязку информационной базы к хранилищу конфигураций.

#### ActionMenuBuildWrapper
```go
func ActionMenuBuildWrapper(ctx *context.Context, l *slog.Logger, cfg *config.Config) error
```
Строит интерактивное меню для выбора операций.

## Config Management API

### Основные структуры

#### Config
```go
type Config struct {
    AppConfig     *AppConfig
    ProjectConfig *ProjectConfig
    SecretConfig  *SecretConfig
    DbConfig      *DbConfig
    // ... другие поля
}
```
Основная структура конфигурации, объединяющая все настройки приложения.

#### AppConfig
```go
type AppConfig struct {
    LogLevel string `yaml:"logLevel"`
    WorkDir  string `yaml:"workDir"`
    TmpDir   string `yaml:"tmpDir"`
    Timeout  int    `yaml:"timeout"`
    Paths    struct {
        Bin1cv8  string `yaml:"bin1cv8"`
        BinIbcmd string `yaml:"binIbcmd"`
        EdtCli   string `yaml:"edtCli"`
        Rac      string `yaml:"rac"`
    } `yaml:"paths"`
    // ... другие поля
}
```
Представляет настройки приложения из файла app.yaml. Содержит конфигурацию уровня логирования, рабочих директорий, таймаутов, путей к исполняемым файлам и настроек подключения к различным сервисам.

### Функции конфигурации

#### MustLoad
```go
func MustLoad() (*Config, error)
```
Загружает конфигурацию из файлов и переменных окружения с применением иерархии приоритетов.

## 1C Integration API

### internal/entity/one/designer

#### OneDb
```go
type OneDb struct {
    DbConnectString   string `json:"Строка соединения,omitempty"`
    User              string `json:"Пользователь,omitempty"`
    Pass              string `json:"Пароль,omitempty"`
    FullConnectString string
    ServerDb          bool
    DbExist           bool
}
```
Представляет конфигурацию базы данных 1С:Предприятие. Содержит параметры подключения и настройки для работы с информационной базой.

#### Create
```go
func (odb *OneDb) Create(ctx *context.Context, l *slog.Logger, cfg *config.Config) error
```
Создает новую информационную базу 1С:Предприятие. Выполняет создание базы данных с указанными параметрами подключения.

**Выполняемая команда:**
```bash
ibcmd infobase create --create-database --db-path="/path/to/database"
```

#### Load
```go
func (odb *OneDb) Load(ctx context.Context, l *slog.Logger, cfg *config.Config, sourcePath string, extensionName ...string) error
```
Загружает конфигурацию из файлов в информационную базу. Может загружать как основную конфигурацию, так и расширения при указании имени расширения.

**Параметры:**
- `sourcePath` — путь к исходным файлам конфигурации
- `extensionName` — опциональное имя расширения для загрузки

#### UpdateCfg
```go
func (odb *OneDb) UpdateCfg(ctx context.Context, l *slog.Logger, cfg *config.Config, extensionName ...string) error
```
Обновляет конфигурацию базы данных, применяя изменения структуры метаданных.

#### Dump
```go
func (odb *OneDb) Dump(ctx context.Context, l *slog.Logger, cfg *config.Config, targetPath string, extensionName ...string) error
```
Выгружает конфигурацию в файл формата .cf.

#### DumpToFiles
```go
func (odb *OneDb) DumpToFiles(ctx context.Context, l *slog.Logger, cfg *config.Config, targetPath string, extensionName ...string) error
```
Выгружает конфигурацию в файлы исходного кода.

#### Add
```go
func (odb *OneDb) Add(ctx context.Context, l *slog.Logger, cfg *config.Config, nameAdd string) error
```
Добавляет новое расширение в информационную базу. Создает расширение с указанным именем в существующей базе данных.

**Параметры:**
- `nameAdd` — имя создаваемого расширения

#### GetDbName
```go
func GetDbName(ctx *context.Context, l *slog.Logger, cfg *config.Config) (string, error)
```
Получает имя информационной базы из конфигурации.

### internal/entity/one/store

#### Store
```go
type Store struct {
    Name    string `json:"Имя хранилища,omitempty"`
    Path    string `json:"Относительный путь"`
    User    string `json:"Пользователь"`
    Pass    string `json:"Пароль"`
    Command string `json:"-"`
}
```
Представляет хранилище конфигурации 1С. Содержит информацию о подключении к хранилищу конфигураций, включая имя, путь, учетные данные и команды для работы с хранилищем.

#### StoreRecord
```go
type StoreRecord struct {
    Version     int       `json:"Версия хранилища"`
    ConfVersion string    `json:"Версия конфигурации"`
    User        string    `json:"Пользователь"`
    Date        time.Time `json:"Дата время"`
    Comment     string    `json:"Комментарий,omitempty"`
}
```
Представляет запись в хранилище конфигураций. Содержит информацию о версии конфигурации, пользователе, дате создания и комментарии к изменениям в хранилище.

#### ReadReport
```go
func (s *Store) ReadReport(l *slog.Logger, dbConnectString string, cfg *config.Config, startVersion int) ([]StoreRecord, []User, int, error)
```
Получает отчет хранилища конфигураций, включая историю изменений и список пользователей.

**Параметры:**
- `startVersion` — начальная версия для отчета

**Возвращает:**
- `[]StoreRecord` — список записей хранилища
- `[]User` — список пользователей
- `int` — максимальная версия
- `error` — ошибка выполнения

#### ParseReport
```go
func ParseReport(path string) ([]StoreRecord, []User, int, error)
```
Парсит текстовый отчет хранилища и возвращает структурированные данные.

### internal/entity/one/convert

#### ConvertConfig
```go
type ConvertConfig struct {
    StoreRoot   string         `json:"Корень хранилища"`
    OneDB       designer.OneDb `json:"Параметры подключения"`
    ConvertPair []ConvertPair  `json:"Сопоставления"`
}
```
Представляет конфигурацию для процесса конвертации данных. Содержит параметры источника, назначения и правила преобразования.

#### ConvertPair
```go
type ConvertPair struct {
    Source Source      `json:"Источник"`
    Store  store.Store `json:"Хранилище"`
}
```
Представляет пару "источник-назначение" для операции конвертации. Определяет связь между исходными и целевыми данными.

#### LoadFromConfig
```go
func LoadFromConfig(ctx *context.Context, l *slog.Logger, cfg *config.Config) (*ConvertConfig, error)
```
Загружает конфигурацию конвертации из основной конфигурации приложения.

#### LoadConfigFromData
```go
func LoadConfigFromData(ctx *context.Context, l *slog.Logger, data []byte) (*ConvertConfig, error)
```
Загружает конфигурацию конвертации из JSON-данных.

## RAC Operations API

### internal/rac

#### Client
```go
type Client struct {
    Server     string
    Port       int
    User       string
    Password   string
    DbUser     string
    DbPassword string
    Timeout    time.Duration
    Logger     *slog.Logger
}
```
Клиент для работы с консолью удаленного администрирования 1С (RAC).

#### ServiceModeStatus
```go
type ServiceModeStatus struct {
    Enabled        bool
    Message        string
    ActiveSessions int
}
```
Представляет статус сервисного режима.

#### SessionInfo
```go
type SessionInfo struct {
    SessionID    string
    UserName     string
    AppID        string
    StartedAt    time.Time
    LastActiveAt time.Time
}
```
Представляет информацию о сессии.

#### EnableServiceMode
```go
func (c *Client) EnableServiceMode(ctx context.Context, clusterUUID, infobaseUUID string, terminateSessions bool) error
```
Включает сервисный режим для указанной информационной базы. Устанавливает блокировку сессий и регламентных заданий с заданным сообщением.

**Параметры:**
- `clusterUUID` — уникальный идентификатор кластера
- `infobaseUUID` — уникальный идентификатор информационной базы
- `terminateSessions` — флаг принудительного завершения активных сессий

#### DisableServiceMode
```go
func (c *Client) DisableServiceMode(ctx context.Context, clusterUUID, infobaseUUID string) error
```
Отключает сервисный режим для указанной информационной базы. Снимает блокировку сессий и регламентных заданий.

#### GetServiceModeStatus
```go
func (c *Client) GetServiceModeStatus(ctx context.Context, clusterUUID, infobaseUUID string) (*ServiceModeStatus, error)
```
Получает текущий статус сервисного режима для указанной информационной базы. Возвращает информацию о состоянии блокировки, сообщении и количестве активных сессий.

#### GetSessions
```go
func (c *Client) GetSessions(ctx context.Context, clusterUUID, infobaseUUID string) ([]SessionInfo, error)
```
Получает список активных сессий для указанной информационной базы.

#### TerminateSession
```go
func (c *Client) TerminateSession(ctx context.Context, clusterUUID, sessionUUID string) error
```
Завершает конкретную сессию по ее идентификатору.

#### TerminateAllSessions
```go
func (c *Client) TerminateAllSessions(ctx context.Context, clusterUUID, infobaseUUID string) error
```
Завершает все активные сессии для указанной информационной базы.

## Gitea Integration API

### internal/entity/gitea

#### Client
```go
type Client struct {
    BaseURL string
    Token   string
    Owner   string
    Repo    string
}
```
Клиент для работы с Gitea API.

#### Repository
```go
type Repository struct {
    ID          int    `json:"id"`
    Name        string `json:"name"`
    FullName    string `json:"full_name"`
    Description string `json:"description"`
}
```
Информация о репозитории.

#### PullRequest
```go
type PullRequest struct {
    ID     int    `json:"id"`
    Number int    `json:"number"`
    Title  string `json:"title"`
    State  string `json:"state"`
}
```
Представляет Pull Request в Gitea.

#### GetRepository
```go
func (c *Client) GetRepository(ctx context.Context) (*Repository, error)
```
Получает информацию о репозитории.

#### CreatePR
```go
func (c *Client) CreatePR(ctx context.Context, title, head, base, body string) (*PullRequest, error)
```
Создает новый Pull Request.

#### ActivePR
```go
func (c *Client) ActivePR(ctx context.Context) ([]PullRequest, error)
```
Получает список активных Pull Request'ов.

#### ClosePR
```go
func (c *Client) ClosePR(ctx context.Context, prNumber int) error
```
Закрывает указанный Pull Request.

#### CreateIssueComment
```go
func (c *Client) CreateIssueComment(ctx context.Context, issueNumber int, body string) error
```
Создает комментарий к issue.

## Utilities API

### internal/util/runner

#### Runner
```go
type Runner struct {
    RunString  string   // Путь к исполняемому файлу
    Params     []string // Параметры командной строки
    WorkDir    string   // Рабочая директория
    TmpDir     string   // Временная директория
    ConsoleOut []byte   // Вывод в консоль
    FileOut    []byte   // Вывод в файл
}
```
Исполнитель внешних команд с возможностью захвата вывода.

#### RunCommand
```go
func (r *Runner) RunCommand(l *slog.Logger) ([]byte, error)
```
Выполняет команду с заданными параметрами и возвращает результат.

### internal/util/template_processor

#### TemplateResult
```go
type TemplateResult struct {
    Name    string
    Content string
}
```
Результат обработки шаблона.

#### ProcessTemplates
```go
func (tp *TemplateProcessor) ProcessTemplates(ctx context.Context, templates []string, data interface{}) ([]TemplateResult, error)
```
Обрабатывает шаблоны конфигурации с подстановкой данных.

## Примеры использования API

### Создание и настройка информационной базы

```go
package main

import (
    "context"
    "log/slog"
    "github.com/Kargones/apk-ci/internal/config"
    "github.com/Kargones/apk-ci/internal/entity/one/designer"
)

func createAndSetupDatabase() error {
    ctx := context.Background()
    cfg, err := config.MustLoad()
    if err != nil {
        return err
    }
    
    logger := cfg.Logger
    
    // Создание новой информационной базы
    oneDb := &designer.OneDb{}
    err = oneDb.Create(&ctx, logger, cfg)
    if err != nil {
        return err
    }
    
    // Загрузка конфигурации из файлов
    err = oneDb.Load(ctx, logger, cfg, "/path/to/config/files")
    if err != nil {
        return err
    }
    
    // Обновление конфигурации БД
    err = oneDb.UpdateCfg(ctx, logger, cfg)
    if err != nil {
        return err
    }
    
    return nil
}
```

### Управление сервисным режимом

```go
package main

import (
    "context"
    "log/slog"
    "github.com/Kargones/apk-ci/internal/config"
    "github.com/Kargones/apk-ci/internal/rac"
)

func manageServiceMode() error {
    ctx := context.Background()
    cfg, err := config.MustLoad()
    if err != nil {
        return err
    }
    
    // Создание RAC клиента
    racClient := &rac.Client{
        Server:   "localhost",
        Port:     1545,
        User:     "admin",
        Password: "password",
        Logger:   cfg.Logger,
    }
    
    // Получение UUID кластера и базы
    clusterUUID, err := racClient.GetClusterUUID(ctx)
    if err != nil {
        return err
    }
    
    infobaseUUID, err := racClient.GetInfobaseUUID(ctx, clusterUUID, "MyDatabase")
    if err != nil {
        return err
    }
    
    // Включение сервисного режима с завершением сессий
    err = racClient.EnableServiceMode(ctx, clusterUUID, infobaseUUID, true)
    if err != nil {
        return err
    }
    
    // Получение статуса
    status, err := racClient.GetServiceModeStatus(ctx, clusterUUID, infobaseUUID)
    if err != nil {
        return err
    }
    
    cfg.Logger.Info("Service mode status", 
        "enabled", status.Enabled,
        "message", status.Message,
        "active_sessions", status.ActiveSessions)
    
    return nil
}
```

### Работа с Gitea API

```go
package main

import (
    "context"
    "github.com/Kargones/apk-ci/internal/entity/gitea"
)

func workWithGitea() error {
    ctx := context.Background()
    
    // Создание Gitea клиента
    client := &gitea.Client{
        BaseURL: "https://git.company.ru",
        Token:   "your-api-token",
        Owner:   "myorg",
        Repo:    "myrepo",
    }
    
    // Получение информации о репозитории
    repo, err := client.GetRepository(ctx)
    if err != nil {
        return err
    }
    
    // Создание Pull Request
    pr, err := client.CreatePR(ctx, 
        "Feature: New functionality", 
        "feature-branch", 
        "main", 
        "Description of changes")
    if err != nil {
        return err
    }
    
    // Получение списка активных PR
    activePRs, err := client.ActivePR(ctx)
    if err != nil {
        return err
    }
    
    return nil
}
```

## Обработка ошибок

Все функции API возвращают ошибки в стандартном формате Go. Ошибки оборачиваются с контекстом для лучшей диагностики:

```go
if err != nil {
    return fmt.Errorf("failed to create database %s: %w", dbName, err)
}
```

### Типы ошибок

- **ConfigError** — ошибки конфигурации
- **CommandError** — ошибки выполнения внешних команд
- **NetworkError** — сетевые ошибки
- **ValidationError** — ошибки валидации данных

## Логирование

Все функции API принимают структурированный логгер `*slog.Logger` для записи отладочной информации:

```go
logger.Debug("Starting operation", 
    "operation", "create_database",
    "database", dbName)

logger.Error("Operation failed",
    "operation", "create_database", 
    "error", err.Error())
```

## Контекст выполнения

Все долгосрочные операции поддерживают контекст для отмены и установки таймаутов:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := oneDb.Create(&ctx, logger, cfg)
```

---

*Версия документа: 1.0*  
*Последнее обновление: 2025-08-26*