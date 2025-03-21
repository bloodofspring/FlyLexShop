package main

import (
	"encoding/json"
	"log"
	"main/actions"
	"main/controllers"
	"main/database"
	"main/filters"
	"main/handlers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

const debug = true

func connect() *tgbotapi.BotAPI {
	envFile, _ := godotenv.Read(".env")

	bot, err := tgbotapi.NewBotAPI(envFile["API_KEY"])
	if err != nil {
		panic(err)
	}

	log.Printf("Successfully authorized on account @%s", bot.Self.UserName)

	return bot
}

func getBotActions(bot tgbotapi.BotAPI) handlers.ActiveHandlers {
	act := handlers.ActiveHandlers{Handlers: []handlers.Handler{
		handlers.CommandHandler.Product(actions.SayHi{Name: "start-cmd", Client: bot}, []handlers.Filter{filters.StartFilter}),
		handlers.CallbackQueryHandler.Product(actions.RegisterUser{Name: "reg-user", Client: bot}, []handlers.Filter{filters.RegisterUserFilter}),
		handlers.CommandHandler.Product(actions.MainMenu{Name: "main-menu-cmd", Client: bot}, []handlers.Filter{filters.ToMainMenuFilter}),
		handlers.CallbackQueryHandler.Product(actions.MainMenu{Name: "main-menu-btn", Client: bot}, []handlers.Filter{filters.MainMenuFilter}),
		handlers.CallbackQueryHandler.Product(actions.ProfileSettings{Name: "profile-settings", Client: bot}, []handlers.Filter{filters.ProfileSettingsFilter}),
		handlers.CallbackQueryHandler.Product(actions.Shop{Name: "shop", Client: bot}, []handlers.Filter{filters.ShopFilter}),
		handlers.CallbackQueryHandler.Product(actions.About{Name: "about", Client: bot}, []handlers.Filter{filters.AboutFilter}),
		handlers.CallbackQueryHandler.Product(actions.ChangeName{Name: "change-name", Client: bot}, []handlers.Filter{filters.ChangeNameFilter}),
		handlers.CallbackQueryHandler.Product(actions.ChangePhone{Name: "change-phone", Client: bot}, []handlers.Filter{filters.ChangePhoneFilter}),
		handlers.CallbackQueryHandler.Product(actions.ChangeDeliveryAddress{Name: "change-delivery-address", Client: bot}, []handlers.Filter{filters.ChangeDeliveryAddressFilter}),
		handlers.CallbackQueryHandler.Product(actions.ChangeDeliveryService{Name: "change-delivery-service", Client: bot}, []handlers.Filter{filters.ChangeDeliveryServiceFilter}),
	}}

	return act
}

func printUpdate(update *tgbotapi.Update) {
	updateJSON, err := json.MarshalIndent(update, "", "    ")
	if err != nil {
		return
	}

	log.Println(string(updateJSON))
}

func main() {
	err := database.InitDb()
	if err != nil {
		panic(err)
	}

	if debug {
		log.Println("\033[1m\033[93mWARNING! Set debug to false before push!\033[0m")
	}
	log.Println("Database init finished without errors!")

	client := connect()
	act := getBotActions(*client)

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	stepManager := controllers.GetNextStepManager()

	updates := client.GetUpdatesChan(updateConfig)
	for update := range updates {
		if debug {
			printUpdate(&update)
		}

		_ = act.HandleAll(update)

		controllers.RunStepUpdates(update, stepManager, *client)
	}
}
