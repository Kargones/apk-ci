# Архитектурное описание решения: ActionMenuBuild

## 1. Общая архитектура

### 1.1. Модульная структура
**Реализует требования:** 1.1, 1.2.1, 1.2.2, 1.2.3, 10.1.1, 10.2.3  
**Инструкция:** При выполнении данного пункта загрузите в контекст требования 1.1, 1.2.1, 1.2.2, 1.2.3, 10.1.1, 10.2.3

Функция ActionMenuBuild будет интегрирована в модуль app как основная точка входа. Архитектура предусматривает разделение на следующие компоненты:
- Основная функция ActionMenuBuild (публичная)
- Набор приватных функций по ответственности
- Использование модуля constants для хранения шаблонов
- Интеграция с существующими модулями для работы с Gitea API

### 1.2. Принципы проектирования
**Реализует требования:** 10.1.2, 10.1.3, 10.2.1, 10.2.2  
**Инструкция:** При выполнении данного пункта загрузите в контекст требования 10.1.2, 10.1.3, 10.2.1, 10.2.2

- Единственная ответственность для каждой функции
- Минимизация внешних зависимостей
- Предпочтение стандартной библиотеки Go
- Обеспечение читаемости и тестируемости кода

## 2. Компоненты системы

### 2.1. Модуль проверки изменений
**Реализует требования:** 2.1.1, 2.1.2, 2.2.1, 2.2.2, 2.2.3  
**Инструкция:** При выполнении данного пункта загрузите в контекст требования 2.1.1, 2.1.2, 2.2.1, 2.2.2, 2.2.3

```go
func checkProjectYamlChanges(giteaClient *gitea.Client, repo string) (bool, error)
```

Ответственность:
- Определение ветки по умолчанию через Gitea API
- Получение информации о последнем коммите
- Проверка изменений файла project.yaml
- Логирование результата проверки

### 2.2. Модуль анализа конфигурации
**Реализует требования:** 3.1.1, 3.1.2, 3.1.3, 3.2.1, 3.2.2, 3.2.3  
**Инструкция:** При выполнении данного пункта загрузите в контекст требования 3.1.1, 3.1.2, 3.1.3, 3.2.1, 3.2.2, 3.2.3

```go
func analyzeProjectConfig(config *Config.ProjectConfig) (*TemplateData, error)
```

Ответственность:
- Валидация данных конфигурации
- Извлечение списков тестовых и продуктивных баз
- Подготовка данных для генерации шаблонов
- Возврат структурированных данных

### 2.3. Модуль управления файлами Actions
**Реализует требования:** 4.1.1, 4.1.2, 4.1.3, 4.2.1, 4.2.2, 7.2.1, 7.2.2, 7.2.3  
**Инструкция:** При выполнении данного пункта загрузите в контекст требования 4.1.1, 4.1.2, 4.1.3, 4.2.1, 4.2.2, 7.2.1, 7.2.2, 7.2.3

```go
func manageActionsFiles(giteaClient *gitea.Client, repo string, newFileNames []string) error
```

Ответственность:
- Получение списка существующих файлов через Gitea API
- Формирование массива CurrentActions
- Исключение файла ActionMenuBuildName
- Удаление устаревших файлов

## 3. Система генерации содержимого

### 3.1. Модуль хранения шаблонов
**Реализует требования:** 5.1.1, 5.1.2, 5.1.3, 6.1.1, 6.1.2, 6.1.3, 6.2.1, 6.2.2, 6.2.3, 6.3.1, 6.3.2, 6.3.3  
**Инструкция:** При выполнении данного пункта загрузите в контекст требования 5.1.1, 5.1.2, 5.1.3, 6.1.1, 6.1.2, 6.1.3, 6.2.1, 6.2.2, 6.2.3, 6.3.1, 6.3.2, 6.3.3

В модуле constants будут определены константы с шаблонами:

