# Архитектура системы конфигурации

## Обзор

Данный документ описывает архитектурное решение для системы загрузки и управления конфигурацией приложения apk-ci. Система обеспечивает гибкую и безопасную загрузку настроек из различных источников с поддержкой иерархии приоритетов.

## Структура файлов конфигурации

Система использует четыре основных файла конфигурации:

### 1. app.yaml - Конфигурация приложения
Содержит настройки, относящиеся к работе самого приложения:
- Пути к исполняемым файлам 1С
- Настройки логирования
- Рабочие директории
- Таймауты и повторы
- Настройки EDT

```yaml
# Пример app.yaml
app:
  logLevel: "Debug"
  workDir: "/tmp/benadis"
  tmpDir: "/tmp/benadis/temp"
  timeout: 30

paths:
  bin1cv8: "/opt/1cv8/x86_64/8.3.27.1606/1cv8"
  binIbcmd: "/opt/1cv8/x86_64/8.3.27.1606/ibcmd"
  edtCli: "/opt/1C/1CE/components/1c-edt-2024.2.6+7-x86_64/1cedtcli"
  rac: "/opt/1cv8/x86_64/8.3.27.1606/rac"

rac:
  port: 1545
  timeout: 30
  retries: 3

users:
  rac: "admin"
  db: "db_user"
  mssql: "mssql_user"
  storeAdmin: "store_admin"
```

### 2. project.yaml - Конфигурация проекта
Содержит настройки, специфичные для конкретного проекта:
- Базы данных проекта
- Связи между продуктивными и тестовыми базами
- Настройки расширений
- Конфигурация хранилища

```yaml
# Пример project.yaml (существующий формат)
store-db: V8_DEV_DSBEKETOV_STORE_ERP
prod:
  V8_OPER_APK_TOIR3:
    dbName: "База ТОИР"
    add-disable:
      - апк_ДоработкиТОИР
    related:
      V8_DEV_DSBEKETOV_APK_TOIR3:
      V8_DEV_DSBEKETOV_APK_TOIR3_1:
```

### 3. secret.yaml - Конфигурация секретов
Содержит конфиденциальные данные:
- Пароли пользователей
- Токены доступа
- Строки подключения к базам данных
- Ключи API

```yaml
# Пример secret.yaml
passwords:
  rac: "secure_password"
  db: "db_password"
  mssql: "mssql_password"
  storeAdminPassword: "store_admin_password"

gitea:
  accessToken: "gitea_token_here"
```

### 4. dbconfig.yaml - Конфигурация баз данных
Содержит информацию о серверах и базах данных:
- Сервер 1С для каждой базы (используется также как RAC сервер)
- Признак продуктивности базы
- Сервер MS SQL для каждой базы

```yaml
# Пример dbconfig.yaml (существующий формат)
V8_DEV_DSBEKETOV_STORE_ERP:
  one-server: DEV-16-AS-003
  prod: false
  dbserver: MSK-DV-SQL-01

V8_OPER_APK_TOIR3:
  one-server: MSK-TS-AS-001
  prod: true
  dbserver: MSK-TS-SQL-01
```

## Архитектура загрузки конфигурации

### Иерархия приоритетов

1. **Переменные окружения** (высший приоритет)
2. **GitHub Actions inputs**
3. **Файлы конфигурации** (app.yaml, project.yaml, secret.yaml)
4. **Значения по умолчанию** (низший приоритет)

### Структура данных

