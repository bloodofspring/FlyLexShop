package actions

import (
	"context"
	"fmt"
	"main/controllers"
	"main/database"
	"main/database/models"
	"regexp"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// RegisterUser представляет собой структуру для обработки регистрации пользователя
// Name - имя команды
// Client - экземпляр Telegram бота
type RegisterUser struct {
	Name   string
	Client tgbotapi.BotAPI
	mu     *sync.Mutex
}

func NewRegisterUserHandler(client tgbotapi.BotAPI) *RegisterUser {
	return &RegisterUser{
		Name:   "registerUser",
		Client: client,
		mu:     &sync.Mutex{},
	}
}

// RegistrationCompleted завершает процесс регистрации пользователя
// client - экземпляр Telegram бота
// update - обновление от Telegram API
// stepParams - параметры шага регистрации
// Возвращает ошибку, если что-то пошло не так
func RegistrationCompleted(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	fmt.Println(1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	mu := sync.Mutex{}
	var err error

	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			return
		default:
			db := database.Connect()
			defer db.Close()

			user := models.TelegramUser{ID: update.Message.From.ID}
			_ = user.GetOrCreate(update.Message.From, *db)

			_, err = db.Model(&user).
				WherePK().
				Set("delivery_address = ?", update.Message.Text).
				Set("is_authorized = ?", true).
				Update()
			if err != nil {
				return
			}

			message := tgbotapi.NewMessage(update.Message.Chat.ID, "Вы успешно зарегистрированы! Нажмите «Главное меню» чтобы продолжить.")

			callbackData := "mainMenu"
			message.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					{{Text: "Главное меню", CallbackData: &callbackData}},
				},
			}

			mu.Lock()
			_, err = client.Send(message)
			mu.Unlock()
			if err != nil {
				return
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

type GetPVZ struct {
	Name string
	Client tgbotapi.BotAPI
	mu *sync.Mutex
}

func NewGetPVZHandler(client tgbotapi.BotAPI) *GetPVZ {
	return &GetPVZ{
		Name: "getPVZ",
		Client: client,
		mu: &sync.Mutex{},
	}
}

// GetPVZFunc обрабатывает ввод номера телефона и запрашивает адрес ПВЗ
// client - экземпляр Telegram бота
// update - обновление от Telegram API
// stepParams - параметры шага регистрации
// Возвращает ошибку, если что-то пошло не так
func (g GetPVZ) Run(update tgbotapi.Update) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    var wg sync.WaitGroup
	mu := sync.Mutex{}
    var err error

    wg.Add(1)
    go func() {
        defer wg.Done()
        select {
        case <-ctx.Done():
            return
        default:
			data := ParseCallData(update.CallbackQuery.Data)
			service := data["service"]

			servisePVZName := ""
			switch service {
			case "cdek":
				servisePVZName = "CDEK"
			case "yandex":
				servisePVZName = "Яндекс доставки"
			}

			db := database.Connect()
			defer db.Close()
		
			message := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("Введите адрес пвз для сервиса %s (не забудьте указать город) ", servisePVZName))
			mu.Lock()
			_, err = g.Client.Send(message)
			mu.Unlock()
			if err != nil {
				return
			}
		
			user := models.TelegramUser{ID: update.CallbackQuery.From.ID}
			err = user.GetOrCreate(update.CallbackQuery.From, *db)
			if err != nil {
				return
			}
		
			_, err = db.Model(&user).
				WherePK().
				Set("delivery_service = ?", service).
				Update()
			if err != nil {
				return
			}
				
			stepKey := controllers.NextStepKey{
				ChatID: update.CallbackQuery.Message.Chat.ID,
				UserID: update.CallbackQuery.From.ID,
			}
			stepAction := controllers.NextStepAction{
				Func:        RegistrationCompleted,
				Params:      make(map[string]any),
				CreatedAtTS: time.Now().Unix(),
			}
		
			mu.Lock()
			controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)
			mu.Unlock()
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

func (g GetPVZ) GetName() string {
	return g.Name
}

func GetDeliveryServiceFunc(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    var wg sync.WaitGroup
	mu := sync.Mutex{}
    var err error

    wg.Add(1)
    go func() {
        defer wg.Done()
        select {
        case <-ctx.Done():
            return
        default:
			regex := regexp.MustCompile(`^[0-9]{11}$`)
			if !regex.MatchString(update.Message.Text) {
				message := tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный формат ввода!\n\nВведите номер телефона в формате 89991234567:")
				
				mu.Lock()
				_, err = client.Send(message)
				mu.Unlock()
				if err != nil {
					return
				}
				
				stepKey := controllers.NextStepKey{
					ChatID: update.Message.Chat.ID,
					UserID: update.Message.From.ID,
				}
				stepAction := controllers.NextStepAction{
					Func:        GetDeliveryServiceFunc,
					Params:      make(map[string]any),
					CreatedAtTS: time.Now().Unix(),
				}
		
				mu.Lock()
				controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)
				mu.Unlock()
		
				return
			}
		
			db := database.Connect()
			defer db.Close()
		
			message := tgbotapi.NewMessage(update.Message.Chat.ID, "выберите сервис доставки")
			cdekCallbackData := "selectDeliveryService?service=cdek"
			yandexCallbackData := "selectDeliveryService?service=yandex"
			message.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{
					{{Text: "CDEK", CallbackData: &cdekCallbackData}},
					{{Text: "Яндекс доставка", CallbackData: &yandexCallbackData}},
				},
			}
			mu.Lock()
			_, err = client.Send(message)
			mu.Unlock()
			if err != nil {
				return
			}
		
			user := models.TelegramUser{ID: update.Message.From.ID}
			err = user.GetOrCreate(update.Message.From, *db)
			if err != nil {
				return
			}
		
			_, err = db.Model(&user).
				WherePK().
				Set("phone = ?", update.Message.Text).
				Update()
			if err != nil {
				return
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

// RegisterPhoneNumberFunc обрабатывает ввод ФИО и запрашивает номер телефона
// client - экземпляр Telegram бота
// update - обновление от Telegram API
// stepParams - параметры шага регистрации
// Возвращает ошибку, если что-то пошло не так
func RegisterPhoneNumberFunc(client tgbotapi.BotAPI, update tgbotapi.Update, stepParams map[string]any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    var wg sync.WaitGroup
	mu := sync.Mutex{}
    var err error

    wg.Add(1)
    go func() {
        defer wg.Done()
        select {
        case <-ctx.Done():
            return
        default:
			db := database.Connect()
			defer db.Close()

			message := tgbotapi.NewMessage(update.Message.Chat.ID, "Введите номер телефона:\n<i>Пример ввода: 89991234567</i>")
			message.ParseMode = "HTML"
			mu.Lock()
			_, err = client.Send(message)
			mu.Unlock()
			if err != nil {
				return
			}

			user := models.TelegramUser{ID: update.Message.From.ID}
			err = user.GetOrCreate(update.Message.From, *db)
			if err != nil {
				return
			}

			_, err = db.Model(&user).
				WherePK().
				Set("fio = ?", update.Message.Text).
				Update()
			if err != nil {
				return
			}

			stepKey := controllers.NextStepKey{
				ChatID: update.Message.Chat.ID,
				UserID: update.Message.From.ID,
			}
			stepAction := controllers.NextStepAction{
				Func:        GetDeliveryServiceFunc,
				Params:      make(map[string]any),
				CreatedAtTS: time.Now().Unix(),
			}

			mu.Lock()
			controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)
			mu.Unlock()
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

// Run запускает процесс регистрации пользователя
// update - обновление от Telegram API
// Возвращает ошибку, если что-то пошло не так
func (r RegisterUser) Run(update tgbotapi.Update) error {
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
			r.mu.Lock()
			ClearNextStepForUser(update, &r.Client, true)
			r.mu.Unlock()
		
			message := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Введите ФИО")
			r.mu.Lock()
			_, err = r.Client.Send(message)
			r.mu.Unlock()
			if err != nil {
				return
			}
		
			stepKey := controllers.NextStepKey{
				ChatID: update.CallbackQuery.Message.Chat.ID,
				UserID: update.CallbackQuery.From.ID,
			}
			stepAction := controllers.NextStepAction{
				Func:        RegisterPhoneNumberFunc,
				Params:      make(map[string]any),
				CreatedAtTS: time.Now().Unix(),
			}
		
			r.mu.Lock()
			controllers.GetNextStepManager().RegisterNextStepAction(stepKey, stepAction)
			r.mu.Unlock()
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
func (r RegisterUser) GetName() string {
	return r.Name
}
