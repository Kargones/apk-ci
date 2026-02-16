# Тест эмуляции Gitea Action

## Что было создано

Создан тест `cmd/github.com/Kargones/apk-ci/main_test.go`, который эмулирует запуск приложения benadis-runner из Gitea Action с параметрами из `action.yaml`.

## Файлы

1. **`cmd/github.com/Kargones/apk-ci/main_test.go`** - Основной тестовый файл
2. **`docs/testing-gitea-action.md`** - Подробная документация по тестированию

## Функциональность

### TestMainWithGiteaActionParams
Тестирует три сценария:
- **convert** - Конвертация конфигурации (dev окружение)
- **dbrestore** - Восстановление базы данных (dev окружение)
- **service-mode-enable** - Включение сервисного режима (prod окружение)

### TestConfigLoadingWithGiteaActionParams
Проверяет корректность загрузки конфигурации с параметрами из Gitea Action.

## Как запустить

```bash
# Запуск всех тестов
go test -v ./cmd/github.com/Kargones/apk-ci/

# Запуск конкретного теста
go test -v ./cmd/github.com/Kargones/apk-ci/ -run TestMainWithGiteaActionParams
```

## Результат последнего запуска

```
=== RUN   TestMainWithGiteaActionParams
=== RUN   TestMainWithGiteaActionParams/convert_command_with_gitea_action_params
=== RUN   TestMainWithGiteaActionParams/dbrestore_command_with_gitea_action_params
=== RUN   TestMainWithGiteaActionParams/service_mode_enable_with_gitea_action_params
--- PASS: TestMainWithGiteaActionParams (0.00s)
=== RUN   TestConfigLoadingWithGiteaActionParams
--- PASS: TestConfigLoadingWithGiteaActionParams (0.00s)
PASS
ok      github.com/Kargones/apk-ci/cmd/benadis-runner       0.046s
```

## Особенности

- Тест **НЕ вызывает** функцию `main()` для безопасности
- Проверяет только корректность установки переменных окружения
- Эмулирует реальные параметры из `action.yaml`
- Покрывает основные сценарии использования

## Соответствие action.yaml

Тест использует переменные окружения, которые точно соответствуют входным параметрам из `action.yaml`:

- `INPUT_*` переменные (от Gitea Action)
- `BR_*` переменные (внутренние переменные benadis-runner)
- Пути к исполняемым файлам и конфигурациям

Подробная документация доступна в `docs/testing-gitea-action.md`.