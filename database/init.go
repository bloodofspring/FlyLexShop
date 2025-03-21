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
		Password: envFile["db_password"],
		Database: envFile["db_name"], // bigBrotherBotDb
	})

	return db
}

func InitDb() error {
	db := Connect()
	defer db.Close()

	models := []interface{}{
		(*models.TelegramUser)(nil),
	}

	for _, model := range models {
		err := db.Model(model).CreateTable(&orm.CreateTableOptions{
			Temp:        false, // Временные таблицы
			IfNotExists: true,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
