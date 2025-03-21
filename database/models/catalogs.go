package models

import "github.com/go-pg/pg/v10"

type Catalog struct {
	ID int `json:"id"`
	Name string `json:"name"`
}

func (c *Catalog) GetProductCount(db pg.DB) (int, error) {
	products := []Product{}
	err := db.Model(&products).Where("catalog_id = ?", c.ID).Select()
	if err != nil {
		return 0, err
	}

	return len(products), nil
}

type Product struct {
	ID int `json:"id"`
	ImageFileID string `json:"image_file_id"`
	Name string `json:"name"`
	Description string `json:"description"`
	Price int `json:"price"`
	CatalogID int `json:"catalog_id"`
	Catalog *Catalog `pg:"rel:has-one"`
}

func (p *Product) InUserCart(userId int64, db pg.DB) (bool, error) {
	cart := []ShoppingCart{}
	err := db.Model(&cart).Where("user_id = ?", userId).Where("product_id = ?", p.ID).Select()
	if err != nil {
		return false, err
	}

	return len(cart) > 0, nil
}