```go
const (
    ActionMenuBuildName = "action-menu-build.yaml"
    
    TemplateTestDBUpdate = `#1. Обновление тестовых баз.yaml
    // ... полный шаблон с использованием raw string literal
    `
    
    TemplateProdDBUpdate = `#2. Обновление прод баз.yaml
    // ... полный шаблон с использованием raw string literal
    `
    
    TemplateConvertStore = `#авто. Конвертация и перенос в хранилище.yaml
    // ... статический шаблон
    `
)
```

### 3.2. Модуль парсинга имен файлов
**Реализует требования:** 5.2.1, 5.2.2, 5.2.3  
**Инструкция:** При выполнении данного пункта загрузите в контекст требования 5.2.1, 5.2.2, 5.2.3

```go
func parseFileName(template string) (string, error)
```

Ответственность:
- Использование регулярного выражения для извлечения имени файла
- Парсинг первой строки шаблона после символа '#'
- Извлечение всех символов, включая цифры и точки
- Обеспечение надежности парсинга

### 3.3. Модуль генерации содержимого
**Реализует требования:** 5.3.1, 5.3.2, 5.3.3  
**Инструкция:** При выполнении данного пункта загрузите в контекст требования 5.3.1, 5.3.2, 5.3.3

```go
func generateFileContent(template string, data *TemplateData) (string, error)
```

Ответственность:
- Использование text/template из стандартной библиотеки Go
- Подстановка данных из Config.ProjectConfig
- Генерация содержимого для каждого типа файла
- Валидация результата генерации

## 4. Система создания и управления файлами

### 4.1. Модуль создания файлов
**Реализует требования:** 7.1.1, 7.1.2, 7.1.3  
**Инструкция:** При выполнении данного пункта загрузите в контекст требования 7.1.1, 7.1.2, 7.1.3

```go
func createWorkflowFiles(giteaClient *gitea.Client, repo string, files map[string]string) error
```

Ответственность:
- Создание файлов в каталоге .gitea/workflows через Gitea API
- Использование имен файлов из парсинга шаблонов
- Запись сгенерированного содержимого
- Обработка ошибок создания

### 4.2. Модуль коммита изменений
**Реализует требования:** 8.1.1, 8.1.2, 8.2.1, 8.2.2  
**Инструкция:** При выполнении данного пункта загрузите в контекст требования 8.1.1, 8.1.2, 8.2.1, 8.2.2

```go
func commitChanges(giteaClient *gitea.Client, repo string, projectYamlContent string) error
```

Ответственность:
- Формирование сообщения коммита с содержимым project.yaml
- Исключение комментариев и пустых строк
- Выполнение коммита через Gitea API
- Финализация изменений

## 5. Система обработки ошибок

### 5.1. Стратегия обработки ошибок
**Реализует требования:** 9.1.1, 9.1.2, 9.1.3, 9.2.1, 9.2.2, 9.2.3, 9.2.4  
**Инструкция:** При выполнении данного пункта загрузите в контекст требования 9.1.1, 9.1.2, 9.1.3, 9.2.1, 9.2.2, 9.2.3, 9.2.4

```go
type ActionMenuBuildError struct {
    Operation string
    Cause     error
    Context   map[string]interface{}
}

func (e *ActionMenuBuildError) Error() string {
    return fmt.Sprintf("ActionMenuBuild %s failed: %v", e.Operation, e.Cause)
}
```

Категории ошибок:
- Ошибки валидации конфигурации
- Ошибки работы с Gitea API
- Ошибки парсинга шаблонов
- Ошибки генерации содержимого

## 6. Структуры данных

### 6.1. Основные структуры
**Реализует требования:** 3.2.3, 5.3.2  
**Инструкция:** При выполнении данного пункта загрузите в контекст требования 3.2.3, 5.3.2

```go
type TemplateData struct {
    TestDatabases []string
    ProdDatabases []ProdDatabase
}

type ProdDatabase struct {
    Key    string
    DbName string
}

type FileInfo struct {
    Name     string
    Content  string
    Template string
}
```

## 7. Последовательность выполнения

### 7.1. Основной алгоритм
**Реализует требования:** Все требования в логической последовательности  
**Инструкция:** При выполнении данного пункта загрузите в контекст все функциональные требования

