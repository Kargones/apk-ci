# Rollback NR-команд на legacy-версию

## Обзор

Данный runbook описывает процедуру отката (rollback) с NR-команд на предыдущую стабильную версию apk-ci при обнаружении проблем в production.

### Что такое NR-команды

NR (New Registry) — обновлённая архитектура команд apk-ci с самостоятельной регистрацией, структурированным выводом (JSON/текст), трассировкой и логированием. Все NR-команды имеют префикс `nr-` (например, `nr-service-mode-status`).

### Предварительные условия

- Доступ к Gitea Actions workflow файлам проекта
- Знание текущей версии apk-ci и предыдущей стабильной версии
- Доступ к переменным окружения CI/CD пайплайна

### Важно

**DeprecatedBridge НЕ является механизмом rollback.** Вызов deprecated-алиаса (например, `service-mode-status` вместо `nr-service-mode-status`) выполняет тот же NR handler с предупреждением в stderr. Для настоящего rollback необходим откат на предыдущую версию бинарника.

---

## Процедура rollback

### Шаг 1: Определить предыдущую стабильную версию

```bash
# Посмотреть доступные релизы
gh release list -R <owner>/<repo>

# Или через Gitea API
curl -s https://git.benadis.ru/api/v1/repos/<owner>/<repo>/releases | jq '.[].tag_name'
```

### Шаг 2: Откатить версию apk-ci в CI/CD

В файле Gitea Actions workflow (`.gitea/workflows/*.yaml`) измените версию apk-ci:

> **Примечание:** Версии `v2.0.0` и `v1.9.0` в примерах ниже — условные. Замените на актуальные версии из вашего реестра релизов.

**До (текущая версия с проблемой):**
```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Выполнить команду
        uses: actions/apk-ci@v2.0.0  # замените на актуальную версию
        env:
          BR_COMMAND: nr-service-mode-status
          BR_INFOBASE_NAME: ${{ vars.INFOBASE_NAME }}
```

**После (откат на предыдущую стабильную версию):**
```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Выполнить команду
        uses: actions/apk-ci@v1.9.0  # замените на предыдущую стабильную версию
        env:
          BR_COMMAND: nr-service-mode-status
          BR_INFOBASE_NAME: ${{ vars.INFOBASE_NAME }}
```

### Шаг 3: Замена BR_COMMAND (если требуется)

При откате на версию, которая ещё не поддерживала NR-команды, замените значение `BR_COMMAND`:

| NR-команда (текущая) | Legacy-команда (для rollback) |
|---|---|
| `nr-service-mode-status` | `service-mode-status` |
| `nr-service-mode-enable` | `service-mode-enable` |
| `nr-service-mode-disable` | `service-mode-disable` |
| `nr-dbrestore` | `dbrestore` |
| `nr-dbupdate` | `dbupdate` |
| `nr-create-temp-db` | `create-temp-db` |
| `nr-store2db` | `store2db` |
| `nr-storebind` | `storebind` |
| `nr-create-stores` | `create-stores` |
| `nr-convert` | `convert` |
| `nr-git2store` | `git2store` |
| `nr-execute-epf` | `execute-epf` |
| `nr-sq-scan-branch` | `sq-scan-branch` |
| `nr-sq-scan-pr` | `sq-scan-pr` |
| `nr-sq-report-branch` | `sq-report-branch` |
| `nr-sq-project-update` | `sq-project-update` |
| `nr-test-merge` | `test-merge` |
| `nr-action-menu-build` | `action-menu-build` |

**Команды без legacy-аналога (rollback недоступен):**

| NR-команда | Статус rollback |
|---|---|
| `nr-version` | Rollback недоступен — команда добавлена только в NR-архитектуре |
| `help` | Rollback недоступен — команда не имеет `nr-` префикса, зарегистрирована как `help` |
| `nr-force-disconnect-sessions` | Rollback недоступен — команда добавлена только в NR-архитектуре |
| `nr-migrate` | Rollback недоступен — утилита миграции, добавлена только в NR-архитектуре |
| `nr-deprecated-audit` | Rollback недоступен — утилита аудита, добавлена только в NR-архитектуре |

### Шаг 4: Закоммитить и задеплоить изменения

```bash
git add .gitea/workflows/
git commit -m "rollback: откат apk-ci на v1.9.0"
git push
```

---

## Верификация rollback

После выполнения rollback необходимо убедиться, что legacy-команда работает корректно.

