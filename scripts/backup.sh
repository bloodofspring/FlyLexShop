#!/bin/bash

set -e

# Загрузка переменных окружения
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
else
  echo "Файл .env не найден"
  exit 1
fi

# Создание директории для бэкапов, если её нет
mkdir -p backups

# Формирование имени файла бэкапа
BACKUP_FILE="backups/backup_$(date +%Y%m%d_%H%M%S).sql"

# Создание бэкапа
echo "Создание бэкапа базы данных..."
docker exec fly-lex-shop-db pg_dump -U "$DB_USER" -d "$DB_NAME" > "$BACKUP_FILE"

# Проверка успешности создания бэкапа
if [ $? -eq 0 ]; then
  echo "Бэкап успешно создан: $BACKUP_FILE"
  
  # Удаление старых бэкапов (оставляем последние 7)
  echo "Удаление старых бэкапов..."
  ls -t backups/backup_*.sql | tail -n +8 | xargs -r rm
  
  # Сжатие бэкапа
  echo "Сжатие бэкапа..."
  gzip "$BACKUP_FILE"
else
  echo "Ошибка при создании бэкапа"
  exit 1
fi 