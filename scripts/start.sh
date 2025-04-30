#!/bin/bash

# Проверяем наличие Homebrew
if ! command -v brew &> /dev/null; then
    echo "Homebrew не установлен. Устанавливаем..."
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >> ~/.zshrc
    eval "$(/opt/homebrew/bin/brew shellenv)"
fi

# Проверяем наличие Go и его версию
if ! command -v go &> /dev/null; then
    echo "Go не установлен. Устанавливаем Go 1.23..."
    brew install go@1.23
    echo 'export PATH="/opt/homebrew/opt/go@1.23/bin:$PATH"' >> ~/.zshrc
    source ~/.zshrc
else
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    REQUIRED_VERSION="1.23"
    
    if [[ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]]; then
        echo "Требуется Go версии $REQUIRED_VERSION или выше. Устанавливаем..."
        brew install go@1.23
        echo 'export PATH="/opt/homebrew/opt/go@1.23/bin:$PATH"' >> ~/.zshrc
        source ~/.zshrc
    fi
fi

# Даем права на выполнение всем скриптам
chmod +x scripts/*.sh

# Запускаем скрипт настройки cron
./scripts/setup-cron.sh

# Запускаем контейнеры
docker-compose up -d
