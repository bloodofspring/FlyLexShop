package filters

import (
	"log"
	"main/controllers"
	"main/database"
	"main/database/models"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var RegisterUserFilter = func(update tgbotapi.Update) bool {
	return update.CallbackQuery.Data == "registerUser"
}

var MainMenuFilter = func(update tgbotapi.Update) bool {
	return update.CallbackQuery.Data == "mainMenu"
}

var ProfileSettingsFilter = func(update tgbotapi.Update) bool {
	if update.CallbackQuery.Data == "profileSettings" {
		controllers.GetNextStepManager().RemoveNextStepAction(controllers.NextStepKey{ChatID: update.CallbackQuery.Message.Chat.ID, UserID: update.CallbackQuery.From.ID})
		return true
	}

	return false
}

var ViewCatalogFilter = func(update tgbotapi.Update) bool {
	return strings.HasPrefix(update.CallbackQuery.Data, "toCat")
}

var ShopFilter = func(update tgbotapi.Update) bool {
	return update.CallbackQuery.Data == "shop"
}

var AboutFilter = func(update tgbotapi.Update) bool {
	return update.CallbackQuery.Data == "about"
}

var ChangeNameFilter = func(update tgbotapi.Update) bool {
	return update.CallbackQuery.Data == "changeName"
}

var ChangePhoneFilter = func(update tgbotapi.Update) bool {
	return update.CallbackQuery.Data == "changePhone"
}

var ChangeDeliveryAddressFilter = func(update tgbotapi.Update) bool {
	return update.CallbackQuery.Data == "changeDeliveryAddress"
}

var ChangeDeliveryServiceFilter = func(update tgbotapi.Update) bool {
	if !strings.HasPrefix(update.CallbackQuery.Data, "changeDeliveryService") {
		return false
	}

	db := database.Connect()
	defer db.Close()

	user := models.TelegramUser{ID: update.CallbackQuery.From.ID}
	err := user.GetOrCreate(update.CallbackQuery.From, *db)
	if err != nil {
		log.Println("ChangeDeliveryServiceFilter says: ошибка при получении или создании пользователя", err)
		return false
	}

	params := ParseCallbackData(update.CallbackQuery.Data)
	service, ok := params["service"]
	if !ok {
		return true
	}

	user.DeliveryService = service

	_, err = db.Model(&user).WherePK().Update()

	return err == nil
}
