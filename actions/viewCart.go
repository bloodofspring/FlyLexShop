package actions

import (
	"encoding/json"
	"fmt"
	"log"
	"main/database"
	"main/database/models"
	"main/filters"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ViewCart struct {
	Name   string
	Client tgbotapi.BotAPI
}

// ToDo: Отсмотреть это дерьмо со свежей головой
func (v ViewCart) Run(update tgbotapi.Update) error {
	db := database.Connect()
	defer db.Close()

	pars := filters.ParseCallbackData(update.CallbackQuery.Data)

	itemIdStr, ok := pars["itemId"]
	if !ok {
		itemIdStr = "0"
	}

	itemId, err := strconv.Atoi(itemIdStr)
	if err != nil {
		return err
	}

	var items []models.Product
	err = db.Model(&items).Where("id IN (SELECT product_id FROM shopping_carts WHERE user_id = ?)", update.CallbackQuery.From.ID).Select()
	if err != nil {
		return err
	}

	if len(items) == 0 {
		_, err = v.Client.Send(tgbotapi.CallbackConfig{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "В корзине пока что нет товаров",
		})
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

	_, ok = pars["remove"]
	if ok {
		_, err = db.Model(&models.ShoppingCart{}).Where("user_id = ?", update.CallbackQuery.From.ID).Where("product_id = ?", item.ID).Delete()
		if err != nil {
			return err
		}

		if err = db.Model(&items).Where("id IN (SELECT product_id FROM shopping_carts WHERE user_id = ?)", update.CallbackQuery.From.ID).Select(); err != nil {
			if len(items) == 0 {
				_, err = v.Client.Send(tgbotapi.CallbackConfig{
					CallbackQueryID: update.CallbackQuery.ID,
					Text:            "Теперь ваша карзина пуста",
					ShowAlert:       true,
				})
				if err != nil {
					return err
				}

				err = Shop{Name: "reset-to-shop", Client: v.Client}.Run(update)

				return err
			}
		} else {
			return err
		}

		printUpdate := func(update *tgbotapi.Update) {
			updateJSON, err := json.MarshalIndent(update, "", "    ")
			if err != nil {
				return
			}
		
			log.Println(string(updateJSON))
		}
		printUpdate(&update)
		err = ViewCart{Name: "view-cart", Client: v.Client}.Run(update)

		return err
	}

	keyboard := [][]tgbotapi.InlineKeyboardButton{}

	if ok, err := items[itemId].InUserCart(update.CallbackQuery.From.ID, *db); ok && err == nil {
		callbackData := fmt.Sprintf("viewCart?itemId=%d&remove=1", itemId)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			{Text: "Удалить из корзины", CallbackData: &callbackData},
		})
	} else if err != nil {
		return err
	}

	if len(items) > 1 {
		nextItemCallbackData := fmt.Sprintf("viewCart?itemId=%d", itemId+1)
		prevItemCallbackData := fmt.Sprintf("viewCart?itemId=%d", itemId-1)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			{Text: "<--<<", CallbackData: &prevItemCallbackData},
			{Text: ">>-->", CallbackData: &nextItemCallbackData},
		})
	}

	toShop := "shop"
	makeOrder := "makeOrder"
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{{Text: "К списку каталогов", CallbackData: &toShop}, {Text: "Оформить заказ", CallbackData: &makeOrder}})


	content := fmt.Sprintf("<b>%s</b>\nЦена: %d₽\n\n%s", item.Name, item.Price, item.Description)

	if update.CallbackQuery.Message.Caption != "" {
		editMeida := tgbotapi.EditMessageMediaConfig{
			BaseEdit: tgbotapi.BaseEdit{
				ChatID:    update.CallbackQuery.Message.Chat.ID,
				MessageID: update.CallbackQuery.Message.MessageID,
			},
			Media: tgbotapi.NewInputMediaPhoto(tgbotapi.FileID(item.ImageFileID)),
		}
		_, err = v.Client.Send(editMeida)
		if err != nil {
			return err
		}

		editCaption := tgbotapi.NewEditMessageCaption(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, content)
		editCaption.ParseMode = "HTML"
		editCaption.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}

		_, err = v.Client.Send(editCaption)
		if err != nil {
			return err
		}

	} else {
		v.Client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))
		photoMsg := tgbotapi.NewPhoto(update.CallbackQuery.Message.Chat.ID, tgbotapi.FileID(item.ImageFileID))
		photoMsg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}
		photoMsg.Caption = content
		photoMsg.ParseMode = "HTML"

		_, err = v.Client.Send(photoMsg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (v ViewCart) GetName() string {
	return v.Name
}

