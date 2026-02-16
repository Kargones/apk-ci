# Makefile для проекта benadis-runner
# Упрощает сборку, тестирование и развертывание

# Переменные
APP_NAME := benadis-runner
CMD_DIR := ./cmd/benadis-runner
BUILD_DIR := ./build
DOCS_DIR := ./docs
EXAMPLES_DIR := ./examples

# Go параметры
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Версия и метаданные
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Флаги сборки
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Цели по умолчанию
.PHONY: all build clean test deps help
.DEFAULT_GOAL := help

## help: Показать справку
help:
	@echo "Доступные команды:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## all: Выполнить полную сборку (deps + test + build)
all: deps test build

## build: Собрать приложение
build:
	@echo "Сборка $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)
	@echo "Сборка завершена: $(BUILD_DIR)/$(APP_NAME)"

## build-all: Собрать для всех платформ
build-all: build-linux build-windows build-darwin

## build-linux: Собрать для Linux
build-linux:
	@echo "Сборка для Linux..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux $(CMD_DIR)

## test-smb: Запустить тесты SMB модуля
test-smb:
	@echo "Запуск тестов SMB модуля..."
	$(GOTEST) -v -race -coverprofile=smb_coverage.out ./internal/smb/...

## check-smb-deps: Проверить системные зависимости для SMB
check-smb-deps:
	@echo "Проверка системных зависимостей SMB..."
	@command -v smbclient >/dev/null 2>&1 || echo "WARNING: smbclient не найден"
	@command -v mount.cifs >/dev/null 2>&1 || echo "WARNING: mount.cifs не найден"
	@command -v kinit >/dev/null 2>&1 || echo "WARNING: kinit не найден"
	@command -v klist >/dev/null 2>&1 || echo "WARNING: klist не найден"
	@echo "Проверка системных зависимостей завершена"

## install-smb-deps-ubuntu: Установить зависимости SMB для Ubuntu/Debian
install-smb-deps-ubuntu:
	@echo "Установка зависимостей SMB для Ubuntu/Debian..."
	sudo apt-get update
	sudo apt-get install -y samba-client cifs-utils krb5-user libkrb5-dev

## install-smb-deps-centos: Установить зависимости SMB для CentOS/RHEL
install-smb-deps-centos:
	@echo "Установка зависимостей SMB для CentOS/RHEL..."
	sudo yum install -y samba-client cifs-utils krb5-workstation krb5-devel
	@echo "Сборка для Linux..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(CMD_DIR)

## build-windows: Собрать для Windows
build-windows:
	@echo "Сборка для Windows..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(CMD_DIR)

## build-darwin: Собрать для macOS
build-darwin:
	@echo "Сборка для macOS..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(CMD_DIR)

## clean: Очистить артефакты сборки
clean:
	@echo "Очистка..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "Очистка завершена"

## test: Запустить тесты
test:
	@echo "Запуск тестов..."
	$(GOTEST) -v ./...

## test-coverage: Запустить тесты с покрытием
test-coverage:
	@echo "Запуск тестов с анализом покрытия..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Отчет о покрытии: coverage.html"

## test-integration: Запустить интеграционные тесты
test-integration:
	@echo "Запуск интеграционных тестов..."
	@echo "Убедитесь, что настроены переменные окружения для тестирования"
	$(GOTEST) -v -tags=integration ./internal/servicemode/

## deps: Установить зависимости
deps:
	@echo "Установка зависимостей..."
	$(GOGET) -d ./...
	$(GOMOD) tidy
	$(GOMOD) verify

## deps-update: Обновить зависимости
deps-update:
	@echo "Обновление зависимостей..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

## lint: Запустить линтеры
lint:
	@echo "Запуск линтеров..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint не установлен. Установите: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## lint-no-test: Запустить линтеры для всех файлов кроме тестов
lint-no-test:
	@echo "Запуск линтеров (исключая тестовые файлы)..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --tests=false; \
	else \
		echo "golangci-lint не установлен. Установите: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## fmt: Форматировать код
fmt:
	@echo "Форматирование кода..."
	$(GOCMD) fmt ./...

