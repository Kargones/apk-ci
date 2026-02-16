#!/bin/bash

# Переменные для репозитория и пути установки
REPO_OWNER="Kilo-Org"
REPO_NAME="kilocode"
TARGET_DIR="/root/r/benadis-runner"
VSIX_FILE="$TARGET_DIR/kilo-code.vsix"

# Создаем целевой каталог, если он не существует
mkdir -p "$TARGET_DIR"

# Получаем информацию о последнем релизе из GitHub API
echo "Получаем информацию о последнем релизе Kilo Code..."
RELEASE_JSON=$(curl -s "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest")

# Проверяем, успешно ли получен JSON
if [ -z "$RELEASE_JSON" ]; then
    echo "Ошибка: Не удалось получить данные о релизе. Проверьте сеть или имя репозитория."
    exit 1
fi

# Извлекаем URL первого актива (предполагаем, что это VSIX-файл)
VSIX_URL=$(echo "$RELEASE_JSON" | grep -o '"browser_download_url": "[^"]*' | head -1 | cut -d'"' -f4)

# Если URL не найден, пытаемся получить через теги
if [ -z "$VSIX_URL" ]; then
    echo "Не удалось найти URL для скачивания в релизах. Пытаемся получить через теги..."
    TAGS_JSON=$(curl -s "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/tags")
    LATEST_TAG=$(echo "$TAGS_JSON" | grep -o '"name": "[^"]*' | head -1 | cut -d'"' -f4)
    if [ -n "$LATEST_TAG" ]; then
        VSIX_URL="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$LATEST_TAG/kilo-code.vsix"
    else
        echo "Ошибка: Не удалось найти теги для репозитория."
        exit 1
    fi
fi

# Скачиваем VSIX-файл
echo "Скачиваем расширение из: $VSIX_URL"
curl -L -o "$VSIX_FILE" "$VSIX_URL"

# Проверяем, успешно ли скачан файл
if [ ! -f "$VSIX_FILE" ]; then
    echo "Ошибка: Не удалось скачать файл расширения."
    exit 1
fi

echo "Файл расширения скачан: $VSIX_FILE"

# Устанавливаем расширение в VS Code внутри контейнера
echo "Устанавливаем расширение в VS Code..."
# Предполагаем, что VS Code запущен в контейнере и доступен через код команды
code --install-extension "$VSIX_FILE"

# Если контейнер не запущен, потребуется сначала запустить его с монтированием каталога
# docker run -v "$TARGET_DIR:$TARGET_DIR" -it <vscode_image> code --install-extension "$VSIX_FILE"

echo "Расширение Kilo Code установлено в VS Code."