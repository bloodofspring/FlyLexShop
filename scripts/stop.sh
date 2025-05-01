#!/bin/bash

echo "Удаление существующих заданий crontab..."
crontab -r 2>/dev/null || true

# Останавливаем контейнеры без их удаления
docker compose stop
