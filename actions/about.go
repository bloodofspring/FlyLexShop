package actions

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type About struct {
	Name   string
	Client tgbotapi.BotAPI
}

func (a About) Run(update tgbotapi.Update) error {
	return nil
}

func (a About) GetName() string {
	return a.Name
}
