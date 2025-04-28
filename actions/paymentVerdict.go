package actions

import (
	"context"
	"main/database"
	"main/database/models"
	"main/filters"
	"strconv"
	"sync"
	"time"

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
	mu     *sync.Mutex
}

func NewPaymentVerdictHandler(client tgbotapi.BotAPI) *PaymentVerdict {
	return &PaymentVerdict{
		Name:   "paymentVerdict",
		Client: client,
		mu:     &sync.Mutex{},
	}
}

// Run обрабатывает результат проверки оплаты администратором
// update - обновление от Telegram API
// Возвращает ошибку, если что-то пошло не так
func (p PaymentVerdict) Run(update tgbotapi.Update) error {
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

			data := filters.ParseCallbackData(update.CallbackQuery.Data)

			var userId int64
			userId, err = strconv.ParseInt(data["userId"], 10, 64)
			if err != nil {
				return
			}

			paymentAccepted := data["ok"]

			if paymentAccepted == "true" {
				message := tgbotapi.NewMessage(userId, paymentAcceptedMessageText)
				p.mu.Lock()
				_, err = p.Client.Send(message)
				p.mu.Unlock()
				if err != nil {
					return
				}

				p.mu.Lock()
				_, err = p.Client.Send(tgbotapi.NewEditMessageCaption(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, update.CallbackQuery.Message.Caption+"\n\nОплата принята✅"))
				p.mu.Unlock()
				if err != nil {
					return
				}

				db := database.Connect()
				defer db.Close()

				_, err = db.Model(&models.ShoppingCart{}).Where("user_id = ?", userId).Delete()
				if err != nil {
					return
				}

				return
			}

			message := tgbotapi.NewMessage(userId, paymentRejectedMessageText)
			p.mu.Lock()
			_, err = p.Client.Send(message)
			p.mu.Unlock()
			if err != nil {
				return
			}

			p.mu.Lock()
			_, err = p.Client.Send(tgbotapi.NewEditMessageCaption(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, update.CallbackQuery.Message.Caption+"\n\nОплата отклонена❌"))
			p.mu.Unlock()
			if err != nil {
				return
			}
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
func (p PaymentVerdict) GetName() string {
	return p.Name
}