## vet: Запустить go vet
vet:
	@echo "Запуск go vet..."
	$(GOCMD) vet ./...

## mod-graph: Показать граф зависимостей
mod-graph:
	$(GOMOD) graph

## install: Установить приложение в GOPATH/bin
install:
	@echo "Установка $(APP_NAME)..."
	$(GOCMD) install $(LDFLAGS) $(CMD_DIR)

## run: Запустить приложение (требует настройки переменных окружения)
run:
	@echo "Запуск $(APP_NAME)..."
	@if [ -f "$(EXAMPLES_DIR)/service-mode-config.env" ]; then \
		echo "Загрузка конфигурации из $(EXAMPLES_DIR)/service-mode-config.env"; \
		set -a && source $(EXAMPLES_DIR)/service-mode-config.env && set +a && $(GOBUILD) $(LDFLAGS) -o $(APP_NAME) $(CMD_DIR) && ./$(APP_NAME); \
	else \
		echo "Файл конфигурации не найден. Создайте $(EXAMPLES_DIR)/service-mode-config.env"; \
		echo "Или настройте переменные окружения вручную"; \
		exit 1; \
	fi

## demo: Запустить демонстрацию
demo: build
	@echo "Запуск демонстрации..."
	@if [ -f "$(EXAMPLES_DIR)/service-mode-demo.sh" ]; then \
		chmod +x $(EXAMPLES_DIR)/service-mode-demo.sh; \
		cp $(BUILD_DIR)/$(APP_NAME) ./$(APP_NAME); \
		$(EXAMPLES_DIR)/service-mode-demo.sh; \
		rm -f ./$(APP_NAME); \
	else \
		echo "Демонстрационный скрипт не найден: $(EXAMPLES_DIR)/service-mode-demo.sh"; \
		exit 1; \
	fi

## docs: Генерировать документацию
docs:
	@echo "Генерация документации..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Запуск godoc сервера на http://localhost:6060"; \
		godoc -http=:6060; \
	else \
		echo "godoc не установлен. Установите: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

## version: Показать информацию о версии
version:
	@echo "Версия: $(VERSION)"
	@echo "Время сборки: $(BUILD_TIME)"
	@echo "Git коммит: $(GIT_COMMIT)"

## check: Выполнить все проверки (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "Все проверки пройдены успешно"

## setup-dev: Настроить среду разработки
setup-dev:
	@echo "Настройка среды разработки..."
	$(GOGET) -u golang.org/x/tools/cmd/godoc
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint
	@echo "Создание примера конфигурации..."
	@if [ ! -f "$(EXAMPLES_DIR)/service-mode-config.env" ]; then \
		echo "Файл конфигурации уже существует: $(EXAMPLES_DIR)/service-mode-config.env"; \
	else \
		echo "Скопируйте и настройте $(EXAMPLES_DIR)/service-mode-config.env для вашей среды"; \
	fi
	@echo "Среда разработки настроена"

## release: Подготовить релиз
release: clean check build-all
	@echo "Релиз готов в директории $(BUILD_DIR)/"
	@ls -la $(BUILD_DIR)/

# Служебные цели
.PHONY: _check-env
_check-env:
	@if [ -z "$(BR_COMMAND)" ] || [ -z "$(BR_INFOBASE_NAME)" ]; then \
		echo "Ошибка: Не настроены обязательные переменные окружения"; \
		echo "Настройте BR_COMMAND и BR_INFOBASE_NAME"; \
		echo "Или загрузите конфигурацию: source $(EXAMPLES_DIR)/service-mode-config.env"; \
		exit 1; \
	fi

# Цели для управления сервисным режимом
## service-enable: Включить сервисный режим
service-enable: build _check-env
	@export BR_COMMAND=service-mode-enable && $(BUILD_DIR)/$(APP_NAME)

## service-disable: Отключить сервисный режим
service-disable: build _check-env
	@export BR_COMMAND=service-mode-disable && $(BUILD_DIR)/$(APP_NAME)

## service-status: Проверить статус сервисного режима
service-status: build _check-env
	@export BR_COMMAND=service-mode-status && $(BUILD_DIR)/$(APP_NAME)