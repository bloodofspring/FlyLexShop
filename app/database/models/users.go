package models

import (
	"sync"

	"github.com/go-pg/pg/v10"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramUser struct {
	ID int64 `json:"id"`

	mu sync.Mutex `pg:"-" json:"-"` // мьютекс для защиты конкурентного доступа

	CreatedAtTS int64 `pg:",default:extract(epoch from now())" json:"created_at_ts"`
	UpdatedAtTS int64 `pg:",default:extract(epoch from now())" json:"updated_at_ts"`

	FIO             string `pg:",default:null" json:"name"`
	Phone           string `pg:",default:null" json:"phone"`
	DeliveryAddress string `pg:",default:null" json:"delivery_address"`
	DeliveryService string `pg:",default:'cdek'" json:"delivery_service"`

	IsAuthorized bool `pg:",default:false" json:"is_authorized"`

	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	IsAdmin   bool   `pg:",default:false" json:"is_admin"`

	ShopSession *ShopViewSession `pg:"rel:has-one,fk:id,join_fk:user_id"`
}

func (u *TelegramUser) UpdateProfileData(apiUser *tgbotapi.User, db *pg.DB) error {
	u.Username = apiUser.UserName
	u.FirstName = apiUser.FirstName
	u.LastName = apiUser.LastName

	_, err := db.Model(u).
		Where("id = ?", u.ID).
		Column("username", "first_name", "last_name").
		Update()

	return err
}

func (u *TelegramUser) Get(db pg.DB) error {
	err := db.Model(u).Where("id = ?", u.ID).Select()

	return err
}

func (u *TelegramUser) GetOrCreate(apiUser *tgbotapi.User, db pg.DB) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	// Сначала пытаемся получить пользователя
	err := u.Get(db)

	// Если пользователь не найден, создаем нового
	if err == pg.ErrNoRows {
		u.ID = apiUser.ID
		u.Username = apiUser.UserName
		u.FirstName = apiUser.FirstName
		u.LastName = apiUser.LastName

		_, err = db.Model(u).Insert()
		if err != nil {
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	// Если пользователь найден, обновляем его данные
	return u.UpdateProfileData(apiUser, &db)
}

func (u *TelegramUser) GetOrCreateTransaction(db pg.DB) (Transaction, error, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()

	var activeTransactions []Transaction

	err := db.Model(&activeTransactions).
		Where("user_id = ?", u.ID).
		Where("is_waiting_for_approval = ?", false).
		Select()
	if err != nil {
		return Transaction{}, err, false
	}

	if len(activeTransactions) > 0 {
		return activeTransactions[0], nil, false
	}

	transaction := Transaction{
		UserID: u.ID,
	}

	_, err = db.Model(&transaction).Insert()
	if err != nil {
		return Transaction{}, err, false
	}

	return transaction, nil, true
}

func (u *TelegramUser) GetProductInCartCount(db pg.DB, productID int) (int, error) {
	transaction, err, created := u.GetOrCreateTransaction(db)
	if err != nil {
		return 0, err
	}

	if created {
		return 0, nil
	}

	var product AddedProducts
	err = db.Model(&product).
		Where("user_id = ?", u.ID).
		Where("product_id = ?", productID).
		Where("transaction_id = ?", transaction.ID).
		Select()
	if err != nil {
		return 0, err
	}

	return product.ProductCount, nil
}

func (u *TelegramUser) AddProductToCart(db pg.DB, productID int) error {
	transaction, err, _ := u.GetOrCreateTransaction(db)
	if err != nil {
		return err
	}

	if productInCartCount, err := u.GetProductInCartCount(db, productID); err != nil {
		return err
	} else if productInCartCount > 0 {
		_, err := db.Model(&AddedProducts{}).
			Where("user_id = ?", u.ID).
			Where("product_id = ?", productID).
			Where("transaction_id = ?", transaction.ID).
			Set("product_count = product_count + 1").
			Update()
		if err != nil {
			return err
		}
	} else {
		_, err := db.Model(&AddedProducts{
			UserID:    u.ID,
			ProductID: productID,
			TransactionID: transaction.ID,
		}).Insert()
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *TelegramUser) RemoveProductFromCart(db pg.DB, productID int) error {
	transaction, err, _ := u.GetOrCreateTransaction(db)
	if err != nil {
		return err
	}

	if productInCartCount, err := u.GetProductInCartCount(db, productID); err != nil {
		return err
	} else if productInCartCount > 1 {
		_, err := db.Model(&AddedProducts{}).
			Where("user_id = ?", u.ID).
			Where("product_id = ?", productID).
			Where("transaction_id = ?", transaction.ID).
			Set("product_count = product_count - 1").
			Update()
		if err != nil {
			return err
		}
	} else if productInCartCount == 1 {
		_, err = db.Model(&AddedProducts{}).
			Where("user_id = ?", u.ID).
			Where("product_id = ?", productID).
			Where("transaction_id = ?", transaction.ID).
			Delete()
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *TelegramUser) GetTotalCartPrice(db pg.DB) (int, error) {
	transaction, err, _ := u.GetOrCreateTransaction(db)
	if err != nil {
		return 0, err
	}

	cart := []AddedProducts{}
	err = db.Model(&cart).
		Where("user_id = ?", u.ID).
		Where("transaction_id = ?", transaction.ID).
		Relation("Product").
		Select()
	if err != nil {
		return 0, err
	}

	totalPrice := 0
	for _, item := range cart {
		totalPrice += item.Product.Price * item.ProductCount
	}
	return totalPrice, nil
}

type Transaction struct {
	ID int `json:"id"`

	CreatedAtTS int64 `pg:",default:extract(epoch from now())" json:"created_at_ts"`
	UpdatedAtTS int64 `pg:",default:extract(epoch from now())" json:"updated_at_ts"`

	UserID int64         `json:"user_id"`
	User   *TelegramUser `pg:"rel:has-one,fk:user_id"`

	IsWaitingForApproval bool `pg:",default:false" json:"is_waiting_for_approval"`

	AddedProducts []*AddedProducts `pg:"rel:has-many,join_fk:transaction_id"`
}

type AddedProducts struct {
	ID int `json:"id"`

	UserID int64         `json:"user_id"`
	User   *TelegramUser `pg:"rel:has-one,fk:user_id"`

	ProductID    int      `json:"product_id"`
	Product      *Product `pg:"rel:has-one,fk:product_id"`
	ProductCount int      `pg:",default:1" json:"product_count"`

	TransactionID int       `json:"transaction_id"`
	Transaction   *Transaction `pg:"rel:has-one,fk:transaction_id"`
}
