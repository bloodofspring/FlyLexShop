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
  su - postgres -c "initdb -D /var/lib/postgresql/data"
  
  # Настройка PostgreSQL
  echo "host all all all md5" >> /var/lib/postgresql/data/pg_hba.conf
  echo "listen_addresses = '*'" >> /var/lib/postgresql/data/postgresql.conf
  
  # Запуск PostgreSQL
  su - postgres -c "pg_ctl -D /var/lib/postgresql/data start"
  
  # Создание пользователя и базы данных
  su - postgres -c "psql -c \"CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';\""
  su - postgres -c "createdb -O $DB_USER $DB_NAME"
  
  echo "PostgreSQL инициализирован и настроен."
else
  # Запуск PostgreSQL, если база данных уже существует
  su - postgres -c "pg_ctl -D /var/lib/postgresql/data start"
fi

# Проверка наличия исполняемого файла
if [ ! -f "./fly-lex-shop-bot" ]; then
  echo "Ошибка: Исполняемый файл fly-lex-shop-bot не найден в текущей директории"
  exit 1
fi

# Обработка сигналов завершения
trap 'echo "Получен сигнал завершения. Завершаю работу..."; su - postgres -c "pg_ctl -D /var/lib/postgresql/data stop"; exit 0' SIGTERM SIGINT

echo "Запуск приложения..."
./fly-lex-shop-bot
