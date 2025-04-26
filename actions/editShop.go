package actions

import (
	"main/controllers"
	"main/database"
	"main/database/models"
	"main/filters"
	"strconv"
	"time"

	"github.com/go-pg/pg/v10"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type EditShop struct {
	Name   string
	Client tgbotapi.BotAPI
}

func (e EditShop) Run(update tgbotapi.Update) error {
	ClearNextStepForUser(update, &e.Client, true)

	db := database.Connect()
	defer db.Close()

	data := filters.ParseCallbackData(update.CallbackQuery.Data)

	switch data["a"] {
	case "removeCatalog":
		removeCatalog(update, e.Client, data["sessionId"], *db)
	case "removeProduct":
		removeProduct(update, e.Client, data["productId"], *db)
	case "changePhoto":
		changePhoto(update, e.Client, data["productId"])
	case "changePrice":
		changePrice(update, e.Client, data["productId"])
	case "changeName":
		changeName(update, e.Client, data["productId"])
	case "changeDescription":
		changeDescription(update, e.Client, data["productId"])
	case "createProduct":
		createProduct(update, e.Client, data["sessionId"])
	}

	return nil
}

func removeCatalog(update tgbotapi.Update, client tgbotapi.BotAPI, sessionId string, db pg.DB) error {
	var session models.ShopViewSession
	err := db.Model(&session).Where("id = ?", sessionId).Select()
	if err != nil {
		return err
	}

	_, err = db.Model(&models.Catalog{}).Where("id = ?", session.CatId).Delete()
	if err != nil {
		return err
	}

	_, err = db.Model(&session).WherePK().Delete()
	if err != nil {
		return err
	}

	client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))

	msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Каталог удален")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("К списку каталогов", "shop"),
		),
	)
	_, err = client.Send(msg)

	return err
}

func removeProduct(update tgbotapi.Update, client tgbotapi.BotAPI, productId string, db pg.DB) error {
	_, err := db.Model(&models.Product{}).Where("id = ?", productId).Delete()
	if err != nil {
		return err
	}

	client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))

	_, err = client.Request(tgbotapi.CallbackConfig{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            "Товар удален!",
		ShowAlert:       true,
	})

	if err != nil {
		return err
	}

	return Shop{Name: "shop", Client: client}.Run(update)
}

func baseForm(client tgbotapi.BotAPI, update tgbotapi.Update, params map[string]any, formText, CancelMessage string, formHandler controllers.NextStepFunc) error {
	client.Send(tgbotapi.NewDeleteMessage(GetMessage(update).Chat.ID, GetMessage(update).MessageID))

	msg := tgbotapi.NewMessage(GetMessage(update).Chat.ID, formText)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Отмена", "shop"),
		),
	)
	_, err := client.Send(msg)
	if err != nil {
		return err
	}

	stepKey := controllers.NextStepKey{
		UserID: GetMessage(update).From.ID,
		ChatID: GetMessage(update).Chat.ID,
	}

	stepAction := controllers.NextStepAction{
		Func: formHandler,
		Params: params,
		CreatedAtTS: time.Now().Unix(),
		CancelMessage: CancelMessage,
	}

	controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)

	return nil
}

func baseFormSuccess(client tgbotapi.BotAPI, update tgbotapi.Update, successMessage string) error {
	ClearNextStepForUser(update, &client, false)

	msg := tgbotapi.NewMessage(GetMessage(update).Chat.ID, successMessage)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("К списку товаров", "shop"),
		),
	)
	_, err := client.Send(msg)

	return err
}

func baseFormResend(client tgbotapi.BotAPI, update tgbotapi.Update, formText, CancelMessage string, stepParams map[string]any, formHandler controllers.NextStepFunc) error {
	msg := tgbotapi.NewMessage(GetMessage(update).Chat.ID, formText)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Отмена", "shop"),
		),
	)
	_, err := client.Send(msg)
	if err != nil {
		return err
	}

	stepKey := controllers.NextStepKey{
		UserID: GetMessage(update).From.ID,
		ChatID: GetMessage(update).Chat.ID,
	}

	stepAction := controllers.NextStepAction{
		Func: formHandler,
		Params: stepParams,
		CreatedAtTS: time.Now().Unix(),
		CancelMessage: CancelMessage,
	}

	controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)

	return nil
}

func changePhoto(update tgbotapi.Update, client tgbotapi.BotAPI, productId string) error {
	return baseForm(client, update, map[string]any{
		"productId": productId,
	}, "Отправьте ниже новое фото товара", "Фото не обновлено", changePhotoHandler)
}

func changePhotoHandler(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID-1))
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))

	photo := update.Message.Photo
	if len(photo) == 0 {
		return baseFormResend(client, update, "Отправьте ниже новое фото товара", "Фото не обновлено", stepParams, changePhotoHandler)
	}

	photoID := photo[len(photo)-1].FileID

	db := database.Connect()
	defer db.Close()

	_, err := db.Model(&models.Product{}).Where("id = ?", stepParams["productId"]).Set("image_file_id = ?", photoID).Update()
	if err != nil {
		return err
	}

	return baseFormSuccess(client, update, "Фото обновлено!")
}

