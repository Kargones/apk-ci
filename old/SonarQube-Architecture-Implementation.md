# Архитектура реализации интеграции с SonarQube

## 1. Введение

Данный документ описывает архитектуру реализации интеграции проекта benadis-runner с системой SonarQube на основе требований, изложенных в документе SRS-2-refact-04.md и диаграммах классов, компонентов и последовательностей.

## 2. Общее описание архитектуры

Интеграция с SonarQube представляет собой набор компонентов, которые обеспечивают взаимодействие между benadis-runner и SonarQube API, а также управление процессом сканирования кода с использованием sonar-scanner. Архитектура построена на принципах SOLID, с использованием интерфейсов для обеспечения гибкости и тестируемости.

## 3. Основные компоненты

### 3.1. Интерфейсы

#### 3.1.1. SonarQubeAPIInterface

Интерфейс для взаимодействия с API SonarQube:

```go
interface "SonarQubeAPIInterface" {
    +CreateProject(owner: string, repo: string, branch: string): string, error
    +GetProjectList(): []Project, error
    +GetProjectToken(projectName: string): string, error
    +GetScanList(projectName: string): []Scan, error
    +Authenticate(token: string): error
}
```

#### 3.1.2. SonarScannerInterface

Интерфейс для управления sonar-scanner:

```go
interface "SonarScannerInterface" {
    +DownloadScanner(giteaURL: string, scannerRepo: string): error
    +ConfigureScanner(config: ScannerConfig): error
    +ExecuteScan(): error
}
```

#### 3.1.3. ConfigurationManagerInterface

Интерфейс для работы с конфигурацией:

```go
interface "ConfigurationManagerInterface" {
    +LoadAppConfig(): *AppConfig, error
    +LoadSecretConfig(): *SecretConfig, error
    +ValidateConfig(): error
}
```

### 3.2. Сервисы

#### 3.2.1. SonarQubeService

Сервис для работы с SonarQube API:

```go
class "SonarQubeService" {
    -api: SonarQubeAPIInterface
    -config: *SonarQubeConfig
    +NewSonarQubeService(api: SonarQubeAPIInterface, cfg: *SonarQubeConfig): *SonarQubeService
    +CreateProject(owner: string, repo: string, branch: string): string, error
    +GetProjectList(): []Project, error
    +GetProjectToken(projectName: string): string, error
    +GetScanList(projectName: string): []Scan, error
    +Authenticate(): error
}
```

#### 3.2.2. SonarScannerManager

Сервис для управления sonar-scanner:

```go
class "SonarScannerManager" {
    -scanner: SonarScannerInterface
    -config: *SonarScannerConfig
    +NewSonarScannerManager(scanner: SonarScannerInterface, cfg: *SonarScannerConfig): *SonarScannerManager
    +DownloadScanner(): error
    +ConfigureScanner(config: ScannerConfig): error
    +ExecuteScan(): error
}
```

#### 3.2.3. ConfigurationManager

Сервис для работы с конфигурацией:

```go
class "ConfigurationManager" {
    -config: *Config
    +NewConfigurationManager(cfg: *Config): *ConfigurationManager
    +LoadAppConfig(): *AppConfig, error
    +LoadSecretConfig(): *SecretConfig, error
    +ValidateConfig(): error
}
```

### 3.3. Обработчики команд

#### 3.3.1. SQCommandHandler

Основной обработчик команд SonarQube:

```go
class "SQCommandHandler" {
    -giteaService: *GiteaService
    -sonarQubeService: *SonarQubeService
    -sonarScannerManager: *SonarScannerManager
    -configManager: *ConfigurationManager
    +NewSQCommandHandler(giteaService: *GiteaService, sonarQubeService: *SonarQubeService, sonarScannerManager: *SonarScannerManager, configManager: *ConfigurationManager): *SQCommandHandler
    +HandleSQScanBranch(owner: string, repo: string, branch: string, commitHash: string): error
    +HandleSQScanPR(owner: string, repo: string, pr: string): error
    +HandleSQProjectUpdate(owner: string, repo: string): error
    +HandleSQRepoSync(owner: string, repo: string): error
    +HandleSQRepoClear(owner: string, repo: string, force: bool): error
    +HandleSQReportPR(owner: string, repo: string, pr: string): error
    +HandleSQReportBranch(owner: string, repo: string, branch: string, firstCommitHash: string, lastCommitHash: string): error
    +HandleSQReportProject(owner: string, repo: string): error
}
```

#### 3.3.2. Специализированные обработчики команд

Для каждой команды SonarQube создан отдельный обработчик:

