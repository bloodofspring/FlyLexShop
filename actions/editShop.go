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
		changePhoto(update, e.Client, data["productId"], *db)
	case "changePrice":
		changePrice(update, e.Client, data["productId"], *db)
	case "changeName":
		changeName(update, e.Client, data["productId"], *db)
	case "changeDescription":
		changeDescription(update, e.Client, data["productId"], *db)
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

func baseForm(client tgbotapi.BotAPI, update tgbotapi.Update, productId string, db pg.DB, formText, CancelMessage string, formHandler controllers.NextStepFunc) error {
	client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))

	msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, formText)
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
		UserID: update.CallbackQuery.From.ID,
		ChatID: update.CallbackQuery.Message.Chat.ID,
	}

	stepAction := controllers.NextStepAction{
		Func: formHandler,
		Params: map[string]interface{}{
			"productId": productId,
			"db": db,
		},
		CreatedAtTS: time.Now().Unix(),
		CancelMessage: CancelMessage,
	}

	controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)

	return nil
}

func baseFormSuccess(client tgbotapi.BotAPI, update tgbotapi.Update, successMessage string) error {
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))

	_, err := client.Request(tgbotapi.CallbackConfig{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            successMessage,
		ShowAlert:       true,
	})

	if err != nil {
		return err
	}

	return Shop{Name: "shop", Client: client}.Run(update)
}

func baseFormResend(client tgbotapi.BotAPI, update tgbotapi.Update, formText, CancelMessage string, stepParams map[string]any, formHandler controllers.NextStepFunc) error {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, formText)
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
		UserID: update.Message.From.ID,
		ChatID: update.Message.Chat.ID,
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

func changePhoto(update tgbotapi.Update, client tgbotapi.BotAPI, productId string, db pg.DB) error {
	return baseForm(client, update, productId, db, "Отправьте ниже новое фото товара", "Фото не обновлено", changePhotoHandler)
}

func changePhotoHandler(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	photo := update.Message.Photo
	if len(photo) == 0 {
		return baseFormResend(client, update, "Отправьте ниже новое фото товара", "Фото не обновлено", stepParams, changePhotoHandler)
	}

	photoID := photo[len(photo)-1].FileID

	_, err := stepParams["db"].(*pg.DB).Model(&models.Product{}).Where("id = ?", stepParams["productId"]).Set("image_file_id = ?", photoID).Update()
	if err != nil {
		return err
	}

	return baseFormSuccess(client, update, "Фото обновлено!")
}

func changePrice(update tgbotapi.Update, client tgbotapi.BotAPI, productId string, db pg.DB) error {
	return baseForm(client, update, productId, db, "Отправьте ниже новую цену товара", "Цена не обновлена", changePriceHandler)
}

func changePriceHandler(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	price := update.Message.Text

	priceInt, err := strconv.Atoi(price)
	
	if err != nil {
		return baseFormResend(client, update, "Отправьте ниже новую цену товара (целое число!)", "Цена не обновлена", stepParams, changePriceHandler)
	}

	_, err = stepParams["db"].(*pg.DB).Model(&models.Product{}).Where("id = ?", stepParams["productId"]).Set("price = ?", priceInt).Update()
	if err != nil {
		return err
	}

	return baseFormSuccess(client, update, "Цена обновлена!")
}

func changeName(update tgbotapi.Update, client tgbotapi.BotAPI, productId string, db pg.DB) error {
	return baseForm(client, update, productId, db, "Отправьте ниже новое название товара", "Название не обновлено", changeNameHandler)
}

func changeNameHandler(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	name := update.Message.Text

	if name == "" {
		return baseFormResend(client, update, "Название не может быть пустым", "Название не обновлено", stepParams, changeNameHandler)
	}

	_, err := stepParams["db"].(*pg.DB).Model(&models.Product{}).Where("id = ?", stepParams["productId"]).Set("name = ?", name).Update()
	if err != nil {
		return err
	}

	return baseFormSuccess(client, update, "Название обновлено!")
}

func changeDescription(update tgbotapi.Update, client tgbotapi.BotAPI, productId string, db pg.DB) error {
	return baseForm(client, update, productId, db, "Отправьте ниже новое описание товара", "Описание не обновлено", changeDescriptionHandler)
}

func changeDescriptionHandler(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	description := update.Message.Text

	if description == "" {
		return baseFormResend(client, update, "Описание не может быть пустым", "Описание не обновлено", stepParams, changeDescriptionHandler)
	}
	
	_, err := stepParams["db"].(*pg.DB).Model(&models.Product{}).Where("id = ?", stepParams["productId"]).Set("description = ?", description).Update()
	if err != nil {
		return err
	}

	return baseFormSuccess(client, update, "Описание обновлено!")
}


func (e EditShop) GetName() string {
	return e.Name
}
