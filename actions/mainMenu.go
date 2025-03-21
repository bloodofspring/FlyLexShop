package actions

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"


type MainMenu struct {
	Name   string
	Client tgbotapi.BotAPI
}

func (m MainMenu) Run(update tgbotapi.Update) error {
	const text = "<b>Главное меню</b>\nВыберите опцию:"

	settingsCallbackData := "profileSettings"
	shopCallbackData := "shop"
	aboutCallbackData := "about"

	keyboard := tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{{Text: "Настройки", CallbackData: &settingsCallbackData}},
			{{Text: "Магазин", CallbackData: &shopCallbackData}},
			{{Text: "О нас", CallbackData: &aboutCallbackData}},
		},
	}

	if update.CallbackQuery != nil {
		message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, text)
		message.ParseMode = "HTML"

		message.ReplyMarkup = &keyboard

		_, err := m.Client.Send(message)

		return err
	}

	message := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	message.ParseMode = "HTML"

	message.ReplyMarkup = keyboard

	_, err := m.Client.Send(message)

	return err
}

func (m MainMenu) GetName() string {
	return m.Name
}
