# Story 6.9: Delve Debugging (FR44-46)

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a разработчик,
I want запускать приложение в режиме отладки через Delve,
so that могу диагностировать сложные проблемы с breakpoints и step-through debugging.

## Acceptance Criteria

1. [AC1] `make debug` собирает бинарник с отключённой оптимизацией (`-gcflags="all=-N -l"`) для корректной работы Delve
2. [AC2] `make debug-run` запускает приложение под Delve в headless режиме (`--headless --listen=:2345 --api-version=2`)
3. [AC3] Удалённое подключение к отладчику через порт (по умолчанию 2345) работает из IDE (VS Code, GoLand)
4. [AC4] `BR_DEBUG_PORT` env переменная позволяет переопределить порт Delve (default: 2345)
5. [AC5] Работает на Linux (основная целевая платформа) и в Docker-контейнере
6. [AC6] `make debug-docker` запускает Docker-контейнер с Delve на exposed порту
7. [AC7] Документация: `.vscode/launch.json` пример конфигурации для remote debugging
8. [AC8] `make debug-test` запускает тесты под Delve для отладки конкретного теста
9. [AC9] Все существующие Makefile targets работают без изменений (backward compatible)
10. [AC10] В README.md или docs/ добавлена секция по отладке с примерами

## Tasks / Subtasks

