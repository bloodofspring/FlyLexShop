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

// RegisterPaymentPhoto обрабатывает фотографию чека об оплате или PDF файл
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
			// Проверяем наличие фото или PDF документа
			hasValidAttachment := (update.Message != nil && update.Message.Photo != nil) ||
				(update.Message != nil && update.Message.Document != nil && update.Message.Document.MimeType == "application/pdf")

			if !hasValidAttachment {
				message := tgbotapi.NewMessage(update.Message.Chat.ID, "Пожалуйста, пришлите фото чека или PDF файл на проверку.")
				toMainMenuCallbackData := "mainMenu?resetAvailablity=true"
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

			user := models.TelegramUser{ID: update.Message.From.ID}
			err = user.Get(*db)
			if err != nil {
				return
			}

			var totalPrice int
			totalPrice, err = user.GetTotalCartPrice(*db)
			if err != nil {
				return
			}

			var cartDesc string
			cartDesc, err = user.GetCartDescription(*db)
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

			var msg tgbotapi.Chattable
			if update.Message.Photo != nil {
				photoMsg := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(update.Message.Photo[len(update.Message.Photo)-1].FileID))
				photoMsg.ParseMode = "HTML"
				photoMsg.Caption = cartDesc
				msg = photoMsg
			} else {
				docMsg := tgbotapi.NewDocument(chatID, tgbotapi.FileID(update.Message.Document.FileID))
				docMsg.ParseMode = "HTML"
				docMsg.Caption = cartDesc
				msg = docMsg
			}

			var transaction models.Transaction
			transaction, err, _ = user.GetOrCreateTransaction(*db)
			if err != nil {
				return
			}

			db.Model(&transaction).WherePK().Set("is_waiting_for_approval = ?", true).Update()

			acceptData := fmt.Sprintf("paymentVerdict?ok=true&tid=%d&userId=%d", transaction.ID, update.Message.From.ID)
			rejectData := fmt.Sprintf("paymentVerdict?ok=false&tid=%d&userId=%d", transaction.ID, update.Message.From.ID)

			// Создаем клавиатуру для обоих типов сообщений
			keyboard := tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					{
						{Text: "Принять заявку", CallbackData: &acceptData},
						{Text: "Отклонить заявку", CallbackData: &rejectData},
					},
				},
			}

			// Устанавливаем клавиатуру в зависимости от типа сообщения
			if photoMsg, ok := msg.(tgbotapi.PhotoConfig); ok {
				photoMsg.ReplyMarkup = keyboard
				msg = photoMsg
			} else if docMsg, ok := msg.(tgbotapi.DocumentConfig); ok {
				docMsg.ReplyMarkup = keyboard
				msg = docMsg
			}

			mu.Lock()
			_, err = client.Send(msg)
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

			var cartChanged bool
			cartChanged, err = user.TidyCart(*db)
			if err != nil {
				return
			}

			if cartChanged {
				_, err := p.Client.Request(tgbotapi.CallbackConfig{
					CallbackQueryID: update.CallbackQuery.ID,
					Text:            "Количество некторых товаров уменьшилось. Проверьте корзину перед покупкой",
					ShowAlert:       true,
				})
				if err != nil {
					return
				}
			}

			var transaction models.Transaction
			transaction, err, _ = user.GetOrCreateTransaction(*db)
			if err != nil {
				return
			}

			err = user.DecreaseProductAvailbleForPurchase(*db, transaction.ID)
			if err != nil {
				return
			}

			var totalPrice int
			totalPrice, err = user.GetTotalCartPrice(*db)
			if err != nil {
				return
			}

			if totalPrice == 0 {
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Корзина пуста")
				msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{
					{Text: "К списку каталогов", CallbackData: &toListofCats},
				}}}

				p.mu.Lock()
				_, err = p.Client.Send(msg)
				p.mu.Unlock()

				return
			}

			pageText := fmt.Sprintf(processOrderPageText, totalPrice, os.Getenv("PAYMENT_CARD_NUMBER"), os.Getenv("PAYMENT_PHONE_NUMBER"), os.Getenv("PAYMENT_BANK"))

			msg := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, pageText)
			toMainMenuCallbackData := "mainMenu?resetAvailablity=true"
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
