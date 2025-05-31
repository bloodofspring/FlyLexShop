package actions

import (
	"main/controllers"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ClearNextStepForUser очищает следующий шаг для пользователя
// update - обновление от Telegram API
// client - экземпляр Telegram бота
// sendCancelMessage - флаг, указывающий, нужно ли отправлять сообщение об отмене
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

// GetMessageAndType возвращает сообщение и его тип из обновления
// update - обновление от Telegram API
// Возвращает сообщение и его тип
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

// GetMessage возвращает сообщение из обновления
// update - обновление от Telegram API
// Возвращает сообщение
func GetMessage(update tgbotapi.Update) *tgbotapi.Message {
	message, _ := GetMessageAndType(update)
	return message
}

func ParseCallData(s string) map[string]string {
	res := make(map[string]string, 0)
	if len(strings.Split(s, "?")) != 2 {
		return res
	}
	params := strings.Trim(strings.Split(s, "?")[1], " ")

	for _, p := range strings.Split(params, "&") {
		if len(strings.Split(p, "=")) != 2 {
			continue
		}
		key := strings.Split(p, "=")[0]
		value := strings.Split(p, "=")[1]

		res[strings.Trim(key, " ")] = strings.Trim(value, " ")
	}

	return res
}