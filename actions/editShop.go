package actions

import (
	"fmt"
	"main/controllers"
	"main/database"
	"main/database/models"
	"main/filters"
	"strconv"
	"time"

	"github.com/go-pg/pg/v10"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// EditShop управляет редактированием каталогов и товаров в магазине.
// Name - имя команды.
// Client - экземпляр Telegram бота.
type EditShop struct {
	Name   string
	Client tgbotapi.BotAPI
}

// Run обрабатывает действие редактирования магазина на основе параметра a.
// update - обновление от Telegram API.
// Возвращает ошибку, если что-то пошло не так.
func (e EditShop) Run(update tgbotapi.Update) error {
	ClearNextStepForUser(update, &e.Client, true)

	db := database.Connect()
	defer db.Close()

	userDb := models.TelegramUser{ID: update.CallbackQuery.From.ID}
	err := userDb.Get(*db)
	if err != nil {
		return err
	}
	err = db.Model(&userDb).
		WherePK().
		Relation("ShopSession").
		Relation("ShopSession.ProductAt").
		Relation("ShopSession.Catalog").
		Select()
	if err != nil {
		return err
	}

	fmt.Println("userDb.ShopSession: ", userDb.ShopSession)

	if userDb.ShopSession == nil {
		return ViewCatalog{Name: "viewCatalog", Client: e.Client}.Run(update)
	}

	data := filters.ParseCallbackData(update.CallbackQuery.Data)
	session := *userDb.ShopSession

	switch data["a"] {
	case "removeCatalog":
		err = removeCatalog(update, e.Client, session, *db)
	case "removeProduct":
		err = removeProduct(update, e.Client, session, *db)
	case "changePhoto":
		err = changePhoto(update, e.Client, session)
	case "changePrice":
		err = changePrice(update, e.Client, session)
	case "changeName":
		err = changeName(update, e.Client, session)
	case "changeDescription":
		changeDescription(update, e.Client, session)
	case "createProduct":
		err = createProduct(update, e.Client, session)
	}

	return err
}

// removeCatalog удаляет каталог и возвращает пользователя к списку каталогов.
func removeCatalog(update tgbotapi.Update, client tgbotapi.BotAPI, session models.ShopViewSession, db pg.DB) error {
	_, err := db.Model(&models.Catalog{}).Where("id = ?", session.CatalogID).Delete()
	if err != nil {
		return err
	}

	session.CatalogID = 0
	_, err = db.Model(&session).WherePK().Column("catalog_id").Update()
	if err != nil {
		return err
	}

	_, err = client.Request(tgbotapi.CallbackConfig{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            "Каталог удален!",
		ShowAlert:       true,
	})

	if err != nil {
		return err
	}

	return Shop{Name: "shop", Client: client}.Run(update)
}

// removeProduct удаляет текущий товар и возвращает пользователя к просмотру каталога.
func removeProduct(update tgbotapi.Update, client tgbotapi.BotAPI, session models.ShopViewSession, db pg.DB) error {
	_, err := db.Model(session.ProductAt).WherePK().Delete()
	if err != nil {
		return err
	}

	_, err = client.Request(tgbotapi.CallbackConfig{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            "Товар удален!",
		ShowAlert:       true,
	})

	if err != nil {
		return err
	}

	return ViewCatalog{Name: "viewCatalog", Client: client}.Run(update)
}

// baseForm отображает форму ввода с кнопкой отмены и регистрирует следующий шаг.
func baseForm(client tgbotapi.BotAPI, update tgbotapi.Update, params map[string]any, formText, CancelMessage string, formHandler controllers.NextStepFunc) error {
	client.Send(tgbotapi.NewDeleteMessage(GetMessage(update).Chat.ID, GetMessage(update).MessageID))

	msg := tgbotapi.NewMessage(GetMessage(update).Chat.ID, formText)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Отмена", "toCat"),
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
		Func:          formHandler,
		Params:        params,
		CreatedAtTS:   time.Now().Unix(),
		CancelMessage: CancelMessage,
	}

	controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)

	return nil
}

