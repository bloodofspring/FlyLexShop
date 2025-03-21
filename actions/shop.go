package actions

import (
	"fmt"
	"main/database"
	"main/database/models"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Shop struct {
	Name   string
	Client tgbotapi.BotAPI
}

func (s Shop) Run(update tgbotapi.Update) error {
	db := database.Connect()
	defer db.Close()

	catalogs := []models.Catalog{}
	err := db.Model(&catalogs).Select()
	if err != nil {
		return err
	}

	keyboard := [][]tgbotapi.InlineKeyboardButton{}

	for _, cat := range catalogs {
		callbackData := fmt.Sprintf("toCat?id=%d", cat.ID)
		productCount, err := cat.GetProductCount(*db)
		if err != nil {
			return err
		}

		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			{Text: cat.Name + " (" + strconv.Itoa(productCount) + ")", CallbackData: &callbackData},
		})
	}

	var text string
	if len(catalogs) == 0 {
		text = "Пока что каталогов не добавлено"
	} else {
		text = "Выберите каталог"
	}

	message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, text)

	toMainMenuCallbackData := "mainMenu"
	message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: append(keyboard, []tgbotapi.InlineKeyboardButton{{Text: "На главную", CallbackData: &toMainMenuCallbackData}}),
	}

	_, err = s.Client.Send(message)
	
	return err
}

func (s Shop) GetName() string {
	return s.Name
}