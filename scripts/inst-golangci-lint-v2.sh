#!/bin/bash

# inst-golangci-lint-v2.sh
# Скрипт для установки golangci-lint с поддержкой конфигурационного формата версии 2

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

# Проверка, что Go установлен
check_go() {
    if ! command -v go &> /dev/null; then
        log_error "Go не установлен. Пожалуйста, установите Go перед запуском этого скрипта."
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Найдена версия Go: $GO_VERSION"
}

# Установка golangci-lint с поддержкой конфигурационного формата версии 2
install_golangci_lint() {
    log_info "Установка golangci-lint с поддержкой конфигурационного формата версии 2..."
    
    # Проверяем, установлена ли уже golangci-lint
    if command -v golangci-lint &> /dev/null; then
        INSTALLED_VERSION=$(golangci-lint --version | awk '{print $4}')
        log_info "Установлена версия: $INSTALLED_VERSION"
        
        # Проверяем, является ли версия достаточно новой для поддержки формата v2
                if [[ $INSTALLED_VERSION == v1.* ]] || [[ $INSTALLED_VERSION == v2.* ]] || [[ $INSTALLED_VERSION == 1.* ]] || [[ $INSTALLED_VERSION == 2.* ]]; then
                    # Версии golangci-lint v1 и v2 поддерживают конфигурационный формат v2
                    log_success "golangci-lint уже установлена и поддерживает конфигурационный формат версии 2"
                    return 0
                else
                    log_warning "Установлена несовместимая версия. Будет выполнена установка совместимой версии."
                fi
    fi
    
    # Установка golangci-lint v2 через go install
        log_info "Установка golangci-lint v2 через go install..."
        if go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2; then
            log_success "golangci-lint v2 успешно установлена"
        else
            log_error "Ошибка при установке golangci-lint v2"
            exit 1
        fi
}

# Проверка установки
verify_installation() {
    log_info "Проверка установки..."
    
    if command -v golangci-lint &> /dev/null; then
        VERSION=$(golangci-lint --version | awk '{print $4}')
        log_success "golangci-lint доступен. Версия: $VERSION"
        
        # Проверяем, что версия начинается с v1, v2, 1. или 2. (поддерживают конфигурационный формат v2)
                if [[ $VERSION == v1.* ]] || [[ $VERSION == v2.* ]] || [[ $VERSION == 1.* ]] || [[ $VERSION == 2.* ]]; then
                    log_success "Установлена корректная версия golangci-lint"
                else
                    log_warning "Версия golangci-lint может быть несовместима: $VERSION"
                fi
    else
        log_error "golangci-lint не найден в PATH"
        exit 1
    fi
}

# Главная функция
main() {
    log_info "Начало установки golangci-lint (с поддержкой конфигурационного формата версии 2)..."
    echo
    
    check_go
    install_golangci_lint
    verify_installation
    
    echo
    log_success "Установка golangci-lint завершена!"
    log_info "Примечание: 'version: 2' в .golangci.yml относится к формату конфигурационного файла, а не к версии инструмента"
}

# Запуск скрипта
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi