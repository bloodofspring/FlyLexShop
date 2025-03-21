package filters

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

var StartFilter = func(update tgbotapi.Update) bool {
	return update.Message.Command() == "start"
}