// baseFormSuccess очищает следующий шаг и показывает сообщение об успехе.
func baseFormSuccess(client tgbotapi.BotAPI, update tgbotapi.Update, successMessage string) error {
	ClearNextStepForUser(update, &client, false)

	msg := tgbotapi.NewMessage(GetMessage(update).Chat.ID, successMessage)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("К списку товаров", "toCat"),
		),
	)
	_, err := client.Send(msg)

	return err
}

// baseFormResend повторно отображает форму ввода при ошибке.
func baseFormResend(client tgbotapi.BotAPI, update tgbotapi.Update, formText, CancelMessage string, stepParams map[string]any, formHandler controllers.NextStepFunc) error {
	msg := tgbotapi.NewMessage(GetMessage(update).Chat.ID, formText)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Отмена", "toCat"),
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
		Func:          formHandler,
		Params:        stepParams,
		CreatedAtTS:   time.Now().Unix(),
		CancelMessage: CancelMessage,
	}

	controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)

	return nil
}

// changePhoto инициирует изменение фото товара.
func changePhoto(update tgbotapi.Update, client tgbotapi.BotAPI, session models.ShopViewSession) error {
	return baseForm(client, update, map[string]any{
		"session": session,
	}, "Отправьте ниже новое фото товара", "Фото не обновлено", changePhotoHandler)
}

// changePhotoHandler обрабатывает загрузку нового фото и сохраняет его.
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

	_, err := db.Model(stepParams["session"].(models.ShopViewSession).ProductAt).WherePK().Set("image_file_id = ?", photoID).Update()
	if err != nil {
		return err
	}

	return baseFormSuccess(client, update, "Фото обновлено!")
}

// changePrice инициирует изменение цены товара.
func changePrice(update tgbotapi.Update, client tgbotapi.BotAPI, session models.ShopViewSession) error {
	return baseForm(client, update, map[string]any{
		"session": session,
	}, "Отправьте ниже новую цену товара", "Цена не обновлена", changePriceHandler)
}

// changePriceHandler обрабатывает ввод новой цены и сохраняет её.
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

	_, err = db.Model(stepParams["session"].(models.ShopViewSession).ProductAt).WherePK().Set("price = ?", priceInt).Update()
	if err != nil {
		return err
	}

	return baseFormSuccess(client, update, "Цена обновлена!")
}

// changeName инициирует изменение названия товара.
// update - обновление от Telegram API.
// client - экземпляр Telegram бота.
// session - текущая сессия просмотра магазина.
func changeName(update tgbotapi.Update, client tgbotapi.BotAPI, session models.ShopViewSession) error {
	return baseForm(client, update, map[string]any{
		"session": session,
	}, "Отправьте ниже новое название товара", "Название не обновлено", changeNameHandler)
}

// changeNameHandler обрабатывает ввод нового названия товара и сохраняет его.
// client - экземпляр Telegram бота.
// update - обновление от Telegram API.
// stepParams - параметры шага, содержащие session.
func changeNameHandler(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID-1))
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))

	name := update.Message.Text

	if name == "" {
		return baseFormResend(client, update, "Название не может быть пустым", "Название не обновлено", stepParams, changeNameHandler)
	}

	db := database.Connect()
	defer db.Close()

	_, err := db.Model(stepParams["session"].(models.ShopViewSession).ProductAt).WherePK().Set("name = ?", name).Update()
	if err != nil {
		return err
	}

	return baseFormSuccess(client, update, "Название обновлено!")
}

// changeDescription инициирует изменение описания товара.
// update - обновление от Telegram API.
// client - экземпляр Telegram бота.
// session - текущая сессия просмотра магазина.
func changeDescription(update tgbotapi.Update, client tgbotapi.BotAPI, session models.ShopViewSession) error {
	return baseForm(client, update, map[string]any{
		"session": session,
	}, "Отправьте ниже новое описание товара", "Описание не обновлено", changeDescriptionHandler)
}

