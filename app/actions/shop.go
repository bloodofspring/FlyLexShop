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

// Shop представляет собой структуру для работы с магазином
// Name - имя команды
// Client - экземпляр Telegram бота
type Shop struct {
	Name   string
	Client tgbotapi.BotAPI
	mu     *sync.Mutex
}

func NewShopHandler(client tgbotapi.BotAPI) *Shop {
	return &Shop{
		Name:   "shop",
		Client: client,
		mu:     &sync.Mutex{},
	}
}

// Run запускает отображение каталогов магазина
// update - обновление от Telegram API
// Возвращает ошибку, если что-то пошло не так
func (s Shop) Run(update tgbotapi.Update) error {
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
			s.mu.Lock()
			ClearNextStepForUser(update, &s.Client, true)
			s.mu.Unlock()

			data := ParseCallData(update.CallbackQuery.Data)
			if catIdStr, ok := data["catId"]; ok {
				update.CallbackQuery.Data = "toCat?catId=" + catIdStr
				handler := NewViewCatalogHandler(s.Client)
				err = handler.Run(update)
				return
			}

			db := database.Connect()
			defer db.Close()

			var session models.ShopViewSession

			userDb := models.TelegramUser{ID: update.CallbackQuery.From.ID}
			err = userDb.Get(*db)
			if err != nil {
				return
			}
			err = db.Model(&userDb).
				WherePK().
				Relation("ShopSession").
				Relation("ShopSession.Catalog").
				Select()
			if err != nil {
				return
			}

			if userDb.ShopSession != nil {
				session = *userDb.ShopSession

				session.CatalogID = 0
				session.Catalog = nil
				session.ProductAtID = 0
				session.ProductAt = nil
				_, err = db.Model(&session).
					WherePK().
					Column("catalog_id").
					Column("product_at_id").
					Update()
				if err != nil {
					return
				}
			} else {
				session = models.ShopViewSession{
					UserID: update.CallbackQuery.From.ID,
					ChatID: update.CallbackQuery.Message.Chat.ID,
				}
				_, err = db.Model(&session).Insert()

				if err != nil {
					return
				}

				err = db.Model(&session).Where("id = ?", session.ID).Select()
				if err != nil {
					return
				}
			}

			catalogs := []models.Catalog{}
			err = db.Model(&catalogs).Order("created_at ASC").Select()
			if err != nil {
				return
			}

			keyboard := [][]tgbotapi.InlineKeyboardButton{}

			for _, cat := range catalogs {
				callbackData := fmt.Sprintf("toCat?catId=%d", cat.ID)
				var productCount int
				productCount, err = cat.GetProductCount(db)
				if err != nil {
					return
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

				var transaction models.Transaction
				transaction, err, _ = (&models.TelegramUser{ID: update.CallbackQuery.From.ID}).GetOrCreateTransaction(*db)
				if err != nil {
					return
				}

				err = db.Model(&transaction).
					WherePK().
					Relation("AddedProducts").
					Relation("AddedProducts.Product").
					Select()
				if err != nil {
					return
				}

				if len(transaction.AddedProducts) > 0 {
					toCartCallbackData := "viewCart?backIsMainMenu=true"

					var total int
					for _, item := range transaction.AddedProducts {
						total += item.Product.Price * item.ProductCount
					}

					keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{{Text: fmt.Sprintf("Корзина (%d₽)", total), CallbackData: &toCartCallbackData}})
				}
			}

			if userDb.IsAdmin {
				addCatalogCallbackData := "addCatalog"
				changeCatalogNameCallbackData := "changeCatalogName"
				keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
					{Text: "Добавить каталог", CallbackData: &addCatalogCallbackData},
				}, []tgbotapi.InlineKeyboardButton{
					{Text: "Изменить каталог", CallbackData: &changeCatalogNameCallbackData},
				})
			}

			toMainMenuCallbackData := "mainMenu"
			keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{{Text: "На главную", CallbackData: &toMainMenuCallbackData}})

			if update.CallbackQuery.Message.Caption != "" {
				s.mu.Lock()
				s.Client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))
				s.Client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))
				message := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, text)
				message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}
				_, err = s.Client.Send(message)
				s.mu.Unlock()
			} else {
				message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, text)
				message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}
				s.mu.Lock()
				_, err = s.Client.Send(message)
				s.mu.Unlock()
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
func (s Shop) GetName() string {
	return s.Name
}

// ViewCatalog представляет собой структуру для просмотра каталога
// Name - имя команды
// Client - экземпляр Telegram бота
type ViewCatalog struct {
	Name   string
	Client tgbotapi.BotAPI
	mu     *sync.Mutex
}

