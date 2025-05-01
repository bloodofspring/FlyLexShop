#!/bin/bash

# Получаем абсолютный путь к директории проекта
PROJECT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

# Удаляем все существующие задачи crontab
echo "Удаление существующих заданий crontab..."
crontab -r 2>/dev/null || true


# Создаем запись в crontab для бэкапа каждые 30 дней в 3:00
(crontab -l 2>/dev/null; echo "0 2 * * * cd $PROJECT_DIR && echo \"\$(date): Starting backup\" >> $PROJECT_DIR/backups/backup.log && ./scripts/backup.sh >> $PROJECT_DIR/backups/backup.log 2>&1") | crontab -


echo "Задача бэкапа успешно добавлена в crontab"
echo "Бэкапы будут создаваться каждый день в 2:00"
echo "Логи бэкапов будут сохраняться в $PROJECT_DIR/backups/backup.log" 