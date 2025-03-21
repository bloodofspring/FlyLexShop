package models

import (
	"github.com/go-pg/pg/v10"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramUser struct {
	ID int64 `json:"id"`

	CreatedAtTS int64 `pg:",default:extract(epoch from now())" json:"created_at_ts"`
	UpdatedAtTS int64 `pg:",default:extract(epoch from now())" json:"updated_at_ts"`

	FIO string `json:"name"`
	Phone string `json:"phone"`
	DeliveryAddress string `json:"delivery_address"`
	DeliveryService string `json:"delivery_service"`

	Username string `json:"username"`
	FirstName string `json:"first_name"`
	LastName string `json:"last_name"`
	IsAdmin bool `pg:",default:false" json:"is_admin"`
}

func (u *TelegramUser) UpdateProfileData(apiUser *tgbotapi.User) {
	u.Username = apiUser.UserName
	u.FirstName = apiUser.FirstName
	u.LastName = apiUser.LastName
}

func (u *TelegramUser) GetOrCreate(apiUser *tgbotapi.User, db pg.DB) error {
	err := db.Model(u).Where("id = ?", u.ID).Select()

	u.UpdateProfileData(apiUser)

	if err == pg.ErrNoRows {
		_, err = db.Model(u).Insert()
	}

	return err
}
