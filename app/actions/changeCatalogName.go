package actions

import (
	"context"
	"fmt"
	"main/controllers"
	"main/database"
	"main/database/models"
	"strconv"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ChangeCatalogName struct {
	Name   string
	Client tgbotapi.BotAPI
	mu     *sync.Mutex
}

func NewChangeCatalogNameHandler(client tgbotapi.BotAPI) *ChangeCatalogName {
	return &ChangeCatalogName{
		Name:   "changeCatalogName",
		Client: client,
		mu:     &sync.Mutex{},
	}
}

func ChangeCatalogNameStep(client tgbotapi.BotAPI, stepUpdate tgbotapi.Update, stepParams map[string]any) error {
	if stepUpdate.Message == nil || stepUpdate.Message.Text == "" {
		message := tgbotapi.NewMessage(stepUpdate.Message.Chat.ID, "Введите новое название каталога")
		toListofCats := "shop"
		message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
				{{Text: "Отмена", CallbackData: &toListofCats}},
			},
		}
		client.Send(message)

		stepKey := controllers.NextStepKey{
			UserID: stepUpdate.Message.From.ID,
			ChatID: stepUpdate.Message.Chat.ID,
		}
		
		stepAction := controllers.NextStepAction{
			Func:          ChangeCatalogNameStep,
			Params:        stepParams,
			CreatedAtTS:   time.Now().Unix(),
			CancelMessage: "Изменение названия каталога отменено",
		}
		
		controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)

		return nil
	}

	db := database.Connect()
	defer db.Close()

	catalogId, ok := stepParams["catalogId"]
	if !ok {
		return fmt.Errorf("catalogId is required")
	}

	catalogIdInt, err := strconv.Atoi(catalogId.(string))
	if err != nil {
		return err
	}

	var catalog models.Catalog
	_, err = db.Model(&catalog).
		Where("id = ?", catalogIdInt).
		Set("name = ?", stepUpdate.Message.Text).
		Update()
	if err != nil {
		return err
	}

	text := fmt.Sprintf("Название каталога изменено на %s", stepUpdate.Message.Text)
	message := tgbotapi.NewMessage(stepUpdate.Message.Chat.ID, text)
	toListofCats := "shop"
	message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{{Text: "К списку каталогов", CallbackData: &toListofCats}},
		},
	}
	client.Send(message)

	return nil	
}

func (c ChangeCatalogName) Run(update tgbotapi.Update) error {
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
			ClearNextStepForUser(update, &c.Client, true)
			c.mu.Unlock()

			db := database.Connect()
			defer db.Close()

			data := ParseCallData(update.CallbackQuery.Data)
			catalogId, ok := data["catalogId"]
			
			if !ok {
				message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, "Выберите каталог:")

				catalogs := []models.Catalog{}
				err = db.Model(&catalogs).Order("created_at ASC").Select()
				if err != nil {
					return
				}

				keyboard := [][]tgbotapi.InlineKeyboardButton{}

				for _, cat := range catalogs {
					callbackData := fmt.Sprintf("changeCatalogName?catalogId=%d", cat.ID)

					keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
						{Text: cat.Name, CallbackData: &callbackData},
					})
				}

				toListofCats := "shop"
				keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
					{Text: "Отмена", CallbackData: &toListofCats},
				})

				message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
					InlineKeyboard: keyboard,
				}
				
				c.mu.Lock()
				_, err = c.Client.Send(message)
				c.mu.Unlock()

				return
			}

			const text = "Введите новое название каталога"
			message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, text)
			toListofCats := "shop"
			message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					{{Text: "Отмена", CallbackData: &toListofCats}},
				},
			}

			c.mu.Lock()
			_, err = c.Client.Send(message)
			c.mu.Unlock()

			stepKey := controllers.NextStepKey{
				UserID: update.CallbackQuery.From.ID,
				ChatID: update.CallbackQuery.Message.Chat.ID,
			}
	
			stepAction := controllers.NextStepAction{
				Func:          ChangeCatalogNameStep,
				Params:        map[string]interface{}{"catalogId": catalogId},
				CreatedAtTS:   time.Now().Unix(),
				CancelMessage: "Изменение названия каталога отменено",
			}
	
			c.mu.Lock()
			controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)
			c.mu.Unlock()

			return
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

func (c ChangeCatalogName) GetName() string {
	return c.Name
}