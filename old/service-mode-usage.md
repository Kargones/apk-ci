# Управление сервисным режимом 1С

Данная документация описывает функциональность управления сервисным режимом информационных баз 1С через RAC (Remote Administration Console).

## Обзор

Библиотека `benadis-rac` предоставляет возможности для:
- Включения сервисного режима информационной базы
- Отключения сервисного режима
- Получения статуса сервисного режима
- Управления пользовательскими сессиями

## Архитектура

### Основные компоненты

1. **`internal/rac/rac.go`** - основной клиент для работы с RAC
2. **`internal/rac/service_mode.go`** - функции управления сервисным режимом
3. **`internal/servicemode/servicemode.go`** - высокоуровневый интерфейс
4. **`internal/app/app.go`** - интеграция с основным приложением

### Структуры данных

```go
// Конфигурация RAC клиента
type ServiceModeConfig struct {
    RacPath     string        // Путь к исполняемому файлу rac
    RacServer   string        // Адрес сервера 1С
    RacPort     int           // Порт RAC (по умолчанию 1545)
    RacUser     string        // Пользователь кластера
    RacPassword string        // Пароль пользователя кластера
    RacTimeout  time.Duration // Таймаут выполнения команд
    RacRetries  int           // Количество повторных попыток
}

// Статус сервисного режима
type ServiceModeStatus struct {
    Enabled        bool   // Включен ли сервисный режим
    Message        string // Сообщение сервисного режима
    ActiveSessions int    // Количество активных сессий
}
```

## Использование

### Переменные окружения

Для работы с сервисным режимом необходимо настроить следующие переменные окружения:

```bash
# Обязательные переменные
export BR_COMMAND="service-mode-enable"  # или service-mode-disable, service-mode-status
export BR_INFOBASE_NAME="MyInfobase"     # Имя информационной базы

# Конфигурация RAC
export RAC_PATH="/opt/1cv8/x86_64/8.3.27.1606/rac"  # Путь к rac
export RAC_SERVER="localhost"                        # Сервер 1С
export RAC_PORT="1545"                              # Порт RAC
export RAC_USER="admin"                             # Пользователь кластера
export RAC_PASSWORD="password"                      # Пароль
export RAC_TIMEOUT="30"                             # Таймаут в секундах
export RAC_RETRIES="3"                              # Количество попыток

# Дополнительные параметры для включения сервисного режима
export BR_SERVICE_MODE_MESSAGE="Техническое обслуживание"  # Сообщение
export BR_TERMINATE_SESSIONS="true"                        # Завершить сессии
```

### Команды

#### 1. Включение сервисного режима

```bash
# Установка переменных
export BR_COMMAND="service-mode-enable"
export BR_INFOBASE_NAME="TestBase"
export BR_SERVICE_MODE_MESSAGE="Плановое техническое обслуживание"
export BR_TERMINATE_SESSIONS="true"

# Запуск
./apk-ci
```

#### 2. Отключение сервисного режима

```bash
# Установка переменных
export BR_COMMAND="service-mode-disable"
export BR_INFOBASE_NAME="TestBase"

# Запуск
./apk-ci
```

#### 3. Проверка статуса сервисного режима

```bash
# Установка переменных
export BR_COMMAND="service-mode-status"
export BR_INFOBASE_NAME="TestBase"

# Запуск
./apk-ci
```

### Программное использование

```go
package main

import (
    "context"
    "log/slog"
    "time"
    
    "github.com/Kargones/apk-ci/internal/servicemode"
)

func main() {
    ctx := context.Background()
    logger := slog.Default()
    
    // Конфигурация
    config := servicemode.ServiceModeConfig{
        RacPath:     "rac",
        RacServer:   "localhost",
        RacPort:     1545,
        RacUser:     "admin",
        RacPassword: "password",
        RacTimeout:  30 * time.Second,
        RacRetries:  3,
    }
    
    // Включение сервисного режима
    err := servicemode.ManageServiceMode(
        ctx, 
        "enable", 
        "MyInfobase", 
        "Техническое обслуживание", 
        true, // завершить сессии
        config, 
        logger,
    )
    if err != nil {
        logger.Error("Failed to enable service mode", "error", err)
        return
    }
    
    logger.Info("Service mode enabled successfully")
}
```

## Безопасность

### Рекомендации по безопасности

1. **Не храните пароли в коде** - используйте переменные окружения
2. **Ограничьте права пользователя RAC** - предоставьте минимально необходимые права
3. **Используйте защищенные соединения** - при возможности используйте TLS
4. **Логирование** - не логируйте пароли и другую чувствительную информацию

### Пример безопасной конфигурации

```bash
# Используйте отдельного пользователя для автоматизации
export RAC_USER="automation_user"
# Пароль из защищенного хранилища
export RAC_PASSWORD="$(cat /secure/rac_password)"
# Ограниченные права доступа
export RAC_SERVER="internal-1c-server.local"
```

## Обработка ошибок

### Типичные ошибки и их решения

1. **"RAC command failed"**
   - Проверьте путь к исполняемому файлу rac
   - Убедитесь, что сервер 1С доступен
   - Проверьте правильность учетных данных

2. **"Cluster UUID not found"**
   - Проверьте подключение к серверу 1С
   - Убедитесь, что кластер запущен

3. **"Infobase UUID not found"**
   - Проверьте правильность имени информационной базы
   - Убедитесь, что база существует в кластере

4. **"Failed to terminate sessions"**
   - Некоторые сессии могут быть заблокированы
   - Попробуйте выполнить операцию без завершения сессий

### Логирование

Библиотека использует структурированное логирование через `slog`:

```go
// Уровни логирования
logger.Debug("RAC command executed", "command", cmd, "output", output)
logger.Info("Service mode enabled", "infobase", name)
logger.Warn("Retry attempt", "attempt", i, "error", err)
logger.Error("Operation failed", "error", err)
```

## Тестирование

### Модульные тесты

```bash
# Запуск модульных тестов
go test ./internal/servicemode/
```

### Интеграционные тесты

```bash
# Настройка переменных для интеграционных тестов
export TEST_RAC_PATH="/opt/1cv8/x86_64/8.3.27.1606/rac"
export TEST_RAC_SERVER="localhost"
export TEST_INFOBASE_NAME="TestBase"
export TEST_RAC_USER="admin"
export TEST_RAC_PASSWORD="password"

# Запуск интеграционных тестов
go test ./internal/servicemode/ -v
```

## Мониторинг и метрики

### Рекомендуемые метрики для мониторинга

1. **Время выполнения операций** - для выявления проблем с производительностью
2. **Количество ошибок** - для мониторинга стабильности
3. **Количество активных сессий** - для контроля нагрузки
4. **Статус сервисного режима** - для операционного мониторинга

### Пример интеграции с Prometheus

```go
// Добавление метрик (пример)
var (
    serviceModeOperations = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "service_mode_operations_total",
            Help: "Total number of service mode operations",
        },
        []string{"operation", "status"},
    )
    
    serviceModeOperationDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "service_mode_operation_duration_seconds",
            Help: "Duration of service mode operations",
        },
        []string{"operation"},
    )
)
```

## Заключение

Библиотека управления сервисным режимом предоставляет надежный и безопасный способ автоматизации операций с информационными базами 1С. Следуйте рекомендациям по безопасности и мониторингу для обеспечения стабильной работы в продуктивной среде.