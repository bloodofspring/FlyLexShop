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

# Проверка наличия исполняемого файла
if [ ! -f "./fly-lex-shop-bot" ]; then
  echo "Ошибка: Исполняемый файл fly-lex-shop-bot не найден"
  exit 1
fi

echo "Запуск приложения..."
chmod +x ./fly-lex-shop-bot
./fly-lex-shop-bot
