package actions

import (
	"main/controllers"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func ClearNextStepForUser(update tgbotapi.Update, client *tgbotapi.BotAPI, sendCancelMessage bool) {
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
	}, *client, sendCancelMessage)
}

func GetMessageAndType(update tgbotapi.Update) (*tgbotapi.Message, string) {
	switch {
	case update.CallbackQuery != nil:
		message := update.CallbackQuery.Message
		message.From = update.CallbackQuery.From
		return message, "CallbackQuery"
	case update.Message != nil:
		return update.Message, "Message"
	case update.EditedMessage != nil:
		return update.EditedMessage, "EditedMessage"
	default:
		return nil, "Unknown"
	}
}

func GetMessage(update tgbotapi.Update) *tgbotapi.Message {
	message, _ := GetMessageAndType(update)
	return message
}