```go
// Основная структура конфигурации
type Config struct {
    // Системные настройки
    Actor         string `env:"BR_ACTOR"`
    Env           string `env:"BR_ENV"`
    Command       string `env:"BR_COMMAND"`
    Logger        *slog.Logger
    
    // Пути к файлам конфигурации
    ConfigSystem  string `env:"BR_CONFIG_SYSTEM"`
    ConfigProject string `env:"BR_CONFIG_PROJECT"`
    ConfigSecret  string `env:"BR_CONFIG_SECRET"`
    
    // Настройки приложения (из app.yaml)
    AppConfig     *AppConfig
    
    // Настройки проекта (из project.yaml)
    ProjectConfig *ProjectConfig
    
    // Секреты (из secret.yaml)
    SecretConfig  *SecretConfig
    
    // Конфигурация баз данных
    DbConfig      map[string]*DatabaseInfo
}

// Конфигурация приложения
type AppConfig struct {
    LogLevel  string        `yaml:"logLevel"`
    WorkDir   string        `yaml:"workDir"`
    TmpDir    string        `yaml:"tmpDir"`
    Timeout   time.Duration `yaml:"timeout"`
    
    Paths struct {
        Bin1cv8   string `yaml:"bin1cv8"`
        BinIbcmd  string `yaml:"binIbcmd"`
        EdtCli    string `yaml:"edtCli"`
        Rac       string `yaml:"rac"`
    } `yaml:"paths"`
    
    Rac struct {
        Port    int    `yaml:"port"`
        Timeout int    `yaml:"timeout"`
        Retries int    `yaml:"retries"`
    } `yaml:"rac"`
    
    Users struct {
        Rac        string `yaml:"rac"`
        Db         string `yaml:"db"`
        Mssql      string `yaml:"mssql"`
        StoreAdmin string `yaml:"storeAdmin"`
    } `yaml:"users"`
}

// Конфигурация проекта
type ProjectConfig struct {
    StoreDb string                    `yaml:"store-db"`
    Prod    map[string]*ProductionDb  `yaml:"prod"`
}

type ProductionDb struct {
    DbName     string              `yaml:"dbName"`
    AddDisable []string            `yaml:"add-disable"`
    Related    map[string]struct{} `yaml:"related"`
}

// Конфигурация секретов
type SecretConfig struct {
    Passwords struct {
        Rac                string `yaml:"rac"`
        Db                 string `yaml:"db"`
        Mssql              string `yaml:"mssql"`
        StoreAdminPassword string `yaml:"storeAdminPassword"`
    } `yaml:"passwords"`
    
    Gitea struct {
        AccessToken string `yaml:"accessToken"`
    } `yaml:"gitea"`
}

// Информация о базе данных
type DatabaseInfo struct {
    OneServer string `yaml:"one-server"`
    Prod      bool   `yaml:"prod"`
    DbServer  string `yaml:"dbserver"`
}
```

### Процесс загрузки конфигурации

```go
func LoadConfiguration() (*Config, error) {
    cfg := &Config{}
    
    // 1. Инициализация логгера
    cfg.Logger = initLogger()
    
    // 2. Загрузка базовых настроек из переменных окружения
    if err := cleanenv.ReadEnv(cfg); err != nil {
        return nil, fmt.Errorf("failed to read env vars: %w", err)
    }
    
    // 3. Загрузка путей к файлам конфигурации
    cfg.loadConfigPaths()
    
    // 4. Загрузка конфигурации приложения
    if err := cfg.loadAppConfig(); err != nil {
        cfg.Logger.Warn("Failed to load app config", "error", err)
    }
    
    // 5. Загрузка конфигурации проекта
    if err := cfg.loadProjectConfig(); err != nil {
        cfg.Logger.Warn("Failed to load project config", "error", err)
    }
    
    // 6. Загрузка секретов
    if err := cfg.loadSecretConfig(); err != nil {
        cfg.Logger.Warn("Failed to load secret config", "error", err)
    }
    
    // 7. Загрузка конфигурации баз данных
    if err := cfg.loadDbConfig(); err != nil {
        cfg.Logger.Warn("Failed to load db config", "error", err)
    }
    
    // 8. Валидация конфигурации
    if err := cfg.validate(); err != nil {
        return nil, fmt.Errorf("config validation failed: %w", err)
    }
    
    return cfg, nil
}
```

### Методы загрузки отдельных конфигураций

