# Отчет об анализе проблемы с загрузкой конфигурации в тесте TestMain_WithRealYamlFile

**Дата:** 2025-10-14
**Версия:** 5.10.13.1:0ebc454-debug
**Тест:** TestMain_WithRealYamlFile
**Команда:** git2store

---

## 1. Описание проблемы

При запуске интеграционного теста `TestMain_WithRealYamlFile` возникает ошибка:

```
Файл не обнаружен '/tmp/4del/s405964516/src/cfg'
```

**Контекст ошибки:**
- Команда: `git2store`
- Репозиторий: `test/TOIR3`
- Операция: Загрузка конфигурации из файлов в базу данных 1С
- Файл: `internal/entity/one/designer/designer.go:162`

---

## 2. Пошаговый анализ выполнения

### 2.1. Успешные шаги:
1. ✅ Загрузка конфигурации из Gitea API
2. ✅ Клонирование репозитория в `/tmp/4del/s405964516/`
3. ✅ Переключение на ветку `main` (формат EDT)
4. ✅ Создание временной базы данных `/tmp/4del/temp/temp_db_20251014_163928`
5. ✅ Добавление расширения `апк_ДоработкиТОИР` в базу данных

### 2.2. Проблемный шаг:
6. ❌ Переключение на ветку `xml` - создана НОВАЯ пустая ветка от `main`
7. ❌ Попытка загрузки конфигурации из `/tmp/4del/s405964516/src/cfg` - файл не найден

---

## 3. Корневая причина

### 3.1. Структура репозитория

Репозиторий имеет две ветки с разной структурой:

**Ветка `main` (формат EDT):**
```
/tmp/4del/s405964516/
├── .git/
├── .gitea/
├── .gitignore
├── project.yaml
├── README.md
├── TOIR3/                    ← Основная конфигурация в формате EDT
└── TOIR3.апк_ДоработкиТОИР/  ← Расширение в формате EDT
```

**Удаленная ветка `origin/xml` (формат XML):**
```
/tmp/4del/s405964516/ (на ветке origin/xml)
├── .git/
├── .gitea/
├── .gitignore
└── src/
    ├── cfg/                   ← Основная конфигурация в формате XML
    └── cfe/
        └── апк_ДоработкиТОИР/ ← Расширение в формате XML
```

### 3.2. Проблемный код

**Файл:** `internal/git/git.go:495-607`
**Функция:** `SwitchOrCreateBranch`

**Строка 568:** Проверка существования только ЛОКАЛЬНОЙ ветки:
```go
cmdCheck := exec.CommandContext(ctx, GitCommand, "rev-parse", "--verify", branchName)
if err := cmdCheck.Run(); err == nil {
    // Ветка существует, переключаемся на неё
    ...
}
```

**Строка 584:** Создание новой ветки от ТЕКУЩЕЙ ветки:
```go
if err := executeGitCommandWithRetry(repoPath, []string{"checkout", "-b", branchName}); err != nil {
    // Создаем новую ветку от текущей (main)
    ...
}
```

**Проблема:** Функция `SwitchOrCreateBranch` НЕ проверяет существование удаленной ветки `origin/xml`. Когда локальная ветка `xml` не найдена, создается новая ветка от текущей (`main`), наследуя структуру EDT вместо XML.

### 3.3. Workflow команды Git2Store

**Файл:** `internal/app/app.go:420-534`
**Функция:** `Git2Store`

```
1. Клонирование репозитория (строка 441)
   → Текущая ветка: main (формат EDT)

2. Переключение на ветку EDT (строка 448-455)
   → g.Branch = constants.EdtBranch (= "main")

3. Создание временной базы данных (строка 457-476)
   → Успешно

4. Переключение на ветку 1C (строка 484-491)
   → g.Branch = constants.OneCBranch (= "xml")
   → Вызов g.Switch() → SwitchOrCreateBranch()
   → Локальная ветка xml не существует
   → ❌ Создается от main (НЕ от origin/xml!)
   → Структура: TOIR3/, TOIR3.апк_ДоработкиТОИР/ (EDT)
   → ОТСУТСТВУЕТ: src/cfg/ (XML)

5. Загрузка конфигурации из файлов (строка 504-506)
   → cc.LoadDb(ctx, l, cfg)
   → Попытка загрузить из /tmp/4del/s405964516/src/cfg
   → ❌ ОШИБКА: Файл не найден
```

