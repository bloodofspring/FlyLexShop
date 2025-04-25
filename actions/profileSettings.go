package actions

import (
	"fmt"
	"main/controllers"
	"main/database"
	"main/database/models"
	"regexp"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ProfileSettings struct {
	Name   string
	Client tgbotapi.BotAPI
}

func (p ProfileSettings) Run(update tgbotapi.Update) error {
	const text = "<b>Настройки профиля</b>\nВыберите опцию:"

	message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, text)
	message.ParseMode = "HTML"

	changeNameCallbackData := "changeName"
	changePhoneCallbackData := "changePhone"
	changeDeliveryAddressCallbackData := "changeDeliveryAddress"
	changeDeliveryServiceCallbackData := "changeDeliveryService"
	toMainMenuCallbackData := "mainMenu"

	message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{{Text: "Изменить ФИО", CallbackData: &changeNameCallbackData}},
			{{Text: "Изменить номер телефона", CallbackData: &changePhoneCallbackData}},
			{{Text: "Добавить/изменить адрес доставки", CallbackData: &changeDeliveryAddressCallbackData}},
			{{Text: "Изменить сервис доставки", CallbackData: &changeDeliveryServiceCallbackData}},
			{{Text: "На главную", CallbackData: &toMainMenuCallbackData}},
		},
	}

	_, err := p.Client.Send(message)

	return err
}

func (p ProfileSettings) GetName() string {
	return p.Name
}

type ChangeName struct {
	Name   string
	Client tgbotapi.BotAPI
}

func (c ChangeName) Run(update tgbotapi.Update) error {
	c.Client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))
	const text = "<b>Ваше ФИО сейчас: %s</b>\nВведите новое ФИО:"

	message := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "")
	message.ParseMode = "HTML"

	db := database.Connect()
	defer db.Close()

	user := models.TelegramUser{ID: update.CallbackQuery.Message.Chat.ID}
	err := user.GetOrCreate(update.CallbackQuery.Message.From, *db)
	if err != nil {
		return err
	}

	message.Text = fmt.Sprintf(text, user.FIO)

	toSettingsCallbackData := "profileSettings"
	message.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{{Text: "Отмена", CallbackData: &toSettingsCallbackData}},
		},
	}

	_, err = c.Client.Send(message)

	if err != nil {
		return err
	}

	stepManager := controllers.GetNextStepManager()
	stepKey := controllers.NextStepKey{
		ChatID: update.CallbackQuery.Message.Chat.ID,
		UserID: update.CallbackQuery.From.ID,
	}
	stepAction := controllers.NextStepAction{
		Func: func(client tgbotapi.BotAPI, stepUpdate tgbotapi.Update, stepParams map[string]any) error {
			db := database.Connect()
			defer db.Close()

			user := models.TelegramUser{ID: stepUpdate.Message.From.ID}
			err := user.GetOrCreate(stepUpdate.Message.From, *db)
			if err != nil {
				return err
			}

			user.FIO = stepUpdate.Message.Text

			_, err = db.Model(&user).WherePK().Update()
			if err != nil {
				return err
			}

			message := tgbotapi.NewMessage(stepUpdate.Message.Chat.ID, "<b>ФИО успешно изменено</b>")
			message.ParseMode = "HTML"

			toSettingsCallbackData := "profileSettings"
			toMainMenuCallbackData := "mainMenu"

			message.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					{{Text: "К настройкам", CallbackData: &toSettingsCallbackData}},
					{{Text: "На главную", CallbackData: &toMainMenuCallbackData}},
				},
			}

			_, err = client.Send(message)

			return err
		},
		Params: map[string]any{},
		CreatedAtTS: time.Now().Unix(),
	}
	stepManager.RegisterNextStepAction(stepKey, stepAction)

	return nil
}

func (c ChangeName) GetName() string {
	return c.Name
}

type ChangePhone struct {
	Name   string
	Client tgbotapi.BotAPI
}

func (c ChangePhone) Run(update tgbotapi.Update) error {
	c.Client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))
	const text = "<b>Ваш номер телефона сейчас: %v(%v)%v-%v</b>\nВведите новый номер телефона:"

	message := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "")
	message.ParseMode = "HTML"

	db := database.Connect()
	defer db.Close()

	user := models.TelegramUser{ID: update.CallbackQuery.Message.Chat.ID}
	err := user.GetOrCreate(update.CallbackQuery.From, *db)
	if err != nil {
		return err
	}

	message.Text = fmt.Sprintf(text, user.Phone[0:1], user.Phone[1:4], user.Phone[4:7], user.Phone[7:])

	toSettingsCallbackData := "profileSettings"
	message.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{{Text: "Отмена", CallbackData: &toSettingsCallbackData}},
		},
	}

	_, err = c.Client.Send(message)

	if err != nil {
		return err
	}

	stepManager := controllers.GetNextStepManager()
	stepKey := controllers.NextStepKey{
		ChatID: update.CallbackQuery.Message.Chat.ID,
		UserID: update.CallbackQuery.From.ID,
	}
	stepAction := controllers.NextStepAction{
		Func: func(client tgbotapi.BotAPI, stepUpdate tgbotapi.Update, stepParams map[string]any) error {
			db := database.Connect()
			defer db.Close()

			regex := regexp.MustCompile(`^[0-9]{11}$`)
			if !regex.MatchString(stepUpdate.Message.Text) {
				message := tgbotapi.NewMessage(stepUpdate.Message.Chat.ID, "Введите номер телефона в формате 89991234567")

				tryAgainCallbackData := "changePhone"
				message.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
					InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
						{{Text: "Попробовать снова", CallbackData: &tryAgainCallbackData}},
						{{Text: "Отмена", CallbackData: &toSettingsCallbackData}},
					},
				}

				_, err := client.Send(message)
				
				return err
			}

			user := models.TelegramUser{ID: stepUpdate.Message.From.ID}
			err := user.GetOrCreate(stepUpdate.Message.From, *db)
			if err != nil {
				return err
			}

			user.Phone = stepUpdate.Message.Text

			_, err = db.Model(&user).WherePK().Update()
			if err != nil {
				return err
			}

			message := tgbotapi.NewMessage(stepUpdate.Message.Chat.ID, "<b>Номер телефона успешно изменен</b>")
			message.ParseMode = "HTML"

			toSettingsCallbackData := "profileSettings"
			toMainMenuCallbackData := "mainMenu"

			message.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					{{Text: "К настройкам", CallbackData: &toSettingsCallbackData}},
					{{Text: "На главную", CallbackData: &toMainMenuCallbackData}},
				},
			}

			_, err = client.Send(message)

			return err
		},
		Params: map[string]any{},
		CreatedAtTS: time.Now().Unix(),
	}
	stepManager.RegisterNextStepAction(stepKey, stepAction)

	return nil
}

