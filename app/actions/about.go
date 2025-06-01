package actions

import (
	"context"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// About представляет собой структуру для отображения информации о боте
// Name - имя команды
// Client - экземпляр Telegram бота
type About struct {
	Name   string
	Client tgbotapi.BotAPI
	mu     *sync.Mutex
}

func NewAboutHandler(client tgbotapi.BotAPI) *About {
	return &About{
		Name:   "about",
		Client: client,
		mu:     &sync.Mutex{},
	}
}

// Run запускает отображение информации о боте
// update - обновление от Telegram API
// Возвращает ошибку, если что-то пошло не так
func (a About) Run(update tgbotapi.Update) error {
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
			a.mu.Lock()
			ClearNextStepForUser(update, &a.Client, true)
			a.mu.Unlock()
		
			const text = `🔥Вас приветствует команда FlyLex в боте для совершения покупок нашей продукции!🎯FlyLex отличается от других тем, что наша продукция является отечественной, так как она производится на территории РФ.
			
			🥇FlyLex - выбор лучших! Рама Pike5’ используется топ-пилотами, в том числе, Платоном Черемных.
			
			✅<a href="https://t.me/FlyLex_official">Телеграмм канал</a>
			✅<a href="https://t.me/FlyLex_response">Чат с отзывами</a>
			✅<a href="https://t.me/FlyLex_chat">Чат</a>
			
			⚙️Контакты для уточнения вопросов по заказам и продукции:
			✅Телеграмм: <b>@FlyLex_Admin</b>
			✅Телефон: <b>8(925)-222-58-10</b>
			
			👨‍💻Рабочее время
			<b>8:00 - 22:00 по МСК</b>`
			message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, text)
			message.ParseMode = "HTML"
			message.DisableWebPagePreview = true
		
			toMainMenuCallbackData := "mainMenu"
			message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					{{Text: "На главную", CallbackData: &toMainMenuCallbackData}},
				},
			}
		
			a.mu.Lock()
			_, err = a.Client.Send(message)
			a.mu.Unlock()
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
func (a About) GetName() string {
	return a.Name
}