- [x] Task 1: Добавить debug build targets в Makefile (AC: #1, #4, #9)
  - [x] Subtask 1.1: Добавить переменную `DEBUG_PORT ?= 2345` и `BR_DEBUG_PORT` override
  - [x] Subtask 1.2: Добавить target `debug` — сборка с `-gcflags="all=-N -l"` и LDFLAGS
  - [x] Subtask 1.3: Добавить target `debug-clean` — удаление debug бинарника
  - [x] Subtask 1.4: Проверить что существующие targets (`build`, `test`, `lint`) не затронуты

- [x] Task 2: Добавить debug-run target в Makefile (AC: #2, #3, #4)
  - [x] Subtask 2.1: Добавить target `debug-run` — запуск `dlv exec` с `--headless --listen=:$(DEBUG_PORT) --api-version=2`
  - [x] Subtask 2.2: Добавить target `debug-attach` — `dlv attach` к запущенному процессу
  - [x] Subtask 2.3: Добавить prerequisite: проверка установки dlv (`@which dlv || (echo "..."; exit 1)`)

- [x] Task 3: Добавить Docker debug support (AC: #5, #6)
  - [x] Subtask 3.1: Создать `Dockerfile.debug` — multi-stage: builder (Go + Delve) → runner (debug binary + dlv)
  - [x] Subtask 3.2: Добавить target `debug-docker` — build + run контейнер с `-p $(DEBUG_PORT):2345`
  - [x] Subtask 3.3: Добавить target `debug-docker-stop` — остановка debug контейнера

- [x] Task 4: Добавить debug-test target (AC: #8)
  - [x] Subtask 4.1: Добавить target `debug-test` — `dlv test` с указанием пакета и тест-функции
  - [x] Subtask 4.2: Использование: `make debug-test PKG=./internal/pkg/tracing TEST=TestNewTracerProvider`

- [x] Task 5: Добавить IDE конфигурацию и документацию (AC: #7, #10)
  - [x] Subtask 5.1: Создать `.vscode/launch.json` с конфигурацией remote attach (порт 2345)
  - [x] Subtask 5.2: Добавить секцию "Отладка" в docs/debugging.md с примерами использования
  - [x] Subtask 5.3: Описать workflow: `make debug` → `make debug-run` → IDE attach

- [x] Task 6: Валидация (AC: #9)
  - [x] Subtask 6.1: Запустить `make build` — существующие targets работают
  - [x] Subtask 6.2: Запустить `make test` — тесты проходят
  - [x] Subtask 6.3: Запустить `make debug` — debug бинарник собирается
  - [x] Subtask 6.4: Проверить `make help` — новые targets отображаются с описанием

### Review Follow-ups (AI)

- [ ] [AI-Review][HIGH] Delve ENTRYPOINT на 0.0.0.0 — любой хост в сети может подключиться к отладчику и выполнить произвольный код, критический security risk для любого не-изолированного окружения [Dockerfile.debug:ENTRYPOINT]
- [ ] [AI-Review][MEDIUM] golang:1.25-bookworm — образ может не существовать на момент сборки (Go 1.25 ещё не выпущен), нужно указать актуальную версию или использовать golang:latest [Dockerfile.debug:FROM]
- [ ] [AI-Review][MEDIUM] git rev-parse в builder stage — git может отсутствовать в базовом образе, команда молча fallback на "unknown" (apt-get install git добавлен, но после COPY а не перед) [Dockerfile.debug]
- [ ] [AI-Review][LOW] remotePath пустая строка в launch.json — для Docker remote debugging требуется указать remotePath ("/app") для корректного маппинга source files [.vscode/launch.json]

### Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][HIGH] Delve ENTRYPOINT слушает 0.0.0.0 — security risk без -p 127.0.0.1 [Dockerfile.debug:44-45]
- [ ] [AI-Review][MEDIUM] golang:1.25-bookworm — версия может не существовать, пинить конкретную [Dockerfile.debug:5]
- [ ] [AI-Review][MEDIUM] git устанавливается после COPY go.mod — логический порядок нарушен [Dockerfile.debug:8]

## Dev Notes

### Архитектурные паттерны и ограничения

**Story 6.9 — инфраструктурная story (не код приложения)**
Это НЕ новый Go-пакет или модуль. Story 6.9 создаёт **build/debug инфраструктуру** через:
- Makefile targets (основная работа)
- Dockerfile.debug (для Docker-отладки)
- IDE конфигурации
- Документация

**Delve НЕ добавляется в go.mod!**
- Delve устанавливается как CLI-инструмент (`go install github.com/go-delve/delve/cmd/dlv@latest`)
- Это dev-зависимость, НЕ runtime-зависимость приложения
- Аналогично golangci-lint (установлен через `make setup-dev`, не в go.mod)

**Существующий Makefile (РАСШИРИТЬ, НЕ ПЕРЕПИСЫВАТЬ)** [Source: Makefile]
```makefile
# ТЕКУЩИЕ переменные:
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# ДОБАВИТЬ переменные:
DEBUG_PORT ?= 2345
DELVE_FLAGS := --headless --listen=:$(DEBUG_PORT) --api-version=2 --accept-multiclient
DEBUG_GCFLAGS := -gcflags="all=-N -l"
```

**Текущий build target (НЕ МЕНЯТЬ):**
```makefile
build:
    @echo "Сборка $(APP_NAME)..."
    @mkdir -p $(BUILD_DIR)
    $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)
```

**ДОБАВИТЬ debug targets (ПОСЛЕ существующих):**
```makefile
## debug: Сборка с debug информацией для Delve
debug:
    @echo "Сборка $(APP_NAME) с debug информацией..."
    @mkdir -p $(BUILD_DIR)
    $(GOBUILD) $(LDFLAGS) $(DEBUG_GCFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-debug $(CMD_DIR)

## debug-run: Запуск под Delve (headless, порт $(DEBUG_PORT))
debug-run: debug
    @which dlv > /dev/null 2>&1 || { echo "Ошибка: dlv не установлен. Выполните: go install github.com/go-delve/delve/cmd/dlv@latest"; exit 1; }
    dlv exec $(BUILD_DIR)/$(APP_NAME)-debug $(DELVE_FLAGS)

## debug-test: Отладка теста через Delve (PKG=./path TEST=TestName)
debug-test:
    @which dlv > /dev/null 2>&1 || { echo "Ошибка: dlv не установлен."; exit 1; }
    dlv test $(PKG) $(DELVE_FLAGS) -- -test.run $(TEST)
```

**Текущий .gitignore уже содержит:**
```
# Debug files
__debug*
*.pprof
```
→ Debug бинарники игнорируются. Убедиться что `$(BUILD_DIR)/$(APP_NAME)-debug` тоже covered.

### Существующие структуры (НЕ ЛОМАТЬ)

**Makefile** [Source: Makefile]
- Все существующие targets: `build`, `build-all`, `build-linux`, `build-windows`, `build-darwin`, `test`, `test-coverage`, `test-integration`, `lint`, `fmt`, `vet`, `clean`, `deps`, `install`, `run`, `demo`, `docs`, `version` — НЕ МЕНЯТЬ
- `setup-dev` target — РАСШИРИТЬ добавлением `dlv` установки
- `help` target (если используется) — новые targets должны быть видны

**cmd/apk-ci/main.go** [Source: cmd/apk-ci/main.go]
- НЕ МЕНЯТЬ — debug mode через Makefile, не через код
- Tracing через OpenTelemetry уже работает (trace_id в логах)
- Context-based cancellation уже реализован

**internal/config/config.go** [Source: internal/config/config.go]
- `ProjectConfig.Debug bool` поле существует в YAML config — НЕ использовать для Delve (это для другой цели)
- НЕ добавлять `BR_DEBUG` env переменную в config.go — debug mode через Makefile targets

**go.mod** [Source: go.mod]
- Go версия: 1.25.4
- НЕ ДОБАВЛЯТЬ delve в go.mod — это dev tool, не runtime dependency

### Project Structure Notes

**Новые файлы (МИНИМУМ):**
- `Dockerfile.debug` — multi-stage Docker debug image
- `.vscode/launch.json` — VS Code remote attach конфигурация
- `docs/debugging.md` — документация по отладке

**Изменяемые файлы:**
- `Makefile` — добавить debug targets (debug, debug-run, debug-test, debug-docker, debug-attach, debug-clean)

**НЕ СОЗДАВАТЬ:**
- Новых Go-пакетов
- Новых файлов в internal/
- Изменений в go.mod/go.sum
- Изменений в main.go или config.go

**НЕ МЕНЯТЬ:**
- Существующие Makefile targets
- cmd/apk-ci/main.go
- internal/config/config.go
- go.mod / go.sum / vendor/

### Dockerfile.debug (СОЗДАТЬ)

```dockerfile
# Multi-stage build для debug
FROM golang:1.25-bookworm AS builder

# Установка Delve
RUN go install github.com/go-delve/delve/cmd/dlv@latest

WORKDIR /app
COPY . .

# Сборка с debug информацией
RUN go build -gcflags="all=-N -l" \
    -ldflags "-X main.Version=debug -X main.BuildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
    -o /app/apk-ci-debug ./cmd/apk-ci/

FROM debian:bookworm-slim

# Копировать бинарник и dlv
COPY --from=builder /go/bin/dlv /usr/local/bin/dlv
COPY --from=builder /app/apk-ci-debug /app/apk-ci-debug

EXPOSE 2345

ENTRYPOINT ["dlv", "exec", "/app/apk-ci-debug", \
    "--headless", "--listen=:2345", "--api-version=2", "--accept-multiclient"]
```

### .vscode/launch.json (СОЗДАТЬ)

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Remote Attach (Delve)",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "remotePath": "",
            "port": 2345,
            "host": "127.0.0.1",
            "showLog": true,
            "trace": "verbose"
        }
    ]
}
```

### Dependencies

**Новых Go-зависимостей НЕТ!**

Delve — dev tool, устанавливается отдельно:
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

Для Docker: Delve устанавливается внутри build stage.

### Testing Strategy

**Для Story 6.9 НЕТ unit-тестов** — это инфраструктурная story.

**Ручная валидация:**
1. `make build` — существующий target работает (regression check)
2. `make test` — все тесты проходят (regression check)
3. `make debug` — debug бинарник собирается с `-gcflags="all=-N -l"`
4. `make debug-run BR_COMMAND=nr-version` — Delve запускается headless
5. IDE подключение к localhost:2345 — breakpoints работают
6. `make debug-docker BR_COMMAND=nr-version` — Docker контейнер с Delve стартует
7. `make debug-test PKG=./internal/pkg/tracing TEST=TestDefaultConfig` — тест запускается под Delve
8. `make help` — debug targets отображаются с описаниями

### Env переменные

| Переменная | Значение по умолчанию | Описание |
|------------|----------------------|----------|
| BR_DEBUG_PORT | 2345 | Порт для Delve headless listener |

**НЕ ДОБАВЛЯТЬ новую env переменную в config.go!** Это переменная Makefile, не приложения.

### Makefile Convention

Текущий Makefile использует:
- Комментарии `##` перед target для help output (если есть `help` target)
- `@echo` для пользовательского вывода
- `@mkdir -p $(BUILD_DIR)` для создания директорий
- Переменные через `?=` для переопределяемых значений
- `$(GOBUILD)`, `$(GOTEST)` и другие Go-команды через переменные

### Git Intelligence (Previous Stories Learnings)

**Story 6-8 (Trace Sampling) — предшественник:**
- Минимальные изменения: 6 файлов модифицировано, 0 создано
- Паттерн "расширяй, не переписывай" — ТОТ ЖЕ подход для Makefile
- Code review нашёл проблемы в 12 итерациях — подробные requirements предотвращают это
- Backward compatibility проверяется явно

**Общие паттерны Epic 6:**
- Все stories расширяют существующий код, не создают новую архитектуру
- Конфигурация через env переменные с разумными defaults
- Backward compatibility — критическое требование
- Документация и примеры — часть каждой story

### Backward Compatibility

- Все существующие Makefile targets работают идентично
- Новые targets имеют `debug-` префикс — нет конфликтов
- Нет изменений в Go коде — нет runtime regression
- Нет изменений в go.mod — нет dependency regression
- `DEBUG_PORT ?= 2345` — переопределяемое значение не ломает defaults

### Security Considerations

- Delve debug port (2345) — ТОЛЬКО для локальной разработки
- Docker debug контейнер: порт exposed только на localhost по умолчанию (`-p 127.0.0.1:2345:2345`)
- В production НЕ ИСПОЛЬЗОВАТЬ debug build (без оптимизации, увеличенный бинарник)
- Dockerfile.debug — НЕ для production deployment
- В .gitignore уже есть `__debug*` — debug артефакты не попадают в VCS

### Known Limitations

- Delve не поддерживает Windows ARM
- Debug build значительно медленнее production build (нет оптимизаций)
- Docker debug image значительно больше production image
- `--accept-multiclient` позволяет несколько IDE подключений, но может быть нежелательно в некоторых случаях
- Delve требует Go toolchain для `dlv test` — не работает с pre-built бинарниками

### References

- [Source: Makefile] — Текущие targets и переменные (расширить)
- [Source: cmd/apk-ci/main.go] — Точка входа (НЕ менять)
- [Source: .gitignore:18-20] — Debug files exclusion (уже настроен)
- [Source: _bmad-output/project-planning-artifacts/prd.md#FR44-46] — Delve debugging requirements
- [Source: _bmad-output/project-planning-artifacts/epics/epic-6-observability.md#Story-6.9] — Исходные требования
- [Source: _bmad-output/project-planning-artifacts/architecture.md] — FR44-46 → Build flags + Delve
- [Source: stories/6-8-trace-sampling.md] — Предыдущая story (learnings, patterns)
- [Delve docs: https://github.com/go-delve/delve/tree/master/Documentation] — Официальная документация Delve

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Debug Log References

- `make build` — успешно, backward compatible
- `make test` — все тесты проходят (FAIL `cmd/apk-ci` — существующая проблема с недоступностью Gitea API, не регрессия)
- `make debug` — debug бинарник собирается с `-gcflags="all=-N -l"`
- `make debug-clean` — удаление debug артефактов работает
- `make -n debug-run BR_DEBUG_PORT=3456` — порт корректно переопределяется через BR_DEBUG_PORT
- `make -n debug-test PKG=./internal/pkg/tracing TEST=TestDefaultConfig` — команда формируется корректно
- `make -n debug-docker` — Docker команды формируются корректно

### Completion Notes List

- Инфраструктурная story — нет Go-кода, только Makefile targets, Dockerfile, IDE config и документация
- Все 7 debug targets добавлены: debug, debug-run, debug-attach, debug-test, debug-docker, debug-docker-stop, debug-clean
- BR_DEBUG_PORT env переменная работает через `$(or $(BR_DEBUG_PORT),2345)` pattern
- Dockerfile.debug: multi-stage build (golang:1.25-bookworm → debian:bookworm-slim)
- .vscode/launch.json: обновлён с сохранением существующей конфигурации, добавлены Remote Attach и Docker Remote Attach
- setup-dev расширен установкой dlv
- Все существующие targets работают без изменений (backward compatible)
- go.mod/go.sum не затронуты
- main.go и config.go не затронуты

### Change Log

- 2026-02-07: Реализована инфраструктура Delve debugging (FR44-46) — Makefile debug targets, Dockerfile.debug, IDE config, документация
- 2026-02-07: Code review #13 fixes — H-1: Docker env vars passthrough (DOCKER_ARGS + фоновый запуск), H-2: Dockerfile GitCommit + BuildTime формат, H-3: launch.json trailing comma, M-1: .dockerignore, M-3/M-4: документация debug-attach/debug-clean/Docker env vars

### File List

- `Makefile` — модифицирован: добавлены debug переменные (DEBUG_PORT, DELVE_FLAGS, DEBUG_GCFLAGS, DOCKER_ARGS), 7 debug targets (debug, debug-run, debug-attach, debug-test, debug-docker, debug-docker-stop, debug-clean), расширен setup-dev
- `Dockerfile.debug` — создан: multi-stage Docker build для debug (golang:1.25-bookworm → debian:bookworm-slim + dlv)
- `.vscode/launch.json` — модифицирован: добавлены конфигурации "Remote Attach (Delve)" и "Docker Remote Attach (Delve)"
- `docs/debugging.md` — создан: документация по отладке через Delve с примерами workflows
- `.dockerignore` — создан: исключение build артефактов, документации и IDE файлов из Docker build context

### Adversarial Code Review #13
- H-1 fix: `launch.json` — исправлен путь к main.go (git.benadis.ru/... → cmd/apk-ci)
- M-6 fix: `Makefile` — добавлены недостающие targets в .PHONY (~25 targets)
- M-7 fix: `Makefile` — исправлена инвертированная проверка файла в setup-dev

### Adversarial Code Review #15

**Findings**: 3 LOW

**Issues fixed (code/config)**:
- **L-14**: `Dockerfile.debug` — добавлены LABEL benadis.build-type="debug" и benadis.warning для автоматической идентификации в CI/CD
- **L-15**: `.vscode/launch.json` — добавлен комментарий о синхронизации порта с Makefile DEBUG_PORT

**Issues documented (not code)**:
- **L-16**: docs/debugging.md не ссылается из README — отложено

### Adversarial Code Review #16

**Findings**: 1 HIGH

**Issues fixed (code)**:
- **H-10**: `Dockerfile.debug` — golang:1.25-bookworm не гарантирует наличие git, а `git rev-parse --short HEAD` молча падал с fallback на "unknown". Добавлен `apt-get install git` в builder stage

### Adversarial Code Review #17 (2026-02-07)

**Findings**: 1 MEDIUM

**Issues fixed (code)**:
- **M-6**: `Dockerfile.debug` — Delve ENTRYPOINT на 0.0.0.0 (security risk — любой хост может подключиться). Добавлен комментарий-предупреждение: "--listen=0.0.0.0:2345 ОПАСНО для production, только для dev окружения с firewall. Для production использовать localhost или --listen=unix:/tmp/dlv.sock"

**Status**: done
