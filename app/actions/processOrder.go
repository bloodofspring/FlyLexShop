package actions

import (
	"context"
	"fmt"
	"main/controllers"
	"main/database"
	"main/database/models"
	"os"
	"strconv"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	// processOrderPageText - шаблон текста для страницы оплаты заказа
	processOrderPageText = "<b>Итог:</b> %d\n\nОплата осуществляется переводом по номеру карты или телефона:\n|_<b>Номер карты:</b> %s\n|_<b>Номер телефона:</b> %s\n|_<b>Банк:</b> %s\n\n<b>!!!После оплаты пришлите боту чек на проверку сообщением ниже!!!</b>"
)

// RegisterPaymentPhoto обрабатывает фотографию чека об оплате
// client - экземпляр Telegram бота
// update - обновление от Telegram API
// stepParams - параметры шага обработки заказа
// Возвращает ошибку, если что-то пошло не так
func RegisterPaymentPhoto(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	mu := sync.Mutex{}
	var err error

	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			return
		default:
			if update.Message == nil || update.Message.Photo == nil {
				message := tgbotapi.NewMessage(update.Message.Chat.ID, "Пожалуйста, пришлите фото чека на проверку.")
				toMainMenuCallbackData := "mainMenu"
				message.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
					InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
						{
							{Text: "На главную", CallbackData: &toMainMenuCallbackData},
						},
					},
				}
				mu.Lock()
				_, err = client.Send(message)
				mu.Unlock()
				if err != nil {
					return
				}
		
				stepKey := controllers.NextStepKey{
					ChatID: update.Message.Chat.ID,
					UserID: update.Message.From.ID,
				}
				stepAction := controllers.NextStepAction{
					Func:          RegisterPaymentPhoto,
					Params:        make(map[string]any),
					CreatedAtTS:   time.Now().Unix(),
					CancelMessage: "Оформление заказа прервано! Вы можете совершить покупку позже в этом же разделе.",
				}
		
				mu.Lock()
				controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)
				mu.Unlock()
		
				return
			}

			adminChatID := os.Getenv("ADMIN_CHAT_ID")
		
			db := database.Connect()
			defer db.Close()
		
			var items []models.Product
			err = db.Model(&items).Where("id IN (SELECT product_id FROM added_products WHERE user_id = ?)", update.Message.From.ID).Select()
			if err != nil {
				return
			}
		
			cartDesc := "Список товаров:\n"
			totalPrice := 0
			for _, item := range items {
				cartDesc += fmt.Sprintf("|_ %s - %d₽\n", item.Name, item.Price)
				totalPrice += item.Price
			}
		
			user := models.TelegramUser{ID: update.Message.From.ID}
			err = user.Get(*db)
			if err != nil {
				return
			}
		
			cartDesc += fmt.Sprintf("\nИтоговая сумма: %d₽", totalPrice)
			cartDesc += "\n<b>Дополнительная информация:</b>"
			cartDesc += "\n|_ Адрес доставки: " + user.DeliveryAddress
			cartDesc += "\n|_ Сервис доставки: " + user.DeliveryService
			cartDesc += "\n|_ Номер телефона: " + user.Phone
			cartDesc += "\n|_ ФИО: " + user.FIO
		
			var chatID int64
			chatID, err = strconv.ParseInt(adminChatID, 10, 64)
			if err != nil {
				return
			}
		
			photoMsg := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(update.Message.Photo[len(update.Message.Photo)-1].FileID))
			photoMsg.ParseMode = "HTML"
			photoMsg.Caption = cartDesc
		
			acceptData := "paymentVerdict?ok=true&userId=" + strconv.FormatInt(update.Message.From.ID, 10)
			rejectData := "paymentVerdict?ok=false&userId=" + strconv.FormatInt(update.Message.From.ID, 10)
			photoMsg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					{
						{Text: "Принять заявку", CallbackData: &acceptData},
						{Text: "Отклонить заявку", CallbackData: &rejectData},
					},
				},
			}
		
			mu.Lock()
			_, err = client.Send(photoMsg)
			mu.Unlock()
			if err != nil {
				return
			}
		
			successMsg := tgbotapi.NewMessage(update.Message.Chat.ID, "Спасибо, администратор скоро проверит оплату!")
			mainMenuCallbackData := "mainMenu"
			successMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					{{Text: "На главную", CallbackData: &mainMenuCallbackData}},
				},
			}
			mu.Lock()
			_, err = client.Send(successMsg)
			mu.Unlock()
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

// ProcessOrder представляет собой структуру для обработки заказа
// Name - имя команды
// Client - экземпляр Telegram бота
type ProcessOrder struct {
	Name   string
	Client tgbotapi.BotAPI
	mu     *sync.Mutex
}

func NewProcessOrderHandler(client tgbotapi.BotAPI) *ProcessOrder {
	return &ProcessOrder{
		Name:   "processOrder",
		Client: client,
		mu:     &sync.Mutex{},
	}
}

// Run запускает процесс обработки заказа
// update - обновление от Telegram API
// Возвращает ошибку, если что-то пошло не так
func (p ProcessOrder) Run(update tgbotapi.Update) error {
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
			p.mu.Lock()
			ClearNextStepForUser(update, &p.Client, true)
			p.mu.Unlock()

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

			pageText := fmt.Sprintf(processOrderPageText, totalPrice, os.Getenv("PAYMENT_CARD_NUMBER"), os.Getenv("PAYMENT_PHONE_NUMBER"), os.Getenv("PAYMENT_BANK"))

			msg := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, pageText)
			toMainMenuCallbackData := "mainMenu"
			msg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					{
						{Text: "На главную", CallbackData: &toMainMenuCallbackData},
					},
				},
			}
			msg.ParseMode = "HTML"

			p.mu.Lock()
			_, err = p.Client.Send(msg)
			p.mu.Unlock()
			if err != nil {
				return
			}

			stepKey := controllers.NextStepKey{
				ChatID: update.CallbackQuery.Message.Chat.ID,
				UserID: update.CallbackQuery.From.ID,
			}
			stepAction := controllers.NextStepAction{
				Func:          RegisterPaymentPhoto,
				Params:        make(map[string]any),
				CreatedAtTS:   time.Now().Unix(),
				CancelMessage: "Оформление заказа прервано! Вы можете совершить покупку позже в этом же разделе.",
			}

			p.mu.Lock()
			controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)
			p.mu.Unlock()
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
func (p ProcessOrder) GetName() string {
	return p.Name
}
