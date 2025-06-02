package actions

import (
	"fmt"
	"main/controllers"
	"main/database/models"
	"os"
	"strconv"
	"strings"

	"github.com/go-pg/pg/v10"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ClearNextStepForUser очищает следующий шаг для пользователя
// update - обновление от Telegram API
// client - экземпляр Telegram бота
// sendCancelMessage - флаг, указывающий, нужно ли отправлять сообщение об отмене
func ClearNextStepForUser(update tgbotapi.Update, client *tgbotapi.BotAPI, sendCancelMessage bool) {
	var user *tgbotapi.User
	var chat *tgbotapi.Chat

	switch {
	case update.Message != nil:
		user = update.Message.From
	case update.CallbackQuery != nil:
		user = update.CallbackQuery.From
	default:
		return
	}

	switch {
	case update.Message != nil:
		chat = update.Message.Chat
	case update.CallbackQuery != nil:
		chat = update.CallbackQuery.Message.Chat
	}

	controllers.GetNextStepManager().RemoveNextStepAction(controllers.NextStepKey{
		ChatID: chat.ID,
		UserID: user.ID,
	}, *client, sendCancelMessage)
}

// GetMessageAndType возвращает сообщение и его тип из обновления
// update - обновление от Telegram API
// Возвращает сообщение и его тип
func GetMessageAndType(update tgbotapi.Update) (*tgbotapi.Message, string) {
	switch {
	case update.CallbackQuery != nil:
		message := update.CallbackQuery.Message
		message.From = update.CallbackQuery.From
		return message, "CallbackQuery"
	case update.Message != nil:
		return update.Message, "Message"
	case update.EditedMessage != nil:
		return update.EditedMessage, "EditedMessage"
	default:
		return nil, "Unknown"
	}
}

// GetMessage возвращает сообщение из обновления
// update - обновление от Telegram API
// Возвращает сообщение
func GetMessage(update tgbotapi.Update) *tgbotapi.Message {
	message, _ := GetMessageAndType(update)
	return message
}

func ParseCallData(s string) map[string]string {
	res := make(map[string]string, 0)
	if len(strings.Split(s, "?")) != 2 {
		return res
	}
	params := strings.Trim(strings.Split(s, "?")[1], " ")

	for _, p := range strings.Split(params, "&") {
		if len(strings.Split(p, "=")) != 2 {
			continue
		}
		key := strings.Split(p, "=")[0]
		value := strings.Split(p, "=")[1]

		res[strings.Trim(key, " ")] = strings.Trim(value, " ")
	}

	return res
}

func NumberToEmoji(n int) string {
	numbersMap := map[int]string{
		0: "0️⃣",
		1: "1️⃣",
		2: "2️⃣",
		3: "3️⃣",
		4: "4️⃣",
		5: "5️⃣",
		6: "6️⃣",
		7: "7️⃣",
		8: "8️⃣",
		9: "9️⃣",
	}

	digits := []int{}
	for n > 0 {
		digits = append(digits, n%10)
		n /= 10
	}

	var result string

	for i := len(digits) - 1; i >= 0; i-- {
		result += numbersMap[digits[i]]
	}

	return result
}

func DeleteProductFromUsersCarts(db *pg.DB, productID int, client *tgbotapi.BotAPI) error {
	addedTo := []models.AddedProducts{}
	err := db.Model(&addedTo).
		Where("product_id = ?", productID).
		Relation("Transaction").
		Relation("Transaction.AddedProducts").
		Relation("Product").
		Relation("User").
		Select()
	if err != nil {
		return err
	}

	for _, item := range addedTo {
		if item.Transaction.IsWaitingForApproval {
			adminChatIdStr := os.Getenv("ADMIN_CHAT_ID")

			adminChatId, err := strconv.ParseInt(adminChatIdStr, 10, 64)
			if err != nil {
				return err
			}

			var userName string
			if item.User.Username != "" {
				userName = "@" + item.User.Username
			} else {
				userName = "<a href='tg://user?id=" + strconv.FormatInt(item.User.ID, 10) + "'>" + item.User.FirstName + " " + item.User.LastName + "</a>"
			}

			message := tgbotapi.NewMessage(adminChatId, fmt.Sprintf("Товар удалён из корзины пользователя %s который уже оплатил заказ! Неоходимо осуществить возврат средств на сумму %d₽", userName, item.Product.Price*item.ProductCount))

			_, err = client.Send(message)
			if err != nil {
				return err
			}
		}

		if len(item.Transaction.AddedProducts) == 1 {
			db.Model(item.Transaction).WherePK().Delete()
		}

		db.Model(item).WherePK().Delete()
	}

	return nil
}