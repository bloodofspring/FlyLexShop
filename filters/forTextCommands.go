package filters

import (
	"main/database"
	"main/database/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var StartFilter = func(update tgbotapi.Update, _ tgbotapi.BotAPI) bool {
	db := database.Connect()
	defer db.Close()

	user := models.TelegramUser{ID: update.Message.From.ID}
	_ = user.GetOrCreate(update.Message.From, *db)

	return update.Message.Command() == "start" && !user.IsAuthorized
}

var ToMainMenuFilter = func(update tgbotapi.Update, _ tgbotapi.BotAPI) bool {
	db := database.Connect()
	defer db.Close()

	user := models.TelegramUser{ID: update.Message.From.ID}
	_ = user.GetOrCreate(update.Message.From, *db)

	return update.Message.Command() == "start" && user.IsAuthorized
}
