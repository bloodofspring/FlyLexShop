package actions

import (
	"main/database"
	"main/database/models"
	"main/filters"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	paymentAcceptedMessageText = "Администрация приняла ваш чек! Ожидайте доставку в указанный пункт выдачи."
	paymentRejectedMessageText = "Администрация отклонила ваш чек! Попробуйте ещё раз."
)

type PaymentVerdict struct {
	Name   string
	Client tgbotapi.BotAPI
}

func (p PaymentVerdict) Run(update tgbotapi.Update) error {
	ClearNextStepForUser(update, &p.Client)

	data := filters.ParseCallbackData(update.CallbackQuery.Data)

	userId, err := strconv.ParseInt(data["userId"], 10, 64)
	if err != nil {
		return err
	}

	paymentAccepted := data["ok"]

	if paymentAccepted == "true" {
		message := tgbotapi.NewMessage(userId, paymentAcceptedMessageText)
		_, err := p.Client.Send(message)
		if err != nil {
			return err
		}

		_, err = p.Client.Send(tgbotapi.NewEditMessageCaption(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, update.CallbackQuery.Message.Caption + "\n\nОплата принята✅"))
		if err != nil {
			return err
		}

		db := database.Connect()
		defer db.Close()

		_, err = db.Model(&models.ShoppingCart{}).Where("user_id = ?", userId).Delete()
		if err != nil {
			return err
		}

		return nil
	}

	message := tgbotapi.NewMessage(userId, paymentRejectedMessageText)
	_, err = p.Client.Send(message)
	if err != nil {
		return err
	}

	_, err = p.Client.Send(tgbotapi.NewEditMessageCaption(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, update.CallbackQuery.Message.Caption + "\n\nОплата отклонена❌"))
	if err != nil {
		return err
	}

	return nil
}

func (p PaymentVerdict) GetName() string {
	return p.Name
}
