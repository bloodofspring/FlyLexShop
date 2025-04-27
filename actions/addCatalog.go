package actions

import (
	"fmt"
	"main/controllers"
	"main/database"
	"main/database/models"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// AddCatalog управляет процессом добавления нового каталога.
// Name - имя команды.
// Client - экземпляр Telegram бота.
type AddCatalog struct {
	Name   string
	Client tgbotapi.BotAPI
}

var (
	cancelCallbackData = "cancel"
)

// Run отправляет запрос ввода названия каталога и регистрирует следующий шаг создания каталога.
func (a AddCatalog) Run(update tgbotapi.Update) error {
	msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Введите название каталога")
	msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{{Text: "Отменить", CallbackData: &cancelCallbackData}},
		},
	}
	_, err := a.Client.Send(msg)
	if err != nil {
		return err
	}

	stepKey := controllers.NextStepKey{
		UserID: update.CallbackQuery.From.ID,
		ChatID: update.CallbackQuery.Message.Chat.ID,
	}

	stepAction := controllers.NextStepAction{
		Func:          CreateCatalog,
		Params:        make(map[string]interface{}),
		CreatedAtTS:   time.Now().Unix(),
		CancelMessage: "Создание каталога отменено",
	}

	controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)

	return nil
}

// GetName возвращает имя команды AddCatalog.
func (a AddCatalog) GetName() string {
	return a.Name
}

// CreateCatalog обрабатывает ввод названия каталога и сохраняет новый каталог в базе данных.
func CreateCatalog(client tgbotapi.BotAPI, stepUpdate tgbotapi.Update, stepParams map[string]any) error {
	if stepUpdate.Message.Text == "" {
		msg := tgbotapi.NewMessage(stepUpdate.Message.Chat.ID, "Введите название каталога")
		msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
				{{Text: "Отменить", CallbackData: &cancelCallbackData}},
			},
		}
		_, err := client.Send(msg)
		if err != nil {
			return err
		}

		stepKey := controllers.NextStepKey{
			UserID: stepUpdate.Message.From.ID,
			ChatID: stepUpdate.Message.Chat.ID,
		}

		stepAction := controllers.NextStepAction{
			Func:          CreateCatalog,
			Params:        make(map[string]interface{}),
			CreatedAtTS:   time.Now().Unix(),
			CancelMessage: "Создание каталога отменено",
		}

		controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)

		return nil
	}

	db := database.Connect()
	defer db.Close()

	_, err := db.Model(&models.Catalog{
		Name: stepUpdate.Message.Text,
	}).Insert()
	if err != nil {
		return err
	}

	ClearNextStepForUser(stepUpdate, &client, false)

	msg := tgbotapi.NewMessage(stepUpdate.Message.Chat.ID, fmt.Sprintf("Каталог с названием \"%s\" успешно создан", stepUpdate.Message.Text))
	toCatalogListCallbackData := "shop"
	msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{{Text: "К списку каталогов", CallbackData: &toCatalogListCallbackData}},
		},
	}
	_, err = client.Send(msg)
	if err != nil {
		return err
	}

	return nil
}
