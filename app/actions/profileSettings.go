package actions

import (
	"fmt"
	"main/controllers"
	"main/database"
	"main/database/models"
	"regexp"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ProfileSettings представляет собой структуру для управления настройками профиля пользователя.
// Name - имя команды.
// Client - экземпляр Telegram бота.
type ProfileSettings struct {
	Name   string
	Client tgbotapi.BotAPI
}

// Run отображает меню настроек профиля.
// update - обновление от Telegram API.
// Возвращает ошибку, если отправка сообщения не удалась.
func (p ProfileSettings) Run(update tgbotapi.Update) error {
	const text = "<b>Настройки профиля</b>\nВыберите опцию:"

	message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, text)
	message.ParseMode = "HTML"

	data := ParseCallData(update.CallbackQuery.Data)
	showBackButton := data["showBackButton"] == "true"

	changeNameCallbackData := "changeName?showBackButton=" + strconv.FormatBool(showBackButton)
	changePhoneCallbackData := "changePhone?showBackButton=" + strconv.FormatBool(showBackButton)
	changeDeliveryAddressCallbackData := "changeDeliveryAddress?showBackButton=" + strconv.FormatBool(showBackButton)
	changeDeliveryServiceCallbackData := "changeDeliveryService?showBackButton=" + strconv.FormatBool(showBackButton)
	toMainMenuCallbackData := "mainMenu"
	processOrderCallbackData := "makeOrder"

	keyboard := [][]tgbotapi.InlineKeyboardButton{
		{{Text: "Изменить ФИО", CallbackData: &changeNameCallbackData}},
		{{Text: "Изменить номер телефона", CallbackData: &changePhoneCallbackData}},
		{{Text: "Добавить/изменить адрес доставки", CallbackData: &changeDeliveryAddressCallbackData}},
		{{Text: "Изменить сервис доставки", CallbackData: &changeDeliveryServiceCallbackData}},
	}

	if !showBackButton {
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{{Text: "На главную", CallbackData: &toMainMenuCallbackData}})
	} else {
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{{Text: "К оформлению заказа", CallbackData: &processOrderCallbackData}})
	}

	message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}

	_, err := p.Client.Send(message)

	return err
}

// GetName возвращает имя команды ProfileSettings.
func (p ProfileSettings) GetName() string {
	return p.Name
}

// ChangeName представляет собой структуру для изменения ФИО пользователя.
// Name - имя команды.
// Client - экземпляр Telegram бота.
type ChangeName struct {
	Name   string
	Client tgbotapi.BotAPI
}

