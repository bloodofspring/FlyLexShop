package models


type ShopViewSession struct {
	Id int
	UserId int64
	ChatId int64

	CreatedAt int64 `pg:",default:extract(epoch from now())"`
	UpdatedAt int64 `pg:",default:extract(epoch from now())"`

	CatId int `pg:",default:null"`
	ProductAtId int `pg:",default:null"`
	PageNo int `pg:",default:null"`
}
