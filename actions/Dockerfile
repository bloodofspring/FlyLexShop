# Используем официальный образ Go с поддержкой модулей
FROM golang:1.21 AS builder

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /app

# Копируем файлы зависимостей Go (go.mod и go.sum)
COPY go.mod go.sum ./

# Загружаем зависимости Go
RUN go mod download

# Копируем исходный код приложения
COPY . .

# Собираем Go-приложение
RUN go build -o fly-lex-shop-bot .

# Создаем новый образ для запуска приложения (минимальный размер)
FROM alpine:latest

# Устанавливаем bash и необходимые утилиты
RUN apk update && apk add --no-cache bash postgresql-client

# Копируем скомпилированное приложение из предыдущего образа
COPY --from=builder /app/fly-lex-shop-bot /app/fly-lex-shop-bot

# Копируем скрипт для настройки базы данных
COPY scripts/setup.sh /app/setup.sh

# Делаем скрипт исполняемым
RUN chmod +x /app/setup.sh

# Задаем точку входа контейнера (скрипт и затем приложение)
ENTRYPOINT ["/app/setup.sh"]
