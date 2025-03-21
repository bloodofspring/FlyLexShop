package actions

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type Shop struct {
	Name   string
	Client tgbotapi.BotAPI
}

func (s Shop) Run(update tgbotapi.Update) error {
	return nil
}

func (s Shop) GetName() string {
	return s.Name
}