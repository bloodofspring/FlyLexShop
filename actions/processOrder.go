package actions

import (
	"fmt"
	"main/controllers"
	"main/database"
	"main/database/models"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

const (
	processOrderPageText = "<b>Итог:</b> %d\n\nОплата осуществляется переводом по номеру карты или телефона:\n|_<b>Номер карты:</b> %s\n|_<b>Номер телефона:</b> %s\n|_<b>Банк:</b> %s\n\n<b>!!!После оплаты пришлите боту чек на проверку сообщением ниже!!!</b>"
)

func RegisterPaymentPhoto(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	if update.Message == nil {
		return nil
	}

	if update.Message.Photo == nil {
		message := tgbotapi.NewMessage(update.Message.Chat.ID, "Пожалуйста, пришлите фото чека на проверку.")
		_, err := client.Send(message)
		if err != nil {
			return err
		}

		stepKey := controllers.NextStepKey{
			ChatID: update.Message.Chat.ID,
			UserID: update.Message.From.ID,
		}
		stepAction := controllers.NextStepAction{
			Func:        RegisterPaymentPhoto,
			Params:      make(map[string]any),
			CreatedAtTS: time.Now().Unix(),
			CancelMessage: "Оформление заказа прервано! Вы можете совершить покупку позже в этом же разделе.",
		}

		controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)

		return nil
	}

	envFile, err := godotenv.Read(".env")
	if err != nil {
		return err
	}

	adminChatID := envFile["admin_chat_id"]

	db := database.Connect()
	defer db.Close()

	var items []models.Product
	err = db.Model(&items).Where("id IN (SELECT product_id FROM shopping_carts WHERE user_id = ?)", update.Message.From.ID).Select()
	if err != nil {
		return err
	}

	cartDesc := "Список товаров:\n\n"
	totalPrice := 0
	for _, item := range items {
		cartDesc += fmt.Sprintf("|_ %s - %d₽\n", item.Name, item.Price)
		totalPrice += item.Price
	}
	cartDesc += fmt.Sprintf("\nИтоговая сумма: %d₽", totalPrice)

	chatID, err := strconv.ParseInt(adminChatID, 10, 64)
	if err != nil {
		return err
	}

	photoMsg := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(update.Message.Photo[len(update.Message.Photo)-1].FileID))
	photoMsg.Caption = cartDesc

	acceptData := "paymentVerdict?ok=true&userId=" + strconv.FormatInt(update.Message.From.ID, 10)
	rejectData := "paymentVerdict?ok=false&userId=" + strconv.FormatInt(update.Message.From.ID, 10)
	photoMsg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
			{
				{Text: "Принять заявку", CallbackData: &acceptData},
				{Text: "Отклонить заявку", CallbackData: &rejectData},
			},
		},
	}

	_, err = client.Send(photoMsg)
	if err != nil {
		return err
	}

	successMsg := tgbotapi.NewMessage(update.Message.Chat.ID, "Спасибо, администратор скоро проверит оплату!")
	_, err = client.Send(successMsg)

	return err
}

type ProcessOrder struct {
	Name string
	Client tgbotapi.BotAPI
}

func (p ProcessOrder) Run(update tgbotapi.Update) error {
	ClearNextStepForUser(update, &p.Client)
	db := database.Connect()
	defer db.Close()

	user := models.TelegramUser{ID: update.CallbackQuery.From.ID}
	err := user.Get(*db)
	if err != nil {
		return err
	}

	totalPrice, err := user.GetTotalCartPrice(*db)
	if err != nil {
		return err
	}

	envFile, _ := godotenv.Read(".env")

	pageText := fmt.Sprintf(processOrderPageText, totalPrice, envFile["payment_card_number"], envFile["payment_phone_number"], envFile["payment_bank"])

	msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, pageText)
	msg.ParseMode = "HTML"

	_, err = p.Client.Send(msg)
	if err != nil {
		return err
	}

	stepKey := controllers.NextStepKey{
		ChatID: update.CallbackQuery.Message.Chat.ID,
		UserID: update.CallbackQuery.From.ID,
	}
	stepAction := controllers.NextStepAction{
		Func:        RegisterPaymentPhoto,
		Params:      make(map[string]any),
		CreatedAtTS: time.Now().Unix(),
		CancelMessage: "Оформление заказа прервано! Вы можете совершить покупку позже в этом же разделе.",
	}

	controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)

	return nil
}

func (p ProcessOrder) GetName() string {
	return p.Name
}