func (c ChangePhone) GetName() string {
	return c.Name
}

type ChangeDeliveryAddress struct {
	Name   string
	Client tgbotapi.BotAPI
}

func (c ChangeDeliveryAddress) Run(update tgbotapi.Update) error {
	c.Client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))
	const text = "<b>Ваш адрес доставки сейчас: %s</b>\nВведите новый адрес доставки:"

	message := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "")
	message.ParseMode = "HTML"

	db := database.Connect()
	defer db.Close()

	user := models.TelegramUser{ID: update.CallbackQuery.Message.Chat.ID}
	err := user.GetOrCreate(update.CallbackQuery.From, *db)
	if err != nil {
		return err
	}

	message.Text = fmt.Sprintf(text, user.DeliveryAddress)

	toSettingsCallbackData := "profileSettings"
	message.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{{Text: "Отмена", CallbackData: &toSettingsCallbackData}},
		},
	}

	_, err = c.Client.Send(message)
	
	if err != nil {
		return err
	}

	stepManager := controllers.GetNextStepManager()
	stepKey := controllers.NextStepKey{
		ChatID: update.CallbackQuery.Message.Chat.ID,
		UserID: update.CallbackQuery.From.ID,
	}
	stepAction := controllers.NextStepAction{
		Func: func(client tgbotapi.BotAPI, stepUpdate tgbotapi.Update, stepParams map[string]any) error {
			db := database.Connect()
			defer db.Close()

			user := models.TelegramUser{ID: stepUpdate.Message.From.ID}
			err := user.GetOrCreate(stepUpdate.Message.From, *db)
			if err != nil {
				return err
			}

			user.DeliveryAddress = stepUpdate.Message.Text

			_, err = db.Model(&user).WherePK().Update()
			if err != nil {
				return err
			}
			
			message := tgbotapi.NewMessage(stepUpdate.Message.Chat.ID, "<b>Адрес доставки успешно изменен</b>")
			message.ParseMode = "HTML"

			toSettingsCallbackData := "profileSettings"
			toMainMenuCallbackData := "mainMenu"
			
			message.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					{{Text: "К настройкам", CallbackData: &toSettingsCallbackData}},
					{{Text: "На главную", CallbackData: &toMainMenuCallbackData}},
				},
			}
			_, err = client.Send(message)

			return err
		},
		Params: map[string]any{},
		CreatedAtTS: time.Now().Unix(),
	}
	stepManager.RegisterNextStepAction(stepKey, stepAction)

	return nil
}

func (c ChangeDeliveryAddress) GetName() string {
	return c.Name
}

type ChangeDeliveryService struct {
	Name   string
	Client tgbotapi.BotAPI
}

func (c ChangeDeliveryService) GetKeyboard(userDb models.TelegramUser) [][]tgbotapi.InlineKeyboardButton {
	type buttonConfig struct {
		Text string
		Setting string
	}

	createButton := func(cfg buttonConfig) tgbotapi.InlineKeyboardButton {
		if cfg.Setting == userDb.DeliveryService {
			cfg.Text += " ✅"
		}

		callQuery := fmt.Sprintf("changeDeliveryService?service=%s", cfg.Setting)

		return tgbotapi.InlineKeyboardButton{
			Text: cfg.Text,
			CallbackData: &callQuery,
		}
	}

	buttons := []buttonConfig{
		{Text: "CDEK", Setting: "cdek"},
		{Text: "Яндекс доставка", Setting: "yandex"},
	}

	keyboard := [][]tgbotapi.InlineKeyboardButton{}

	for _, b := range buttons {
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{createButton(b)})
	}

	return keyboard
}

func (c ChangeDeliveryService) Run(update tgbotapi.Update) error {
	ClearNextStepForUser(update, &c.Client)

	const text = "<b>Ваш сервис доставки сейчас: %s</b>\nВыберите новый сервис доставки:"

	message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, "")
	message.ParseMode = "HTML"

	db := database.Connect()
	defer db.Close()

	user := models.TelegramUser{ID: update.CallbackQuery.From.ID}
	err := user.GetOrCreate(update.CallbackQuery.From, *db)
	if err != nil {
		return err
	}

	message.Text = fmt.Sprintf(text, user.DeliveryService)
	
	toSettingsCallbackData := "profileSettings"
	message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: append(c.GetKeyboard(user), []tgbotapi.InlineKeyboardButton{{Text: "К настройкам", CallbackData: &toSettingsCallbackData}}),
	}

	_, err = c.Client.Send(message)

	return err	
}

func (c ChangeDeliveryService) GetName() string {
	return c.Name
}
