package actions

import (
	"context"
	"main/database"
	"main/database/models"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SayHi представляет собой структуру для обработки команды /start
// Name - имя команды
// Client - экземпляр Telegram бота
type SayHi struct {
	Name   string
	Client tgbotapi.BotAPI
	mu     *sync.Mutex
}

func NewSayHiHandler(client tgbotapi.BotAPI) *SayHi {
	return &SayHi{
		Name:   "sayHi",
		Client: client,
		mu:     &sync.Mutex{},
	}
}

// fabricateAnswer создает ответное сообщение на команду /start
// update - обновление от Telegram API
// Возвращает сконфигурированное сообщение с приветствием и кнопкой регистрации
func (e SayHi) fabricateAnswer(update tgbotapi.Update) tgbotapi.MessageConfig {
	ClearNextStepForUser(update, &e.Client, true)
	const text = "Добрый день!👋\nВы попали в бота компании FlyLex🔥\n\nНажмите кнопку «Регистрация» чтобы продолжить!"
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)

	callbackData := "registerUser"
	msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{{Text: "Регистрация", CallbackData: &callbackData}},
		},
	}

	db := database.Connect()
	defer db.Close()

	user := models.TelegramUser{ID: update.Message.From.ID}
	_ = user.GetOrCreate(update.Message.From, *db)

	return msg
}

// Run выполняет обработку команды /start
// update - обновление от Telegram API
// Возвращает ошибку, если отправка сообщения не удалась
func (e SayHi) Run(update tgbotapi.Update) error {
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
			resp := e.fabricateAnswer(update)

			e.mu.Lock()
			_, err = e.Client.Send(resp)
			e.mu.Unlock()
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
func (e SayHi) GetName() string {
	return e.Name
}
