package actions

import (
	"main/controllers"
	"main/database"
	"main/database/models"
	"regexp"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type RegisterUser struct {
	Name   string
	Client tgbotapi.BotAPI
}

func RegistrationCompleted(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	message := tgbotapi.NewMessage(update.Message.Chat.ID, "Вы успешно зарегистрированы! Нажмите «Главное меню» чтобы продолжить.")

	callbackData := "mainMenu"
	message.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{{Text: "Главное меню", CallbackData: &callbackData}},
		},
	}

	_, err := client.Send(message)
	if err != nil {
		return err
	}

	return nil
}

func GetPVZFunc(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	regex := regexp.MustCompile(`^[0-9]{11}$`)
	if !regex.MatchString(update.Message.Text) {
		message := tgbotapi.NewMessage(update.Message.Chat.ID, "Введите номер телефона в формате 89991234567")
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
			Func:        RegisterPhoneNumberFunc,
			Params:      make(map[string]any),
			CreatedAtTS: time.Now().Unix(),
		}

		stepManager.RegisterNextStepAction(stepKey, stepAction)

		return nil
	}

	db := database.Connect()
	defer db.Close()

	message := tgbotapi.NewMessage(update.Message.Chat.ID, "Введите адрес ближайшего ПВЗ для дальнейшего оформления заказов (CDEK или Яндекс доставка)")
	_, err := client.Send(message)
	if err != nil {
		return err
	}

	user := models.TelegramUser{ID: update.Message.From.ID}
	err = user.GetOrCreate(update.Message.From, *db)
	if err != nil {
		return err
	}

	user.Phone = update.Message.Text
	_, err = db.Model(&user).WherePK().Update()
	if err != nil {
		return err
	}

	stepManager := controllers.GetNextStepManager()

	stepKey := controllers.NextStepKey{
		ChatID: update.Message.Chat.ID,
		UserID: update.Message.From.ID,
	}
	stepAction := controllers.NextStepAction{
		Func:        RegistrationCompleted,
		Params:      make(map[string]any),
		CreatedAtTS: time.Now().Unix(),
	}

	stepManager.RegisterNextStepAction(stepKey, stepAction)

	return nil
}

func RegisterPhoneNumberFunc(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	db := database.Connect()
	defer db.Close()

	message := tgbotapi.NewMessage(update.Message.Chat.ID, "Введите номер телефона:")
	_, err := client.Send(message)
	if err != nil {
		return err
	}

	user := models.TelegramUser{ID: update.Message.From.ID}
	err = user.GetOrCreate(update.Message.From, *db)
	if err != nil {
		return err
	}

	user.FIO = update.Message.Text
	_, err = db.Model(&user).WherePK().Update()
	if err != nil {
		return err
	}

	stepManager := controllers.GetNextStepManager()

	stepKey := controllers.NextStepKey{
		ChatID: update.Message.Chat.ID,
		UserID: update.Message.From.ID,
	}
	stepAction := controllers.NextStepAction{
		Func:        GetPVZFunc,
		Params:      make(map[string]any),
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
		Func:        RegisterPhoneNumberFunc,
		Params:      make(map[string]any),
		CreatedAtTS: time.Now().Unix(),
	}

	stepManager.RegisterNextStepAction(stepKey, stepAction)

	return nil
}

func (r RegisterUser) GetName() string {
	return r.Name
}
