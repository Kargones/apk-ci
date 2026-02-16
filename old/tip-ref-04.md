# Детальный план реализации библиотечных функций benadis-rac

## Обзор

Данный документ содержит детальный план реализации библиотечных функций для управления сервисным режимом 1C через RAC (Remote Administration Console), основанный на анализе существующей архитектуры проекта и найденных библиотек в директории `/tmp`.

## Анализ существующих библиотек

### Структура библиотеки benadis-rac

В директории `/tmp` обнаружена полнофункциональная библиотека для управления сервисным режимом 1C:

#### Основные файлы:
- `benadis_rac.go` - основной интерфейс библиотеки
- `benadis_rac_helpers.go` - вспомогательные функции
- `internal/rac/rac.go` - внутренний клиент RAC
- `internal/rac/service_mode.go` - функции управления сервисным режимом
- `internal/config/config.go` - конфигурация
- `internal/constants/constants.go` - константы

#### Ключевые интерфейсы и структуры:

```go
// ServiceModeManager - основной интерфейс для управления сервисным режимом
type ServiceModeManager interface {
    ManageServiceMode(ctx context.Context, enable bool) error
    EnableServiceMode(ctx context.Context) error
    DisableServiceMode(ctx context.Context) error
    GetServiceModeStatus(ctx context.Context) (bool, error)
}

// Client - основная структура клиента
type Client struct {
    config *Config
    logger Logger
}

// Config - конфигурация RAC клиента
type Config struct {
    ServerHost            string
    RACPort              int
    RACPath              string
    BaseName             string
    ClusterAdmin         string
    ClusterAdminPassword string
    DBAdmin              string
    DBAdminPassword      string
    ConnectionTimeout    time.Duration
    CommandTimeout       time.Duration
    RetryCount           int
    RetryDelay           time.Duration
    DeniedMessage        string
    PermissionCode       string
}
```

## План интеграции в существующую архитектуру

### 1. Интеграция с системой логирования

#### Использование существующего логгера
Библиотека уже содержит интерфейс `Logger` и реализацию `SlogLogger`, которые совместимы с существующей системой логирования проекта:

```go
// Интерфейс логгера из библиотеки
type Logger interface {
    Debug(msg string, args ...interface{})
    Info(msg string, args ...interface{})
    Warn(msg string, args ...interface{})
    Error(msg string, args ...interface{})
    DebugCommand(msg string, command []string)
}
```

**Рекомендации по интеграции:**
- Использовать существующую функцию `NewLogger()` из `config.go` проекта
- Адаптировать интерфейс логгера библиотеки к существующему логгеру проекта
- Обеспечить единообразие уровней логирования

### 2. Интеграция с системой конфигурации

#### Использование существующей конфигурации
Библиотека поддерживает загрузку конфигурации из YAML файлов, что соответствует архитектуре проекта:

**Структура конфигурационных файлов:**
- `config.yaml` - основные настройки RAC
- `secret.yaml` - учетные данные
- `project.yaml` - настройки проекта

**Рекомендации по интеграции:**
- Расширить существующую структуру `Config` в `config.go` проекта
- Добавить секцию `ServiceMode` в конфигурацию
- Использовать существующие функции загрузки конфигурации

### 3. Реализация функции ManageServiceMode

#### Основная функция управления сервисным режимом

```go
// ManageServiceMode - главная функция управления сервисным режимом
func ManageServiceMode(ctx context.Context, enable bool) error {
    // 1. Загрузка конфигурации
    config, err := loadServiceModeConfig()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }
    
    // 2. Создание логгера
    logger := NewLogger(config.LogLevel)
    
    // 3. Создание RAC клиента
    client := NewClient(config, logger)
    
    // 4. Получение UUID кластера и информационной базы
    clusterUUID, err := client.getClusterUUID(ctx)
    if err != nil {
        return fmt.Errorf("failed to get cluster UUID: %w", err)
    }
    
    infobaseUUID, err := client.getInfobaseUUID(ctx, clusterUUID)
    if err != nil {
        return fmt.Errorf("failed to get infobase UUID: %w", err)
    }
    
    // 5. Управление сервисным режимом
    if enable {
        return client.EnableServiceMode(ctx)
    } else {
        return client.DisableServiceMode(ctx)
    }
}
```

### 4. Интеграция с CLI интерфейсом

#### Расширение команд в main.go

```go
// Добавление новых команд в switch statement
switch action {
    // ... существующие команды ...
    
    case "service-mode-enable":
        return app.EnableServiceMode()
    case "service-mode-disable":
        return app.DisableServiceMode()
    case "service-mode-status":
        return app.GetServiceModeStatus()
}
```

#### Новые функции в internal/app/app.go

```go
// EnableServiceMode включает сервисный режим
func EnableServiceMode() error {
    ctx := context.Background()
    return ManageServiceMode(ctx, true)
}

// DisableServiceMode отключает сервисный режим
func DisableServiceMode() error {
    ctx := context.Background()
    return ManageServiceMode(ctx, false)
}

// GetServiceModeStatus получает статус сервисного режима
func GetServiceModeStatus() error {
    ctx := context.Background()
    
    config, err := loadServiceModeConfig()
    if err != nil {
        return err
    }
    
    logger := NewLogger(config.LogLevel)
    client := NewClient(config, logger)
    
    status, err := client.GetServiceModeStatus(ctx)
    if err != nil {
        return err
    }
    
    if status {
        logger.Info("Service mode: ENABLED")
    } else {
        logger.Info("Service mode: DISABLED")
    }
    
    return nil
}
```

