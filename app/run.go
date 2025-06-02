package main

import (
	"context"
	"encoding/json"
	"main/actions"
	"main/controllers"
	"main/database"
	"main/filters"
	"main/handlers"
	"main/logger"
	"main/metrics"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

const (
	debug = false
	maxWorkers      = 50
	shutdownTimeout = 5 * time.Second
	metricsInterval = 12 * time.Hour
)


func connect() *tgbotapi.BotAPI {
	_ = godotenv.Load() // Для dev-режима подхватит .env, в проде проигнорирует

	bot, err := tgbotapi.NewBotAPI(os.Getenv("API_KEY"))
	if err != nil {
		panic(err)
	}

	logger.GetLogger().Info("Successfully authorized on account @%s", bot.Self.UserName)

	return bot
}

func getBotActions(bot tgbotapi.BotAPI) handlers.ActiveHandlers {
	act := handlers.ActiveHandlers{Handlers: []handlers.Handler{
		handlers.CommandHandler.Product(actions.NewSayHiHandler(bot), []handlers.Filter{filters.StartFilter}),
		handlers.CallbackQueryHandler.Product(actions.NewRegisterUserHandler(bot), []handlers.Filter{filters.RegisterUserFilter}),
		handlers.CallbackQueryHandler.Product(actions.NewGetPVZHandler(bot), []handlers.Filter{filters.SelectDeliveryServiceFilter}),
		
		handlers.CommandHandler.Product(actions.NewMainMenuHandler(bot), []handlers.Filter{filters.ToMainMenuFilter}),
		handlers.CallbackQueryHandler.Product(actions.NewMainMenuHandler(bot), []handlers.Filter{filters.MainMenuFilter}),

		handlers.CallbackQueryHandler.Product(actions.NewAboutHandler(bot), []handlers.Filter{filters.AboutFilter}),

		handlers.CallbackQueryHandler.Product(actions.ProfileSettings{Name: "profile-settings", Client: bot}, []handlers.Filter{filters.ProfileSettingsFilter}),
		handlers.CallbackQueryHandler.Product(actions.ChangeName{Name: "change-name", Client: bot}, []handlers.Filter{filters.ChangeNameFilter}),
		handlers.CallbackQueryHandler.Product(actions.ChangePhone{Name: "change-phone", Client: bot}, []handlers.Filter{filters.ChangePhoneFilter}),
		handlers.CallbackQueryHandler.Product(actions.ChangeDeliveryAddress{Name: "change-delivery-address", Client: bot}, []handlers.Filter{filters.ChangeDeliveryAddressFilter}),
		handlers.CallbackQueryHandler.Product(actions.ChangeDeliveryService{Name: "change-delivery-service", Client: bot}, []handlers.Filter{filters.ChangeDeliveryServiceFilter}),

		handlers.CallbackQueryHandler.Product(actions.NewShopHandler(bot), []handlers.Filter{filters.ShopFilter}),
		handlers.CallbackQueryHandler.Product(actions.NewViewCatalogHandler(bot), []handlers.Filter{filters.ViewCatalogFilter}),
		handlers.CallbackQueryHandler.Product(actions.NewViewCartHandler(bot), []handlers.Filter{filters.ViewCartFilter}),

		handlers.CallbackQueryHandler.Product(actions.NewMakeOrderHandler(bot), []handlers.Filter{filters.MakeOrderFilter}),
		handlers.CallbackQueryHandler.Product(actions.NewProcessOrderHandler(bot), []handlers.Filter{filters.ProcessOrderFilter}),
		handlers.CallbackQueryHandler.Product(actions.NewPaymentVerdictHandler(bot), []handlers.Filter{filters.PaymentVerdictFilter}),
	
		handlers.CallbackQueryHandler.Product(actions.NewAddCatalogHandler(bot), []handlers.Filter{filters.AddCatalogFilter}),
		handlers.CallbackQueryHandler.Product(actions.NewEditShopHandler(bot), []handlers.Filter{filters.EditShopFilter}),

		handlers.CallbackQueryHandler.Product(actions.NewChangeCatalogNameHandler(bot), []handlers.Filter{filters.ChangeCatalogNameFilter}),

		handlers.CallbackQueryHandler.Product(actions.NewCancelHandler(bot), []handlers.Filter{filters.CancelFilter}),
	}}

	return act
}

func printUpdate(update *tgbotapi.Update) {
	updateJSON, err := json.MarshalIndent(update, "", "    ")
	if err != nil {
		return
	}

	logger.GetLogger().Debug("Update: %s", string(updateJSON))
}

func main() {
	log := logger.GetLogger()
	if debug {
		log.SetLevel(logger.Debug)
	}

	metrics := metrics.GetMetrics()


	err := database.InitDb()
	if err != nil {
		log.Fatal("Failed to initialize database: %v", err)
	}

	client := connect()
	act := getBotActions(*client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(metricsInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				stats := metrics.GetStats()
				log.Info("Metrics: %+v", stats)
			case <-ctx.Done():
				return
			}
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Info("Received signal: %v", sig)
		cancel()
	}()

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := client.GetUpdatesChan(updateConfig)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)
	stepManager := controllers.GetNextStepManager()

	for {
		select {
		case update := <-updates:
			if debug {
				printUpdate(&update)
			}

			select {
			case semaphore <- struct{}{}:
				wg.Add(1)
				startTime := time.Now()

				go func(update tgbotapi.Update) {
					defer func() {
						<-semaphore
						wg.Done()

						duration := time.Since(startTime)
						metrics.RecordMessageProcessing(duration, true)
						metrics.RecordGoroutineCount(runtime.NumGoroutine())
					}()

					defer func() {
						if r := recover(); r != nil {
							log.Error("Panic in handler: %v", r)
							metrics.RecordError("panic")
						}
					}()

					_, updateCancel := context.WithTimeout(ctx, 30*time.Second)
					defer updateCancel()

					afterUpdate := act.HandleAll(update, *client)
					for _, afterUpdate := range afterUpdate {
						if afterUpdate.Error != nil {
							log.Error("Error handling %s update: %v", afterUpdate.Name, afterUpdate.Error)
							metrics.RecordError("handler_error")
						}
					}

					controllers.RunStepUpdates(update, stepManager, *client)
				}(update)
			case <-ctx.Done():
				goto shutdown
			}
		case <-ctx.Done():
			goto shutdown
		}
	}

shutdown:
	log.Info("Shutting down...")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("All handlers completed successfully")
	case <-time.After(shutdownTimeout):
		log.Warning("Shutdown timeout reached, some handlers may not have completed")
	}

	stats := metrics.GetStats()
	log.Info("Final metrics: %+v", stats)

	log.Info("Bot shutdown complete")
}