// Run инициирует процесс изменения ФИО.
// update - обновление от Telegram API.
// Возвращает ошибку, если отправка сообщения не удалась.
func (c ChangeName) Run(update tgbotapi.Update) error {
	c.Client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))
	const text = "Ваше ФИО сейчас:\n<b>%s</b>\n\n<i>Введите новое ФИО:</i>"

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

	data := ParseCallData(update.CallbackQuery.Data)
	showBackButton := data["showBackButton"] == "true"

	toSettingsCallbackData := "profileSettings?showBackButton=" + strconv.FormatBool(showBackButton)
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

			_, err = db.Model(&user).WherePK().Column("fio").Update()
			if err != nil {
				return err
			}

			message := tgbotapi.NewMessage(stepUpdate.Message.Chat.ID, "<b>ФИО успешно изменено</b>✅")
			message.ParseMode = "HTML"

			toSettingsCallbackData := "profileSettings?showBackButton=" + strconv.FormatBool(showBackButton)
			toMainMenuCallbackData := "mainMenu"

			keyboard := [][]tgbotapi.InlineKeyboardButton{
				{{Text: "⚙️Настройки", CallbackData: &toSettingsCallbackData}},
			}

			if !showBackButton {
				keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{{Text: "На главную", CallbackData: &toMainMenuCallbackData}})
			} else {
				keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{{Text: "К оформлению заказа", CallbackData: &processOrderCallbackData}})
			}

			message.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: keyboard,
			}

			_, err = client.Send(message)

			return err
		},
		Params:      map[string]any{},
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
	const text = "Ваш номер телефона сейчас:\n<b>%v(%v)%v-%v</b>\n\n<i>Введите новый номер телефона:</i>"

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

	data := ParseCallData(update.CallbackQuery.Data)
	showBackButton := data["showBackButton"] == "true"

	toSettingsCallbackData := "profileSettings?showBackButton=" + strconv.FormatBool(showBackButton)
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

				tryAgainCallbackData := "changePhone?showBackButton=" + strconv.FormatBool(showBackButton)
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

			_, err = db.Model(&user).WherePK().Column("phone").Update()
			if err != nil {
				return err
			}

			message := tgbotapi.NewMessage(stepUpdate.Message.Chat.ID, "<b>Номер телефона успешно изменен</b>✅")
			message.ParseMode = "HTML"

			toSettingsCallbackData := "profileSettings?showBackButton=" + strconv.FormatBool(showBackButton)
			toMainMenuCallbackData := "mainMenu"

			keyboard := [][]tgbotapi.InlineKeyboardButton{
				{{Text: "⚙️Настройки", CallbackData: &toSettingsCallbackData}},
			}

			if !showBackButton {
				keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{{Text: "На главную", CallbackData: &toMainMenuCallbackData}})
			} else {
				keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{{Text: "К оформлению заказа", CallbackData: &processOrderCallbackData}})
			}

			message.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: keyboard,
			}

			_, err = client.Send(message)

			return err
		},
		Params:      map[string]any{},
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
	const text = "Ваш адрес доставки сейчас:\n<b>%s</b>\n\n<i>Введите новый адрес доставки для сервиса %s:</i>"

	message := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "")
	message.ParseMode = "HTML"

	db := database.Connect()
	defer db.Close()

	user := models.TelegramUser{ID: update.CallbackQuery.Message.Chat.ID}
	err := user.GetOrCreate(update.CallbackQuery.From, *db)
	if err != nil {
		return err
	}

	devServiceName := ""
	switch user.DeliveryService {
	case "cdek":
		devServiceName = "CDEK"
	case "yandex":
		devServiceName = "Яндекс доставка"
	}

	message.Text = fmt.Sprintf(text, user.DeliveryAddress, devServiceName)

	data := ParseCallData(update.CallbackQuery.Data)
	showBackButton := data["showBackButton"] == "true"

	toSettingsCallbackData := "profileSettings?showBackButton=" + strconv.FormatBool(showBackButton)
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

			_, err = db.Model(&user).WherePK().Column("delivery_address").Update()
			if err != nil {
				return err
			}

			message := tgbotapi.NewMessage(stepUpdate.Message.Chat.ID, "<b>Адрес доставки успешно изменен</b>✅")
			message.ParseMode = "HTML"

			toSettingsCallbackData := "profileSettings?showBackButton=" + strconv.FormatBool(showBackButton)
			toMainMenuCallbackData := "mainMenu"

			keyboard := [][]tgbotapi.InlineKeyboardButton{
				{{Text: "⚙️Настройки", CallbackData: &toSettingsCallbackData}},
			}

			if !showBackButton {
				keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{{Text: "На главную", CallbackData: &toMainMenuCallbackData}})
			} else {
				keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{{Text: "К оформлению заказа", CallbackData: &processOrderCallbackData}})
			}

			message.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: keyboard,
			}
			_, err = client.Send(message)

			return err
		},
		Params:      map[string]any{},
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

func (c ChangeDeliveryService) GetKeyboard(userDb models.TelegramUser, showBackButton bool) [][]tgbotapi.InlineKeyboardButton {
	type buttonConfig struct {
		Text    string
		Setting string
	}

	createButton := func(cfg buttonConfig, showBackButton bool) tgbotapi.InlineKeyboardButton {
		if cfg.Setting == userDb.DeliveryService {
			cfg.Text += " ✅"
		}

		callQuery := fmt.Sprintf("changeDeliveryService?service=%s&showBackButton=%t", cfg.Setting, showBackButton)

		return tgbotapi.InlineKeyboardButton{
			Text:         cfg.Text,
			CallbackData: &callQuery,
		}
	}

	buttons := []buttonConfig{
		{Text: "CDEK", Setting: "cdek"},
		{Text: "Яндекс доставка", Setting: "yandex"},
	}

	keyboard := [][]tgbotapi.InlineKeyboardButton{}

	for _, b := range buttons {
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{createButton(b, showBackButton)})
	}

	return keyboard
}

func (c ChangeDeliveryService) Run(update tgbotapi.Update) error {
	ClearNextStepForUser(update, &c.Client, true)

	const text = "Ваш сервис доставки сейчас:\n<b>%s</b>\n\n<i>Выберите новый сервис доставки:</i>"

	message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, "")
	message.ParseMode = "HTML"

	db := database.Connect()
	defer db.Close()

	user := models.TelegramUser{ID: update.CallbackQuery.From.ID}
	err := user.GetOrCreate(update.CallbackQuery.From, *db)
	if err != nil {
		return err
	}

	devServiceName := ""
	switch user.DeliveryService {
	case "cdek":
		devServiceName = "CDEK"
	case "yandex":
		devServiceName = "Яндекс доставка"
	}

	message.Text = fmt.Sprintf(text, devServiceName)

	data := ParseCallData(update.CallbackQuery.Data)
	showBackButton := data["showBackButton"] == "true"

	toSettingsCallbackData := "profileSettings?showBackButton=" + strconv.FormatBool(showBackButton)
	message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: append(c.GetKeyboard(user, showBackButton), []tgbotapi.InlineKeyboardButton{{Text: "⚙️Настройки", CallbackData: &toSettingsCallbackData}}),
	}

	_, err = c.Client.Send(message)

	return err
}

func (c ChangeDeliveryService) GetName() string {
	return c.Name
}
