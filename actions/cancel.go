package actions

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Cancel struct {
	Name string
	Client tgbotapi.BotAPI
}

func (c Cancel) Run(update tgbotapi.Update) error {
	ClearNextStepForUser(update, &c.Client, false)
	c.Client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))

	_, err := c.Client.Request(tgbotapi.CallbackConfig{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            "Действие отменено",
		ShowAlert:       false,
	})
	if err != nil {
		return err
	}

	return nil
}

func (c Cancel) GetName() string {
	return c.Name
}