// changeDescriptionHandler обрабатывает ввод нового описания товара и сохраняет его.
// client - экземпляр Telegram бота.
// update - обновление от Telegram API.
// stepParams - параметры шага, содержащие session.
func changeDescriptionHandler(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID-1))
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))

	description := update.Message.Text

	if description == "" {
		return baseFormResend(client, update, "Описание не может быть пустым", "Описание не обновлено", stepParams, changeDescriptionHandler)
	}

	db := database.Connect()
	defer db.Close()

	_, err := db.Model(stepParams["session"].(models.ShopViewSession).ProductAt).WherePK().Set("description = ?", description).Update()
	if err != nil {
		return err
	}

	return baseFormSuccess(client, update, "Описание обновлено!")
}

// createProduct инициирует создание нового товара.
// update - обновление от Telegram API.
// client - экземпляр Telegram бота.
// session - текущая сессия просмотра магазина.
func createProduct(update tgbotapi.Update, client tgbotapi.BotAPI, session models.ShopViewSession) error {
	return baseForm(client, update, map[string]any{
		"session": session,
	}, "Отправьте ниже название товара", "Товар не создан", registerNewProductName)
}

// registerNewProductName обрабатывает ввод названия нового товара при создании.
// client - экземпляр Telegram бота.
// update - обновление от Telegram API.
// stepParams - параметры шага, содержащие session.
func registerNewProductName(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID-1))
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))

	name := update.Message.Text

	if name == "" {
		return baseFormResend(client, update, "Название не может быть пустым", "Товар не создан", stepParams, registerNewProductName)
	}

	stepParams["productName"] = name
	return baseForm(client, update, stepParams, "Отправьте ниже цену товара", "Товар не создан", registerNewProductPrice)
}

// registerNewProductPrice обрабатывает ввод цены нового товара при создании.
// client - экземпляр Telegram бота.
// update - обновление от Telegram API.
// stepParams - параметры шага, содержащие session и productName.
func registerNewProductPrice(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID-1))
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))

	price := update.Message.Text

	priceInt, err := strconv.Atoi(price)
	if err != nil {
		return baseFormResend(client, update, "Цена должна быть числом", "Товар не создан", stepParams, registerNewProductPrice)
	}

	stepParams["productPrice"] = priceInt
	return baseForm(client, update, stepParams, "Отправьте ниже описание товара", "Товар не создан", registerNewProductDescription)
}

// registerNewProductDescription обрабатывает ввод описания нового товара при создании.
// client - экземпляр Telegram бота.
// update - обновление от Telegram API.
// stepParams - параметры шага, содержащие session, productName и productPrice.
func registerNewProductDescription(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID-1))
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))

	description := update.Message.Text

	if description == "" {
		return baseFormResend(client, update, "Описание не может быть пустым", "Товар не создан", stepParams, registerNewProductDescription)
	}

	stepParams["productDescription"] = description
	return baseForm(client, update, stepParams, "Отправьте ниже фото товара", "Товар не создан", registerNewProductPhoto)
}

// registerNewProductPhoto обрабатывает загрузку фото нового товара и сохраняет его.
// client - экземпляр Telegram бота.
// update - обновление от Telegram API.
// stepParams - параметры шага, содержащие session, productName, productPrice и productDescription.
func registerNewProductPhoto(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID-1))
	client.Send(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))
	photo := update.Message.Photo
	if len(photo) == 0 {
		return baseFormResend(client, update, "Отправьте ниже фото товара", "Товар не создан", stepParams, registerNewProductPhoto)
	}

	photoID := photo[len(photo)-1].FileID

	db := database.Connect()
	defer db.Close()

	fmt.Println("stepParams: ", stepParams["session"])

	_, err := db.Model(&models.Product{
		ImageFileID: photoID,
		Name:        stepParams["productName"].(string),
		Price:       stepParams["productPrice"].(int),
		Description: stepParams["productDescription"].(string),
		CatalogID:   stepParams["session"].(models.ShopViewSession).Catalog.ID,
	}).Insert()
	if err != nil {
		return err
	}

	return baseFormSuccess(client, update, "Товар успешно создан!")
}

// GetName возвращает имя команды EditShop.
func (e EditShop) GetName() string {
	return e.Name
}