```go
func ActionMenuBuild() error {
    // 1. Проверка изменений project.yaml (требования 2.1, 2.2)
    changed, err := checkProjectYamlChanges(giteaClient, repo)
    if err != nil {
        return wrap(err, "checking project.yaml changes")
    }
    if !changed {
        log.Info("Файл project.yaml не изменился")
        return nil
    }
    
    // 2. Анализ конфигурации (требования 3.1, 3.2)
    templateData, err := analyzeProjectConfig(Config.ProjectConfig)
    if err != nil {
        return wrap(err, "analyzing project config")
    }
    
    // 3. Получение текущих файлов (требования 4.1)
    currentActions, err := getCurrentActions(giteaClient, repo)
    if err != nil {
        return wrap(err, "getting current actions")
    }
    
    // 4. Генерация новых файлов (требования 5, 6)
    newFiles, err := generateFiles(templateData)
    if err != nil {
        return wrap(err, "generating files")
    }
    
    // 5. Создание файлов (требования 7.1)
    err = createWorkflowFiles(giteaClient, repo, newFiles)
    if err != nil {
        return wrap(err, "creating workflow files")
    }
    
    // 6. Удаление устаревших файлов (требования 7.2)
    err = removeObsoleteFiles(giteaClient, repo, currentActions, newFiles)
    if err != nil {
        return wrap(err, "removing obsolete files")
    }
    
    // 7. Коммит изменений (требования 8)
    err = commitChanges(giteaClient, repo, projectYamlContent)
    if err != nil {
        return wrap(err, "committing changes")
    }
    
    return nil
}
```

## 8. Интеграционные точки

### 8.1. Интеграция с Gitea API
**Реализует требования:** 1.2.1, 1.2.2, 10.2.3  
**Инструкция:** При выполнении данного пункта загрузите в контекст требования 1.2.1, 1.2.2, 10.2.3

- Использование существующих модулей проекта для работы с Gitea API
- Запрет на локальное клонирование репозитория
- Все операции выполняются через API вызовы

### 8.2. Интеграция с модулем constants
**Реализует требования:** 4.2.1, 4.2.2, 5.1.1, 5.1.2, 5.1.3  
**Инструкция:** При выполнении данного пункта загрузите в контекст требования 4.2.1, 4.2.2, 5.1.1, 5.1.2, 5.1.3

- Размещение всех шаблонов и констант в модуле constants
- Использование raw string literals для шаблонов
- Централизованное управление константами

### 8.3. Интеграция с системой конфигурации
**Реализует требования:** 3.1.1, 3.1.2, 3.2.1, 3.2.2  
**Инструкция:** При выполнении данного пункта загрузите в контекст требования 3.1.1, 3.1.2, 3.2.1, 3.2.2

- Использование существующей структуры Config.ProjectConfig
- Извлечение данных для генерации шаблонов
- Валидация конфигурационных данных

## 9. Тестирование и валидация

### 9.1. Стратегия тестирования
**Реализует требования:** 10.1.2, 9.1.3  
**Инструкция:** При выполнении данного пункта загрузите в контекст требования 10.1.2, 9.1.3

- Модульные тесты для каждой приватной функции
- Интеграционные тесты для работы с Gitea API
- Тесты валидации шаблонов и генерации содержимого
- Тесты обработки ошибочных ситуаций

### 9.2. Критерии качества
**Реализует требования:** 5.2.3, 5.3.3, 7.2.3  
**Инструкция:** При выполнении данного пункта загрузите в контекст требования 5.2.3, 5.3.3, 7.2.3

- Надежность парсинга имен файлов
- Корректность генерации содержимого
- Безопасность операций удаления файлов
- Предсказуемость поведения системы

