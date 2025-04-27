package actions

import (
	"main/database"
	"main/database/models"
	"main/filters"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	// paymentAcceptedMessageText - сообщение об успешном принятии оплаты
	paymentAcceptedMessageText = "Администрация приняла ваш чек! Ожидайте доставку в указанный пункт выдачи."
	// paymentRejectedMessageText - сообщение об отклонении оплаты
	paymentRejectedMessageText = "Администрация отклонила ваш чек! Попробуйте ещё раз."
)

// PaymentVerdict представляет собой структуру для обработки результатов проверки оплаты
// Name - имя команды
// Client - экземпляр Telegram бота
type PaymentVerdict struct {
	Name   string
	Client tgbotapi.BotAPI
}

// Run обрабатывает результат проверки оплаты администратором
// update - обновление от Telegram API
// Возвращает ошибку, если что-то пошло не так
func (p PaymentVerdict) Run(update tgbotapi.Update) error {
	ClearNextStepForUser(update, &p.Client, true)

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

		_, err = p.Client.Send(tgbotapi.NewEditMessageCaption(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, update.CallbackQuery.Message.Caption+"\n\nОплата принята✅"))
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

	_, err = p.Client.Send(tgbotapi.NewEditMessageCaption(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, update.CallbackQuery.Message.Caption+"\n\nОплата отклонена❌"))
	if err != nil {
		return err
	}

	return nil
}

// GetName возвращает имя команды
func (p PaymentVerdict) GetName() string {
	return p.Name
}
