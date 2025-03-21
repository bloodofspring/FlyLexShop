package filters

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var RegisterUserFilter = func(update tgbotapi.Update) bool {
	return update.CallbackQuery.Data == "registerUser"
}
