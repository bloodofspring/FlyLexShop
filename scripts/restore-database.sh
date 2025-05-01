#!/bin/bash

set -e

# Загрузка переменных окружения
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
else
  echo "Файл .env не найден"
  exit 1
fi

# Определение пути к Docker
DOCKER_CMD="/usr/local/bin/docker"

# Проверка наличия бэкапов
if [ -z "$(ls -A backups/*.sql 2>/dev/null)" ]; then
    echo "Бэкапы не найдены в директории backups/"
    exit 1
fi

# Получение последнего бэкапа
LATEST_BACKUP=$(ls -t backups/*.sql | head -n1)

echo "Восстановление базы данных из бэкапа: $LATEST_BACKUP"

# Восстановление базы данных из бэкапа
$DOCKER_CMD exec -i fly-lex-shop-db psql -U $DB_USER -p $DB_PORT -d $DB_NAME < "$LATEST_BACKUP"

if [ $? -eq 0 ]; then
    echo "База данных успешно восстановлена из бэкапа"
else
    echo "Ошибка при восстановлении базы данных"
    exit 1
fi
