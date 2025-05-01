#!/bin/bash

# Получаем абсолютный путь к директории проекта
PROJECT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

# Создаем запись в crontab для бэкапа каждые 30 дней в 3:00
(crontab -l 2>/dev/null; echo "0 3 */30 * * cd $PROJECT_DIR && ./scripts/backup.sh >> $PROJECT_DIR/backups/backup.log 2>&1") | crontab -

echo "Задача бэкапа успешно добавлена в crontab"
echo "Бэкапы будут создаваться каждые 30 дней в 3:00"
echo "Логи бэкапов будут сохраняться в $PROJECT_DIR/backups/backup.log" 