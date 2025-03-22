package models

import (
	"github.com/go-pg/pg/v10"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramUser struct {
	ID int64 `json:"id"`

	CreatedAtTS int64 `pg:",default:extract(epoch from now())" json:"created_at_ts"`
	UpdatedAtTS int64 `pg:",default:extract(epoch from now())" json:"updated_at_ts"`

	FIO string `pg:",default:null" json:"name"`
	Phone string `pg:",default:null" json:"phone"`
	DeliveryAddress string `pg:",default:null" json:"delivery_address"`
	DeliveryService string `pg:",default:cdek" json:"delivery_service"`

	IsAuthorized bool `pg:",default:false" json:"is_authorized"`

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

func (u *TelegramUser) GetTotalCartPrice(db pg.DB) (int, error) {
	cart := []ShoppingCart{}
	err := db.Model(&cart).Where("user_id = ?", u.ID).Select()
	if err != nil {
		return 0, err
	}

	totalPrice := 0
	for _, item := range cart {
		var product Product
		err = db.Model(&product).Where("id = ?", item.ProductID).Select()
		if err != nil {
			continue
		}
		totalPrice += product.Price
	}
	return totalPrice, nil
}

type ShoppingCart struct {
	ID int `json:"id"`
	UserID int64 `json:"user_id"`
	User *TelegramUser `pg:"rel:has-one"`
	ProductID int `json:"product_id"`
	Product *Product `pg:"rel:has-one"`
}
