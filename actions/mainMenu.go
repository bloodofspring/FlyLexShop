package actions

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"


type MainMenu struct {
	Name   string
	Client tgbotapi.BotAPI
}

func (m MainMenu) Run(update tgbotapi.Update) error {
	message := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "<b>Главное меню</b>/n Выберите опцию:")
	message.ParseMode = "HTML"


	settingsCallbackData := "profileSettings"
	shopCallbackData := "shop"
	aboutCallbackData := "about"

	message.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{{Text: "Настройки", CallbackData: &settingsCallbackData}},
			{{Text: "Магазин", CallbackData: &shopCallbackData}},
			{{Text: "О нас", CallbackData: &aboutCallbackData}},
		},
	}

	_, err := m.Client.Send(message)

	return err
}

func (m MainMenu) GetName() string {
	return m.Name
}
