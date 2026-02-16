#!/bin/bash

# install-dev-tools-no-sudo.sh
# Скрипт для установки расширений Go и Task без использования sudo (для контейнерных сред)

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Функция для вывода сообщений
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Проверка операционной системы
detect_os() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        OS="linux"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        OS="darwin"
    elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
        OS="windows"
    else
        log_error "Неподдерживаемая операционная система: $OSTYPE"
        exit 1
    fi
    
    # Определение архитектуры
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l)
            ARCH="armv6l"
            ;;
        *)
            log_error "Неподдерживаемая архитектура: $ARCH"
            exit 1
            ;;
    esac
    
    log_info "Обнаружена система: $OS/$ARCH"
}

# Создание директорий для локальной установки
setup_local_dirs() {
    LOCAL_BIN="$HOME/.local/bin"
    LOCAL_GO="$HOME/.local/go"
    
    mkdir -p "$LOCAL_BIN"
    mkdir -p "$LOCAL_GO"
    
    log_info "Локальные директории созданы: $LOCAL_BIN, $LOCAL_GO"
}

# Установка Go
install_go() {
    log_info "Проверка установки Go..."
    
    if command -v go &> /dev/null; then
        CURRENT_GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        log_info "Go уже установлен: версия $CURRENT_GO_VERSION"
        
        # Проверяем, нужно ли обновление
        LATEST_GO_VERSION=$(curl -s https://go.dev/VERSION?m=text | tr -d '\n\r' | sed 's/go//' | sed 's/time.*//')
        if [[ "$CURRENT_GO_VERSION" != "$LATEST_GO_VERSION" ]]; then
            log_warning "Доступна новая версия Go: $LATEST_GO_VERSION"
            echo "Обновляем Go..."
            install_go_binary
        else
            log_success "Go уже установлен последней версии"
        fi
    else
        log_info "Go не найден, устанавливаем..."
        install_go_binary
    fi
}

# Установка бинарного файла Go
install_go_binary() {
    log_info "Получение последней версии Go..."
    LATEST_GO_VERSION=$(curl -s https://go.dev/VERSION?m=text | tr -d '\n\r' | sed 's/time.*//')
    
    if [[ "$OS" == "windows" ]]; then
        GO_ARCHIVE="${LATEST_GO_VERSION}.${OS}-${ARCH}.zip"
    else
        GO_ARCHIVE="${LATEST_GO_VERSION}.${OS}-${ARCH}.tar.gz"
    fi
    
    GO_URL="https://go.dev/dl/${GO_ARCHIVE}"
    
    log_info "Скачивание Go: $GO_URL"
    
    # Создаем временную директорию
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"
    
    # Скачиваем Go
    if ! curl -L -o "$GO_ARCHIVE" "$GO_URL"; then
        log_error "Ошибка при скачивании Go"
        exit 1
    fi
    
    # Удаляем старую установку Go (если есть)
    if [[ -d "$LOCAL_GO" ]]; then
        log_info "Удаление старой установки Go..."
        rm -rf "$LOCAL_GO"
    fi
    
    # Извлекаем архив
    log_info "Установка Go в $LOCAL_GO..."
    if [[ "$OS" == "windows" ]]; then
        unzip -q "$GO_ARCHIVE"
    else
        tar -xzf "$GO_ARCHIVE"
    fi
    
    # Перемещаем в локальную директорию
    mv go "$LOCAL_GO"
    
    # Очистка
    cd - > /dev/null
    rm -rf "$TEMP_DIR"
    
    log_success "Go успешно установлен в $LOCAL_GO"
}

# Настройка PATH для Go
setup_go_path() {
    log_info "Настройка PATH для Go..."
    
    GO_PATH="$LOCAL_GO/bin"
    GOPATH="$HOME/go"
    GOBIN="$GOPATH/bin"
    
    # Создаем GOPATH если не существует
    mkdir -p "$GOPATH"
    mkdir -p "$GOBIN"
    
    # Определяем файл профиля
    if [[ -f "$HOME/.bashrc" ]]; then
        PROFILE_FILE="$HOME/.bashrc"
    elif [[ -f "$HOME/.zshrc" ]]; then
        PROFILE_FILE="$HOME/.zshrc"
    elif [[ -f "$HOME/.profile" ]]; then
        PROFILE_FILE="$HOME/.profile"
    else
        PROFILE_FILE="$HOME/.bashrc"
        touch "$PROFILE_FILE"
    fi
    
    # Проверяем, добавлены ли уже пути
    if ! grep -q "export PATH.*$GO_PATH" "$PROFILE_FILE"; then
        log_info "Добавление Go в PATH..."
        {
            echo ""
            echo "# Go environment (local installation)"
            echo "export PATH=\$PATH:$GO_PATH"
            echo "export GOPATH=$GOPATH"
            echo "export GOBIN=$GOBIN"
            echo "export PATH=\$PATH:$GOBIN"
            echo "export PATH=\$PATH:$LOCAL_BIN"
        } >> "$PROFILE_FILE"
        
        log_success "Go PATH настроен в $PROFILE_FILE"
    else
        log_info "Go PATH уже настроен"
    fi
    
    # Экспортируем переменные для текущей сессии
    export PATH="$PATH:$GO_PATH:$GOBIN:$LOCAL_BIN"
    export GOPATH="$GOPATH"
    export GOBIN="$GOBIN"
}

# Установка Go tools
install_go_tools() {
    log_info "Установка Go tools..."
    
    # Список основных Go tools
    GO_TOOLS=(
        "golang.org/x/tools/gopls@latest"                    # Language Server
        "github.com/go-delve/delve/cmd/dlv@latest"           # Debugger
        "golang.org/x/tools/cmd/goimports@latest"            # Import formatter
        "github.com/golangci/golangci-lint/cmd/golangci-lint@latest" # Advanced linter
        "golang.org/x/tools/cmd/godoc@latest"                # Documentation
        "github.com/fatih/gomodifytags@latest"               # Struct tag modifier
        "github.com/josharian/impl@latest"                   # Interface implementation generator
        "github.com/ramya-rao-a/go-outline@latest"           # Go outline
        "github.com/uudashr/gopkgs/v2/cmd/gopkgs@latest"      # Package list
        "github.com/cweill/gotests/gotests@latest"           # Test generator
    )
    
    for tool in "${GO_TOOLS[@]}"; do
        log_info "Установка $tool..."
        if go install "$tool"; then
            log_success "Установлен: $tool"
        else
            log_warning "Ошибка при установке: $tool"
        fi
    done
}

# Установка Task
install_task() {
    log_info "Проверка установки Task..."
    
    if command -v task &> /dev/null; then
        CURRENT_TASK_VERSION=$(task --version | awk '{print $3}')
        log_info "Task уже установлен: версия $CURRENT_TASK_VERSION"
        
        # Проверяем последнюю версию
        LATEST_TASK_VERSION=$(curl -s https://api.github.com/repos/go-task/task/releases/latest | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
        if [[ "$CURRENT_TASK_VERSION" != "$LATEST_TASK_VERSION" ]]; then
            log_warning "Доступна новая версия Task: $LATEST_TASK_VERSION"
            echo "Обновляем Task..."
            install_task_binary
        else
            log_success "Task уже установлен последней версии"
        fi
    else
        log_info "Task не найден, устанавливаем..."
        install_task_binary
    fi
}

# Установка бинарного файла Task
install_task_binary() {
    log_info "Получение последней версии Task..."
    
    # Получаем последнюю версию
    LATEST_TASK_VERSION=$(curl -s https://api.github.com/repos/go-task/task/releases/latest | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
    
    if [[ "$OS" == "windows" ]]; then
        TASK_ARCHIVE="task_${OS}_${ARCH}.zip"
    else
        TASK_ARCHIVE="task_${OS}_${ARCH}.tar.gz"
    fi
    
    TASK_URL="https://github.com/go-task/task/releases/download/v${LATEST_TASK_VERSION}/${TASK_ARCHIVE}"
    
    log_info "Скачивание Task: $TASK_URL"
    
    # Создаем временную директорию
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"
    
    # Скачиваем Task
    if ! curl -L -o "$TASK_ARCHIVE" "$TASK_URL"; then
        log_error "Ошибка при скачивании Task"
        exit 1
    fi
    
    # Извлекаем архив
    if [[ "$OS" == "windows" ]]; then
        unzip -q "$TASK_ARCHIVE"
        TASK_BINARY="task.exe"
    else
        tar -xzf "$TASK_ARCHIVE"
        TASK_BINARY="task"
    fi
    
    # Устанавливаем в локальную директорию
    log_info "Установка Task в $LOCAL_BIN..."
    mv "$TASK_BINARY" "$LOCAL_BIN/"
    chmod +x "$LOCAL_BIN/$TASK_BINARY"
    
    # Очистка
    cd - > /dev/null
    rm -rf "$TEMP_DIR"
    
    log_success "Task успешно установлен в $LOCAL_BIN"
}

# Проверка установки
verify_installation() {
    log_info "Проверка установки..."
    
    echo
    log_info "=== Проверка Go ==="
    if command -v go &> /dev/null; then
        go version
        log_success "Go установлен и доступен"
    else
        log_error "Go не найден в PATH"
    fi
    
    echo
    log_info "=== Проверка Task ==="
    if command -v task &> /dev/null; then
        task --version
        log_success "Task установлен и доступен"
    else
        log_error "Task не найден в PATH"
    fi
    
    echo
    log_info "=== Go tools ==="
    GO_TOOLS_CHECK=(
        "gopls"
        "dlv"
        "goimports"
        "golangci-lint"
    )
    
    for tool in "${GO_TOOLS_CHECK[@]}"; do
        if command -v "$tool" &> /dev/null; then
            log_success "$tool установлен"
        else
            log_warning "$tool не найден"
        fi
    done
    
    echo
    log_info "=== Переменные окружения ==="
    echo "GOPATH: $GOPATH"
    echo "GOBIN: $GOBIN"
    echo "PATH содержит Go: $(echo $PATH | grep -o '[^:]*go[^:]*' | head -1 || echo 'не найдено')"
    echo "PATH содержит local/bin: $(echo $PATH | grep -o '[^:]*\.local/bin[^:]*' | head -1 || echo 'не найдено')"
}

# Главная функция
main() {
    log_info "Начало установки инструментов разработки (без sudo)..."
    echo
    
    detect_os
    setup_local_dirs
    
    install_go
    setup_go_path
    install_go_tools
    
    install_task
    
    verify_installation
    
    echo
    log_success "Установка завершена!"
    log_info "Перезапустите терминал или выполните: source ~/.bashrc (или ~/.zshrc)"
    log_info "Для применения изменений PATH"
    
    echo
    log_info "Установленные директории:"
    log_info "  Go: $LOCAL_GO"
    log_info "  Локальные бинарные файлы: $LOCAL_BIN"
    log_info "  Go workspace: $GOPATH"
}

# Запуск скрипта
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi