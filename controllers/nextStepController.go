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
	Func        func(stepUpdate tgbotapi.Update, stepParams map[string]any) error
	Params      map[string]any
	CreatedAtTS int64
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

func (n NextStepManager) RemoveNextStepAction(stepKey NextStepKey) {
	delete(n.nextStepActions, stepKey)
}

func (n NextStepManager) RunUpdates(update tgbotapi.Update) error {
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

	return action.Func(update, action.Params)
}

func (n *NextStepManager) ClearOldSteps() (int, error) {
	now := time.Now().Unix()
	deleted := 0

	for key, action := range n.nextStepActions {
		if now-action.CreatedAtTS > StepTimeout {
			delete(n.nextStepActions, key)
			deleted++
		}
	}

	return deleted, nil
}

func RunStepUpdates(update tgbotapi.Update, stepManager *NextStepManager) {
	err := stepManager.RunUpdates(update)
	if err != nil {
		log.Printf("run next steps says: %v\n", err)
	}

	stepsCleaned, err := stepManager.ClearOldSteps()
	if err != nil {
		log.Printf("clear old steps says: %v\n", err)
	} else if stepsCleaned != 0 {
		log.Printf("Cleaned %d old steps\n", stepsCleaned)
	}
}
