package models

import (
	"github.com/go-pg/pg/v10"
)

type Catalog struct {
	ID int `json:"id"`

	CreatedAt int64 `pg:",default:extract(epoch from now())"`

	Name string `json:"name"`

	Products     []*Product         `pg:"rel:has-many,join_fk:catalog_id"`
	ShopSessions []*ShopViewSession `pg:"rel:has-many,join_fk:catalog_id"`
}

func (c *Catalog) GetProductCount(db *pg.DB) (int, error) {
	count, err := db.Model(&[]Product{}).
		Where("catalog_id = ?", c.ID).
		Count()
	if err != nil {
		return 0, err
	}

	return count, nil
}

type Product struct {
	ID int `json:"id"`

	CreatedAt int64 `pg:",default:extract(epoch from now())"`

	ImageFileID string `json:"image_file_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int    `json:"price"`
	CatalogID   int    `json:"catalog_id"`

	AvailbleForPurchase int

	Catalog      *Catalog           `pg:"rel:has-one,fk:catalog_id"`
	ShopSessions []*ShopViewSession `pg:"rel:has-many,join_fk:product_at_id"`
}

func (p *Product) InUserCart(userId int64, db pg.DB) (bool, error) {
	cart := []AddedProducts{}
	err := db.Model(&cart).Where("user_id = ?", userId).Where("product_id = ?", p.ID).Select()
	if err != nil {
		return false, err
	}

	return len(cart) > 0, nil
}
