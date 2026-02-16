# Анализ проблемы: Ошибка загрузки конфигурации из файлов в интеграционном тесте

## Описание проблемы

При запуске интеграционного теста `TestMain_WithRealYamlFile` возникает ошибка при попытке сборки конфигурации из исходных файлов. В логах присутствует сообщение:

```
Файл не обнаружен '/tmp/4del/s1290398063/src/cfg'
```

## Анализ проблемы

### 1. Структура репозитория

После клонирования репозитория была проанализирована структура каталогов:

```
/tmp/4del/s665597546/
├── .git/
├── .gitea/
├── .gitignore
├── README.md
├── project.yaml
├── TOIR3/                    # Основная конфигурация
│   ├── .project
│   ├── .settings/
│   ├── DT-INF/
│   └── src/                  # Исходные файлы конфигурации
│       ├── AccumulationRegisters/
│       ├── BusinessProcesses/
│       ├── Catalogs/
│       ├── ... (другие объекты)
│       └── Configuration/
└── TOIR3.апк_ДоработкиТОИР/   # Расширение
    ├── .project
    ├── .settings/
    ├── DT-INF/
    └── src/                  # Исходные файлы расширения
```

### 2. Несоответствие структуры ожиданиям кода

Код ожидает следующую структуру:
- Основная конфигурация: `src/cfg`
- Расширения: `src/cfe/<имя_расширения>`

Фактическая структура репозитория:
- Основная конфигурация: `TOIR3/src/`
- Расширения: `TOIR3.апк_ДоработкиТОИР/src/`

### 3. Анализ кода

В файле [`internal/entity/one/convert/convert.go`](internal/entity/one/convert/convert.go) в функции [`LoadFromConfig()`](internal/entity/one/convert/convert.go:67) формируются пути к исходным файлам:

```go
// Основная конфигурация
mainPair := Pair{
    Source: Source{
        Name:    cfg.ProjectName,
        RelPath: "src/cfg",  // Ожидаемый путь
        Main:    true,
    },
    // ...
}

// Расширения
for _, addName := range cfg.AddArray {
    addPair := Pair{
        Source: Source{
            Name:    addName,
            RelPath: "src/cfe/" + addName,  // Ожидаемый путь
            Main:    false,
        },
        // ...
    }
}
```

В функции [`LoadDb()`](internal/entity/one/convert/convert.go:385) происходит загрузка конфигурации:

```go
err = cc.OneDB.Load(*ctx, l, cfg, path.Join(cfg.RepPath, cp.Source.RelPath))
```

### 4. Причина ошибки

Ошибка возникает из-за несоответствия между фактической структурой репозитория и ожидаемой структурой в коде. Код ищет файлы в `src/cfg`, но они находятся в `TOIR3/src/`.

## Решения проблемы

### Решение 1: Адаптация кода к фактической структуре репозитория

Изменить логику формирования путей в функции [`LoadFromConfig()`](internal/entity/one/convert/convert.go:67):

```go
// Основная конфигурация
mainPair := Pair{
    Source: Source{
        Name:    cfg.ProjectName,
        RelPath: cfg.ProjectName + "/src",  // Измененный путь
        Main:    true,
    },
    // ...
}

// Расширения
for _, addName := range cfg.AddArray {
    addPair := Pair{
        Source: Source{
            Name:    addName,
            RelPath: cfg.ProjectName + "." + addName + "/src",  // Измененный путь
            Main:    false,
        },
        // ...
    }
}
```

### Решение 2: Создание символических ссылок

Добавить в процесс клонирования репозитория создание символических ссылок для совместимости:

```go
// После клонирования репозитория
os.Symlink(cfg.RepPath+"/"+cfg.ProjectName+"/src", cfg.RepPath+"/src/cfg")
for _, addName := range cfg.AddArray {
    os.MkdirAll(cfg.RepPath+"/src/cfe", 0750)
    os.Symlink(
        cfg.RepPath+"/"+cfg.ProjectName+"."+addName+"/src", 
        cfg.RepPath+"/src/cfe/"+addName
    )
}
```

### Решение 3: Конфигурируемые пути

Добавить в конфигурацию возможность указания путей к исходным файлам:

```yaml
# в project.yaml
project:
  name: "TOIR3"
  source_paths:
    main: "TOIR3/src"
    extensions:
      - name: "апк_ДоработкиТОИР"
        path: "TOIR3.апк_ДоработкиТОИР/src"
```

### Решение 4: Автоопределение структуры

Добавить логику автоопределения структуры репозитория:

```go
func detectProjectStructure(repPath string, projectName string) (string, []ExtensionPath, error) {
    // Проверка различных вариантов структуры
    mainPath := path.Join(repPath, projectName, "src")
    if _, err := os.Stat(mainPath); os.IsNotExist(err) {
        // Альтернативный вариант
        mainPath = path.Join(repPath, "src", "cfg")
        if _, err := os.Stat(mainPath); os.IsNotExist(err) {
            return "", nil, fmt.Errorf("не найдена структура проекта")
        }
    }
    
    // Поиск расширений
    extensions := []ExtensionPath{}
    // ...
    
    return mainPath, extensions, nil
}
```

## Рекомендуемое решение

Рекомендуется использовать **Решение 1** как наиболее простое и надежное, поскольку:
1. Не требует дополнительных операций с файловой системой
2. Легко поддерживать и отлаживать
3. Соответствует фактической структуре репозитория

## Дополнительные проблемы

При анализе была обнаружена дополнительная проблема в функции [`CreateTempDb()`](internal/entity/one/designer/designer.go:424):

```go
if !strings.Contains(string(r.ConsoleOut), constants.SearchMsgBaseCreateOk) {
    logger.Error("Неопознанная ошибка создания временной базы данных",
        slog.String("output", string(r.ConsoleOut)))
    return odb, fmt.Errorf("неопознанная ошибка создания временной базы данных: %s", string(r.ConsoleOut))
}
```

Несмотря на успешное создание базы данных (в выводе присутствует "Infobase created"), проверка на успешность завершается ошибкой, так как ищет другую строку. Необходимо исправить константу `SearchMsgBaseCreateOk` или добавить проверку на "Infobase created".

## Итог

Основная проблема заключается в несоответствии структуры репозитория ожиданиям кода. Рекомендуется адаптировать код к фактической структуре репозитория, изменив логику формирования путей в функции [`LoadFromConfig()`](internal/entity/one/convert/convert.go:67).