#!/bin/bash

# Получаем абсолютный путь к директории проекта
PROJECT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

# Переходим в директорию проекта
cd "$PROJECT_DIR"

# Получаем последние изменения
git pull origin main

# Перезапускаем контейнеры с новым кодом
docker-compose down
docker-compose up -d --build

echo "Код успешно обновлен и контейнеры перезапущены" 