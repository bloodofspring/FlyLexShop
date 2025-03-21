package actions

import (
	"main/database"
	"main/database/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type SayHi struct {
	Name   string
	Client tgbotapi.BotAPI
}

func (e SayHi) fabricateAnswer(update tgbotapi.Update) tgbotapi.MessageConfig {
	const text = "Добрый день! Вы попали в бота компании FlyLex! Здесь вы можете приобрести нашу продукцию. \n Нажмите кнопку «Регистрация» чтобы продолжить"
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)

	db := database.Connect()
	defer db.Close()

	user := models.TelegramUser{ID: update.Message.From.ID}
	_ = user.GetOrCreate(update.Message.From, *db)

	return msg
}

func (e SayHi) Run(update tgbotapi.Update) error {
	if _, err := e.Client.Send(e.fabricateAnswer(update)); err != nil {
		return err
	}

	return nil
}

func (e SayHi) GetName() string {
	return e.Name
}