> **Примечание:** Примеры ниже используют `./apk-ci` для локальной проверки. В CI/CD путь к бинарнику определяется Gitea Actions автоматически.

### 1. Проверка exit code

```bash
# Запуск команды и проверка exit code
BR_COMMAND=service-mode-status BR_INFOBASE_NAME=MyInfobase ./apk-ci
echo "Exit code: $?"
# Ожидание: 0 (успех)
```

### 2. Проверка warning deprecated (если используется текущая версия с deprecated-алиасом)

```bash
# Если используется текущая версия и deprecated-алиас:
BR_COMMAND=service-mode-status BR_INFOBASE_NAME=MyInfobase ./apk-ci 2>&1 | grep -i deprecated
# Ожидание: "WARNING: command 'service-mode-status' is deprecated, use 'nr-service-mode-status' instead"
# Наличие этого warning подтверждает что deprecated bridge работает.
```

### 3. Проверка вывода

```bash
# Текстовый вывод (по умолчанию)
BR_COMMAND=service-mode-status BR_INFOBASE_NAME=MyInfobase ./apk-ci

# JSON вывод
BR_COMMAND=service-mode-status BR_INFOBASE_NAME=MyInfobase BR_OUTPUT_FORMAT=json ./apk-ci
```

### 4. Запуск smoke-тестов

```bash
# Smoke-тесты проверяют базовую работоспособность системы
BR_COMMAND=nr-version ./apk-ci
# Ожидание: информация о версии без ошибок
```

### 5. Запуск shadow-run для сравнения

Если доступен shadow-run, используйте его для сравнения NR и legacy выводов:

```bash
BR_SHADOW_RUN=true BR_COMMAND=nr-service-mode-status BR_INFOBASE_NAME=MyInfobase ./apk-ci
# Shadow-run выполнит обе версии и покажет различия
```

---

## Возврат на NR

После устранения проблемы, вызвавшей rollback, выполните обратную миграцию.

### Шаг 1: Убедиться что проблема исправлена

- Проверьте changelog новой версии
- Убедитесь что issue/баг-трекер отмечает проблему как исправленную

### Шаг 2: Обновить версию apk-ci

```yaml
# Обновить на исправленную версию
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Выполнить команду
        uses: actions/apk-ci@v2.0.1  # исправленная версия
        env:
          BR_COMMAND: nr-service-mode-status
          BR_INFOBASE_NAME: ${{ vars.INFOBASE_NAME }}
```

### Шаг 3: Вернуть BR_COMMAND на NR-версию (если менялся)

Замените legacy-имена обратно на NR:
```yaml
env:
  BR_COMMAND: nr-service-mode-status  # вместо service-mode-status
```

### Шаг 4: Валидация с shadow-run

Перед полным переключением рекомендуется запустить shadow-run для проверки идентичности результатов:

```bash
BR_SHADOW_RUN=true BR_COMMAND=nr-service-mode-status BR_INFOBASE_NAME=MyInfobase ./apk-ci
```

### Шаг 5: Валидация с plan-only

Используйте plan-only режим для проверки плана операций без реального выполнения:

```bash
BR_PLAN_ONLY=true BR_COMMAND=nr-dbrestore BR_INFOBASE_NAME=MyInfobase ./apk-ci
```

---

## Gitea Actions примеры

### Пример 1: Workflow с NR-командой (до rollback)

```yaml
name: Обновление конфигурации
on:
  push:
    branches: [main]

jobs:
  update-config:
    runs-on: self-hosted
    steps:
      - uses: actions/checkout@v4

      - name: Включить сервисный режим
        uses: actions/apk-ci@v2.0.0
        env:
          BR_COMMAND: nr-service-mode-enable
          BR_INFOBASE_NAME: ${{ vars.INFOBASE_NAME }}
          BR_OUTPUT_FORMAT: json

      - name: Восстановить БД
        uses: actions/apk-ci@v2.0.0
        env:
          BR_COMMAND: nr-dbrestore
          BR_INFOBASE_NAME: ${{ vars.INFOBASE_NAME }}

      - name: Обновить БД
        uses: actions/apk-ci@v2.0.0
        env:
          BR_COMMAND: nr-dbupdate
          BR_INFOBASE_NAME: ${{ vars.INFOBASE_NAME }}

      - name: Отключить сервисный режим
        uses: actions/apk-ci@v2.0.0
        env:
          BR_COMMAND: nr-service-mode-disable
          BR_INFOBASE_NAME: ${{ vars.INFOBASE_NAME }}
```

