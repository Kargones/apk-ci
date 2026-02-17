# Инвентаризация структур данных apk-ci

## Обзор

Данный документ содержит полную инвентаризацию всех структур данных, найденных в проекте apk-ci. Структуры сгруппированы по пакетам и функциональному назначению.

## Структуры по пакетам

### 1. internal/config/config.go

#### Основные конфигурационные структуры:

**AppConfig** - Конфигурация приложения
```go
type AppConfig struct {
    LogLevel string `yaml:"logLevel"`
    WorkDir  string `yaml:"workDir"`
    TmpDir   string `yaml:"tmpDir"`
    Timeout  int    `yaml:"timeout"`
    Paths    struct { ... } `yaml:"paths"`
    Rac      struct { ... } `yaml:"rac"`
    Users    struct { ... } `yaml:"users"`
    Dbrestore struct { ... } `yaml:"dbrestore"`
}
```

**ProjectConfig** - Конфигурация проекта
```go
type ProjectConfig struct {
    StoreDb string `yaml:"store-db"`
    Prod    map[string]struct { ... } `yaml:"prod"`
}
```

**SecretConfig** - Конфигурация секретов
```go
type SecretConfig struct {
    Passwords struct { ... } `yaml:"passwords"`
    Gitea     struct { ... } `yaml:"gitea"`
}
```

**DatabaseInfo** - Информация о базе данных
```go
type DatabaseInfo struct {
    OneServer string `yaml:"one-server"`
    Prod      bool   `yaml:"prod"`
    DbServer  string `yaml:"dbserver"`
}
```

**Config** - Главная структура конфигурации (245+ полей)
```go
type Config struct {
    ProjectName string
    AddArray    []string
    Actor       string `env:"BR_ACTOR"`
    Env         string `env:"BR_ENV"`
    Command     string `env:"BR_COMMAND"`
    Logger      *slog.Logger
    // ... множество других полей для различных компонентов
}
```

#### Специализированные конфигурации:

**ConvertConfig** - Конфигурация конвертации
```go
type ConvertConfig struct {
    Branch     string `json:"branch" yaml:"branch"`
    CommitSHA1 string `json:"commitSHA1" yaml:"commitSHA1"`
    OneDB      bool   `json:"oneDB" yaml:"oneDB"`
    DbUser     string `json:"dbUser" yaml:"dbUser"`
    DbPassword string `json:"dbPassword" yaml:"dbPassword"`
    DbServer   string `json:"dbServer" yaml:"dbServer"`
    DbName     string `json:"dbName" yaml:"dbName"`
}
```

**DBRestoreConfig** - Конфигурация восстановления БД
```go
type DBRestoreConfig struct {
    Server      string        `yaml:"server"`
    User        string        `yaml:"user"`
    Password    string        `yaml:"password"`
    Database    string        `yaml:"database"`
    Backup      string        `yaml:"backup"`
    Timeout     time.Duration `yaml:"timeout"`
    SrcServer   string        `yaml:"srcServer"`
    SrcDB       string        `yaml:"srcDB"`
    DstServer   string        `yaml:"dstServer"`
    DstDB       string        `yaml:"dstDB"`
    Autotimeout bool          `yaml:"autotimeout"`
}
```

**ServiceModeConfig** - Конфигурация сервисного режима
```go
type ServiceModeConfig struct {
    RacPath     string        `yaml:"racPath"`
    RacServer   string        `yaml:"racServer"`
    RacPort     int           `yaml:"racPort"`
    RacUser     string        `yaml:"racUser"`
    RacPassword string        `yaml:"racPassword"`
    DbUser      string        `yaml:"dbUser"`
    DbPassword  string        `yaml:"dbPassword"`
    RacTimeout  time.Duration `yaml:"racTimeout"`
    RacRetries  int           `yaml:"racRetries"`
}
```

**EdtConfig** - Конфигурация EDT
```go
type EdtConfig struct {
    EdtCli     string `yaml:"edtCli"`
    Workspace  string `yaml:"workspace"`
    ProjectDir string `yaml:"projectDir"`
}
```

#### API структуры:

**Repo** - Информация о репозитории
```go
type Repo struct {
    DefaultBranch string `json:"default_branch"`
}
```

