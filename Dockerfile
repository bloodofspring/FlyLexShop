FROM alpine:latest

# Установка необходимых пакетов
RUN apk add --no-cache postgresql postgresql-client su-exec bash
# Установка Go 1.23
RUN apk add --no-cache 'go>=1.23.0' 'go<1.24.0'
RUN apk add --no-cache git


# Создание необходимых директорий для PostgreSQL
RUN mkdir -p /var/lib/postgresql/data && \
    mkdir -p /run/postgresql && \
    chown -R postgres:postgres /var/lib/postgresql && \
    chown -R postgres:postgres /run/postgresql && \
    chmod -R 755 /var/lib/postgresql && \
    chmod -R 755 /run/postgresql

# Копирование исходного кода
COPY . /src
WORKDIR /src

# Сборка приложения
RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -o fly-lex-shop-bot

# Создание директории /app
RUN mkdir -p /app

# Перемещение исполняемого файла в /app
RUN mv /src/fly-lex-shop-bot /app/

# Копирование скриптов и конфигурации
COPY /scripts/setup.sh /app/setup.sh
COPY /scripts/backup.sh /app/backup.sh
COPY /.env /app/.env

# Установка рабочей директории
WORKDIR /app

# Сделаем скрипт исполняемым
RUN chmod +x setup.sh

# Запуск скрипта при старте контейнера
CMD ["/app/setup.sh"]
