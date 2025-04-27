package database

import (
	"main/database/models"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/joho/godotenv"
)

func Connect() *pg.DB {
	envFile, _ := godotenv.Read(".env")
	db := pg.Connect(&pg.Options{
		Addr:     "localhost:5432",
		User:     "postgres",
		Password: envFile["DB_PASSWORD"],
		Database: envFile["DB_NAME"], // bigBrotherBotDb
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
		(*models.ShoppingCart)(nil),
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
