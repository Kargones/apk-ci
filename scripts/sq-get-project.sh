#!/bin/bash

# === Настройки ===
SONAR_HOST="http://sq.apkholding.ru:9000"   # адрес сервера SonarQube
SONAR_TOKEN="squ_5c6ae0f64debadf3b2fe58e93cff01c4a8615d92"               # токен авторизации (создаётся в профиле SonarQube)
PROJECT_KEY="test_SCUD_main"                    # ключ проекта

# === Запрос ===
response=$(curl -s -u "${SONAR_TOKEN}:" \
  "${SONAR_HOST}/api/projects/search?projects=${PROJECT_KEY}")

# === Вывод ответа ===
echo "Информация о проекте с ключом '${PROJECT_KEY}':"
echo "$response" | jq .
