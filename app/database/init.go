package database

import (
	"main/database/models"
	"os"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
)

func Connect() *pg.DB {
	db := pg.Connect(&pg.Options{
		Addr:     os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Database: os.Getenv("DB_NAME"),
	})

	return db
}

func InitDb() error {
	db := Connect()
	defer db.Close()

	models := []interface{}{
		(*models.TelegramUser)(nil),
		(*models.Catalog)(nil),
		(*models.Product)(nil),
		(*models.AddedProducts)(nil),
		(*models.Transaction)(nil),
		(*models.ShopViewSession)(nil),
	}

	for _, model := range models {
		err := db.Model(model).CreateTable(&orm.CreateTableOptions{
			Temp:        false,
			IfNotExists: true,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