### 10. Используемые шаблоны:
Шаблоны файлов:
```yaml
#1. Обновление тестовых баз.yaml
on:
  workflow_dispatch:
    inputs:
      restore_DB:
        description: 'Восстановить базу перед загрузкой конфигурации'
        required: true
        type: boolean
        default: false 
      service_mode_enable:
        description: 'Включить сервисный режим (отключать только для загрузки конфигузации без применения)'
        required: true
        type: boolean
        default: true 
      load_cfg:
        description: 'Загрузить конфигурацию из хранилища'
        required: true
        type: boolean
        default: true 
      DbName:
        description: 'Выберите базу для загрузки конфигурации (Test)'
        required: true
        default: '<Первая база из списка в options>'
        type: choice
        options:
        <Список баз из Config.ProjectConfig.Prod.<Имя прод базы>.related. Например:
        - V8_DEV_DSBEKETOV_APK_TOIR3
        - V8_TEST_KONTUR_APK_TOIR3 >
      update_conf:
        description: 'Применить конфигурацию после загрузки'
        required: true
        type: boolean
        default: true 
jobs:               
  db-update-test:
    runs-on: edt
    steps:
      - name: Восстановление базы данных (Test)
        id: br-dbrestore
        if: ${{ inputs.restore_DB == true }}
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "dbrestore"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}
          
      - name: Включение сервисного режима (Test)
        id: br-service-mode-enable
        if: ${{ inputs.service_mode_enable == true }}
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "service-mode-enable"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}
          
      - name: Загрузка конфигурации из хранилища (Test)
        id: br-store2db
        if: ${{ inputs.load_cfg == true }}
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "store2db"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}

      - name: Применение конфигурации (Test)
        id: br-dbupdate
        if: ${{ inputs.update_conf == true }}
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "dbupdate"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}

      - name: Отключение сервисного режима (Test)
        id: br-service-mode-disable
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "service-mode-disable"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}
```

```yaml
 #2. Обновление прод баз.yaml
on:
  workflow_dispatch:
    inputs:
      service_mode_enable:
        description: 'Включить сервисный режим (отключать только для загрузки конфигузации без применения)'
        required: true
        type: boolean
        default: true 
      load_cfg:
        description: 'Загрузить конфигурацию из хранилища'
        required: true
        type: boolean
        default: true 
      DbName:
        description: 'Выберите базу для загрузки конфигурации (Prod)'
        required: true
        default: '<Первая база из списка в options>'
        type: choice
        options:
        <Список баз из Config.ProjectConfig.Prod.<Имя прод базы> вида <Config.ProjectConfig.Prod.<Имя прод базы>.dbName (Config.ProjectConfig.Prod.<Имя прод базы>)> Например:
        - База ТОИР (V8_OPER_APK_TOIR3) >
      update_conf:
        description: 'Применить конфигурацию после загрузки'
        required: true
        type: boolean
        default: true 
jobs:               
  db-update-test:
    runs-on: edt
    steps:
      - name: Включение сервисного режима (Prod)
        id: br-service-mode-enable
        if: ${{ inputs.service_mode_enable == true }}
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "service-mode-enable"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}
          
      - name: Загрузка конфигурации из хранилища (Prod)
        id: br-store2db
        if: ${{ inputs.load_cfg == true }}
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "store2db"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}

      - name: Применение конфигурации (Prod)
        id: br-dbupdate
        if: ${{ inputs.update_conf == true }}
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "dbupdate"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}

      - name: Отключение сервисного режима (Prod)
        id: br-service-mode-disable
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "service-mode-disable"
          logLevel: "Debug"
          dbName: ${{ inputs.DbName }}
          actor: ${{ gitea.actor }}
```
Шаблон расположенный ниже не изменяется
```yaml
#авто. Конвертация и перенос в хранилище.yaml
name: Конвертация и перенос в хранилище
run-name: ${{ gitea.event_name }} - ${{ gitea.workflow }} - ${{ gitea.actor }}
on:
  release:
    types: [published]
jobs:
  convert-and-store:
    runs-on: edt
    steps:
      - name: Конвертация
        id: br-convert
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ github.server_url }}
          repository: ${{ github.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "convert"
          logLevel: "Debug"
          actor: ${{ gitea.actor }}
      - name: Перенос в хранилище
        id: br-convert
        uses: https://${{ secrets.TOKEN_FULL }}:@regdv.apkholding.ru/gitops-tools/apk-ci@latest
        with:
          giteaURL: ${{ github.server_url }}
          repository: ${{ github.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          command: "git2store"
          logLevel: "Debug"
          actor: ${{ gitea.actor }}
```
