package models

type ShopViewSession struct {
	ID     int
	UserID int64
	User   *TelegramUser `pg:"rel:has-one,fk:user_id"`
	ChatID int64

	CreatedAt int64 `pg:",default:extract(epoch from now())"`
	UpdatedAt int64 `pg:",default:extract(epoch from now())"`

	CatalogID   int      `pg:",fk:catalog_id"`
	Catalog     *Catalog `pg:"rel:belongs-to,fk:catalog_id"`
	ProductAtID int
	ProductAt   *Product `pg:"rel:has-one,fk:product_at_id"`

	Offest int `pg:",default:0"`
}
