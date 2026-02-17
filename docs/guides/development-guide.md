# Руководство по разработке

## Предварительные требования

- **Go** 1.25.1+
- **Make** (для автоматизации сборки)
- **golangci-lint** (для линтинга)
- **Git**

### Опциональные зависимости

- **godoc** - для генерации документации
- **smbclient, mount.cifs, kinit, klist** - для SMB тестов

## Установка

### Клонирование репозитория

```bash
git clone <repository-url>
cd apk-ci
```

### Настройка среды разработки

```bash
make setup-dev
```

Это установит:
- golangci-lint
- godoc

### Установка зависимостей

```bash
make deps
```

## Сборка

### Стандартная сборка

```bash
make build
```

Результат: `./build/apk-ci`

### Кросс-платформенная сборка

```bash
# Все платформы
make build-all

# Конкретная платформа
make build-linux
make build-windows
make build-darwin
```

### Информация о версии

```bash
make version
```

## Тестирование

### Запуск всех тестов

```bash
make test
# или
go test ./...
```

### Тесты с покрытием

```bash
make test-coverage
```

Результат: `coverage.html`

### Интеграционные тесты

```bash
make test-integration
```

Требует настройки переменных окружения для тестирования.

### SMB тесты

```bash
# Проверка зависимостей
make check-smb-deps

# Запуск тестов
make test-smb
```

### Запуск отдельного теста

```bash
go test -v ./internal/config -run TestMustLoad
```

## Качество кода

### Все проверки

```bash
make check
```

Выполняет: fmt, vet, lint, test

### Форматирование

```bash
make fmt
```

### Линтинг

```bash
# Все файлы
make lint

# Без тестов
make lint-no-test
```

### go vet

```bash
make vet
```

## Конфигурация линтера

Настройки в `.golangci.yml`:

### Включенные линтеры (20+)

- govet, errcheck, staticcheck
- unused, ineffassign, nilerr
- rowserrcheck, sqlclosecheck
- durationcheck, predeclared
- bodyclose, goconst
- goprintffuncname, whitespace
- misspell, gosec, prealloc
- revive, gocritic
- forbidigo, dogsled, dupl
- wastedassign

### Отключенные линтеры

- lll (длина строк)
- funlen (длина функций)
- wsl (whitespace linter)
- depguard (зависимости)

## Локальный запуск

### Через Makefile

```bash
make run
```

Требует файл конфигурации `examples/service-mode-config.env`

### Прямой запуск

```bash
BR_COMMAND=service-mode-status BR_INFOBASE_NAME=MyInfobase ./apk-ci
```

## Структура переменных окружения

### Обязательные

- `BR_COMMAND` - команда для выполнения
- `BR_INFOBASE_NAME` - имя информационной базы (для многих команд)

### Gitea

- `BR_GITEA_URL` - URL сервера Gitea
- `BR_ACCESS_TOKEN` - токен доступа
- `BR_OWNER` - владелец репозитория
- `BR_REPO` - имя репозитория

### SonarQube

- `BR_SQ_URL` - URL сервера SonarQube
- `BR_SQ_TOKEN` - токен API

### База данных

- `DBRESTORE_SERVER` - сервер MSSQL
- `DBRESTORE_DATABASE` - имя БД
- `DBRESTORE_USER` - пользователь
- `DBRESTORE_PASSWORD` - пароль

## Добавление новой команды

### 1. Константа

`internal/constants/constants.go`:

```go
const (
    ActNewCommand = "new-command"
)
```

### 2. Функция оркестрации

`internal/app/app.go`:

```go
func NewCommand(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
    l.Info("Starting new command")

    // Бизнес-логика

    return nil
}
```

### 3. Маршрутизация

`cmd/apk-ci/main.go`:

```go
case constants.ActNewCommand:
    err = app.NewCommand(&ctx, l, cfg)
    if err != nil {
        l.Error("Ошибка выполнения новой команды",
            slog.String("error", err.Error()),
            slog.String(constants.MsgErrProcessing, constants.MsgAppExit),
        )
        os.Exit(8)
    }
    l.Info("Новая команда успешно выполнена")
```

### 4. Тесты

`internal/app/new_command_test.go`:

```go
func TestNewCommand(t *testing.T) {
    tests := []struct {
        name    string
        cfg     *config.Config
        wantErr bool
    }{
        {
            name:    "успешное выполнение",
            cfg:     &config.Config{...},
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := context.Background()
            l := slog.Default()

            err := NewCommand(&ctx, l, tt.cfg)

            if (err != nil) != tt.wantErr {
                t.Errorf("NewCommand() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Паттерны тестирования

### Табличные тесты

```go
tests := []struct {
    name    string
    input   string
    want    string
    wantErr bool
}{
    {"valid input", "test", "result", false},
    {"empty input", "", "", true},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got, err := Function(tt.input)
        if (err != nil) != tt.wantErr {
            t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
        }
        if got != tt.want {
            t.Errorf("got = %v, want %v", got, tt.want)
        }
    })
}
```

### Мокирование БД

```go
import "github.com/DATA-DOG/go-sqlmock"

func TestDBOperation(t *testing.T) {
    db, mock, err := sqlmock.New()
    require.NoError(t, err)
    defer db.Close()

    mock.ExpectQuery("SELECT").
        WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

    // тест
}
```

### Мокирование интерфейсов

```go
type MockGiteaAPI struct {
    mock.Mock
}

func (m *MockGiteaAPI) GetCommitFiles(hash string) ([]CommitFile, error) {
    args := m.Called(hash)
    return args.Get(0).([]CommitFile), args.Error(1)
}
```

## Отладка

### Уровни логирования

- `Debug` - детальная информация
- `Info` - информационные сообщения
- `Warn` - предупреждения
- `Error` - ошибки

### Установка уровня

```bash
BR_LOG_LEVEL=Debug ./apk-ci
```

## Релиз

### Подготовка релиза

```bash
make release
```

Выполняет: clean, check, build-all

Результаты в `./build/`:
- apk-ci-linux-amd64
- apk-ci-windows-amd64.exe
- apk-ci-darwin-amd64

## Документация

### Запуск godoc

```bash
make docs
```

Доступно на http://localhost:6060

### Обновление CLAUDE.md

При изменении архитектуры или добавлении новых команд обновите `CLAUDE.md` для корректной работы AI-ассистентов.
