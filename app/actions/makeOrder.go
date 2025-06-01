package actions

import (
	"context"
	"fmt"
	"main/database"
	"main/database/models"
	"sync"
	"time"

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
	changeDataCallbackData = "profileSettings?showBackButton=true"
	toListofCats = "shop"
)

// MakeOrder представляет собой структуру для оформления заказа
// Name - имя команды
// Client - экземпляр Telegram бота
type MakeOrder struct {
	Name   string
	Client tgbotapi.BotAPI
	mu     *sync.Mutex
}

func NewMakeOrderHandler(client tgbotapi.BotAPI) *MakeOrder {
	return &MakeOrder{
		Name:   "makeOrder",
		Client: client,
		mu:     &sync.Mutex{},
	}
}

// Run запускает процесс оформления заказа
// update - обновление от Telegram API
// Возвращает ошибку, если что-то пошло не так
func (m MakeOrder) Run(update tgbotapi.Update) error {
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
			m.mu.Lock()
			ClearNextStepForUser(update, &m.Client, true)
			m.mu.Unlock()

			db := database.Connect()
			defer db.Close()

			user := models.TelegramUser{ID: update.CallbackQuery.From.ID}
			err = user.Get(*db)
			if err != nil {
				return
			}

			var totalPrice int
			totalPrice, err = user.GetTotalCartPrice(*db)
			if err != nil {
				return
			}

			m.mu.Lock()
			m.Client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))
			m.mu.Unlock()

			finalPageText := fmt.Sprintf(makeOrderPageText, totalPrice, user.Phone, user.FIO, user.DeliveryAddress, user.DeliveryService)

			msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, finalPageText)
			msg.ParseMode = "HTML"

			msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{
				{Text: "Да, все верно✅", CallbackData: &processOrderCallbackData},
				{Text: "Изменить данные⚙️", CallbackData: &changeDataCallbackData},
			}, {
				{Text: "К списку каталогов", CallbackData: &toListofCats},
			}}}

			m.mu.Lock()
			_, err = m.Client.Send(msg)
			m.mu.Unlock()
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

// GetName возвращает имя команды
func (m MakeOrder) GetName() string {
	return m.Name
}
