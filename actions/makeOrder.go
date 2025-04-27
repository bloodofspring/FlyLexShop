package actions

import (
	"fmt"
	"main/database"
	"main/database/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	// makeOrderPageText - шаблон текста для страницы оформления заказа
	makeOrderPageText = "<b>Итог:</b>\nОбщая стоимость товаров: %dр.\n\n<b>Проверьте корректность ваших данных:</b>\n\n|_ Номер телефона: %s\n|_ ФИО: %s\n|_ Адрес ПВЗ: %s\n|_ Сервис доставки: %s"
)

var (
	// processOrderCallbackData - callback data для обработки заказа
	processOrderCallbackData = "processOrder"
	// changeDataCallbackData - callback data для изменения данных пользователя
	changeDataCallbackData = "profileSettings"
)

// MakeOrder представляет собой структуру для оформления заказа
// Name - имя команды
// Client - экземпляр Telegram бота
type MakeOrder struct {
	Name   string
	Client tgbotapi.BotAPI
}

// Run запускает процесс оформления заказа
// update - обновление от Telegram API
// Возвращает ошибку, если что-то пошло не так
func (m MakeOrder) Run(update tgbotapi.Update) error {
	ClearNextStepForUser(update, &m.Client, true)
	db := database.Connect()
	defer db.Close()

	user := models.TelegramUser{ID: update.CallbackQuery.From.ID}
	err := user.Get(*db)
	if err != nil {
		return err
	}

	totalPrice, err := user.GetTotalCartPrice(*db)
	if err != nil {
		return err
	}

	m.Client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))
	finalPageText := fmt.Sprintf(makeOrderPageText, totalPrice, user.Phone, user.FIO, user.DeliveryAddress, user.DeliveryService)

	msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, finalPageText)
	msg.ParseMode = "HTML"

	msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{
		{Text: "Да, все верно", CallbackData: &processOrderCallbackData},
		{Text: "Изменить данные", CallbackData: &changeDataCallbackData},
	}}}

	_, err = m.Client.Send(msg)

	return err
}

// GetName возвращает имя команды
func (m MakeOrder) GetName() string {
	return m.Name
}
