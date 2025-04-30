#!/bin/bash

set -e

echo "Остановка текущих контейнеров..."
bash ./scripts/stop.sh

echo "Обновление образов..."
docker compose build

echo "Запуск обновленных контейнеров..."
bash ./scripts/start.sh

echo "Обновление завершено" 