**FileInfo** - Информация о файле
```go
type FileInfo struct {
    Name        string `json:"name"`
    Path        string `json:"path"`
    SHA         string `json:"sha"`
    Size        int64  `json:"size"`
    URL         string `json:"url"`
    HTMLURL     string `json:"html_url"`
    GitURL      string `json:"git_url"`
    DownloadURL string `json:"download_url"`
    Type        string `json:"type"`
    Content     string `json:"content"`
    Encoding    string `json:"encoding"`
    Target      string `json:"target"`
    Submodule   string `json:"submodule"`
}
```

**GiteaAPI** - API клиент Gitea
```go
type GiteaAPI struct {
    GiteaURL    string
    Owner       string
    Repo        string
    AccessToken string
}
```

### 2. internal/entity/dbrestore/dbrestore.go

**DBRestore** - Клиент восстановления БД
```go
type DBRestore struct {
    Db              *sql.DB
    Server          string        `yaml:"server"`
    User            string        `yaml:"user"`
    Password        string        `yaml:"password"`
    Port            int           `yaml:"port"`
    Database        string        `yaml:"database"`
    Timeout         time.Duration `yaml:"timeout"`
    TimeToRestore   string        `yaml:"time2restore"`
    TimeToStatistic string        `yaml:"time2statistic"`
    AutoTimeOut     bool          `yaml:"autotimeout"`
    Description     string        `yaml:"description"`
    SrcServer       string        `yaml:"srcServer"`
    SrcDB           string        `yaml:"srcDB"`
    DstServer       string        `yaml:"dstServer"`
    DstDB           string        `yaml:"dstDB"`
}
```

**RestoreStats** - Статистика восстановления
```go
type RestoreStats struct {
    AvgRestoreTimeSecond sql.NullInt64
    MaxRestoreTimeSecond sql.NullInt64
}
```

### 3. internal/entity/gitea/gitea.go

**PR** - Pull Request
```go
type PR struct {
    ID     int64
    Number int64
    Base   string
    Head   string
}
```

**PRData** - Данные Pull Request
```go
type PRData struct {
    ID     int64  `json:"id"`
    Number int64  `json:"number"`
    Base   Branch `json:"base"`
    Head   Branch `json:"head"`
}
```

**Branch** - Ветка
```go
type Branch struct {
    Label string `json:"label"`
}
```

**PullRequest** - Запрос на слияние
```go
type PullRequest struct {
    Number    int  `json:"number"`
    Mergeable bool `json:"mergeable"`
}
```

**ConflictFile** - Файл с конфликтом
```go
type ConflictFile struct {
    Filename string `json:"filename"`
}
```

**Issue** - Задача
```go
type Issue struct {
    ID     int64  `json:"id"`
    Number int64  `json:"number"`
    Title  string `json:"title"`
    Body   string `json:"body"`
    State  string `json:"state"`
    User   struct {
        Login string `json:"login"`
        ID    int64  `json:"id"`
    } `json:"user"`
    CreatedAt string `json:"created_at"`
    UpdatedAt string `json:"updated_at"`
}
```

**ProjectAnalysis** - Анализ проекта
```go
type ProjectAnalysis struct {
    ProjectName string   `json:"project_name"`
    Extensions  []string `json:"extensions"`
}
```

### 4. internal/entity/one/convert/convert.go

**ConvertConfig** - Конфигурация конвертации
```go
type ConvertConfig struct {
    StoreRoot   string         `json:"Корень хранилища"`
    OneDB       designer.OneDb `json:"Параметры подключения"`
    ConvertPair []ConvertPair  `json:"Сопоставления"`
}
```

**ConvertPair** - Пара конвертации
```go
type ConvertPair struct {
    Source Source      `json:"Источник"`
    Store  store.Store `json:"Хранилище"`
}
```

**Source** - Источник
```go
type Source struct {
    Name    string `json:"Имя"`
    RelPath string `json:"Относительный путь"`
    Main    bool   `json:"Основная конфигурация"`
}
```

### 5. internal/entity/one/designer/designer.go

**OneDb** - База данных 1С
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

### 6. internal/entity/one/store/store.go