```go
// Загрузка конфигурации приложения
func (cfg *Config) loadAppConfig() error {
    data, err := cfg.getConfigData(cfg.ConfigSystem)
    if err != nil {
        return err
    }
    
    cfg.AppConfig = &AppConfig{}
    return yaml.Unmarshal(data, cfg.AppConfig)
}

// Загрузка конфигурации проекта
func (cfg *Config) loadProjectConfig() error {
    data, err := cfg.getConfigData(cfg.ConfigProject)
    if err != nil {
        return err
    }
    
    cfg.ProjectConfig = &ProjectConfig{}
    return yaml.Unmarshal(data, cfg.ProjectConfig)
}

// Загрузка секретов
func (cfg *Config) loadSecretConfig() error {
    data, err := cfg.getConfigData(cfg.ConfigSecret)
    if err != nil {
        return err
    }
    
    cfg.SecretConfig = &SecretConfig{}
    return yaml.Unmarshal(data, cfg.SecretConfig)
}

// Загрузка конфигурации баз данных
func (cfg *Config) loadDbConfig() error {
    data, err := cfg.getConfigData("dbconfig.yaml")
    if err != nil {
        return err
    }
    
    cfg.DbConfig = make(map[string]*DatabaseInfo)
    return yaml.Unmarshal(data, &cfg.DbConfig)
}
```

### Получение информации о базе данных

```go
// Получение информации о сервере 1С для базы данных
func (cfg *Config) GetOneServer(dbName string) (string, error) {
    if dbInfo, exists := cfg.DbConfig[dbName]; exists {
        return dbInfo.OneServer, nil
    }
    return "", fmt.Errorf("database %s not found in config", dbName)
}

// Получение сервера RAC для конкретной базы данных
func (cfg *Config) GetRacServerForDb(dbName string) (string, error) {
    if dbInfo, exists := cfg.DbConfig[dbName]; exists {
        return dbInfo.OneServer, nil
    }
    return "", fmt.Errorf("database %s not found in config", dbName)
}

// Проверка, является ли база продуктивной
func (cfg *Config) IsProductionDb(dbName string) (bool, error) {
    if dbInfo, exists := cfg.DbConfig[dbName]; exists {
        return dbInfo.Prod, nil
    }
    return false, fmt.Errorf("database %s not found in config", dbName)
}

// Получение информации о сервере MS SQL для базы данных
func (cfg *Config) GetDbServer(dbName string) (string, error) {
    if dbInfo, exists := cfg.DbConfig[dbName]; exists {
        return dbInfo.DbServer, nil
    }
    return "", fmt.Errorf("database %s not found in config", dbName)
}

// Получение полной информации о базе данных
func (cfg *Config) GetDatabaseInfo(dbName string) (*DatabaseInfo, error) {
    if dbInfo, exists := cfg.DbConfig[dbName]; exists {
        return dbInfo, nil
    }
    return nil, fmt.Errorf("database %s not found in config", dbName)
}
```

## Источники конфигурации

### 1. Локальные файлы
- Используются для разработки и тестирования
- Загружаются через `os.ReadFile()`

### 2. Gitea API
- Основной источник для production
- Загружаются через `GetConfigData()` функцию
- Поддержка base64 декодирования

### 3. Переменные окружения
- Имеют наивысший приоритет
- Используются для переопределения настроек

## Безопасность

### Обработка секретов
1. Секреты хранятся в отдельном файле `secret.yaml`
2. Файл секретов загружается через защищенный API
3. Секреты не логируются в открытом виде
4. Поддержка переопределения через переменные окружения

### Валидация
1. Проверка обязательных параметров
2. Валидация форматов данных
3. Проверка доступности файлов и директорий

## Расширяемость

### Добавление новых типов конфигурации
1. Создание новой структуры конфигурации
2. Добавление метода загрузки
3. Интеграция в основной процесс загрузки

### Поддержка новых источников
1. Реализация интерфейса `ConfigSource`
2. Добавление в цепочку загрузки
3. Настройка приоритетов

## Миграция

