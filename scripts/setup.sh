#!/bin/bash

set -e

if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
else
  echo "Файл .env не найден. Создайте файл .env с необходимыми переменными."
  exit 1
fi

required_vars=("DB_HOST" "DB_PORT" "DB_USER" "DB_PASSWORD" "DB_NAME")
for var in "${required_vars[@]}"; do
  if [ -z "${!var}" ]; then
    echo "Ошибка: Переменная $var не определена в .env файле"
    exit 1
  fi
done

# Инициализация PostgreSQL, если это первый запуск
if [ ! -f "/var/lib/postgresql/data/PG_VERSION" ]; then
  echo "Инициализация базы данных PostgreSQL..."
  su-exec postgres initdb -D /var/lib/postgresql/data
  
  # Настройка PostgreSQL
  echo "host all all all md5" >> /var/lib/postgresql/data/pg_hba.conf
  echo "listen_addresses = '*'" >> /var/lib/postgresql/data/postgresql.conf
  
  # Запуск PostgreSQL
  su-exec postgres pg_ctl -D /var/lib/postgresql/data start
  
  # Создание пользователя и базы данных
  su-exec postgres psql -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';"
  su-exec postgres createdb -O $DB_USER $DB_NAME
  
  # Создание базы данных FlyLexShopDb
  su-exec postgres createdb -O $DB_USER FlyLexShopDb
  echo "База данных FlyLexShopDb создана успешно."
  
  echo "PostgreSQL инициализирован и настроен."
else
  # Запуск PostgreSQL, если база данных уже существует
  su-exec postgres pg_ctl -D /var/lib/postgresql/data start
  
  # Проверка и создание базы данных FlyLexShopDb, если она не существует
  if ! su-exec postgres psql -lqt | cut -d \| -f 1 | grep -qw FlyLexShopDb; then
    su-exec postgres createdb -O $DB_USER FlyLexShopDb
    echo "База данных FlyLexShopDb создана успешно."
  fi
fi

# Проверка наличия исполняемого файла
if [ ! -f "./fly-lex-shop-bot" ]; then
  echo "Ошибка: Исполняемый файл fly-lex-shop-bot не найден"
  exit 1
fi

# Обработка сигналов завершения
trap 'echo "Получен сигнал завершения. Завершаю работу..."; su-exec postgres pg_ctl -D /var/lib/postgresql/data stop; exit 0' SIGTERM SIGINT

echo "Запуск приложения..."
chmod +x ./fly-lex-shop-bot
./fly-lex-shop-bot