**Store** - Хранилище конфигурации
```go
type Store struct {
    Name    string `json:"Имя хранилища,omitempty"`
    Path    string `json:"Относительный путь"`
    User    string `json:"Пользователь"`
    Pass    string `json:"Пароль"`
    Command string `json:"-"`
}
```

**StoreRecord** - Запись хранилища
```go
type StoreRecord struct {
    Version     int       `json:"Версия хранилища"`
    ConfVersion string    `json:"Версия конфигурации"`
    User        string    `json:"Пользователь"`
    Date        time.Time `json:"Дата время"`
    Comment     string    `json:"Комментарий,omitempty"`
}
```

**User** - Пользователь хранилища
```go
type User struct {
    StoreUserName string `json:"Имя пользователя в хранилище"`
    AccountName   string `json:"Доменное имя"`
    Email         string `json:"e-mail"`
}
```

### 7. internal/entity/one/edt/edt.go

**Convert** - Конвертация EDT
```go
type Convert struct {
    CommitSha1  string    `json:"Хеш коммита"`
    Source      Data      `json:"Источник"`
    Distination Data      `json:"Приемник"`
    Mappings    []Mapping `json:"Сопоставление путей"`
}
```

**Data** - Данные конвертации
```go
type Data struct {
    Name   string `json:"Имя"`
    Format string `json:"Формат"`
    Branch string `json:"Ветка"`
}
```

**Mapping** - Сопоставление путей
```go
type Mapping struct {
    SourcePath      string `json:"Путь источника"`
    DistinationPath string `json:"Путь приемника"`
}
```

**EdtCli** - CLI EDT
```go
type EdtCli struct {
    RingPath   string
    EdtVersion string
    Direction  string
    PathIn     string
    PathOut    string
    WorkSpace  string
    Operation  string
    LastErr    error
}
```

### 8. internal/git/git.go

**Git** - Git репозиторий
```go
type Git struct {
    RepURL     string
    RepPath    string
    Branch     string
    CommitSHA1 string
    WorkDir    string
    Token      string
}
```

**GitConfigs** - Конфигурации Git
```go
type GitConfigs struct {
    GitConfig []GitConfig
}
```

**GitConfig** - Конфигурация Git
```go
type GitConfig struct {
    Name  string
    Value string
}
```

### 9. internal/rac/rac.go

**Client** - RAC клиент
```go
type Client struct {
    RacPath    string
    Server     string
    Port       int
    User       string
    Password   string
    DbUser     string
    DbPassword string
    Timeout    time.Duration
    Retries    int
    Logger     *slog.Logger
}
```

### 10. internal/structures/types.go (Новые оптимизированные структуры)

**ConvertData** - Оптимизированная структура для данных конвертации
```go
type ConvertData struct {
    SourcePath string
    TargetPath string
    Timestamp  time.Time
    Version    int64
    IsValid    bool
}
```

## Анализ и рекомендации

### Проблемы текущей архитектуры:

1. **Дублирование структур**: Множественные определения похожих структур в разных пакетах
2. **Неконсистентные теги**: Смешение JSON и YAML тегов, различные стили именования
3. **Отсутствие валидации**: Большинство структур не имеют методов валидации
4. **Memory alignment**: Неоптимальное расположение полей в структурах
5. **Отсутствие документации**: Многие структуры не документированы

### Рекомендации по рефакторингу:

1. **Консолидация**: Объединить похожие структуры в общие пакеты
2. **Стандартизация**: Унифицировать теги и стили именования
3. **Валидация**: Добавить методы валидации для всех структур
4. **Оптимизация**: Переупорядочить поля для лучшего memory alignment
5. **Документация**: Добавить комментарии и примеры использования
6. **Тестирование**: Создать unit-тесты для всех структур

### Приоритеты рефакторинга:

1. **Высокий приоритет**: Config, ConvertConfig, DBRestore
2. **Средний приоритет**: Git, Store, OneDb
3. **Низкий приоритет**: API структуры, вспомогательные типы

## Заключение

Проект содержит 30+ основных структур данных, распределенных по 9 пакетам. Необходим комплексный рефакторинг для улучшения читаемости, производительности и поддерживаемости кода.