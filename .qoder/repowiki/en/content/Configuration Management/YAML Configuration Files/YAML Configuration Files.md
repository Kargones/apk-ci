# YAML Configuration Files

<cite>
**Referenced Files in This Document**
- [app.yaml](file://config/app.yaml)
- [app.yaml.example](file://config/app.yaml.example)
- [dbconfig.yaml](file://config/dbconfig.yaml)
- [action.yaml](file://config/action.yaml)
- [menu_main.yaml](file://config/menu_main.yaml)
- [menu_debug.yaml](file://config/menu_debug.yaml)
- [secret.yaml.example](file://config/secret.yaml.example)
- [config.go](file://internal/config/config.go)
- [implementations_test.go](file://internal/config/implementations_test.go)
- [production_config.yaml](file://internal/config/testdata/production_config.yaml)
</cite>

## Update Summary
**Changes Made**
- Added comprehensive documentation for the new ImplementationsConfig structure
- Updated logging configuration section with new defaults (text format, stderr output)
- Enhanced configuration loading process documentation with backward compatibility
- Added examples of the new implementations section in YAML configuration
- Updated configuration validation and environment variable handling

## Table of Contents
1. [Introduction](#introduction)
2. [Configuration Architecture](#configuration-architecture)
3. [Application Configuration (app.yaml)](#application-configuration-appyaml)
4. [Database Configuration (dbconfig.yaml)](#database-configuration-dbconfigyaml)
5. [GitHub Actions Integration (action.yaml)](#github-actions-integration-actionyaml)
6. [Menu Configuration Files](#menu-configuration-files)
7. [Configuration Loading Process](#configuration-loading-process)
8. [Environment Variables and Priority](#environment-variables-and-priority)
9. [Common Configuration Patterns](#common-configuration-patterns)
10. [Error Handling and Validation](#error-handling-and-validation)
11. [Environment-Specific Configuration](#environment-specific-configuration)
12. [Best Practices](#best-practices)

## Introduction

The apk-ci employs a sophisticated YAML-based configuration system that centralizes all application settings, database connections, and operational parameters. This system provides a unified approach to configuration management across all modules, ensuring consistency and maintainability while supporting multiple deployment environments.

The configuration system is built around five primary configuration files, each serving a specific purpose in the application's operation:

- **app.yaml**: Application-wide settings including logging, timeouts, executable paths, and service configurations
- **dbconfig.yaml**: Database connection definitions with host, port, credentials, and database names
- **action.yaml**: GitHub Actions workflow integration and parameter definitions
- **Menu configuration files**: Dynamic workflow templates for different operational scenarios
- **secret.yaml**: Sensitive configuration data including passwords and tokens

## Configuration Architecture

The configuration system follows a hierarchical structure with multiple layers of precedence and enhanced modularity:

```mermaid
graph TD
A["Environment Variables<br/>(Highest Priority)"] --> B["Configuration Files<br/>(Medium Priority)"]
B --> C["Default Values<br/>(Lowest Priority)"]
D["app.yaml<br/>Application Settings"] --> E["Central Config Struct"]
F["dbconfig.yaml<br/>Database Definitions"] --> E
G["action.yaml<br/>GitHub Actions"] --> E
H["Menu Files<br/>Workflow Templates"] --> E
I["secret.yaml<br/>Secrets"] --> E
J["ImplementationsConfig<br/>Operation Implementations"] --> E
K["LoggingConfig<br/>Logging Settings"] --> E
E --> L["Runtime Configuration"]
M["Validation Layer"] --> N["Error Handling"]
N --> O["Fallback Mechanisms"]
```

**Diagram sources**
- [config.go](file://internal/config/config.go#L131-L213)
- [config.go](file://internal/config/config.go#L292-L326)

**Section sources**
- [config.go](file://internal/config/config.go#L131-L213)
- [config.go](file://internal/config/config.go#L292-L326)

## Application Configuration (app.yaml)

The `app.yaml` file serves as the central configuration hub for application-wide settings. It defines system-level parameters that control logging, timeouts, executable paths, and service integrations, now enhanced with modular configuration sections.

### Core Structure

```yaml
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
  rac: "gitops"
  db: "gitops"
  mssql: "gitops"
  storeAdmin: "gitops"

dbrestore:
  database: "master"
  timeout: "3600s"
  autotimeout: true

implementations:
  config_export: "1cv8"
  db_create: "1cv8"

logging:
  level: "info"
  format: "text"
  output: "stderr"
  filePath: "/var/log/apk-ci.log"
  maxSize: 100
  maxBackups: 3
  maxAge: 28
  compress: true
```

### Enhanced Configuration Sections

#### Implementations Configuration
The new `implementations` section allows flexible selection of operation implementations:

```yaml
implementations:
  config_export: "1cv8"  # Options: "1cv8", "ibcmd", "native"
  db_create: "1cv8"      # Options: "1cv8", "ibcmd"
```

#### Logging Configuration
Enhanced logging with improved defaults:
- **level**: "info" (was "debug")
- **format**: "text" (was "json") - more readable output
- **output**: "stderr" (was "stdout") - separates logs from application output

### SonarQube Integration

The application includes comprehensive SonarQube integration configuration:

```yaml
sonarqube:
  url: "http://sq.apkholding.ru:9000"
  timeout: "30s"
  retryAttempts: 3
  retryDelay: "1s"
  projectPrefix: ""
  defaultVisibility: "private"
  qualityGateTimeout: "5m"
```

### Scanner Configuration

```yaml
scanner:
  scannerUrl: "https://regdv.apkholding.ru/gitops-tools/sonar-scanner-cli.git"
  scannerVersion: "7.2.0.5079"
  javaOpts: "-Xmx8192m"
  timeout: "240m"
  workDir: "/tmp/4del/scanner"
  tempDir: "/tmp/4del/scanner/temp"
  properties:
    sonar.sourceEncoding: "UTF-8"
    sonar.sources: "."
    sonar.inclusions: "**/*.bsl,**/*.os,**/*.epf,**/*.erf,**/*.cf,**/*.cfe,**/*.xml,**/*.mdo,**/*.mxl,**/*.dcr,**/*.dcs,**/*.xsd,**/*.go"
    sonar.language: "bsl"
```

### Git Configuration

```yaml
git:
  userName: "apk-ci"
  userEmail: "apk-ci@benadis.ru"
  defaultBranch: "main"
  timeout: "30s"
  credentialHelper: "store"
  credentialTimeout: "300s"
```

**Section sources**
- [app.yaml](file://config/app.yaml#L1-L138)
- [app.yaml.example](file://config/app.yaml.example#L1-L127)
- [config.go](file://internal/config/config.go#L27-L61)

## Database Configuration (dbconfig.yaml)

The `dbconfig.yaml` file defines multiple database connections with comprehensive metadata for each database instance. This file serves as the central registry for all database configurations used by the application.

### Database Definition Format

Each database entry follows a standardized format:

```yaml
DATABASE_NAME:
  one-server: SERVER_NAME
  prod: BOOLEAN
  dbserver: DATABASE_SERVER
```

### Example Database Entries

```yaml
V8_ARCH_APK_CENTER_2IS:
  one-server: MSK-AS-ARCH-001
  prod: false
  dbserver: MSK-SQL-ARCH-01

TEST_DNAVOLOTSKY_SURV:
  one-server: MSK-TS-AS-001
  prod: false
  dbserver: DEV-RZHAVKI-DB1

TEST_IKHRISTENZEN_SURV:
  one-server: MSK-TS-AS-001
  prod: false
  dbserver: TEST-16-DB-001
```

### Database Types and Categories

The configuration supports multiple database categories:

#### Production Databases
- Marked with `prod: true`
- Typically used for live operations
- Require special handling for updates and maintenance

#### Test Databases
- Marked with `prod: false`
- Used for development, testing, and staging
- Allow more flexible update procedures

#### Server Associations
- **one-server**: 1C:Enterprise server hosting the database
- **dbserver**: Microsoft SQL Server instance
- Enables cross-server operations and migrations

### Database Information Structure

```mermaid
classDiagram
class DatabaseInfo {
+string OneServer
+bool Prod
+string DbServer
+getOneServer() string
+isProduction() bool
+getDbServer() string
}
class Config {
+map[string]*DatabaseInfo DbConfig
+GetDatabaseInfo(dbName) *DatabaseInfo
+IsProductionDb(dbName) bool
+GetOneServer(dbName) string
+GetDbServer(dbName) string
}
Config --> DatabaseInfo : "manages"
```

**Diagram sources**
- [config.go](file://internal/config/config.go#L94-L101)
- [config.go](file://internal/config/config.go#L157-L158)

**Section sources**
- [dbconfig.yaml](file://config/dbconfig.yaml#L1-L800)
- [config.go](file://internal/config/config.go#L94-L101)

## GitHub Actions Integration (action.yaml)

The `action.yaml` file defines the GitHub Actions workflow interface, specifying input parameters, environment variables, and execution steps for the apk-ci.

### Action Metadata

```yaml
name: 'gitops commander'
description: 'Запуск команд gitops'
```

### Input Parameters

The action defines comprehensive input parameters for flexible operation:

```yaml
inputs:
  giteaURL:
    description: 'Адрес сервера gitea'
    required: true
  repository:
    description: 'Полное имя репозитория'
    required: true
  accessToken:
    description: 'Токен доступа'
    required: true
  command:
    description: 'Выполняемая команда'
    required: true
```

### Advanced Configuration Parameters

```yaml
  configSystem:
    description: 'Имя файла с системной конфигурацией'
    required: false
    default: 'https://regdv.apkholding.ru/api/v1/repos/gitops-tools/gitops_congif/contents/app.yaml?ref=main'
  configProject:
    description: 'Имя файла с конфигурацией проекта'
    required: false
    default: 'project.yaml'
  configSecret:
    description: 'Имя файла с секретами'
    required: false
    default: 'https://regdv.apkholding.ru/api/v1/repos/gitops-tools/gitops_congif/contents/secret.yaml?ref=main'
  configDbData:
    description: 'Имя файла с конфигурацией базы данных'
    required: false
    default: 'https://regdv.apkholding.ru/api/v1/repos/gitops-tools/gitops_congif/contents/dbconfig.yaml?ref=main'
```

### Environment Variable Mapping

The action automatically maps inputs to environment variables:

```yaml
env:
  GITEA_URL: ${{ inputs.giteaURL }}
  REPOSITORY: ${{ inputs.repository }}
  ACCESS_TOKEN: ${{ inputs.accessToken }}
  COMMAND: ${{ inputs.command }}
  LOG_LEVEL: ${{ inputs.logLevel }}
  ISSUE_NUMBER: ${{ inputs.issueNumber }}
  CONFIG_SYSTEM: ${{ inputs.configSystem }}
  CONFIG_PROJECT: ${{ inputs.configProject }}
  CONFIG_SECRET: ${{ inputs.configSecret }}
  CONFIG_DB_DATA: ${{ inputs.configDbData }}
```

### Execution Steps

The action provides flexible execution modes:

```yaml
steps:
  - name: 'Run apk-ci'
    run: |
      if [ "${{ inputs.debug_port }}" != "0" ]; then
        if [ "${{ inputs.wait }}" = "false" ]; then
          dlv --listen=:${{ inputs.debug_port }} --headless=true --api-version=2 --accept-multiclient --continue exec ${{ github.action_path }}/apk-ci
        else
          dlv --listen=:${{ inputs.debug_port }} --headless=true --api-version=2 --accept-multiclient exec ${{ github.action_path }}/apk-ci
        fi
      else
        ${{ github.action_path }}/apk-ci
      fi
    shell: bash
```

**Section sources**
- [action.yaml](file://config/action.yaml#L1-L121)
- [config.go](file://internal/config/config.go#L106-L127)

## Menu Configuration Files

The menu configuration files provide dynamic workflow templates for different operational scenarios. These files support parameterized workflows with automatic database selection and conditional execution.

### Main Menu Configuration (menu_main.yaml)

The main menu provides structured workflows for common operations:

#### Database Update Workflows

```yaml
1. Обновление тестовых баз.yaml
run-name: ${{ gitea.event_name }} - ${{ gitea.workflow }} - ${{ gitea.actor }}
on:
  workflow_dispatch:
    inputs:
      restore_DB:
        description: 'Восстановить базу перед загрузкой конфигурации'
        required: true
        type: boolean
        default: false 
      service_mode_enable:
        description: 'Включить сервисный режим'
        required: true
        type: boolean
        default: true 
```

#### Parameterized Database Selection

The menu supports dynamic database selection with automatic replacement:

```yaml
      DbName:
        description: 'Выберите базу для загрузки конфигурации (Test)'
        required: true
        default: $TestBaseReplace$
        type: choice
        options:
$TestBaseReplaceAll$
```

### Debug Menu Configuration (menu_debug.yaml)

The debug menu provides comprehensive testing capabilities:

#### Multi-Action Testing

```yaml
name: Test All Actions
on:
  workflow_dispatch:
    inputs:
      version:
        description: 'Version of apk-ci'
        required: true
        default: 'v1.2.4'
        type: string
      action_1:
        description: 'Enable service mode for database'
        required: false
        default: true
        type: boolean
```

#### Conditional Step Execution

```yaml
jobs:
  test-all-actions:
    runs-on: edt
    steps:
      - name: Service Mode Enable DB
        if: ${{ inputs.action_1 == true }}
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@${{ inputs.version }}
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          actor: ${{ gitea.actor }}
          command: 'service-mode-enable'
          dbName: ${{ inputs.action_1_dbName }}
          logLevel: 'Debug'
```

### Workflow Template Structure

```mermaid
flowchart TD
A["Menu Configuration File"] --> B["Parameter Replacement"]
B --> C["Dynamic Workflow Generation"]
C --> D["Conditional Step Execution"]
D --> E["Multi-Database Operations"]
E --> F["GitHub Actions Integration"]
G["Template Variables"] --> B
H["$TestBaseReplace$"] --> G
I["$ProdBaseReplace$"] --> G
J["$TestBaseReplaceAll$"] --> G
K["$ProdBaseReplaceAll$"] --> G
```

**Diagram sources**
- [menu_main.yaml](file://config/menu_main.yaml#L1-L367)
- [menu_debug.yaml](file://config/menu_debug.yaml#L1-L257)

**Section sources**
- [menu_main.yaml](file://config/menu_main.yaml#L1-L367)
- [menu_debug.yaml](file://config/menu_debug.yaml#L1-L257)

## Configuration Loading Process

The configuration loading process follows a systematic approach with multiple fallback mechanisms, validation layers, and enhanced backward compatibility.

### Loading Sequence

```mermaid
sequenceDiagram
participant GH as GitHub Actions
participant CFG as Config Loader
participant GITEA as Gitea API
participant YAML as YAML Parser
participant ENV as Environment Variables
participant DEF as Defaults
GH->>CFG : Initialize with Input Params
CFG->>CFG : Validate Required Parameters
CFG->>GITEA : Load Config System (app.yaml)
GITEA-->>CFG : YAML Data
CFG->>YAML : Parse app.yaml
YAML-->>CFG : AppConfig Struct
CFG->>CFG : Extract ImplementationsConfig
CFG->>CFG : Extract LoggingConfig
CFG->>GITEA : Load Config Project (project.yaml)
GITEA-->>CFG : YAML Data
CFG->>YAML : Parse project.yaml
YAML-->>CFG : ProjectConfig Struct
CFG->>GITEA : Load Config Secret (secret.yaml)
GITEA-->>CFG : YAML Data
CFG->>YAML : Parse secret.yaml
YAML-->>CFG : SecretConfig Struct
CFG->>GITEA : Load DB Config (dbconfig.yaml)
GITEA-->>CFG : YAML Data
CFG->>YAML : Parse dbconfig.yaml
YAML-->>CFG : DbConfig Map
CFG->>ENV : Load Environment Variables
ENV-->>CFG : Override Values
CFG->>DEF : Apply Default Values
DEF-->>CFG : Final Configuration
```

**Diagram sources**
- [config.go](file://internal/config/config.go#L626-L793)
- [config.go](file://internal/config/config.go#L482-L516)

### Enhanced Configuration Loading Functions

Each configuration type has dedicated loading functions with improved error handling:

#### Application Configuration Loading

```go
func loadAppConfig(l *slog.Logger, cfg *Config) (*AppConfig, error) {
    giteaAPI := CreateGiteaAPI(cfg)
    data, err := giteaAPI.GetConfigData(l, cfg.ConfigSystem)
    if err != nil {
        return nil, fmt.Errorf("ошибка получения данных app.yaml: %w", err)
    }

    var appConfig AppConfig
    if err = yaml.Unmarshal(data, &appConfig); err != nil {
        return nil, fmt.Errorf("ошибка парсинга app.yaml: %w", err)
    }

    return &appConfig, nil
}
```

#### Implementations Configuration Loading

```go
func loadImplementationsConfig(l *slog.Logger, cfg *Config) (*ImplementationsConfig, error) {
    // Check if implementations config exists in AppConfig
    if cfg.AppConfig != nil && (cfg.AppConfig.Implementations != ImplementationsConfig{}) {
        implConfig := &cfg.AppConfig.Implementations
        // Apply environment variable overrides
        if err := cleanenv.ReadEnv(implConfig); err != nil {
            l.Warn("Ошибка загрузки Implementations конфигурации из переменных окружения",
                slog.String("error", err.Error()),
            )
        }
        l.Info("Implementations конфигурация загружена из AppConfig",
            slog.String("config_export", implConfig.ConfigExport),
            slog.String("db_create", implConfig.DBCreate),
        )
        return implConfig, nil
    }

    // If not found, use default values
    implConfig := getDefaultImplementationsConfig()
    
    // Apply environment variable overrides
    if err := cleanenv.ReadEnv(implConfig); err != nil {
        l.Warn("Ошибка загрузки Implementations конфигурации из переменных окружения",
            slog.String("error", err.Error()),
        )
    }

    l.Debug("Implementations конфигурация: используются значения по умолчанию",
        slog.String("config_export", implConfig.ConfigExport),
        slog.String("db_create", implConfig.DBCreate),
    )

    return implConfig, nil
}
```

#### Logging Configuration Loading

```go
func loadLoggingConfig(l *slog.Logger, cfg *Config) (*LoggingConfig, error) {
    // Check if logging config exists in AppConfig
    if cfg.AppConfig != nil && (cfg.AppConfig.Logging != LoggingConfig{}) {
        return &cfg.AppConfig.Logging, nil
    }

    loggingConfig := getDefaultLoggingConfig()

    if err := cleanenv.ReadEnv(loggingConfig); err != nil {
        l.Warn("Ошибка загрузки Logging конфигурации из переменных окружения",
            slog.String("error", err.Error()),
        )
    }

    return loggingConfig, nil
}
```

### Backward Compatibility Features

The system maintains backward compatibility with existing configurations:

```go
// Test for backward compatibility - production configs without new sections
func TestConfig_ProductionBackwardCompat(t *testing.T) {
    // Read production-like config from testdata
    data, err := os.ReadFile("testdata/production_config.yaml")
    require.NoError(t, err, "Не удалось прочитать testdata/production_config.yaml")

    // Parse config into AppConfig
    var appConfig AppConfig
    err = parseYAML(data, &appConfig)

    // Verify successful parsing
    require.NoError(t, err, "Production конфиг должен парситься без ошибок")

    // Check that main fields parsed correctly
    assert.Equal(t, "Info", appConfig.LogLevel)
    assert.Equal(t, "/tmp/benadis", appConfig.WorkDir)
    assert.Equal(t, 1545, appConfig.Rac.Port)

    // Verify missing sections have zero values (no panic)
    assert.Equal(t, ImplementationsConfig{}, appConfig.Implementations)
    assert.Equal(t, LoggingConfig{}, appConfig.Logging)
}
```

**Section sources**
- [config.go](file://internal/config/config.go#L482-L516)
- [config.go](file://internal/config/config.go#L518-L534)
- [implementations_test.go](file://internal/config/implementations_test.go#L123-L144)

## Environment Variables and Priority

The configuration system supports environment variables with a clear priority hierarchy and enhanced modular configuration.

### Variable Naming Convention

Each module uses specific prefixes:

- **APP**: Application-level variables
- **BR**: Benadis Runner global variables
- **DBRESTORE**: Database restore operations
- **SERVICE**: Service mode operations
- **GIT**: Git operations
- **LOG**: Logging configuration
- **IMPL**: Implementation configuration

### Priority Order

1. **Environment Variables** (highest priority)
2. **Configuration Files** (medium priority)
3. **Default Values** (lowest priority)

### Environment Variable Mapping

```yaml
# From action.yaml
env:
  GITEA_URL: ${{ inputs.giteaURL }}
  REPOSITORY: ${{ inputs.repository }}
  ACCESS_TOKEN: ${{ inputs.accessToken }}
  COMMAND: ${{ inputs.command }}
  LOG_LEVEL: ${{ inputs.logLevel }}
```

### Enhanced Environment Variable Integration

The system uses the `cleanenv` package for environment variable parsing with enhanced support for nested structures:

```go
func GetInputParams() *InputParams {
    inputParams := &InputParams{}
    if err := cleanenv.ReadEnv(inputParams); err != nil {
        return nil
    }
    return inputParams
}
```

### Implementation Configuration Environment Variables

```yaml
# Implementation configuration environment variables
BR_IMPL_CONFIG_EXPORT: "1cv8"  # Override config_export
BR_IMPL_DB_CREATE: "1cv8"      # Override db_create
```

### Logging Configuration Environment Variables

```yaml
# Logging configuration environment variables
BR_LOG_LEVEL: "info"      # Override logging.level
BR_LOG_FORMAT: "text"     # Override logging.format
BR_LOG_OUTPUT: "stderr"   # Override logging.output
BR_LOG_FILE_PATH: "/var/log/apk-ci.log"  # Override logging.filePath
```

**Section sources**
- [config.go](file://internal/config/config.go#L106-L127)
- [config.go](file://internal/config/config.go#L409-L422)
- [implementations_test.go](file://internal/config/implementations_test.go#L47-L64)

## Common Configuration Patterns

### Database Connection Patterns

#### Production vs Test Separation

```yaml
# Production database
V8_PROD_DB:
  one-server: PROD-SERVER
  prod: true
  dbserver: PROD-MSSQL

# Test database (related to production)
V8_TEST_DB:
  one-server: TEST-SERVER
  prod: false
  dbserver: TEST-MSSQL
```

#### Cross-Server Operations

```yaml
# Define relationships between databases
FindRelatedDatabase(dbName string) (string, error) {
    // Logic to find related databases
    // Returns related database name or error
}
```

### Service Configuration Patterns

#### Timeout Configuration

```yaml
# Consistent timeout patterns
timeout: 30s
timeout: 300s
timeout: 240m
timeout: 5m
```

#### Retry Configuration

```yaml
# Retry patterns for reliability
retryAttempts: 3
retryDelay: "1s"
maxRetryDelay: "30s"
```

### Logging Configuration Patterns

```yaml
logging:
  level: "info"
  format: "text"
  output: "stderr"
  filePath: "/var/log/apk-ci.log"
  maxSize: 100
  maxBackups: 3
  maxAge: 28
  compress: true
```

### Implementation Configuration Patterns

```yaml
implementations:
  config_export: "1cv8"  # Default: "1cv8"
  db_create: "1cv8"      # Default: "1cv8"
```

**Section sources**
- [config.go](file://internal/config/config.go#L94-L101)
- [app.yaml](file://config/app.yaml#L122-L138)
- [implementations_test.go](file://internal/config/implementations_test.go#L14-L34)

## Error Handling and Validation

The configuration system implements comprehensive error handling, validation mechanisms, and enhanced backward compatibility.

### Required Parameter Validation

```go
func validateRequiredParams(inputParams *InputParams, l *slog.Logger) error {
    var missingParams []string

    // Check mandatory parameters
    if inputParams.GHAActor == "" {
        missingParams = append(missingParams, "ACTOR")
    }
    if inputParams.GHAGiteaURL == "" {
        missingParams = append(missingParams, "GITEAURL")
    }
    if inputParams.GHARepository == "" {
        missingParams = append(missingParams, "REPOSITORY")
    }
    if inputParams.GHAAccessToken == "" {
        missingParams = append(missingParams, "ACCESSTOKEN")
    }
    if inputParams.GHAAccessToken == "" {
        missingParams = append(missingParams, "ACCESSTOKEN")
    }
    if inputParams.GHACommand == "" {
        missingParams = append(missingParams, "COMMAND")
    }

    if len(missingParams) > 0 {
        missingParamsStr := strings.Join(missingParams, ", ")
        errorMsg := fmt.Sprintf("Отсутствуют обязательные параметры конфигурации: %s", missingParamsStr)
        l.Error(errorMsg)
        return errors.New(errorMsg)
    }

    return nil
}
```

### Enhanced Configuration Validation

#### Implementations Configuration Validation

```go
func (c *ImplementationsConfig) Validate() error {
    // Apply defaults for empty values
    if c.ConfigExport == "" {
        c.ConfigExport = "1cv8"
    }
    if c.DBCreate == "" {
        c.DBCreate = "1cv8"
    }

    validConfigExport := map[string]bool{"1cv8": true, "ibcmd": true, "native": true}
    validDBCreate := map[string]bool{"1cv8": true, "ibcmd": true}

    if !validConfigExport[c.ConfigExport] {
        return fmt.Errorf("недопустимое значение ConfigExport: %q, допустимые: 1cv8, ibcmd, native", c.ConfigExport)
    }
    if !validDBCreate[c.DBCreate] {
        return fmt.Errorf("недопустимое значение DBCreate: %q, допустимые: 1cv8, ibcmd", c.DBCreate)
    }
    return nil
}
```

#### Database Configuration Validation

```go
func (cfg *Config) IsProductionDb(dbName string) bool {
    if cfg.DbConfig == nil {
        return false
    }
    if dbInfo, exists := cfg.DbConfig[dbName]; exists {
        return dbInfo.Prod
    }
    return false
}
```

#### Configuration Reloading

```go
func (cfg *Config) ReloadConfig() error {
    // Reload application configuration
    appConfig, err := loadAppConfig(cfg.Logger, cfg)
    if err != nil {
        return fmt.Errorf("failed to reload app config: %w", err)
    }
    cfg.AppConfig = appConfig

    // Reload other configurations...
    return nil
}
```

### Error Recovery Strategies

1. **Graceful Degradation**: Use defaults when configuration fails
2. **Logging**: Comprehensive error logging with context
3. **Validation**: Early detection of configuration issues
4. **Reload Capability**: Runtime configuration updates
5. **Backward Compatibility**: Graceful handling of missing configuration sections

**Section sources**
- [config.go](file://internal/config/config.go#L424-L455)
- [config.go](file://internal/config/config.go#L305-L326)
- [implementations_test.go](file://internal/config/implementations_test.go#L151-L181)

## Environment-Specific Configuration

The configuration system supports multiple deployment environments with environment-specific settings and enhanced modular configuration.

### Environment Detection

```go
func (cfg *Config) Environment() string {
    return cfg.Env
}
```

### Environment-Specific Patterns

#### Development Environment

```yaml
# Development settings
workDir: "/tmp/benadis-dev"
tmpDir: "/tmp/benadis-dev/temp"
logLevel: "Debug"
timeout: 60
```

#### Production Environment

```yaml
# Production settings
workDir: "/var/lib/benadis"
tmpDir: "/var/tmp/benadis"
logLevel: "Info"
timeout: 300
```

#### Staging Environment

```yaml
# Staging settings
workDir: "/opt/benadis/staging"
tmpDir: "/opt/benadis/staging/temp"
logLevel: "Warn"
timeout: 120
```

### Configuration File Organization

#### Centralized Configuration

```
config/
├── app.yaml                    # Base application settings
├── dbconfig.yaml              # Database definitions
├── secret.yaml                # Sensitive data
├── project.yaml               # Project-specific settings
├── menu_main.yaml             # Main workflow templates
└── menu_debug.yaml            # Debug workflow templates
```

#### Environment-Specific Overrides

```
environments/
├── dev/
│   ├── app.yaml
│   └── dbconfig.yaml
├── staging/
│   ├── app.yaml
│   └── dbconfig.yaml
└── production/
    ├── app.yaml
    └── dbconfig.yaml
```

### Deployment Strategies

#### GitOps Approach

```yaml
# Using remote configuration files
configSystem: 'https://regdv.apkholding.ru/api/v1/repos/gitops-tools/gitops_congif/contents/app.yaml?ref=main'
configSecret: 'https://regdv.apkholding.ru/api/v1/repos/gitops-tools/gitops_congif/contents/secret.yaml?ref=main'
configDbData: 'https://regdv.apkholding.ru/api/v1/repos/gitops-tools/gitops_congif/contents/dbconfig.yaml?ref=main'
```

#### Local Configuration

```yaml
# Using local configuration files
configSystem: './config/app.yaml'
configProject: './config/project.yaml'
configSecret: './config/secret.yaml'
configDbData: './config/dbconfig.yaml'
```

### Modular Configuration Management

The enhanced system supports modular configuration management:

```yaml
# Example of modular configuration structure
app:
  # Core application settings
  logLevel: "Info"
  workDir: "/var/lib/benadis"
  tmpDir: "/var/tmp/benadis"
  timeout: 300

implementations:
  # Operation implementation selection
  config_export: "1cv8"
  db_create: "1cv8"

logging:
  # Enhanced logging configuration
  level: "info"
  format: "text"
  output: "stderr"
  filePath: "/var/log/apk-ci.log"
  maxSize: 100
  maxBackups: 3
  maxAge: 28
  compress: true
```

**Section sources**
- [config.go](file://internal/config/config.go#L134-L137)
- [action.yaml](file://config/action.yaml#L22-L37)
- [app.yaml](file://config/app.yaml#L1-L138)

## Best Practices

### Configuration Management

#### 1. Use Environment Variables for Secrets

```yaml
# Good: Use environment variables for sensitive data
PASSWORD: ${DB_PASSWORD}

# Avoid: Hardcoding secrets in configuration files
PASSWORD: "supersecret123"
```

#### 2. Implement Proper Validation

```go
// Validate configuration during startup
if err := validateConfiguration(cfg); err != nil {
    log.Fatalf("Invalid configuration: %v", err)
}
```

#### 3. Use Descriptive Names

```yaml
# Good: Descriptive database names
V8_ARCH_APK_CENTER_2IS:
  one-server: MSK-AS-ARCH-001
  prod: false
  dbserver: MSK-SQL-ARCH-01

# Avoid: Generic names
DB1:
  one-server: SERVER1
  prod: false
  dbserver: MSSQL1
```

### Security Considerations

#### 1. Secure Secret Storage

```yaml
# Use encrypted secrets in CI/CD
secrets:
  DB_PASSWORD: ${{ secrets.DB_PASSWORD }}
  GITEA_TOKEN: ${{ secrets.GITEA_TOKEN }}
```

#### 2. Limit Access to Configuration Files

```bash
# Restrict file permissions
chmod 600 config/*.yaml
chown benadis:benadis config/*.yaml
```

#### 3. Audit Configuration Changes

```yaml
# Track configuration changes
audit:
  enabled: true
  log_level: "info"
  retention_days: 30
```

### Performance Optimization

#### 1. Optimize Configuration Loading

```go
// Cache frequently accessed configurations
func (cfg *Config) GetCachedDatabaseInfo(dbName string) *DatabaseInfo {
    if cached, exists := cfg.cache.Get(dbName); exists {
        return cached.(*DatabaseInfo)
    }
    return cfg.GetDatabaseInfo(dbName)
}
```

#### 2. Use Efficient Data Structures

```go
// Use maps for fast database lookups
type Config struct {
    DbConfig map[string]*DatabaseInfo
    // Other fields...
}
```

#### 3. Implement Configuration Validation

```go
// Validate configuration before use
func (cfg *Config) Validate() error {
    if cfg.AppConfig.Timeout <= 0 {
        return errors.New("invalid timeout value")
    }
    return nil
}
```

### Monitoring and Observability

#### 1. Monitor Configuration Changes

```go
// Log configuration changes
func (cfg *Config) ReloadConfig() error {
    oldConfig := cfg.AppConfig
    // Reload logic...
    if !reflect.DeepEqual(oldConfig, cfg.AppConfig) {
        cfg.Logger.Info("Configuration reloaded", "timestamp", time.Now())
    }
    return nil
}
```

#### 2. Implement Health Checks

```go
// Configuration health check
func (cfg *Config) HealthCheck() error {
    if cfg.AppConfig == nil {
        return errors.New("application configuration not loaded")
    }
    if cfg.DbConfig == nil {
        return errors.New("database configuration not loaded")
    }
    return nil
}
```

#### 3. Use Structured Logging

```go
// Log configuration with context
cfg.Logger.Info("Configuration loaded",
    "app_config", cfg.AppConfig,
    "db_count", len(cfg.DbConfig),
    "project_name", cfg.ProjectName,
    "implementations", cfg.ImplementationsConfig,
    "logging", cfg.LoggingConfig,
)
```

### Documentation and Maintenance

#### 1. Document Configuration Options

```yaml
# Document configuration options
# logLevel: Debug | Info | Warn | Error
# timeout: Integer value in seconds
# prod: true | false
# implementations.config_export: 1cv8 | ibcmd | native
# implementations.db_create: 1cv8 | ibcmd
# logging.level: debug | info | warn | error
# logging.format: json | text
# logging.output: stdout | stderr | file
```

#### 2. Maintain Configuration Templates

```yaml
# Keep configuration templates up-to-date
templates:
  - app.yaml.template
  - dbconfig.yaml.template
  - secret.yaml.template
```

#### 3. Implement Configuration Migration

```go
// Handle configuration version upgrades
func MigrateConfig(cfg *Config) error {
    if cfg.Version < 2 {
        return migrateToV2(cfg)
    }
    return nil
}
```

### Backward Compatibility Guidelines

#### 1. Maintain Zero Value Behavior

```go
// Ensure missing sections have zero values, not panics
assert.Equal(t, ImplementationsConfig{}, appConfig.Implementations)
assert.Equal(t, LoggingConfig{}, appConfig.Logging)
```

#### 2. Provide Default Values

```go
// Default implementations configuration
func getDefaultImplementationsConfig() *ImplementationsConfig {
    return &ImplementationsConfig{
        ConfigExport: "1cv8",
        DBCreate:     "1cv8",
    }
}
```

#### 3. Validate Environment Variable Overrides

```go
// Validate environment variable overrides
func (c *ImplementationsConfig) Validate() error {
    // Apply defaults for empty values
    if c.ConfigExport == "" {
        c.ConfigExport = "1cv8"
    }
    if c.DBCreate == "" {
        c.DBCreate = "1cv8"
    }
    // ... validation logic
}
```

**Section sources**
- [config.go](file://internal/config/config.go#L589-L595)
- [implementations_test.go](file://internal/config/implementations_test.go#L96-L105)
- [app.yaml](file://config/app.yaml#L1-L138)