### Переход от текущей архитектуры
1. Постепенное выделение настроек в отдельные файлы
2. Сохранение обратной совместимости
3. Поэтапное обновление модулей

### План миграции
1. **Фаза 1**: Создание новых структур конфигурации
2. **Фаза 2**: Реализация загрузчиков конфигурации
3. **Фаза 3**: Обновление существующих модулей
4. **Фаза 4**: Удаление устаревшего кода

## Примеры использования

### Загрузка конфигурации для модуля Service Mode
```go
func LoadServiceModeConfig(cfg *Config, dbName string) (*ServiceModeConfig, error) {
    // Получение сервера RAC для конкретной базы
    racServer, err := cfg.GetRacServerForDb(dbName)
    if err != nil {
        return nil, fmt.Errorf("failed to get RAC server for db %s: %w", dbName, err)
    }
    
    serviceCfg := &ServiceModeConfig{
        // Значения по умолчанию из app config
        RacPath:   cfg.AppConfig.Paths.Rac,
        RacServer: racServer, // Получаем из dbconfig.yaml для конкретной базы
        RacPort:   cfg.AppConfig.Rac.Port,
        
        // Пользователи из app config
        RacUser:     cfg.AppConfig.Users.Rac,
        DbUser:      cfg.AppConfig.Users.Db,
        MssqlUser:   cfg.AppConfig.Users.Mssql,
        StoreAdmin:  cfg.AppConfig.Users.StoreAdmin,
        
        // Пароли из secret config
        RacPassword:         cfg.SecretConfig.Passwords.Rac,
        DbPassword:          cfg.SecretConfig.Passwords.Db,
        MssqlPassword:       cfg.SecretConfig.Passwords.Mssql,
        StoreAdminPassword:  cfg.SecretConfig.Passwords.StoreAdminPassword,
    }
    
    // Переопределение из переменных окружения
    if err := cleanenv.ReadEnv(serviceCfg); err != nil {
        return nil, err
    }
    
    return serviceCfg, nil
}
```

### Получение информации о базе данных
```go
func ProcessDatabase(cfg *Config, dbName string) error {
    // Получение информации о базе
    dbInfo, err := cfg.GetDatabaseInfo(dbName)
    if err != nil {
        return fmt.Errorf("failed to get db info: %w", err)
    }
    
    // Проверка типа базы
    if dbInfo.Prod {
        cfg.Logger.Info("Processing production database", "db", dbName)
    } else {
        cfg.Logger.Info("Processing test database", "db", dbName)
    }
    
    // Использование серверов
    cfg.Logger.Debug("Database servers", 
        "oneServer", dbInfo.OneServer,
        "dbServer", dbInfo.DbServer)
    
    return nil
}
```

## Заключение

Предложенная архитектура обеспечивает:
- Четкое разделение типов конфигурации
- Гибкую систему приоритетов
- Безопасную обработку секретов
- Простоту расширения и сопровождения
- Обратную совместимость с существующим кодом
- Динамическое получение RAC сервера для каждой базы из dbconfig.yaml
- Разделение пользователей и паролей между app.yaml и secret.yaml
- Поддержку MS SQL пользователей и паролей

Ключевые изменения в архитектуре:
1. **Перенос rac.path в paths** - все пути к исполняемым файлам теперь находятся в одном разделе
2. **Динамический RAC сервер** - сервер RAC определяется для каждой базы на основе поля one-server из dbconfig.yaml
3. **Разделение пользователей и паролей** - имена пользователей хранятся в app.yaml, пароли в secret.yaml
4. **Поддержка MS SQL** - добавлены настройки для пользователей и паролей MS SQL
5. **Упрощение структуры конфигурации** - объединены дублирующиеся секции users и passwords
6. **Поддержка хранилища конфигурации** - добавлены поля storeAdmin и storeAdminPassword для подключения к ConfigurationRepository

Архитектура позволяет легко добавлять новые типы конфигурации и источники данных, обеспечивая масштабируемость системы.