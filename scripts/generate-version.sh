#!/bin/bash

# Скрипт для автоматической генерации файла version.go с константами версии
# на основе текущей даты и информации о git коммите

set -e

# Путь к файлу version.go
VERSION_FILE="internal/constants/version.go"

# Получение текущей даты
CURRENT_DAY=$(date +"%-d")  # Без лидирующих нулей
CURRENT_MONTH=$(date +"%-m")  # Без лидирующих нулей
CURRENT_YEAR=$(date +"%Y")
YEAR_LAST_DIGIT=${CURRENT_YEAR: -1}  # Последняя цифра года

# Получение короткого хеша последнего коммита (7 символов)
GIT_COMMIT_HASH=$(git rev-parse --short=7 HEAD 2>/dev/null || echo "unknown")

# Получение комментария последнего коммита
GIT_COMMIT_MESSAGE=$(git log -1 --pretty=format:'%s' 2>/dev/null || echo "unknown")

# Получение BUILD_NUMBER из существующего файла version.go или инициализация
if [ -f "$VERSION_FILE" ]; then
    # Извлекаем текущий BUILD_NUMBER из файла version.go
    CURRENT_BUILD=$(grep 'versionMinor = ' "$VERSION_FILE" | sed 's/.*versionMinor = "\([0-9]*\)".*/\1/')
    # Проверяем, совпадает ли дата в файле с текущей датой
    CURRENT_DAY_IN_FILE=$(grep 'versionDay = ' "$VERSION_FILE" | sed 's/.*versionDay = "\([0-9]*\)".*/\1/')
    CURRENT_MONTH_IN_FILE=$(grep 'versionMonth = ' "$VERSION_FILE" | sed 's/.*versionMonth = "\([0-9]*\)".*/\1/')
    CURRENT_YEAR_IN_FILE=$(grep 'versionYear = ' "$VERSION_FILE" | sed 's/.*versionYear = "\([0-9]*\)".*/\1/')
    
    if [ "$CURRENT_DAY_IN_FILE" = "$CURRENT_DAY" ] && [ "$CURRENT_MONTH_IN_FILE" = "$CURRENT_MONTH" ] && [ "$CURRENT_YEAR_IN_FILE" = "$YEAR_LAST_DIGIT" ]; then
        # Та же дата - увеличиваем номер сборки
        BUILD_NUMBER=$((CURRENT_BUILD + 1))
    else
        # Новая дата - сбрасываем счетчик
        BUILD_NUMBER=1
    fi
else
    BUILD_NUMBER=1
fi

# Функция для генерации version.go
generate_version_go() {
    local is_debug="$1"
    local debug_suffix=""
    
    if [ "$is_debug" = "true" ]; then
        debug_suffix="-debug"
    fi
    
    echo "Генерация version.go:"
    echo "  Последняя цифра года: $YEAR_LAST_DIGIT"
    echo "  Месяц: $CURRENT_MONTH"
    echo "  День: $CURRENT_DAY"
    echo "  Номер сборки за день: $BUILD_NUMBER"
    echo "  Git коммит: $GIT_COMMIT_HASH"
    if [ "$is_debug" = "true" ]; then
        echo "  Режим отладки: включен"
    fi

    # Создание директории если не существует
    mkdir -p "$(dirname "$VERSION_FILE")"

    # Генерация файла version.go
    cat > "$VERSION_FILE" << EOF
// Package constants содержит константы версии приложения.
// Этот файл автоматически генерируется при сборке.
// НЕ РЕДАКТИРУЙТЕ ЭТОТ ФАЙЛ ВРУЧНУЮ!
package constants

// Константы версии приложения
const (
	// versionMinor - минорная версия (номер сборки за текущий день)
	versionMinor = "$BUILD_NUMBER"
	// versionDay - день версии (текущий день)
	versionDay = "$CURRENT_DAY"
	// versionMonth - месяц версии (текущий месяц)
	versionMonth = "$CURRENT_MONTH"
	// versionYear - последняя цифра года
	versionYear = "$YEAR_LAST_DIGIT"
	// PreCommitHash - хеш последнего коммита на момент сборки
	PreCommitHash = "$GIT_COMMIT_HASH"
	// DebugSuffix - суффикс для отладочной сборки
	DebugSuffix = "$debug_suffix"
	// Version - полная версия приложения в формате: год.месяц.день.сборка:коммит
	Version = versionYear + "." + versionMonth + "." + versionDay + "." + versionMinor + ":" + PreCommitHash + DebugSuffix
)
EOF

    echo "Файл $VERSION_FILE успешно сгенерирован"
    echo "Версия: $YEAR_LAST_DIGIT.$CURRENT_MONTH.$CURRENT_DAY.$BUILD_NUMBER:$GIT_COMMIT_HASH$debug_suffix"
    echo "Коммит: $GIT_COMMIT_HASH$debug_suffix"
}

