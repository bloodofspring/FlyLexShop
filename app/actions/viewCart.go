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

			var cartItems []models.AddedProducts
			err = db.Model(&cartItems).Where("user_id = ?", update.CallbackQuery.From.ID).Relation("Product").Select()
			if err != nil {
				return
			}

			if len(cartItems) == 0 {
				v.mu.Lock()
				_, err = v.Client.Request(tgbotapi.CallbackConfig{
					CallbackQueryID: update.CallbackQuery.ID,
					Text:            "В корзине пока что нет товаров",
				})
				v.mu.Unlock()
				return
			}

			if itemId >= len(cartItems) {
				itemId = 0
			} else if itemId < 0 {
				itemId = len(cartItems) - 1
			}

			cartItem := cartItems[itemId]
			item := cartItem.Product
			if item == nil {
				return
			}

			// Обработка изменения количества товара
			if deltaStr, ok := pars["cartDelta"]; ok {
				var delta int
				delta, err = strconv.Atoi(deltaStr)
				if err == nil {
					user := models.TelegramUser{ID: update.CallbackQuery.From.ID}
					if delta == 1 {
						err = user.AddProductToCart(*db, item.ID)
					} else if delta == -1 {
						err = user.RemoveProductFromCart(*db, item.ID)
					}
					if err != nil {
						return
					}
					// Перезапускаем обработчик для обновления отображения
					update.CallbackQuery.Data = fmt.Sprintf("viewCart?itemId=%d&backIsMainMenu=%t", itemId, backIsMainMenu)
					handler := NewViewCartHandler(v.Client)
					handler.mu = v.mu
					err = handler.Run(update)
					return
				}
			}

			// Клавиатура управления количеством
			keyboard := [][]tgbotapi.InlineKeyboardButton{}
			add1CallbackData := fmt.Sprintf("viewCart?itemId=%d&cartDelta=1&backIsMainMenu=%t", itemId, backIsMainMenu)
			rem1CallbackData := fmt.Sprintf("viewCart?itemId=%d&cartDelta=-1&backIsMainMenu=%t", itemId, backIsMainMenu)
			nullCallbackData := "<null>"
			countBtn := tgbotapi.InlineKeyboardButton{
				Text:         fmt.Sprintf("%s/%s", NumberToEmoji(cartItem.ProductCount), NumberToEmoji(item.AvailbleForPurchase)),
				CallbackData: &nullCallbackData,
			}
			row := []tgbotapi.InlineKeyboardButton{
				{Text: "-", CallbackData: &rem1CallbackData},
				countBtn,
				{Text: "+", CallbackData: &add1CallbackData},
			}
			keyboard = append(keyboard, row)

			// Навигация по товарам в корзине
			if len(cartItems) > 1 {
				nextItemCallbackData := fmt.Sprintf("viewCart?itemId=%d&backIsMainMenu=%t", itemId+1, backIsMainMenu)
				noneCallbackData := "<null>"
				prevItemCallbackData := fmt.Sprintf("viewCart?itemId=%d&backIsMainMenu=%t", itemId-1, backIsMainMenu)
				keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
					{Text: "⬅️", CallbackData: &prevItemCallbackData},
					{Text: fmt.Sprintf("%s/%s", NumberToEmoji(itemId+1), NumberToEmoji(len(cartItems))), CallbackData: &noneCallbackData},
					{Text: "➡️", CallbackData: &nextItemCallbackData},
				})
			}

			// Кнопки возврата и оформления заказа
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
