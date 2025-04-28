#!/bin/bash

set -e

echo "Остановка текущих контейнеров..."
docker-compose stop

echo "Обновление образов..."
docker-compose build

echo "Запуск обновленных контейнеров..."
docker-compose up -d

echo "Обновление завершено" 