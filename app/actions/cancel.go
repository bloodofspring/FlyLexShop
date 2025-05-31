package actions

import (
	"context"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Cancel представляет собой структуру для отмены действий
// Name - имя команды
// Client - экземпляр Telegram бота
type Cancel struct {
	Name   string
	Client tgbotapi.BotAPI
	mu     *sync.Mutex
}

func NewCancelHandler(client tgbotapi.BotAPI) *Cancel {
	return &Cancel{
		Name:   "cancel",
		Client: client,
		mu:     &sync.Mutex{},
	}
}

// Run запускает процесс отмены действия
// update - обновление от Telegram API
// Возвращает ошибку, если что-то пошло не так
func (c Cancel) Run(update tgbotapi.Update) error {
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
			ClearNextStepForUser(update, &c.Client, false)
			c.Client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))
			c.mu.Unlock()

			_, err = c.Client.Request(tgbotapi.CallbackConfig{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "Действие отменено",
				ShowAlert:       false,
			})
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
func (c Cancel) GetName() string {
	return c.Name
}