func changePrice(update tgbotapi.Update, client tgbotapi.BotAPI, productId string) error {
	return baseForm(client, update, map[string]any{
		"productId": productId,
	}, "Отправьте ниже новую цену товара", "Цена не обновлена", changePriceHandler)
}

func changePriceHandler(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID-1))
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))

	price := update.Message.Text

	priceInt, err := strconv.Atoi(price)
	
	if err != nil {
		return baseFormResend(client, update, "Отправьте ниже новую цену товара (целое число!)", "Цена не обновлена", stepParams, changePriceHandler)
	}

	db := database.Connect()
	defer db.Close()

	_, err = db.Model(&models.Product{}).Where("id = ?", stepParams["productId"]).Set("price = ?", priceInt).Update()
	if err != nil {
		return err
	}

	return baseFormSuccess(client, update, "Цена обновлена!")
}

func changeName(update tgbotapi.Update, client tgbotapi.BotAPI, productId string) error {
	return baseForm(client, update, map[string]any{
		"productId": productId,
	}, "Отправьте ниже новое название товара", "Название не обновлено", changeNameHandler)
}

func changeNameHandler(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID-1))
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))

	name := update.Message.Text

	if name == "" {
		return baseFormResend(client, update, "Название не может быть пустым", "Название не обновлено", stepParams, changeNameHandler)
	}

	db := database.Connect()
	defer db.Close()

	_, err := db.Model(&models.Product{}).Where("id = ?", stepParams["productId"]).Set("name = ?", name).Update()
	if err != nil {
		return err
	}

	return baseFormSuccess(client, update, "Название обновлено!")
}

func changeDescription(update tgbotapi.Update, client tgbotapi.BotAPI, productId string) error {
	return baseForm(client, update, map[string]any{
		"productId": productId,
	}, "Отправьте ниже новое описание товара", "Описание не обновлено", changeDescriptionHandler)
}

func changeDescriptionHandler(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID-1))
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))

	description := update.Message.Text

	if description == "" {
		return baseFormResend(client, update, "Описание не может быть пустым", "Описание не обновлено", stepParams, changeDescriptionHandler)
	}
	
	db := database.Connect()
	defer db.Close()

	_, err := db.Model(&models.Product{}).Where("id = ?", stepParams["productId"]).Set("description = ?", description).Update()
	if err != nil {
		return err
	}

	return baseFormSuccess(client, update, "Описание обновлено!")
}

func createProduct(update tgbotapi.Update, client tgbotapi.BotAPI, sessionId string) error {
	return baseForm(client, update, map[string]any{
		"sessionId": sessionId,
	}, "Отправьте ниже название товара", "Товар не создан", registerNewProductName)
}

func registerNewProductName(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID-1))
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))

	name := update.Message.Text

	if name == "" {
		return baseFormResend(client, update, "Название не может быть пустым", "Товар не создан", stepParams, registerNewProductName)
	}
	
	db := database.Connect()
	defer db.Close()

	_, err := db.Model(&models.Product{}).Insert()
	if err != nil {
		return err
	}

	stepParams["productName"] = name
	return baseForm(client, update, stepParams, "Отправьте ниже цену товара", "Товар не создан", registerNewProductPrice)
}

func registerNewProductPrice(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID-1))
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))

	price := update.Message.Text
	
	priceInt, err := strconv.Atoi(price)
	if err != nil {
		return baseFormResend(client, update, "Цена должна быть числом", "Товар не создан", stepParams, registerNewProductPrice)
	}

	db := database.Connect()
	defer db.Close()

	_, err = db.Model(&models.Product{}).Insert()
	if err != nil {
		return err
	}
	
	stepParams["productPrice"] = priceInt
	return baseForm(client, update, stepParams, "Отправьте ниже описание товара", "Товар не создан", registerNewProductDescription)
}

func registerNewProductDescription(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID-1))
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))

	description := update.Message.Text
	
	if description == "" {
		return baseFormResend(client, update, "Описание не может быть пустым", "Товар не создан", stepParams, registerNewProductDescription)
	}

	db := database.Connect()
	defer db.Close()

	_, err := db.Model(&models.Product{}).Insert()
	if err != nil {
		return err
	}

	stepParams["productDescription"] = description
	return baseForm(client, update, stepParams, "Отправьте ниже фото товара", "Товар не создан", registerNewProductPhoto)
}

func registerNewProductPhoto(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID-1))
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))
	photo := update.Message.Photo
	if len(photo) == 0 {
		return baseFormResend(client, update, "Отправьте ниже фото товара", "Товар не создан", stepParams, registerNewProductPhoto)
	}

	photoID := photo[len(photo)-1].FileID

	session := models.ShopViewSession{}

	db := database.Connect()
	defer db.Close()

	err := db.Model(&session).Where("id = ?", stepParams["sessionId"]).Select()
	if err != nil {
		return err
	}
	
	db = database.Connect()
	defer db.Close()

	_, err = db.Model(&models.Product{
		ImageFileID: photoID,
		Name: stepParams["productName"].(string),
		Price: stepParams["productPrice"].(int),
		Description: stepParams["productDescription"].(string),
		CatalogID: session.CatId,
	}).Insert()
	if err != nil {
		return err
	}

	return baseFormSuccess(client, update, "Товар успешно создан!")
}

func (e EditShop) GetName() string {
	return e.Name
}
