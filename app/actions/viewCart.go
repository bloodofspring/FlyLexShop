package actions

import (
	"context"
	"fmt"
	"main/database"
	"main/database/models"
	"main/filters"
	"strconv"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ViewCart представляет собой структуру для просмотра корзины
// Name - имя команды
// Client - экземпляр Telegram бота
type ViewCart struct {
	Name   string
	Client tgbotapi.BotAPI
	mu     *sync.Mutex
}

func NewViewCartHandler(client tgbotapi.BotAPI) *ViewCart {
	return &ViewCart{
		Name:   "view-cart",
		Client: client,
		mu:     &sync.Mutex{},
	}
}

// Run запускает отображение содержимого корзины
// update - обновление от Telegram API
// Возвращает ошибку, если что-то пошло не так
func (v ViewCart) Run(update tgbotapi.Update) error {
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
			v.mu.Lock()
			ClearNextStepForUser(update, &v.Client, true)
			v.mu.Unlock()

			db := database.Connect()
			defer db.Close()

			pars := filters.ParseCallbackData(update.CallbackQuery.Data)
			backIsMainMenu := pars["backIsMainMenu"] == "true"

			itemIdStr, ok := pars["itemId"]
			if !ok {
				itemIdStr = "0"
			}

			var itemId int
			itemId, err = strconv.Atoi(itemIdStr)
			if err != nil {
				return
			}

			var items []models.Product
			err = db.Model(&items).Where("id IN (SELECT product_id FROM shopping_carts WHERE user_id = ?)", update.CallbackQuery.From.ID).Select()
			if err != nil {
				return
			}

			if len(items) == 0 {
				v.mu.Lock()
				_, err = v.Client.Request(tgbotapi.CallbackConfig{
					CallbackQueryID: update.CallbackQuery.ID,
					Text:            "В корзине пока что нет товаров",
				})
				v.mu.Unlock()

				return
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
					return
				}

				err = db.Model(&items).Where("id IN (SELECT product_id FROM shopping_carts WHERE user_id = ?)", update.CallbackQuery.From.ID).Select()
				if err != nil {
					return
				}

				if len(items) == 0 {
					v.mu.Lock()
					_, err = v.Client.Request(tgbotapi.CallbackConfig{
						CallbackQueryID: update.CallbackQuery.ID,
						Text:            "Теперь ваша карзина пуста",
						ShowAlert:       true,
					})
					v.mu.Unlock()
					if err != nil {
						return
					}

					handler := NewShopHandler(v.Client)
					handler.mu = v.mu
					err = handler.Run(update)
					return
				}

				if itemId >= len(items) {
					itemId = len(items) - 1
				}

				update.CallbackQuery.Data = fmt.Sprintf("viewCart?itemId=%d&backIsMainMenu=%t", itemId, backIsMainMenu)

				handler := NewViewCartHandler(v.Client)
				handler.mu = v.mu
				err = handler.Run(update)
				return
			}

			keyboard := [][]tgbotapi.InlineKeyboardButton{}

			ok, err = items[itemId].InUserCart(update.CallbackQuery.From.ID, *db)
			if ok && err == nil {
				callbackData := fmt.Sprintf("viewCart?itemId=%d&remove=true&backIsMainMenu=%t", itemId, backIsMainMenu)
				keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
					{Text: "Удалить из корзины❌", CallbackData: &callbackData},
				})
			} else if err != nil {
				return
			}

			if len(items) > 1 {
				nextItemCallbackData := fmt.Sprintf("viewCart?itemId=%d&backIsMainMenu=%t", itemId+1, backIsMainMenu)
				noneCallbackData := "<null>"
				prevItemCallbackData := fmt.Sprintf("viewCart?itemId=%d&backIsMainMenu=%t", itemId-1, backIsMainMenu)
				keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
					{Text: "⬅️", CallbackData: &prevItemCallbackData},
					{Text: fmt.Sprintf("%s/%s", NumberToEmoji(itemId+1), NumberToEmoji(len(items))), CallbackData: &noneCallbackData},
					{Text: "➡️", CallbackData: &nextItemCallbackData},
				})
			}

			var toShop string
			var buttonText string
			var userDb models.TelegramUser
			err = db.Model(&userDb).
				Where("user_id = ?", update.CallbackQuery.From.ID).
				Relation("ShopSession").
				Select()
			if err == nil && !backIsMainMenu {
				toShop = "shop?catId=" + strconv.Itoa(userDb.ShopSession.CatalogID)
				buttonText = "К списку товаров"
			} else {
				toShop = "shop"
				buttonText = "К списку каталогов"
			}

			makeOrder := "makeOrder"
			keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{{Text: buttonText, CallbackData: &toShop}, {Text: "Оформить заказ✅", CallbackData: &makeOrder}})

			content := fmt.Sprintf("<b>%s</b>\nЦена: %d₽\n\n%s", item.Name, item.Price, item.Description)

			if update.CallbackQuery.Message.Caption != "" {
				editMeida := tgbotapi.EditMessageMediaConfig{
					BaseEdit: tgbotapi.BaseEdit{
						ChatID:    update.CallbackQuery.Message.Chat.ID,
						MessageID: update.CallbackQuery.Message.MessageID,
					},
					Media: tgbotapi.NewInputMediaPhoto(tgbotapi.FileID(item.ImageFileID)),
				}
				v.mu.Lock()
				_, err = v.Client.Send(editMeida)
				v.mu.Unlock()
				if err != nil {
					return
				}

				editCaption := tgbotapi.NewEditMessageCaption(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, content)
				editCaption.ParseMode = "HTML"
				editCaption.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}
				v.mu.Lock()
				_, err = v.Client.Send(editCaption)
				v.mu.Unlock()
				if err != nil {
					return
				}

			} else {
				v.mu.Lock()
				v.Client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))
				v.mu.Unlock()

				photoMsg := tgbotapi.NewPhoto(update.CallbackQuery.Message.Chat.ID, tgbotapi.FileID(item.ImageFileID))
				photoMsg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}
				photoMsg.Caption = content
				photoMsg.ParseMode = "HTML"

				v.mu.Lock()
				_, err = v.Client.Send(photoMsg)
				v.mu.Unlock()
				if err != nil {
					return
				}
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

// GetName возвращает имя команды
func (v ViewCart) GetName() string {
	return v.Name
}
