package actions

import (
	"context"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// MainMenu представляет собой структуру для отображения главного меню
// Name - имя команды
// Client - экземпляр Telegram бота
type MainMenu struct {
	Name   string
	Client tgbotapi.BotAPI
	mu     *sync.Mutex
}

func NewMainMenuHandler(client tgbotapi.BotAPI) *MainMenu {
	return &MainMenu{
		Name:   "mainMenu",
		Client: client,
		mu:     &sync.Mutex{},
	}
}

// Run запускает отображение главного меню
// update - обновление от Telegram API
// Возвращает ошибку, если что-то пошло не так
func (m MainMenu) Run(update tgbotapi.Update) error {
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

			const text = "<b>Главное меню</b>\nВыберите опцию:"

			settingsCallbackData := "profileSettings"
			shopCallbackData := "shop"
			aboutCallbackData := "about"

			keyboard := tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					{{Text: "Настройки", CallbackData: &settingsCallbackData}},
					{{Text: "Магазин", CallbackData: &shopCallbackData}},
					{{Text: "О нас", CallbackData: &aboutCallbackData}},
				},
			}

			if update.CallbackQuery != nil {
				message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, text)
				message.ParseMode = "HTML"

				message.ReplyMarkup = &keyboard

				m.mu.Lock()
				_, err = m.Client.Send(message)
				m.mu.Unlock()

				return
			}

			message := tgbotapi.NewMessage(update.Message.Chat.ID, text)
			message.ParseMode = "HTML"

			message.ReplyMarkup = keyboard

			m.mu.Lock()
			_, err = m.Client.Send(message)
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
func (m MainMenu) GetName() string {
	return m.Name
}
