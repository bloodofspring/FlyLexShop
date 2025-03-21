package main

import (
	"encoding/json"
	"log"
	"main/actions"
	"main/database"
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
	startFilter := func(update tgbotapi.Update) bool { return update.Message.Command() == "start" }

	act := handlers.ActiveHandlers{Handlers: []handlers.Handler{
		// Place your handlers here
		handlers.CommandHandler.Product(actions.SayHi{Name: "start-cmd", Client: bot}, []handlers.Filter{startFilter}),
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

	updates := client.GetUpdatesChan(updateConfig)
	for update := range updates {
		if debug {
			printUpdate(&update)
		}

		_ = act.HandleAll(update)
	}
}