---

## 4. Варианты решения

### 4.1. **Рекомендуемое решение:** Проверка удаленных веток

**Изменить функцию `SwitchOrCreateBranch` в `internal/git/git.go`:**

```go
// Проверяем существование локальной ветки
cmdCheck := exec.CommandContext(ctx, GitCommand, "rev-parse", "--verify", branchName)
if err := cmdCheck.Run(); err == nil {
    // Локальная ветка существует
    slog.Debug("Ветка существует, переключаемся на неё", slog.String("branch", branchName))
    if err := executeGitCommandWithRetry(repoPath, []string{"checkout", branchName}); err != nil {
        return fmt.Errorf("failed to switch to branch %s: %v", branchName, err)
    }
    slog.Info("Успешно переключились на существующую ветку", slog.String("branch", branchName))
} else {
    // Локальная ветка не существует - проверяем удаленную
    slog.Debug("Локальная ветка не существует, проверяем удаленную ветку",
        slog.String("branch", branchName))

    // Проверяем существование удаленной ветки origin/branchName
    cmdCheckRemote := exec.CommandContext(ctx, GitCommand, "rev-parse", "--verify", "origin/"+branchName)
    if err := cmdCheckRemote.Run(); err == nil {
        // Удаленная ветка существует - создаем локальную от неё
        slog.Debug("Удаленная ветка существует, создаем локальную ветку от неё",
            slog.String("branch", branchName),
            slog.String("remote", "origin/"+branchName))

        if err := executeGitCommandWithRetry(repoPath, []string{"checkout", "-b", branchName, "origin/" + branchName}); err != nil {
            slog.Error("Не удалось создать локальную ветку от удаленной",
                slog.String("branch", branchName),
                slog.String("remote", "origin/"+branchName),
                slog.String("error", err.Error()))
            return fmt.Errorf("failed to create branch %s from origin/%s: %v", branchName, branchName, err)
        }
        slog.Info("Успешно создали локальную ветку от удаленной",
            slog.String("branch", branchName),
            slog.String("remote", "origin/"+branchName))
    } else {
        // Ни локальная, ни удаленная ветка не существуют - создаем новую
        slog.Debug("Ни локальная, ни удаленная ветка не существуют, создаем новую ветку",
            slog.String("branch", branchName))

        if err := executeGitCommandWithRetry(repoPath, []string{"checkout", "-b", branchName}); err != nil {
            slog.Error("Не удалось создать новую ветку",
                slog.String("branch", branchName),
                slog.String("error", err.Error()))
            return fmt.Errorf("failed to create and switch to branch %s: %v", branchName, err)
        }
        slog.Info("Успешно создали новую ветку", slog.String("branch", branchName))
    }
}
```

**Преимущества:**
- ✅ Минимальные изменения в коде (только одна функция)
- ✅ Обратная совместимость сохранена
- ✅ Работает для всех случаев использования
- ✅ Правильно обрабатывает существующие удаленные ветки

**Недостатки:**
- Нет

---

### 4.2. Альтернативное решение: Предварительный fetch всех веток

**Добавить после клонирования репозитория в `Git2Store`:**

```go
// После строки 447 в app.go
err = g.Clone(ctx, l)
if err != nil {
    return err
}

// Явный fetch всех веток для обеспечения их доступности
cmdFetch := exec.Command("git", "-C", g.RepPath, "fetch", "--all")
if err := cmdFetch.Run(); err != nil {
    l.Warn("Не удалось выполнить fetch всех веток",
        slog.String("error", err.Error()))
}
```

**Преимущества:**
- Гарантирует наличие всех удаленных веток

