package models

type Message struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	RoomID    uint   `gorm:"index;not null" json:"roomId"`
	UserID    uint   `gorm:"index;not null" json:"userId"`
	Content   string `gorm:"type:text;not null" json:"content"`
	Type      string `gorm:"size:20;default:text" json:"type"`
	CreatedAt string `json:"createdAt"`
}

func (Message) TableName() string {
	return "messages"
}

type MessageDTO struct {
	ID             uint   `json:"id"`
	RoomID         uint   `json:"roomId"`
	RoomIDSnake    uint   `json:"room_id"`
	UserID         uint   `json:"userId"`
	UserIDSnake    uint   `json:"user_id"`
	Content        string `json:"content"`
	Type           string `json:"type"`
	Nickname       string `json:"nickname"`
	Username       string `json:"username"`
	Avatar         string `json:"avatar"`
	RoomName       string `json:"roomName"`
	CreatedAt      string `json:"createdAt"`
	CreatedAtSnake string `json:"created_at"`
	CreatedAtText  string `json:"-"`
}
