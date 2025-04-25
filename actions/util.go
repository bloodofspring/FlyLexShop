package actions

import (
	"main/controllers"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func ClearNextStepForUser(update tgbotapi.Update, client *tgbotapi.BotAPI) {
	var user *tgbotapi.User
	var chat *tgbotapi.Chat

	switch {
		case update.Message != nil:
			user = update.Message.From
		case update.CallbackQuery != nil:
			user = update.CallbackQuery.From
		default:
			return
	}

	switch {
		case update.Message != nil:
			chat = update.Message.Chat
		case update.CallbackQuery != nil:
			chat = update.CallbackQuery.Message.Chat
	}

	controllers.GetNextStepManager().RemoveNextStepAction(controllers.NextStepKey{
		ChatID: chat.ID,
		UserID: user.ID,
	}, *client)
}
