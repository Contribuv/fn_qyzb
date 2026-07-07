package models

type Room struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"size:100;not null" json:"name"`
	Description string `gorm:"size:255" json:"description"`
	CreatedAt   string `json:"createdAt"`
}

func (Room) TableName() string {
	return "rooms"
}
