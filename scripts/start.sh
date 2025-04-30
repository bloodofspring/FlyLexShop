#!/bin/bash
# Даем права на выполнение всем скриптам
chmod +x scripts/*.sh

# Запускаем скрипт настройки cron
./scripts/setup-cron.sh

# Запускаем контейнеры
docker compose up -d
