# Тестирование эмуляции Gitea Action

Этот документ описывает тест `main_test.go`, который эмулирует запуск приложения apk-ci из Gitea Action с параметрами из `action.yaml`.

## Назначение

Тест `TestMainWithGiteaActionParams` создан для:

1. **Эмуляции выполнения Gitea Action** - воспроизводит условия запуска приложения через GitHub/Gitea Actions
2. **Проверки корректности передачи параметров** - убеждается, что все входные параметры из `action.yaml` правильно преобразуются в переменные окружения
3. **Валидации конфигурации** - проверяет, что приложение может корректно загрузить конфигурацию с параметрами из Action

## Структура теста

### TestMainWithGiteaActionParams

Основной тест, который включает три сценария:

#### 1. convert_command_with_gitea_action_params
- **Команда**: `convert`
- **Окружение**: `dev`
- **Уровень логирования**: `Debug`
- **База данных**: `V8_DEV_TEST`
- **Назначение**: Тестирует стандартный сценарий конвертации конфигурации

#### 2. dbrestore_command_with_gitea_action_params
- **Команда**: `dbrestore`
- **Окружение**: `dev`
- **Уровень логирования**: `Info`
- **База данных**: `V8_DEV_TEST_RESTORE`
- **Особенности**: Включает завершение сессий (`terminateSessions: true`)
- **Назначение**: Тестирует сценарий восстановления базы данных

#### 3. service_mode_enable_with_gitea_action_params
- **Команда**: `service-mode-enable`
- **Окружение**: `prod`
- **Уровень логирования**: `Warn`
- **База данных**: `V8_PROD_MAIN`
- **Актор**: `admin`
- **Назначение**: Тестирует сценарий включения сервисного режима в продуктивной среде

### TestConfigLoadingWithGiteaActionParams

Вспомогательный тест, который:
- Устанавливает базовый набор переменных окружения
- Проверяет корректность их установки
- Валидирует соответствие ожидаемым значениям

## Соответствие action.yaml

Тест использует переменные окружения, которые соответствуют входным параметрам из `action.yaml`:

| Параметр action.yaml | Переменная окружения | Переменная apk-ci |
|---------------------|---------------------|---------------------------|
| `giteaURL` | `INPUT_GITEAURL` | - |
| `repository` | `INPUT_REPOSITORY` | - |
| `accessToken` | `INPUT_ACCESSTOKEN` | `BR_ACCESS_TOKEN` |
| `command` | `INPUT_COMMAND` | `BR_COMMAND` |
| `logLevel` | `INPUT_LOGLEVEL` | `BR_LOGLEVEL` |
| `issueNumber` | `INPUT_ISSUENUMBER` | `BR_ISSUE_NUMBER` |
| `configSystem` | `INPUT_CONFIGSYSTEM` | `BR_CONFIG_SYSTEM` |
| `configProject` | `INPUT_CONFIGPROJECT` | `BR_CONFIG_PROJECT` |
| `configSecret` | `INPUT_CONFIGSECRET` | `BR_CONFIG_SECRET` |
| `configDbData` | `INPUT_CONFIGDBDATA` | `BR_CONFIG_DBDATA` |
| `actor` | `INPUT_ACTOR` | `BR_ACTOR` |
| `dbName` | `INPUT_DBNAME` | `BR_INFOBASE_NAME` |
| `terminateSessions` | `INPUT_TERMINATESESSIONS` | `BR_TERMINATE_SESSIONS` |

## Запуск тестов

```bash
# Запуск всех тестов в директории cmd/apk-ci
go test -v ./cmd/github.com/Kargones/apk-ci/

# Запуск конкретного теста
go test -v ./cmd/github.com/Kargones/apk-ci/ -run TestMainWithGiteaActionParams

# Запуск с подробным выводом
go test -v ./cmd/github.com/Kargones/apk-ci/ -run TestMainWithGiteaActionParams -args -test.v
```

## Ожидаемый результат

При успешном выполнении тест должен:

1. **Установить все переменные окружения** согласно сценарию
2. **Вывести информацию о установленных переменных** в лог
3. **Завершиться со статусом PASS** для всех подтестов

Пример успешного вывода:
```
=== RUN   TestMainWithGiteaActionParams/convert_command_with_gitea_action_params
    main_test.go:230: Тест convert_command_with_gitea_action_params: переменные окружения установлены корректно
    main_test.go:231: BR_COMMAND: convert
    main_test.go:232: BR_ACTOR: test-actor
    main_test.go:233: BR_ENV: dev
    main_test.go:234: BR_LOGLEVEL: Debug
    main_test.go:235: BR_INFOBASE_NAME: V8_DEV_TEST
--- PASS: TestMainWithGiteaActionParams/convert_command_with_gitea_action_params (0.00s)
```

## Ограничения

### Безопасность выполнения

Вызов функции `main()` в тесте **закомментирован** по следующим причинам:

1. **Реальные операции**: `main()` может выполнять операции с файловой системой, базами данных и внешними сервисами
2. **Побочные эффекты**: Может создавать/изменять файлы, подключаться к базам данных
3. **Зависимости**: Требует наличия внешних сервисов (Gitea, базы данных, 1C платформы)

### Рекомендации для полного тестирования

Для более полного тестирования рекомендуется:

1. **Выделить логику инициализации** в отдельную функцию
2. **Использовать dependency injection** для внешних зависимостей
3. **Создать моки** для внешних сервисов
4. **Тестировать компоненты по отдельности**

## Использование в CI/CD

Этот тест может быть интегрирован в CI/CD пайплайн для:

1. **Проверки совместимости** с новыми версиями action.yaml
2. **Валидации изменений** в логике обработки параметров
3. **Регрессионного тестирования** при изменении конфигурации

## Расширение тестов

Для добавления новых сценариев:

1. Добавьте новый элемент в массив `tests` в `TestMainWithGiteaActionParams`
2. Определите уникальное имя сценария
3. Установите соответствующие переменные окружения в поле `envs`
4. При необходимости добавьте специфичные проверки

Пример добавления нового сценария:

```go
{
    name: "new_command_scenario",
    envs: map[string]string{
        "INPUT_COMMAND": "new-command",
        "BR_COMMAND": "new-command",
        // ... другие переменные
    },
},
```

## Связанные файлы

- `action.yaml` - Определение Gitea Action с входными параметрами
- `cmd/github.com/Kargones/apk-ci/main.go` - Основная точка входа приложения
- `internal/config/config.go` - Логика загрузки конфигурации
- `internal/config/config_test.go` - Тесты конфигурации