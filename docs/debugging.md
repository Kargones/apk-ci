# Отладка benadis-runner через Delve

## Обзор

Проект поддерживает отладку через [Delve](https://github.com/go-delve/delve) — отладчик для Go. Доступны режимы: локальная отладка, отладка в Docker-контейнере и отладка отдельных тестов.

## Установка Delve

```bash
# Установка через make (рекомендуется)
make setup-dev

# Или вручную
go install github.com/go-delve/delve/cmd/dlv@latest
```

## Доступные make targets

| Target | Описание |
|--------|----------|
| `make debug` | Сборка с debug информацией (`-gcflags="all=-N -l"`) |
| `make debug-run` | Запуск под Delve в headless режиме |
| `make debug-attach PID=<pid>` | Подключение к запущенному процессу |
| `make debug-test PKG=<pkg> TEST=<name>` | Отладка конкретного теста |
| `make debug-docker` | Запуск Docker-контейнера с Delve |
| `make debug-docker-stop` | Остановка debug Docker-контейнера |
| `make debug-clean` | Удаление debug бинарника |

## Workflow: Локальная отладка

### 1. Сборка debug бинарника

```bash
make debug
```

Создаёт `./build/benadis-runner-debug` с отключённой оптимизацией для корректной работы breakpoints.

### 2. Запуск под Delve

```bash
# С переменными окружения для конкретной команды
BR_COMMAND=nr-version make debug-run

# С другим портом
BR_DEBUG_PORT=3456 make debug-run
```

Delve запускается в headless режиме и слушает на порту 2345 (по умолчанию).

### 3. Подключение из IDE

#### VS Code

Используйте конфигурацию "Remote Attach (Delve)" из `.vscode/launch.json`:
1. Запустите `make debug-run` в терминале
2. В VS Code: Run → Start Debugging → "Remote Attach (Delve)"
3. Установите breakpoints и отлаживайте

#### GoLand

1. Run → Edit Configurations → "+" → Go Remote
2. Host: `127.0.0.1`, Port: `2345`
3. Запустите `make debug-run` в терминале
4. Запустите конфигурацию "Go Remote" в GoLand

## Workflow: Подключение к запущенному процессу

```bash
# Найти PID процесса benadis-runner
pgrep -f benadis-runner

# Подключить Delve к процессу
make debug-attach PID=12345

# Подключитесь из IDE к localhost:2345
```

## Workflow: Отладка в Docker

```bash
# Запуск debug контейнера (запускается в фоне)
make debug-docker

# С передачей env переменных приложению
make debug-docker DOCKER_ARGS='-e BR_COMMAND=nr-version -e BR_INFOBASE_NAME=MyBase'

# Подключитесь из IDE к localhost:2345
# VS Code: конфигурация "Docker Remote Attach (Delve)"

# Остановка
make debug-docker-stop
```

Docker образ собирается из `Dockerfile.debug` и включает Delve и debug бинарник.

## Workflow: Отладка тестов

```bash
# Отладка конкретного теста
make debug-test PKG=./internal/pkg/tracing TEST=TestDefaultConfig

# Подключитесь из IDE к localhost:2345
```

## Переменные окружения

| Переменная | По умолчанию | Описание |
|------------|-------------|----------|
| `BR_DEBUG_PORT` | `2345` | Порт для Delve headless listener |

## Безопасность

- Debug порт (2345) предназначен **только для локальной разработки**
- Docker debug контейнер привязывает порт к `127.0.0.1` (только localhost)
- **Не используйте** debug build и `Dockerfile.debug` в production
- Debug build не имеет оптимизаций и значительно медленнее

## Очистка debug артефактов

```bash
# Удаление debug бинарника
make debug-clean

# Полная очистка (включая debug)
make clean
```

## Известные ограничения

- Delve не поддерживает Windows ARM
- Debug build значительно медленнее production build
- Docker debug образ значительно больше production образа
- Delve требует Go toolchain для `dlv test`
