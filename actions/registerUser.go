package actions

import (
	"main/controllers"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type RegisterUser struct {
	Name string
	Client tgbotapi.BotAPI
}

var GetPVZFunc = func(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	message := tgbotapi.NewMessage(update.Message.Chat.ID, "Введите адрес ближайшего ПВЗ для дальнейшего оформления заказов (CDEK или Яндекс доставка)")
	_, err := client.Send(message)
	if err != nil {
		return err
	}

	stepManager := controllers.GetNextStepManager()

	stepKey := controllers.NextStepKey{
		ChatID: update.Message.Chat.ID,
		UserID: update.Message.From.ID,
	}
	stepAction := controllers.NextStepAction{
		Func: nil,
		Params: make(map[string]any),
		CreatedAtTS: time.Now().Unix(),
	}

	stepManager.RegisterNextStepAction(stepKey, stepAction)

	return nil
}

var RegisterPhoneNumberFunc = func(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	message := tgbotapi.NewMessage(update.Message.Chat.ID, "Введите номер телефона:")
	_, err := client.Send(message)
	if err != nil {
		return err
	}

	stepManager := controllers.GetNextStepManager()

	stepKey := controllers.NextStepKey{
		ChatID: update.Message.Chat.ID,
		UserID: update.Message.From.ID,
	}
	stepAction := controllers.NextStepAction{
		Func: GetPVZFunc,
		Params: make(map[string]any),
		CreatedAtTS: time.Now().Unix(),
	}

	stepManager.RegisterNextStepAction(stepKey, stepAction)

	return nil
}

func (r RegisterUser) Run(update tgbotapi.Update) error {
	stepManager := controllers.GetNextStepManager()

	message := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Введите ФИО")
	_, err := r.Client.Send(message)
	if err != nil {
		return err
	}

	stepKey := controllers.NextStepKey{
		ChatID: update.CallbackQuery.Message.Chat.ID,
		UserID: update.CallbackQuery.From.ID,
	}
	stepAction := controllers.NextStepAction{
		Func: RegisterPhoneNumberFunc,
		Params: make(map[string]any),
		CreatedAtTS: time.Now().Unix(),
	}

	stepManager.RegisterNextStepAction(stepKey, stepAction)

	return nil
}

func (r RegisterUser) GetName() string {
	return r.Name
}