**Недостатки:**
- ❌ Не решает корневую проблему в `SwitchOrCreateBranch`
- ❌ Дополнительный сетевой запрос
- ❌ Проблема останется в других местах использования `Switch`

---

### 4.3. Альтернативное решение: Явное указание source для новой ветки

**Изменить вызов Switch в Git2Store:**

```go
// Вместо строки 484-491 в app.go
g.Branch = constants.OneCBranch
g.TrackingBranch = "origin/" + constants.OneCBranch  // Новое поле
if err := g.Switch(*ctx, l); err != nil {
    return err
}
```

**Преимущества:**
- Явный контроль source ветки

**Недостатки:**
- ❌ Требует изменения структуры Git и всех вызовов
- ❌ Больше изменений в коде
- ❌ Усложняет API

---

## 5. Рекомендация

**Рекомендуется реализовать решение 4.1** - модификацию функции `SwitchOrCreateBranch` с проверкой удаленных веток.

**Причины:**
1. Минимальные изменения кода
2. Решает проблему в корне
3. Обратная совместимость
4. Универсальное решение для всех случаев использования
5. Соответствует стандартному поведению Git

---

## 6. План внедрения

1. **Изменить функцию `SwitchOrCreateBranch`** в `internal/git/git.go:495-607`
   - Добавить проверку удаленной ветки `origin/branchName`
   - Создавать локальную ветку от удаленной, если она существует

2. **Добавить тесты**:
   - Тест проверки создания ветки от удаленной
   - Тест создания новой ветки, когда удаленной нет
   - Тест переключения на существующую локальную ветку

3. **Запустить регрессионные тесты**:
   - `TestMain_WithRealYamlFile`
   - Все интеграционные тесты с git операциями

4. **Обновить документацию**:
   - Добавить комментарий о приоритете удаленных веток

---

## 7. Затронутые файлы

**Основные:**
- `internal/git/git.go` - функция `SwitchOrCreateBranch` (строки 495-607)

**Связанные (для понимания контекста):**
- `internal/app/app.go` - функция `Git2Store` (строки 420-534)
- `internal/entity/one/convert/convert.go` - пути к конфигурации (строки 102, 302)
- `cmd/apk-ci/yaml_integration_test.go` - тест (строки 10-96)

---

## 8. Проверка решения

После внедрения проверить:

```bash
# 1. Очистить временные каталоги
rm -rf /tmp/4del/*

# 2. Запустить тест
go test -v -timeout 10m ./cmd/apk-ci -run TestMain_WithRealYamlFile

# 3. Проверить, что ветка xml создана от origin/xml
cd /tmp/4del/s*/
git log --oneline -5
# Должны увидеть: "Конвертирован автоматически" - коммит из origin/xml

# 4. Проверить наличие структуры src/cfg
ls -la src/
# Должны увидеть: cfg/, cfe/
```

---

## 9. Дополнительные наблюдения

### 9.1. Несоответствие в main-test.yaml

В файле `/root/r/apk-ci/main-test.yaml`:
- Строка 11: `INPUT_COMMAND: "git2store"`
- Строка 48: `BR_COMMAND: "convert"`

В тесте ожидается `INPUT_COMMAND: "convert"` (строка 47 теста), но реально запускается `git2store`.

**Рекомендация:** Синхронизировать параметры в main-test.yaml или обновить проверку в тесте.

### 9.2. Консистентность команд

Команды `convert` и `git2store` имеют разную логику:
- `convert`: Конвертирует EDT → XML с созданием новой ветки
- `git2store`: Ожидает существующую ветку XML для загрузки в БД

Эта логика должна быть явно описана в документации.

---

## 10. Выводы

1. **Проблема идентифицирована**: Функция `SwitchOrCreateBranch` не проверяет удаленные ветки
2. **Решение определено**: Добавить проверку `origin/branchName` перед созданием новой ветки
3. **Изменения минимальны**: Одна функция в `internal/git/git.go`
4. **Риск низкий**: Изменения обратно совместимы
5. **Тестирование**: Запустить `TestMain_WithRealYamlFile` после внедрения

---

**Конец отчета**
