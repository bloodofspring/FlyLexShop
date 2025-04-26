package controllers

import (
	"errors"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	StepTimeout = 3600
)

var (
	ErrMessageIsCommand = errors.New("message is command")
)

type NextStepKey struct {
	ChatID int64
	UserID int64
}

type NextStepAction struct {
	Func        func(client tgbotapi.BotAPI, stepUpdate tgbotapi.Update, stepParams map[string]any) error
	Params      map[string]any
	CreatedAtTS int64
	CancelMessage string
}

type NextStepManager struct {
	nextStepActions map[NextStepKey]NextStepAction
}

// Глобальный экземпляр NextStepManager
var GlobalNextStepManager = &NextStepManager{
	nextStepActions: make(map[NextStepKey]NextStepAction),
}

// GetNextStepManager возвращает глобальный экземпляр NextStepManager
func GetNextStepManager() *NextStepManager {
	return GlobalNextStepManager
}

func (n *NextStepManager) RegisterNextStepAction(stepKey NextStepKey, action NextStepAction) {
	n.nextStepActions[stepKey] = action
}

func (n NextStepManager) RemoveNextStepAction(stepKey NextStepKey, bot tgbotapi.BotAPI, sendCancelMessage bool) {
	if sendCancelMessage && n.nextStepActions[stepKey].CancelMessage != "" {
		bot.Send(tgbotapi.NewMessage(stepKey.ChatID, n.nextStepActions[stepKey].CancelMessage))
	}

	delete(n.nextStepActions, stepKey)
}

func (n NextStepManager) RunUpdates(update tgbotapi.Update, client tgbotapi.BotAPI) error {
	if update.Message == nil {
		return nil
	}

	key := NextStepKey{ChatID: update.Message.Chat.ID, UserID: update.Message.From.ID}

	action, ok := n.nextStepActions[key]

	if !ok {
		return nil
	}

	if update.Message.IsCommand() {
		return ErrMessageIsCommand
	}

	return action.Func(client, update, action.Params)
}

func (n *NextStepManager) ClearOldSteps(client tgbotapi.BotAPI) (int, error) {
	now := time.Now().Unix()
	deleted := 0

	for key, action := range n.nextStepActions {
		if now-action.CreatedAtTS > StepTimeout {
			n.RemoveNextStepAction(key, client, true)
			deleted++
		}
	}

	return deleted, nil
}

func RunStepUpdates(update tgbotapi.Update, stepManager *NextStepManager, client tgbotapi.BotAPI) {
	err := stepManager.RunUpdates(update, client)
	if err != nil {
		log.Printf("run next steps says: %v\n", err)
	}

	stepsCleaned, err := stepManager.ClearOldSteps(client)
	if err != nil {
		log.Printf("clear old steps says: %v\n", err)
	} else if stepsCleaned != 0 {
		log.Printf("Cleaned %d old steps\n", stepsCleaned)
	}
}