### 5. Обработка ошибок и безопасность

#### Стратегия обработки ошибок
- Использование wrapped errors с контекстом
- Логирование всех критических операций
- Graceful degradation при сетевых ошибках
- Retry логика для временных сбоев

#### Безопасность
- Маскирование паролей в логах
- Безопасное хранение учетных данных
- Валидация входных параметров
- Таймауты для предотвращения зависания

### 6. Тестирование

#### Unit тесты
```go
// Пример unit теста
func TestManageServiceMode(t *testing.T) {
    tests := []struct {
        name    string
        enable  bool
        wantErr bool
    }{
        {"Enable service mode", true, false},
        {"Disable service mode", false, false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := context.Background()
            err := ManageServiceMode(ctx, tt.enable)
            if (err != nil) != tt.wantErr {
                t.Errorf("ManageServiceMode() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

#### Integration тесты
- Тестирование с реальным RAC сервером
- Проверка корректности выполнения команд
- Валидация состояний сервисного режима

### 7. Документация

#### API документация
- Описание всех публичных функций
- Примеры использования
- Описание конфигурационных параметров

#### Руководство по эксплуатации
- Инструкции по настройке
- Troubleshooting guide
- Best practices

## Детальная реализация компонентов

### 1. Конфигурация (config.go)

```go
// ServiceModeConfig конфигурация для управления сервисным режимом
type ServiceModeConfig struct {
    ServerHost            string        `yaml:"server_host"`
    RACPort              int           `yaml:"rac_port"`
    RACPath              string        `yaml:"rac_path"`
    BaseName             string        `yaml:"base_name"`
    ClusterAdmin         string        `yaml:"cluster_admin"`
    ClusterAdminPassword string        `yaml:"cluster_admin_password"`
    DBAdmin              string        `yaml:"db_admin"`
    DBAdminPassword      string        `yaml:"db_admin_password"`
    ConnectionTimeout    time.Duration `yaml:"connection_timeout"`
    CommandTimeout       time.Duration `yaml:"command_timeout"`
    RetryCount           int           `yaml:"retry_count"`
    RetryDelay           time.Duration `yaml:"retry_delay"`
    DeniedMessage        string        `yaml:"denied_message"`
    PermissionCode       string        `yaml:"permission_code"`
    LogLevel             string        `yaml:"log_level"`
}

// Добавление в основную структуру Config
type Config struct {
    // ... существующие поля ...
    ServiceMode *ServiceModeConfig `yaml:"service_mode"`
}
```

### 2. RAC клиент (rac_client.go)

```go
// RACClient клиент для работы с RAC
type RACClient struct {
    config *ServiceModeConfig
    logger Logger
}

// NewRACClient создает новый RAC клиент
func NewRACClient(config *ServiceModeConfig, logger Logger) *RACClient {
    return &RACClient{
        config: config,
        logger: logger,
    }
}

// ExecuteCommand выполняет RAC команду с retry логикой
func (c *RACClient) ExecuteCommand(ctx context.Context, args []string) (string, error) {
    var lastErr error
    
    for attempt := 1; attempt <= c.config.RetryCount; attempt++ {
        c.logger.Debug("Executing RAC command", "attempt", attempt)
        
        output, err := c.executeCommandOnce(ctx, args)
        if err == nil {
            return output, nil
        }
        
        lastErr = err
        c.logger.Warn("RAC command failed", "attempt", attempt, "error", err)
        
        if attempt < c.config.RetryCount {
            time.Sleep(c.config.RetryDelay)
        }
    }
    
    return "", fmt.Errorf("command failed after %d attempts: %w", c.config.RetryCount, lastErr)
}
```

### 3. Управление сессиями (session_manager.go)

```go
// SessionManager управляет сессиями пользователей
type SessionManager struct {
    client *RACClient
    logger Logger
}

// TerminateAllSessions завершает все активные сессии
func (sm *SessionManager) TerminateAllSessions(ctx context.Context, clusterUUID, infobaseUUID string) error {
    sessions, err := sm.GetActiveSessions(ctx, clusterUUID, infobaseUUID)
    if err != nil {
        return fmt.Errorf("failed to get active sessions: %w", err)
    }
    
    if len(sessions) == 0 {
        sm.logger.Info("No active sessions found")
        return nil
    }
    
    sm.logger.Info("Terminating sessions", "count", len(sessions))
    
    for _, sessionID := range sessions {
        if err := sm.TerminateSession(ctx, clusterUUID, sessionID); err != nil {
            sm.logger.Warn("Failed to terminate session", "session", sessionID, "error", err)
        }
    }
    
    return nil
}
```

## Заключение

Предложенный план реализации обеспечивает:

1. **Полную интеграцию** с существующей архитектурой проекта
2. **Использование существующих** систем логирования и конфигурации
3. **Реализацию функции ManageServiceMode** согласно требованиям
4. **Безопасность и надежность** операций
5. **Расширяемость** для будущих функций
6. **Соответствие** принципам проектирования существующего кода

Библиотека benadis-rac готова к интеграции и содержит все необходимые компоненты для управления сервисным режимом 1C через RAC.