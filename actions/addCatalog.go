package actions

import (
	"context"
	"fmt"
	"main/controllers"
	"main/database"
	"main/database/models"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// AddCatalog управляет процессом добавления нового каталога.
// Name - имя команды.
// Client - экземпляр Telegram бота.
type AddCatalog struct {
	Name   string
	Client tgbotapi.BotAPI
	mu     *sync.Mutex
}

var (
	cancelCallbackData = "cancel"
)

func NewAddCatalogHandler(client tgbotapi.BotAPI) *AddCatalog {
	return &AddCatalog{
		Name:   "addCatalog",
		Client: client,
		mu:     &sync.Mutex{},
	}
}

// Run отправляет запрос ввода названия каталога и регистрирует следующий шаг создания каталога.
func (a AddCatalog) Run(update tgbotapi.Update) error {
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
			msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Введите название каталога")
			msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					{{Text: "Отменить", CallbackData: &cancelCallbackData}},
				},
			}
			a.mu.Lock()
			_, err = a.Client.Send(msg)
			a.mu.Unlock()
			if err != nil {
				return
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

			a.mu.Lock()
			controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)
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

// GetName возвращает имя команды AddCatalog.
func (a AddCatalog) GetName() string {
	return a.Name
}

// CreateCatalog обрабатывает ввод названия каталога и сохраняет новый каталог в базе данных.
func CreateCatalog(client tgbotapi.BotAPI, stepUpdate tgbotapi.Update, stepParams map[string]any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var mu sync.Mutex
	var wg sync.WaitGroup
	var err error

	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			return
		default:
			if stepUpdate.Message.Text == "" {
				msg := tgbotapi.NewMessage(stepUpdate.Message.Chat.ID, "Введите название каталога")
				msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
					InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
						{{Text: "Отменить", CallbackData: &cancelCallbackData}},
					},
				}

				mu.Lock()
				_, err = client.Send(msg)
				mu.Unlock()
				if err != nil {
					return
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
		
				mu.Lock()
				controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)
				mu.Unlock()

				return
			}
		
			db := database.Connect()
			defer db.Close()
		
			_, err = db.Model(&models.Catalog{
				Name: stepUpdate.Message.Text,
			}).Insert()
			if err != nil {
				return
			}

			mu.Lock()
			ClearNextStepForUser(stepUpdate, &client, false)
			mu.Unlock()
		
			msg := tgbotapi.NewMessage(stepUpdate.Message.Chat.ID, fmt.Sprintf("Каталог с названием \"%s\" успешно создан", stepUpdate.Message.Text))
			toCatalogListCallbackData := "shop"
			msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					{{Text: "К списку каталогов", CallbackData: &toCatalogListCallbackData}},
				},
			}
			mu.Lock()
			_, err = client.Send(msg)
			mu.Unlock()
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
