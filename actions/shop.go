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

	userDb := models.TelegramUser{ID: update.CallbackQuery.From.ID}
	err := userDb.Get(*db)
	if err != nil {
		return err
	}
	err = db.Model(&userDb).
		WherePK().
		Relation("ShopSession").
		Relation("ShopSession.Catalog").
		Select()
	if err != nil {
		return err
	}

	if userDb.ShopSession != nil {
		session = *userDb.ShopSession
		fmt.Println("session: ", session)
	} else {
		session = models.ShopViewSession{
			UserID: update.CallbackQuery.From.ID,
			ChatID: update.CallbackQuery.Message.Chat.ID,
		}
		_, err := db.Model(&session).Insert()

		if err != nil {
			return err
		}

		err = db.Model(&session).Where("id = ?", session.ID).Select()
		if err != nil {
			return err
		}
	}

	if showCatStr, ok := data["showCat"]; ok {
		showCat, err := strconv.ParseBool(showCatStr)
		if err == nil && !showCat {
			if session.CatalogID != 0 {
				s.Client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))
				return ViewCatalog{Name: "viewCatalog", Client: s.Client}.Run(update)
			}
		}
	}

	catalogs := []models.Catalog{}
	err = db.Model(&catalogs).Select()
	if err != nil {
		return err
	}

	keyboard := [][]tgbotapi.InlineKeyboardButton{}

	for _, cat := range catalogs {
		callbackData := fmt.Sprintf("toCat?catId=%d", cat.ID)
		productCount, err := cat.GetProductCount(db)
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

	userDb := models.TelegramUser{ID: update.CallbackQuery.From.ID}
	err := userDb.Get(*db)
	if err != nil {
		return err
	}
	err = db.Model(&userDb).
		WherePK().
		Relation("ShopSession").
		Relation("ShopSession.Catalog").
		Select()
	if err != nil {
		return err
	}

	fmt.Println("userDb.ShopSession: ", userDb.ShopSession, userDb.ShopSession.CatalogID)
	fmt.Println("data: ", userDb.ShopSession.CatalogID)

	if userDb.ShopSession != nil && userDb.ShopSession.CatalogID == 0 {
		fmt.Println("Иди нахуй")
		catIdStr, ok := data["catId"]
		if !ok {
			return Shop{Name: "shop", Client: v.Client}.Run(update)
		}

		catalogID, err := strconv.Atoi(catIdStr)
		if err != nil {
			return err
		}

		userDb.ShopSession.CatalogID = catalogID
		_, err = db.Model(userDb.ShopSession).Where("id = ?", userDb.ShopSession.ID).Column("catalog_id").Update()
		if err != nil {
			return err
		}
	}

	session := *userDb.ShopSession
	fmt.Println("session (chosen 65687): ", session, session.CatalogID)

	err = db.Model(&session).
		WherePK().
		Relation("Catalog").
		Select()
	if err != nil {
		return err
	}

	fmt.Println("catalog (chosen 65687): ", session.Catalog, session.Catalog.ID)

	productCount, err := session.Catalog.GetProductCount(db)
	if err != nil {
		return err
	}

	if productCount == 0 {
		message := tgbotapi.NewEditMessageText(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, "В этом каталоге пока что нет товаров")
		toListOfCats := "shop?showCat=true"
		message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{{{Text: "К списку каталогов", CallbackData: &toListOfCats}}},
		}

		if userDb.IsAdmin {
			removeCatalogCallbackData := "editShop?a=removeCatalog"
			addProductCallbackData := "editShop?a=createProduct"
			message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: append(message.ReplyMarkup.InlineKeyboard, []tgbotapi.InlineKeyboardButton{
					{Text: "Удалить каталог", CallbackData: &removeCatalogCallbackData},
					{Text: "Добавить товар", CallbackData: &addProductCallbackData},
				}),
			}
		}

		_, err = v.Client.Send(message)

		return err
	}

	if pageDeltaStr, ok := data["pageDelta"]; ok {
		pageDelta, err := strconv.Atoi(pageDeltaStr)
		if err != nil {
			return err
		}

		session.Offest += pageDelta
	}

	_, err = db.Model(&session).WherePK().Column("offest").Update()
	if err != nil {
		return err
	}

	if session.Offest >= productCount {
		session.Offest = 0
		_, err = db.Model(&session).WherePK().Column("offest").Update()
		if err != nil {
			return err
		}
	} else if session.Offest < 0 {
		session.Offest = productCount - 1
		_, err = db.Model(&session).WherePK().Column("offest").Update()
		if err != nil {
			return err
		}
	}

	var item models.Product
	err = db.Model(&item).
		Where("catalog_id = ?", session.Catalog.ID).
		Offset(session.Offest).
		Limit(1).
		Select()
	if err != nil {
		return err
	}

	session.ProductAtID = item.ID
	_, err = db.Model(&session).WherePK().Column("product_at_id").Update()
	if err != nil {
		return err
	}

	remove, ok := data["removeFromCart"]
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

	if ok, err := item.InUserCart(update.CallbackQuery.From.ID, *db); ok && err == nil {
		callbackData := "toCat?removeFromCart=true"
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			{Text: "Удалить из корзины", CallbackData: &callbackData},
		})
	} else if err != nil {
		return err
	} else {
		callbackData := "toCat?removeFromCart=false"
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
			{Text: "Добавить в корзину", CallbackData: &callbackData},
		})
	}

	if productCount > 1 {
		nextItemCallbackData := "toCat?pageDelta=1"
		prevItemCallbackData := "toCat?pageDelta=-1"
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
			removeCatalogCallbackData     = "editShop?a=removeCatalog"
			removeProductCallbackData     = "editShop?a=removeProduct"
			changePhotoCallbackData       = "editShop?a=changePhoto"
			changePriceCallbackData       = "editShop?a=changePrice"
			changeNameCallbackData        = "editShop?a=changeName"
			changeDescriptionCallbackData = "editShop?a=changeDescription"
			addProductCallbackData        = "editShop?a=createProduct"
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

	toListOfCats := "shop?showCat=true"
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
