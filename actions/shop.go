package actions

import (
	"fmt"
	"main/database"
	"main/database/models"
	"main/filters"
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

	toMainMenuCallbackData := "mainMenu"
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{{Text: "На главную", CallbackData: &toMainMenuCallbackData}})

	if update.CallbackQuery.Message.Caption != "" {
		s.Client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))
		message := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, text)
		message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}
		_, err = s.Client.Send(message)
	} else {
		message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, text)
		message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}
		_, err = s.Client.Send(message)
	}

	return err
}

func (s Shop) GetName() string {
	return s.Name
}

type ViewCatalog struct {
	Name   string
	Client tgbotapi.BotAPI
}

func (v ViewCatalog) Run(update tgbotapi.Update) error {
	db := database.Connect()
	defer db.Close()

	pars := filters.ParseCallbackData(update.CallbackQuery.Data)
	catId, err := strconv.Atoi(pars["id"])
	if err != nil {
		return err
	}

	itemIdStr, ok := pars["itemId"]
	if !ok {
		itemIdStr = "0"
	}

	itemId, err := strconv.Atoi(itemIdStr)
	if err != nil {
		return err
	}

	var items []models.Product
	err = db.Model(&items).Where("catalog_id = ?", catId).Select()
	if err != nil {
		return err
	}

	if len(items) == 0 {
		message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, "В этом каталоге пока что нет товаров")
		toListOfCats := "shop"
		message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{{Text: "К списку каталогов", CallbackData: &toListOfCats}}},
		}
		_, err = v.Client.Send(message)
		if err != nil {
			return err
		}

		return nil
	} else if itemId >= len(items) {
		itemId = 0
	} else if itemId < 0 {
		itemId = len(items) - 1
	}

	item := items[itemId]

	keyboard := [][]tgbotapi.InlineKeyboardButton{}

	if ok, err := items[itemId].InUserCart(update.CallbackQuery.From.ID, *db); ok && err == nil {
		callbackData := fmt.Sprintf("removeFromCart?itemId=%d", item.ID)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			{Text: "Удалить из корзины", CallbackData: &callbackData},
		})
	} else if err != nil {
		return err
	} else {
		callbackData := fmt.Sprintf("addToCart?itemId=%d", item.ID)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			{Text: "Добавить в корзину", CallbackData: &callbackData},
		})
	}

	if len(items) > 1 {
		nextItemCallbackData := fmt.Sprintf("toCat?id=%d&itemId=%d", catId, itemId+1)
		prevItemCallbackData := fmt.Sprintf("toCat?id=%d&itemId=%d", catId, itemId-1)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			{Text: "<--<<", CallbackData: &prevItemCallbackData},
			{Text: ">>-->", CallbackData: &nextItemCallbackData},
		})
	}

	toListOfCats := "shop"
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{{Text: "К списку каталогов", CallbackData: &toListOfCats}})

	v.Client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))
	if update.CallbackQuery.Message.Caption != "" {
		message := tgbotapi.NewEditMessageCaption(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, update.CallbackQuery.Message.Caption)
		message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}

		_, err = v.Client.Send(message)
		if err != nil {
			return err
		}
	} else {
		photoMsg := tgbotapi.NewPhoto(update.CallbackQuery.Message.Chat.ID, tgbotapi.FileID(item.ImageFileID))
		photoMsg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}
		photoMsg.Caption = item.Description
		photoMsg.ParseMode = "HTML"

		_, err = v.Client.Send(photoMsg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (v ViewCatalog) GetName() string {
	return v.Name
}
