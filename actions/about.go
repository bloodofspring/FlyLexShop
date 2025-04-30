package actions

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// About представляет собой структуру для отображения информации о боте
// Name - имя команды
// Client - экземпляр Telegram бота
type About struct {
	Name   string
	Client tgbotapi.BotAPI
}

// Run запускает отображение информации о боте
// update - обновление от Telegram API
// Возвращает ошибку, если что-то пошло не так
func (a About) Run(update tgbotapi.Update) error {
	ClearNextStepForUser(update, &a.Client, true)

	// Выводим информацию о боте
	fmt.Println("хуй")

	return nil
}

// GetName возвращает имя команды
func (a About) GetName() string {
	return a.Name
}