- SQScanBranch - сканирование ветки
- SQScanPR - сканирование PR
- SQProjectUpdate - обновление проекта
- SQRepoSync - синхронизация репозитория
- SQRepoClear - очистка репозитория
- SQReportPR - отчет по PR
- SQReportBranch - отчет по ветке
- SQReportProject - отчет по проекту

## 4. Процессы и потоки данных

### 4.1. Процесс сканирования ветки (sq-scan-branch)

1. Получение данных о ветке из Gitea API
2. Если ветка не main, получение первого коммита ветки
3. Получение данных о проекте из SonarQube API
4. Если проект не существует, создание нового проекта
5. Если commit_hash не передан:
   - Если ветка main, сканирование последнего коммита ветки
   - Если ветка не main, проверка наличия сканирования базового коммита
     - Если сканирования нет, сканирование базового коммита, затем текущего коммита
     - Если сканирование есть, сканирование текущего коммита
6. Если commit_hash передан, сканирование состояния репозитория на указанный коммит

### 4.2. Процесс сканирования PR (sq-scan-pr)

1. Получение данных PR из Gitea API
2. Получение имени ветки источника из данных PR
3. Вызов sq-scan-branch для ветки источника

### 4.3. Процесс обновления проекта (sq-project-update)

1. Получение содержимого README.md из репозитория
2. Обновление описания проекта в SonarQube
3. Синхронизация списка администраторов

### 4.4. Процесс синхронизации репозитория (sq-repo-sync)

1. Получение списка веток из Gitea API
2. Получение списка проектов из SonarQube API
3. Для каждой ветки:
   - Если проект не существует, создание нового проекта
   - Сканирование ветки

### 4.5. Процесс очистки репозитория (sq-repo-clear)

1. Получение списка веток из Gitea API
2. Получение списка проектов из SonarQube API
3. Для каждого проекта, который не соответствует существующей ветке:
   - Если force=true, удаление проекта
   - Если force=false, пометка проекта как устаревшего

### 4.6. Процесс создания отчета по PR (sq-report-pr)

1. Получение данных PR из Gitea API
2. Получение имени ветки источника из данных PR
3. Вызов sq-report-branch для ветки источника

### 4.7. Процесс создания отчета по ветке (sq-report-branch)

1. Получение данных о ветке из Gitea API
2. Получение данных о проекте из SonarQube API
3. Получение списка ошибок из SonarQube API
4. Формирование отчета по ошибкам

### 4.8. Процесс создания отчета по проекту (sq-report-project)

1. Получение списка веток из Gitea API
2. Для каждой ветки вызов sq-report-branch

## 5. Взаимодействие с внешними системами

### 5.1. Взаимодействие с Gitea API

Взаимодействие с Gitea API осуществляется через интерфейс GiteaAPIInterface, который реализуется классом GiteaEntity. Для получения данных о репозитории, ветках, PR и коммитах используются методы этого интерфейса.

### 5.2. Взаимодействие с SonarQube API

Взаимодействие с SonarQube API осуществляется через интерфейс SonarQubeAPIInterface, который реализуется классом SonarQubeService. Для создания проектов, получения списка проектов, получения токена проекта и получения списка сканирований используются методы этого интерфейса.

### 5.3. Взаимодействие с sonar-scanner

Взаимодействие с sonar-scanner осуществляется через интерфейс SonarScannerInterface, который реализуется классом SonarScannerManager. Для загрузки sonar-scanner, настройки параметров сканирования и выполнения сканирования используются методы этого интерфейса.

## 6. Обработка ошибок и механизмы повторных попыток

### 6.1. Обработка ошибок API

Все ошибки API обрабатываются и логируются с информативными сообщениями. Для пользователя выводятся понятные сообщения об ошибках.

### 6.2. Механизм повторных попыток

Для API-запросов реализован механизм повторных попыток с экспоненциально увеличивающейся задержкой. Максимальное количество попыток настраивается в конфигурации.

### 6.3. Обработка ошибок sonar-scanner

При выполнении sonar-scanner перехватывается вывод и анализируются ошибки. В случае ошибок предоставляется детальная диагностическая информация.

## 7. Конфигурация

### 7.1. Конфигурационные файлы

Для конфигурации используются файлы app.yaml и secret.yaml. В app.yaml хранятся общие настройки, а в secret.yaml - секретные данные, такие как токены аутентификации.

### 7.2. Валидация конфигурации

Перед использованием конфигурация валидируется на наличие всех необходимых параметров и их корректность.

## 8. Заключение

Архитектура интеграции с SonarQube для benadis-runner построена на принципах SOLID, с использованием интерфейсов для обеспечения гибкости и тестируемости. Она обеспечивает все необходимые функции для работы с SonarQube API и sonar-scanner, а также предоставляет удобный интерфейс для пользователя через CLI-команды.