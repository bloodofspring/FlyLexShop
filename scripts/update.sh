#!/bin/bash

set -e

# Настраиваем Git для работы в CI/CD
git config --global --add safe.directory "$(pwd)"
git config --global core.autocrlf false
git config --global core.fileMode false

git pull origin main
if [ $? -ne 0 ]; then
  echo "Ошибка при выполнении git pull"
  exit 1
fi

echo "Остановка текущих контейнеров..."
bash ./scripts/stop.sh

echo "Обновление образов..."
docker compose build

echo "Запуск обновленных контейнеров..."
bash ./scripts/start.sh

echo "Обновление завершено" 