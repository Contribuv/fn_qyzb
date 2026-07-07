package models

type User struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Username  string `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Password  string `gorm:"size:255;not null" json:"-"`
	Nickname  string `gorm:"size:50" json:"nickname"`
	Avatar    string `gorm:"size:255" json:"avatar"`
	Role      string `gorm:"size:20;default:user" json:"role"`
	CreatedAt string `json:"createdAt"`
}

func (User) TableName() string {
	return "users"
}
