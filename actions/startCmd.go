package actions

import (
	"main/database"
	"main/database/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SayHi представляет собой структуру для обработки команды /start
// Name - имя команды
// Client - экземпляр Telegram бота
type SayHi struct {
	Name   string
	Client tgbotapi.BotAPI
}

// fabricateAnswer создает ответное сообщение на команду /start
// update - обновление от Telegram API
// Возвращает сконфигурированное сообщение с приветствием и кнопкой регистрации
func (e SayHi) fabricateAnswer(update tgbotapi.Update) tgbotapi.MessageConfig {
	ClearNextStepForUser(update, &e.Client, true)
	const text = "Добрый день! Вы попали в бота компании FlyLex! Здесь вы можете приобрести нашу продукцию.\nНажмите кнопку «Регистрация» чтобы продолжить"
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)

	callbackData := "registerUser"
	msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{{Text: "Регистрация", CallbackData: &callbackData}},
		},
	}

	db := database.Connect()
	defer db.Close()

	user := models.TelegramUser{ID: update.Message.From.ID}
	_ = user.GetOrCreate(update.Message.From, *db)

	return msg
}

// Run выполняет обработку команды /start
// update - обновление от Telegram API
// Возвращает ошибку, если отправка сообщения не удалась
func (e SayHi) Run(update tgbotapi.Update) error {
	if _, err := e.Client.Send(e.fabricateAnswer(update)); err != nil {
		return err
	}

	return nil
}

// GetName возвращает имя команды
func (e SayHi) GetName() string {
	return e.Name
}
