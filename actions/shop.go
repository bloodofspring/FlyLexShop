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
	ClearNextStepForUser(update, &s.Client, true)

	db := database.Connect()
	defer db.Close()

	var data map[string]string = filters.ParseCallbackData(update.CallbackQuery.Data)
	var session models.ShopViewSession

	if sessionIdStr, ok := data["sessionId"]; ok {
		sessionId, err := strconv.Atoi(sessionIdStr)
		if err != nil {
			return err
		}

		err = db.Model(&session).Where("id = ?", sessionId).Select()
		if err != nil {
			return err
		}
	} else {
		session = models.ShopViewSession{
			UserId: update.CallbackQuery.From.ID,
			ChatId: update.CallbackQuery.Message.Chat.ID,
		}
		_, err := db.Model(&session).Insert()

		if err != nil {
			return err
		}

		err = db.Model(&session).Where("id = ?", session.Id).Select()
		if err != nil {
			return err
		}
	}

	if showCatStr, ok := data["showCat"]; ok {
		showCat, err := strconv.ParseBool(showCatStr)
		if err == nil && !showCat {
			if session.CatId != 0 {
				s.Client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))
				return ViewCatalog{Name: "viewCatalog", Client: s.Client}.Run(update)
			}
		}
	}

	catalogs := []models.Catalog{}
	err := db.Model(&catalogs).Select()
	if err != nil {
		return err
	}

	keyboard := [][]tgbotapi.InlineKeyboardButton{}

	for _, cat := range catalogs {
		callbackData := fmt.Sprintf("toCat?catId=%d&sessionId=%d", cat.ID, session.Id)
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

	userDb := models.TelegramUser{ID: update.CallbackQuery.From.ID}
	err = userDb.Get(*db)
	if err != nil {
		return err
	}

	if userDb.IsAdmin {
		addCatalogCallbackData := "addCatalog"
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{{Text: "Добавить каталог", CallbackData: &addCatalogCallbackData}})
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
	ClearNextStepForUser(update, &v.Client, true)
	db := database.Connect()
	defer db.Close()

	var data map[string]string = filters.ParseCallbackData(update.CallbackQuery.Data)
	var session models.ShopViewSession

	if sessionIdStr, ok := data["sessionId"]; ok {
		sessionId, err := strconv.Atoi(sessionIdStr)
		if err != nil {
			return err
		}

		err = db.Model(&session).Where("id = ?", sessionId).Select()
		if err != nil {
			return err
		}

		if catIdStr, ok := data["catId"]; ok {
			catId, err := strconv.Atoi(catIdStr)
			if err != nil {
				return err
			}
			session.CatId = catId
		}

		if pageDeltaStr, ok := data["pageDelta"]; ok {
			pageDelta, err := strconv.Atoi(pageDeltaStr)
			if err != nil {
				return err
			}
			session.ProductAtId += pageDelta
		}

		_, err = db.Model(&session).Where("id = ?", sessionId).Column("cat_id").Column("product_at_id").Update()
		if err != nil {
			return err
		}
	} else {
		return Shop{Name: "shop", Client: v.Client}.Run(update)
	}

	var items []models.Product
	err := db.Model(&items).Where("catalog_id = ?", session.CatId).Select()
	if err != nil {
		return err
	}

	userDb := models.TelegramUser{ID: update.CallbackQuery.From.ID}
	err = userDb.Get(*db)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, "В этом каталоге пока что нет товаров")
		toListOfCats := "shop?showCat=true"
		message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{{Text: "К списку каталогов", CallbackData: &toListOfCats}}},
		}

		if userDb.IsAdmin {
			removeCatalogCallbackData := fmt.Sprintf("editShop?a=removeCatalog&sessionId=%d", session.Id)
			addProductCallbackData := fmt.Sprintf("editShop?a=createProduct&sessionId=%d", session.Id)
			message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: append(message.ReplyMarkup.InlineKeyboard, []tgbotapi.InlineKeyboardButton{
					{Text: "Удалить каталог", CallbackData: &removeCatalogCallbackData},
					{Text: "Добавить товар", CallbackData: &addProductCallbackData},
				}),
			}
		}

		_, err = v.Client.Send(message)
		if err != nil {
			return err
		}

		return nil
	} else if session.ProductAtId >= len(items) {
		session.ProductAtId = 0
	} else if session.ProductAtId < 0 {
		session.ProductAtId = len(items) - 1
	}

	_, err = db.Model(&session).Where("id = ?", session.Id).Column("product_at_id").Update()
	if err != nil {
		return err
	}

	item := items[session.ProductAtId]

	remove, ok := data["remove"]
	if ok {
		removeBool, err := strconv.ParseBool(remove)
		if err != nil {
			return err
		}

		if removeBool {
			_, err = db.Model(&models.ShoppingCart{}).Where("user_id = ?", update.CallbackQuery.From.ID).Where("product_id = ?", item.ID).Delete()
			if err != nil {
				return err
			}
		} else {
			_, err = db.Model(&models.ShoppingCart{
				UserID:    update.CallbackQuery.From.ID,
				ProductID: item.ID,
			}).Insert()
			if err != nil {
				return err
			}
		}
	}

	keyboard := [][]tgbotapi.InlineKeyboardButton{}

	if ok, err := items[session.ProductAtId].InUserCart(update.CallbackQuery.From.ID, *db); ok && err == nil {
		callbackData := "toCat?remove=true&sessionId=" + strconv.Itoa(session.Id)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			{Text: "Удалить из корзины", CallbackData: &callbackData},
		})
	} else if err != nil {
		return err
	} else {
		callbackData := "toCat?remove=false&sessionId=" + strconv.Itoa(session.Id)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			{Text: "Добавить в корзину", CallbackData: &callbackData},
		})
	}

	if len(items) > 1 {
		nextItemCallbackData := fmt.Sprintf("toCat?pageDelta=1&sessionId=%d", session.Id)
		prevItemCallbackData := fmt.Sprintf("toCat?pageDelta=-1&sessionId=%d", session.Id)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			{Text: "⬅️", CallbackData: &prevItemCallbackData},
			{Text: "➡️", CallbackData: &nextItemCallbackData},
		})
	}

	totalPrice, err := userDb.GetTotalCartPrice(*db)
	if err != nil {
		return err
	}

	if userDb.IsAdmin {
		var (
			removeCatalogCallbackData = fmt.Sprintf("editShop?a=removeCatalog&sessionId=%d", session.Id)
			removeProductCallbackData = fmt.Sprintf("editShop?a=removeProduct&productId=%d&sessionId=%d", item.ID, session.Id)
			changePhotoCallbackData = fmt.Sprintf("editShop?a=changePhoto&productId=%d&sessionId=%d", item.ID, session.Id)
			changePriceCallbackData = fmt.Sprintf("editShop?a=changePrice&productId=%d&sessionId=%d", item.ID, session.Id)
			changeNameCallbackData = fmt.Sprintf("editShop?a=changeName&productId=%d&sessionId=%d", item.ID, session.Id)
			changeDescriptionCallbackData = fmt.Sprintf("editShop?a=changeDescription&productId=%d&sessionId=%d", item.ID, session.Id)
			addProductCallbackData = fmt.Sprintf("editShop?a=createProduct&sessionId=%d", session.Id)
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
			},
		)
	}

	toListOfCats := "shop"
	toCart := "viewCart"
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{{Text: "К списку каталогов", CallbackData: &toListOfCats}, {Text: fmt.Sprintf("Корзина (%d₽)", totalPrice), CallbackData: &toCart}})

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

func (v ViewCatalog) GetName() string {
	return v.Name
}