# Функция для создания файла version.md с новой логикой
generate_version_md() {
    local target_dir="$1"
    local is_debug="$2"
    local version_md_file="$target_dir/version.md"
    
    echo "Генерация version.md в каталоге: $target_dir"
    
    # Определяем тип сборки и debug-суффикс
    local build_type="Продакшн"
    local debug_suffix=""
    if [ "$is_debug" = "true" ]; then
        build_type="Отладка"
        debug_suffix="-debug"
    fi
    
    # Определяем дату последнего коммита в целевом каталоге
    local target_last_commit_time=""
    
    if [ -d "$target_dir/.git" ]; then
        # Если в целевом каталоге есть git репозиторий
        target_last_commit_time=$(cd "$target_dir" && git log -1 --format="%ct" 2>/dev/null || echo "")
        echo "Найден git репозиторий в целевом каталоге. Последний коммит: $(date -d @$target_last_commit_time 2>/dev/null || echo 'неизвестно')"
    else
        echo "Git репозиторий в целевом каталоге не найден. Будут получены 10 последних коммитов текущего репозитория."
    fi
    
    # Получаем коммиты из текущего репозитория
    local commits_data=""
    if [ -n "$target_last_commit_time" ]; then
        # Получаем коммиты начиная с TARGET_LAST_COMMIT_TIME
        commits_data=$(git log --since="$target_last_commit_time" --format="%ct|%H|%s" --reverse 2>/dev/null || echo "")
    else
        # Получаем 10 последних коммитов
        commits_data=$(git log -10 --format="%ct|%H|%s" --reverse 2>/dev/null || echo "")
    fi
    
    # Создание директории если не существует
    mkdir -p "$target_dir"
    
    # Генерация файла version.md с улучшенным форматированием
    cat > "$version_md_file" << EOF
# 📦 Информация о сборке

## 🔖 Версия
**Версия сборки:** \`$YEAR_LAST_DIGIT.$CURRENT_MONTH.$CURRENT_DAY.$BUILD_NUMBER:$GIT_COMMIT_HASH$debug_suffix\`

**Дата сборки:** \`$(date '+%Y-%m-%d %H:%M:%S')\`

**Тип сборки:** \`$build_type\`

---

## 📋 История коммитов

EOF
    
    # Обработка коммитов и запись в файл
    if [ -n "$commits_data" ] && [ "$(echo "$commits_data" | wc -l)" -gt 0 ] && [ "$(echo "$commits_data" | head -1)" != "" ]; then
        # Добавляем заголовок таблицы
        cat >> "$version_md_file" << EOF
| № | Дата коммита | Хеш | Описание |
|---|--------------|-----|----------|
EOF
        
        local counter=1
        # Сортируем коммиты от новых к старым (reverse порядок)
        echo "$commits_data" | tac | while IFS='|' read -r timestamp hash message; do
            if [ -n "$timestamp" ] && [ -n "$hash" ]; then
                local commit_date=$(date -d @"$timestamp" "+%Y-%m-%d %H:%M:%S" 2>/dev/null || echo "неизвестно")
                local short_hash=$(echo "$hash" | cut -c1-7)
                # Экранируем символы для markdown таблицы
                local escaped_message=$(echo "$message" | sed 's/|/\\|/g')
                echo "| $counter | $commit_date | \`$short_hash\` | $escaped_message |" >> "$version_md_file"
                counter=$((counter + 1))
            fi
        done
    else
        echo "**Пересборка текущей версии**" >> "$version_md_file"
        echo "" >> "$version_md_file"
    fi
    
    # Добавляем footer с дополнительной информацией
    cat >> "$version_md_file" << EOF

---

## 🛠️ Техническая информация

- **Репозиторий:** benadis-runner
- **Последний коммит:** \`$GIT_COMMIT_HASH\`
- **Сообщение коммита:** \`$GIT_COMMIT_MESSAGE\`
- **Сборка создана:** $(date '+%Y-%m-%d в %H:%M:%S')
- **Тип сборки:** $build_type

> Этот файл автоматически генерируется при сборке проекта.
EOF
    
    echo "Файл $version_md_file успешно создан"
}

# Проверяем параметры командной строки
if [ "$1" = "--version-go-only" ]; then
    # Генерируем только version.go, без version.md
    generate_version_go "false"
elif [ "$1" = "--version-go-debug" ]; then
    # Генерируем только version.go в режиме отладки
    generate_version_go "true"
elif [ "$1" = "--version-md-only" ]; then
    # Генерируем только version.md в указанном каталоге, без version.go
    # Проверяем наличие флага --debug
    if [ "$2" = "--debug" ] && [ -n "$3" ]; then
        TARGET_DIR="$3"
        echo "Создание version.md в целевом каталоге: $TARGET_DIR"
        generate_version_md "$TARGET_DIR" "true"
    elif [ -n "$2" ]; then
        TARGET_DIR="$2"
        echo "Создание version.md в целевом каталоге: $TARGET_DIR"
        generate_version_md "$TARGET_DIR" "false"
    fi
elif [ -n "$1" ]; then
    # Если передан параметр целевого каталога, создаем version.go и version.md только там
    generate_version_go "false"
    TARGET_DIR="$1"
    echo "Создание version.md в целевом каталоге: $TARGET_DIR"
    generate_version_md "$TARGET_DIR" "false"
else
    # Создание version.go и файла version.md в каталоге build (по умолчанию)
    generate_version_go "false"
    BUILD_DIR="build"
    generate_version_md "$BUILD_DIR" "false"
fi