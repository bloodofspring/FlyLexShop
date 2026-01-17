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

	createForeignKeys(db)

	return nil
}

func createForeignKeys(db *pg.DB) error {
	fks := []string{
		`ALTER TABLE added_products
		ADD CONSTRAINT fk_added_products_transaction
		FOREIGN KEY (transaction_id)
		REFERENCES transactions(id)
		ON DELETE CASCADE;`,

		`ALTER TABLE added_products
		ADD CONSTRAINT fk_added_products_product
		FOREIGN KEY (product_id)
		REFERENCES products(id)
		ON DELETE CASCADE;`,

		`ALTER TABLE added_products
		ADD CONSTRAINT fk_added_products_user
		FOREIGN KEY (user_id)
		REFERENCES telegram_users(id)
		ON DELETE CASCADE;`,

		`ALTER TABLE products
		ADD CONSTRAINT fk_products_catalog
		FOREIGN KEY (catalog_id) REFERENCES catalogs(id)
		ON DELETE CASCADE;`,
	}

	for _, fk := range fks {
		db.Exec(fk)
	}

	return nil
}
