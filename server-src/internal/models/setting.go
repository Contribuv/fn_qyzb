package models

type Setting struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	KeyName   string `gorm:"column:key_name;size:100;uniqueIndex;not null" json:"key"`
	Value     string `gorm:"column:value;type:text" json:"value"`
	UpdatedAt string `gorm:"column:updated_at" json:"updatedAt"`
}

func (Setting) TableName() string {
	return "settings"
}