### Пример 2: Тот же workflow после rollback

```yaml
name: Обновление конфигурации
on:
  push:
    branches: [main]

jobs:
  update-config:
    runs-on: self-hosted
    steps:
      - uses: actions/checkout@v4

      - name: Включить сервисный режим
        uses: actions/apk-ci@v1.9.0  # откат версии
        env:
          BR_COMMAND: service-mode-enable     # legacy-команда
          BR_INFOBASE_NAME: ${{ vars.INFOBASE_NAME }}
          # BR_OUTPUT_FORMAT убран — legacy не поддерживает JSON output

      - name: Восстановить БД
        uses: actions/apk-ci@v1.9.0
        env:
          BR_COMMAND: dbrestore               # legacy-команда
          BR_INFOBASE_NAME: ${{ vars.INFOBASE_NAME }}

      - name: Обновить БД
        uses: actions/apk-ci@v1.9.0
        env:
          BR_COMMAND: dbupdate                # legacy-команда
          BR_INFOBASE_NAME: ${{ vars.INFOBASE_NAME }}

      - name: Отключить сервисный режим
        uses: actions/apk-ci@v1.9.0
        env:
          BR_COMMAND: service-mode-disable    # legacy-команда
          BR_INFOBASE_NAME: ${{ vars.INFOBASE_NAME }}
```

### Пример 3: Workflow с проверкой версии

```yaml
name: Проверка версии
on:
  workflow_dispatch:

jobs:
  check-version:
    runs-on: self-hosted
    steps:
      - name: Проверка версии и rollback-маппинга
        uses: actions/apk-ci@v2.0.0
        env:
          BR_COMMAND: nr-version
          BR_OUTPUT_FORMAT: json
      # Вывод содержит rollback_mapping — список всех NR-команд с их legacy-алиасами

      - name: Извлечь rollback-маппинг
        run: |
          # Пример извлечения маппинга из JSON вывода
          BR_COMMAND=nr-version BR_OUTPUT_FORMAT=json ./apk-ci | jq '.data.rollback_mapping'
```

---

## Таблица NR-команд с legacy-алиасами

| # | NR-команда | Legacy-алиас | Категория | Rollback |
|---|---|---|---|---|
| 1 | `nr-service-mode-status` | `service-mode-status` | Сервисный режим | Доступен |
| 2 | `nr-service-mode-enable` | `service-mode-enable` | Сервисный режим | Доступен |
| 3 | `nr-service-mode-disable` | `service-mode-disable` | Сервисный режим | Доступен |
| 4 | `nr-dbrestore` | `dbrestore` | Операции с БД | Доступен |
| 5 | `nr-dbupdate` | `dbupdate` | Операции с БД | Доступен |
| 6 | `nr-create-temp-db` | `create-temp-db` | Операции с БД | Доступен |
| 7 | `nr-store2db` | `store2db` | Синхронизация конфигурации | Доступен |
| 8 | `nr-storebind` | `storebind` | Синхронизация конфигурации | Доступен |
| 9 | `nr-create-stores` | `create-stores` | Синхронизация конфигурации | Доступен |
| 10 | `nr-convert` | `convert` | Синхронизация конфигурации | Доступен |
| 11 | `nr-git2store` | `git2store` | Синхронизация конфигурации | Доступен |
| 12 | `nr-execute-epf` | `execute-epf` | Прочие операции | Доступен |
| 13 | `nr-sq-scan-branch` | `sq-scan-branch` | SonarQube | Доступен |
| 14 | `nr-sq-scan-pr` | `sq-scan-pr` | SonarQube | Доступен |
| 15 | `nr-sq-report-branch` | `sq-report-branch` | SonarQube | Доступен |
| 16 | `nr-sq-project-update` | `sq-project-update` | SonarQube | Доступен |
| 17 | `nr-test-merge` | `test-merge` | Gitea | Доступен |
| 18 | `nr-action-menu-build` | `action-menu-build` | Gitea | Доступен |
| 19 | `nr-version` | — | Системная | Недоступен |
| 20 | `help` | — | Системная | Недоступен |
| 21 | `nr-force-disconnect-sessions` | — | Сервисный режим | Недоступен |
| 22 | `nr-migrate` | — | Утилиты | Недоступен |
| 23 | `nr-deprecated-audit` | — | Утилиты | Недоступен |

> **Примечание:** Команда `nr-version` с опцией `BR_OUTPUT_FORMAT=json` выводит полный rollback-маппинг в поле `rollback_mapping`.
