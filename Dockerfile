# Используем официальный образ Go
FROM golang:1.21-alpine AS builder

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
RUN apk add --no-cache postgresql postgresql-client

# Создание директории для данных PostgreSQL
RUN mkdir -p /var/lib/postgresql/data && \
    chown -R postgres:postgres /var/lib/postgresql

# Копирование бинарного файла из builder
COPY --from=builder /app/fly-lex-shop-bot /app/fly-lex-shop-bot
COPY --from=builder /app/scripts/setup.sh /app/setup.sh

# Установка рабочей директории
WORKDIR /app

# Сделаем скрипт исполняемым
RUN chmod +x setup.sh

# Запуск скрипта при старте контейнера
CMD ["./setup.sh"] 