package actions

import (
	"context"
	"fmt"
	"main/database"
	"main/database/models"
	"main/filters"
	"regexp"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ProfileSettings представляет собой структуру для управления настройками профиля пользователя.
// Name - имя команды.
// Client - экземпляр Telegram бота.
type ProfileSettings struct {
	Name   string
	Client tgbotapi.BotAPI
	mu     *sync.Mutex
}

func NewProfileSettingsHandler(client tgbotapi.BotAPI) *ProfileSettings {
	return &ProfileSettings{
		Name:   "profileSettings",
		Client: client,
		mu:     &sync.Mutex{},
	}
}

// Run отображает меню настроек профиля.
// update - обновление от Telegram API.
// Возвращает ошибку, если отправка сообщения не удалась.
func (p ProfileSettings) Run(update tgbotapi.Update) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	var wg sync.WaitGroup
	var err error
	
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			return
		default:
			data := filters.ParseCallbackData(update.CallbackQuery.Data)

			db := database.Connect()
			defer db.Close()

			userDb := models.TelegramUser{ID: update.CallbackQuery.From.ID}
			err = userDb.GetOrCreate(update.CallbackQuery.From, *db)
			if err != nil {
				return
			}

			if update.CallbackQuery.Data == "profileSettings" {
				err = p.sendChoiceMessage(update)
				return
			}

			p.mu.Lock()
			switch data["a"] {
			case "changeName":
				err = changeUserName(update, p.Client, userDb)
			case "changePhone":
				err = changeUserPhone(update, p.Client, userDb)
			case "changeDeliveryAddress":
				err = changeUserDeliveryAddress(update, p.Client, userDb)
			}
			p.mu.Unlock()

			return
		}
	}()
	
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetName возвращает имя команды ProfileSettings.
func (p ProfileSettings) GetName() string {
	return p.Name
}

func (p ProfileSettings) sendChoiceMessage(update tgbotapi.Update) error {
	p.mu.Lock()
	ClearNextStepForUser(update, &p.Client, true)
	p.mu.Unlock()

	const text = "<b>Настройки профиля</b>\nВыберите опцию:"

	message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, text)
	message.ParseMode = "HTML"

	changeNameCallbackData := "profileSettings?a=changeName"
	changePhoneCallbackData := "profileSettings?a=changePhone"
	changeDeliveryAddressCallbackData := "profileSettings?a=changeDeliveryAddress"
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

	p.mu.Lock()
	_, err := p.Client.Send(message)
	p.mu.Unlock()

	return err
}

func changeUserName(update tgbotapi.Update, client tgbotapi.BotAPI, userDb models.TelegramUser) error {
	return baseForm(client, update, map[string]any{
		"userDb": &userDb,
	}, "Отправьте ниже ваше ФИО", "ФИО не обновлено", "profileSettings", changeUserNameHandler)
}


func changeUserNameHandler(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID-1))
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))

	name := update.Message.Text

	if name == "" {
		return baseFormResend(client, update, "Имя не может быть пустым", "Имя не обновлено", "profileSettings", stepParams, changeUserNameHandler)
	}

	db := database.Connect()
	defer db.Close()

	_, err := db.Model(stepParams["userDb"].(*models.TelegramUser)).WherePK().Set("fio = ?", name).Update()
	if err != nil {
		return err
	}

	return baseFormSuccess(client, update, "ФИО обновлено!", "profileSettings", "К настройкам профиля")
}

func changeUserPhone(update tgbotapi.Update, client tgbotapi.BotAPI, userDb models.TelegramUser) error {
	return baseForm(client, update, map[string]any{
		"userDb": &userDb,
	}, "Отправьте ниже ваш номер телефона", "Номер телефона не обновлен", "profileSettings", changeUserPhoneHandler)
}

func changeUserPhoneHandler(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID-1))
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))

	regex := regexp.MustCompile(`^[0-9]{11}$`)
	if update.Message.Text == "" || !regex.MatchString(update.Message.Text) {
		return baseFormResend(client, update, "Неверный формат ввода!\n\nВведите номер телефона в формате 89991234567:", "Номер телефона не обновлен", "profileSettings", stepParams, changeUserPhoneHandler)
	}

	db := database.Connect()
	defer db.Close()

	_, err := db.Model(stepParams["userDb"].(*models.TelegramUser)).WherePK().Set("phone = ?", update.Message.Text).Update()
	if err != nil {
		return err
	}

	return baseFormSuccess(client, update, "Номер телефона обновлен!", "profileSettings", "К настройкам профиля")
}

func changeUserDeliveryAddress(update tgbotapi.Update, client tgbotapi.BotAPI, userDb models.TelegramUser) error {
	return baseForm(client, update, map[string]any{
		"userDb": &userDb,
	}, "Отправьте ниже ваш адрес доставки", "Адрес доставки не обновлен", "profileSettings", changeUserDeliveryAddressHandler)
}


func changeUserDeliveryAddressHandler(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID-1))
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))

	address := update.Message.Text

	if address == "" {
		return baseFormResend(client, update, "Адрес доставки не может быть пустым", "Адрес доставки не обновлен", "profileSettings", stepParams, changeUserDeliveryAddressHandler)
	}

	db := database.Connect()
	defer db.Close()

	_, err := db.Model(stepParams["userDb"].(*models.TelegramUser)).WherePK().Set("delivery_address = ?", address).Update()
	if err != nil {
		return err
	}

	return baseFormSuccess(client, update, "Адрес доставки обновлен!", "profileSettings", "К настройкам профиля")
}

type ChangeDeliveryService struct {
	Name   string
	Client tgbotapi.BotAPI
	mu     *sync.Mutex
}

func NewChangeDeliveryServiceHandler(client tgbotapi.BotAPI) *ChangeDeliveryService {
	return &ChangeDeliveryService{
		Name:   "changeDeliveryService",
		Client: client,
		mu:     &sync.Mutex{},
	}
}

func (c ChangeDeliveryService) GetKeyboard(userDb models.TelegramUser) [][]tgbotapi.InlineKeyboardButton {
	type buttonConfig struct {
		Text    string
		Setting string
	}

	createButton := func(cfg buttonConfig) tgbotapi.InlineKeyboardButton {
		if cfg.Setting == userDb.DeliveryService {
			cfg.Text += " ✅"
		}

		callQuery := fmt.Sprintf("changeDeliveryService?service=%s", cfg.Setting)

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
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{createButton(b)})
	}

	return keyboard
}

func (c ChangeDeliveryService) Run(update tgbotapi.Update) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	var wg sync.WaitGroup
	var err error
	
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			return
		default:
			c.mu.Lock()
			ClearNextStepForUser(update, &c.Client, true)
			c.mu.Unlock()

			const text = "<b>Ваш сервис доставки сейчас: %s</b>\nВыберите новый сервис доставки:"
		
			message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, "")
			message.ParseMode = "HTML"
		
			db := database.Connect()
			defer db.Close()
		
			user := models.TelegramUser{ID: update.CallbackQuery.From.ID}
			err = user.GetOrCreate(update.CallbackQuery.From, *db)
			if err != nil {
				return
			}
		
			message.Text = fmt.Sprintf(text, user.DeliveryService)
		
			toSettingsCallbackData := "profileSettings"
			message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: append(c.GetKeyboard(user), []tgbotapi.InlineKeyboardButton{{Text: "К настройкам", CallbackData: &toSettingsCallbackData}}),
			}
		
			c.mu.Lock()
			_, err = c.Client.Send(message)
			c.mu.Unlock()
		}
	}()
	
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c ChangeDeliveryService) GetName() string {
	return c.Name
}
