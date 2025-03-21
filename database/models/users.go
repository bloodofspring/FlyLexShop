package models

type TelegramUser struct {
	ID int64 `json:"id"`

	CreatedAtTS int64 `pg:",default:extract(epoch from now())" json:"created_at_ts"`
	UpdatedAtTS int64 `pg:",default:extract(epoch from now())" json:"updated_at_ts"`

	Username string `json:"username"`
	FirstName string `json:"first_name"`
	LastName string `json:"last_name"`
	IsAdmin bool `json:"is_admin"`
}