func NewViewCatalogHandler(client tgbotapi.BotAPI) *ViewCatalog {
	return &ViewCatalog{
		Name:   "viewCatalog",
		Client: client,
		mu:     &sync.Mutex{},
	}
}

// Run запускает отображение содержимого каталога
// update - обновление от Telegram API
// Возвращает ошибку, если что-то пошло не так
func (v ViewCatalog) Run(update tgbotapi.Update) error {
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

			var data map[string]string = filters.ParseCallbackData(update.CallbackQuery.Data)

			userDb := models.TelegramUser{ID: update.CallbackQuery.From.ID}
			err = userDb.Get(*db)
			if err != nil {
				return
			}
			err = db.Model(&userDb).
				WherePK().
				Relation("ShopSession").
				Relation("ShopSession.Catalog").
				Select()
			if err != nil {
				return
			}

			if catIdStr, ok := data["catId"]; ok {
				var catId int
				catId, err = strconv.Atoi(catIdStr)
				if err != nil {
					return
				}
				userDb.ShopSession.CatalogID = catId
				userDb.ShopSession.Catalog = &models.Catalog{ID: catId}
				err = db.Model(userDb.ShopSession.Catalog).Where("id = ?", catId).Select()
				if err != nil {
					return
				}
				_, err = db.Model(userDb.ShopSession).WherePK().Column("catalog_id").Update()
				if err != nil {
					return
				}
			}

			userDb.ShopSession.Catalog = &models.Catalog{ID: userDb.ShopSession.CatalogID}
			err = db.Model(userDb.ShopSession.Catalog).
				Where("id = ?", userDb.ShopSession.CatalogID).
				Select()
			if err != nil {
				return
			}

			var productCount int
			productCount, err = userDb.ShopSession.Catalog.GetProductCount(db)
			if err != nil {
				return
			}

			if productCount == 0 {
				text := "В этом каталоге пока что нет товаров"
				toListOfCats := "shop"
				keyboard := [][]tgbotapi.InlineKeyboardButton{
					{{Text: "К списку каталогов", CallbackData: &toListOfCats}},
				}

				if userDb.IsAdmin {
					removeCatalogCallbackData := "editShop?a=removeCatalog"
					addProductCallbackData := "editShop?a=createProduct"
					keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
						{Text: "Удалить каталог", CallbackData: &removeCatalogCallbackData},
						{Text: "Добавить товар", CallbackData: &addProductCallbackData},
					})
				}

				message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, text)
				message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}

				v.mu.Lock()
				_, err = v.Client.Send(message)
				v.mu.Unlock()

				if err != nil && err.Error() == "Bad Request: there is no text in the message to edit" {
					v.Client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))
					err = nil
				}

				if err != nil && (err.Error() == "Bad Request: message to edit not found" || err.Error() == "Bad Request: there is no text in the message to edit") {
					messageToSend := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, text)
					messageToSend.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}

					v.mu.Lock()
					_, err = v.Client.Send(messageToSend)
					v.mu.Unlock()
				}

				return
			}

			if pageDeltaStr, ok := data["pageDelta"]; ok {
				var pageDelta int
				pageDelta, err = strconv.Atoi(pageDeltaStr)
				if err != nil {
					return
				}

				userDb.ShopSession.Offest += pageDelta
			}

			_, err = db.Model(userDb.ShopSession).WherePK().Column("offest").Update()
			if err != nil {
				return
			}

			if userDb.ShopSession.Offest >= productCount {
				userDb.ShopSession.Offest = 0
				_, err = db.Model(userDb.ShopSession).WherePK().Column("offest").Update()
				if err != nil {
					return
				}
			} else if userDb.ShopSession.Offest < 0 {
				userDb.ShopSession.Offest = productCount - 1
				_, err = db.Model(userDb.ShopSession).WherePK().Column("offest").Update()
				if err != nil {
					return
				}
			}

			var item models.Product
			err = db.Model(&item).
				Where("catalog_id = ?", userDb.ShopSession.Catalog.ID).
				Order("created_at ASC").
				Offset(userDb.ShopSession.Offest).
				Limit(1).
				Select()
			if err != nil {
				return
			}

			userDb.ShopSession.ProductAtID = item.ID
			_, err = db.Model(userDb.ShopSession).WherePK().Column("product_at_id").Update()
			if err != nil {
				return
			}

			cartDelta, ok := data["cartDelta"]
			if ok {
				cartDeltaInt, err := strconv.Atoi(cartDelta)
				if err != nil {
					return
				}

				if cartDeltaInt == 1 {
					err = userDb.AddProductToCart(*db, item.ID)
					if err != nil {
						return
					}
				} else if cartDeltaInt == -1 {
					err = userDb.RemoveProductFromCart(*db, item.ID)
					if err != nil {
						return
					}
				}
			}

			keyboard := [][]tgbotapi.InlineKeyboardButton{}

			var cartChanged bool
			cartChanged, err = userDb.TidyCart(*db)
			if err != nil {
				return
			}

			if cartChanged {
				_, err := v.Client.Request(tgbotapi.CallbackConfig{
					CallbackQueryID: update.CallbackQuery.ID,
					Text:            "Количество некторых товаров уменьшилось. Проверьте корзину перед покупкой",
					ShowAlert:       true,
				})
				if err != nil {
					return
				}
			}

			var productInCartCount int
			productInCartCount, err = userDb.GetProductInCartCount(*db, item.ID)
			if err != nil {
				return
			}

			if ok, err := item.InUserCart(update.CallbackQuery.From.ID, *db); ok && err == nil && item.AvailbleForPurchase > 0 && productInCartCount != 0 {
				add1CallbackData := "toCat?cartDelta=1"
				rem1CallbackData := "toCat?cartDelta=-1"
				nullCallbackData := "<null>"

				buttonRow := []tgbotapi.InlineKeyboardButton{
					{Text: "-", CallbackData: &rem1CallbackData},
					{Text: fmt.Sprintf("%d/%d", productInCartCount, item.AvailbleForPurchase), CallbackData: &nullCallbackData},
				}

				if productInCartCount < item.AvailbleForPurchase {
					buttonRow = append(buttonRow, tgbotapi.InlineKeyboardButton{Text: "+", CallbackData: &add1CallbackData})
				}

				keyboard = append(keyboard, buttonRow)
			} else if err != nil {
				return
			} else if item.AvailbleForPurchase > 0 {
				callbackData := "toCat?cartDelta=1"
				keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
					{Text: "Добавить в корзину✅", CallbackData: &callbackData},
				})
			}

			if productCount > 1 {
				nextItemCallbackData := "toCat?pageDelta=1"
				noneCallbackData := "<null>"
				prevItemCallbackData := "toCat?pageDelta=-1"
				keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
					{Text: "⬅️", CallbackData: &prevItemCallbackData},
					{Text: fmt.Sprintf("%s/%s", NumberToEmoji(userDb.ShopSession.Offest+1), NumberToEmoji(productCount)), CallbackData: &noneCallbackData},
					{Text: "➡️", CallbackData: &nextItemCallbackData},
				})
			}

			var totalPrice int
			totalPrice, err = userDb.GetTotalCartPrice(*db)
			if err != nil {
				return
			}

			if userDb.IsAdmin {
				var (
					removeCatalogCallbackData             = "editShop?a=removeCatalog"
					removeProductCallbackData             = "editShop?a=removeProduct"
					changePhotoCallbackData               = "editShop?a=changePhoto"
					changePriceCallbackData               = "editShop?a=changePrice"
					changeNameCallbackData                = "editShop?a=changeName"
					changeDescriptionCallbackData         = "editShop?a=changeDescription"
					addProductCallbackData                = "editShop?a=createProduct"
					changeAvailbleForPurchaseCallbackData = "editShop?a=changeAvailbleForPurchase"
				)
				keyboard = append(
					keyboard,
					[]tgbotapi.InlineKeyboardButton{
						{Text: "Удалить каталог", CallbackData: &removeCatalogCallbackData},
						{Text: "Удалить товар", CallbackData: &removeProductCallbackData},
					},
					[]tgbotapi.InlineKeyboardButton{
						{Text: "Изменить фото", CallbackData: &changePhotoCallbackData},
						{Text: "Изменить цену", CallbackData: &changePriceCallbackData},
					},
					[]tgbotapi.InlineKeyboardButton{
						{Text: "Изменить название", CallbackData: &changeNameCallbackData},
						{Text: "Изменить описание", CallbackData: &changeDescriptionCallbackData},
					},
					[]tgbotapi.InlineKeyboardButton{
						{Text: "Добавить товар", CallbackData: &addProductCallbackData},
						{Text: "Изменить кол-во товаров", CallbackData: &changeAvailbleForPurchaseCallbackData},
					},
				)
			}

			toListOfCats := "shop"
			toCart := "viewCart"
			keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{{Text: "К списку каталогов", CallbackData: &toListOfCats}, {Text: fmt.Sprintf("Корзина (%d₽)", totalPrice), CallbackData: &toCart}})

			var availablityContent string
			if item.AvailbleForPurchase > 0 {
				availablityContent = fmt.Sprintf("В наличии: %d шт.", item.AvailbleForPurchase)
			} else {
				availablityContent = "Нет в наличии❌"
			}

			content := fmt.Sprintf("<b>%s</b>\nЦена: %d₽\n%s\n\n%s", item.Name, item.Price, availablityContent, item.Description)

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
func (v ViewCatalog) GetName() string {
	return v.Name
}
