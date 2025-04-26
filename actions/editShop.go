package actions

import (
	"main/controllers"
	"main/database"
	"main/database/models"
	"main/filters"
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

func changePhoto(update tgbotapi.Update, client tgbotapi.BotAPI, productId string, db pg.DB) error {
	client.Send(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))

	msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Отправьте ниже новое фото товара")
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
		Func: changePhotoHandler,
		Params: map[string]interface{}{
			"productId": productId,
			"db": db,
		},
		CreatedAtTS: time.Now().Unix(),
		CancelMessage: "Фото не обновлено",
	}

	controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)

	return nil
}

func changePhotoHandler(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	photo := update.Message.Photo
	if len(photo) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Отправьте фото товара")
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
			Func: changePhotoHandler,
			Params: map[string]interface{}{
				"productId": stepParams["productId"],
				"db": stepParams["db"],
			},
			CreatedAtTS: time.Now().Unix(),
			CancelMessage: "Фото не обновлено",
		}

		controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)

		return nil
	}

	photoID := photo[len(photo)-1].FileID

	_, err := stepParams["db"].(*pg.DB).Model(&models.Product{}).Where("id = ?", stepParams["productId"]).Set("image_file_id = ?", photoID).Update()
	if err != nil {
		return err
	}

	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))

	_, err = client.Request(tgbotapi.CallbackConfig{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            "Фото обновлено!",
		ShowAlert:       true,
	})

	if err != nil {
		return err
	}

	return Shop{Name: "shop", Client: client}.Run(update)
}

func changePrice(update tgbotapi.Update, client tgbotapi.BotAPI, productId string, db pg.DB) error {
	return nil
}

func changeName(update tgbotapi.Update, client tgbotapi.BotAPI, productId string, db pg.DB) error {
	return nil
}

func changeDescription(update tgbotapi.Update, client tgbotapi.BotAPI, productId string, db pg.DB) error {
	return nil
}

func (e EditShop) GetName() string {
	return e.Name
}
