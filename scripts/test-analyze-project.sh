#!/bin/bash

# Тестовый скрипт для проверки команды analyze-project

echo "Тестирование команды analyze-project..."

# Устанавливаем переменные окружения
export BR_COMMAND="analyze-project"
export BR_REPO_URL="https://github.com/owner/repo"
export BR_ACCESS_TOKEN="your_token_here"

# Запускаем команду
go run cmd/github.com/Kargones/apk-ci/main.go

echo "Тест завершен."