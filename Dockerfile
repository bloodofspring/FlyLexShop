# Используем официальный образ Go
FROM golang:1.23-alpine AS builder

# Установка необходимых инструментов для сборки
RUN apk add --no-cache git

# Установка рабочей директории
WORKDIR /app

# Копирование файлов проекта
COPY . .

# Сборка приложения
RUN go build -o fly-lex-shop-bot

# Финальный образ
FROM alpine:latest

# Установка необходимых пакетов
RUN apk add --no-cache postgresql postgresql-client su-exec bash

# Создание необходимых директорий для PostgreSQL
RUN mkdir -p /var/lib/postgresql/data && \
    mkdir -p /run/postgresql && \
    chown -R postgres:postgres /var/lib/postgresql && \
    chown -R postgres:postgres /run/postgresql

# Копирование бинарного файла из builder
COPY --from=builder /app/fly-lex-shop-bot /app/fly-lex-shop-bot
COPY --from=builder /app/scripts/setup.sh /app/setup.sh
COPY --from=builder /app/.env /app/.env

# Установка рабочей директории
WORKDIR /app

# Сделаем скрипт исполняемым
RUN chmod +x setup.sh

# Запуск скрипта при старте контейнера
CMD ["/app/setup.sh"] 