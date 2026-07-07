package services

import (
	"time"

	"gorm.io/gorm"

	"qyzb-server/internal/database"
	"qyzb-server/internal/models"
)

type RoomService struct{}

func NewRoomService() *RoomService {
	return &RoomService{}
}

func (s *RoomService) GetAll() ([]models.Room, error) {
	var rooms []models.Room
	result := database.DB.Order("id asc").Find(&rooms)
	return rooms, result.Error
}

func (s *RoomService) GetPaginated(page, pageSize int) ([]models.Room, int64, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	var total int64
	database.DB.Model(&models.Room{}).Count(&total)
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	var rooms []models.Room
	result := database.DB.Order("id asc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rooms)
	return rooms, total, totalPages, result.Error
}

func (s *RoomService) Count() (int64, error) {
	var count int64
	result := database.DB.Model(&models.Room{}).Count(&count)
	return count, result.Error
}

func (s *RoomService) GetByID(id uint) (*models.Room, error) {
	var room models.Room
	result := database.DB.First(&room, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &room, nil
}

func (s *RoomService) Create(name, description string) error {
	room := models.Room{
		Name:        name,
		Description: description,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}
	return database.DB.Create(&room).Error
}

func (s *RoomService) Update(id uint, name, description string) error {
	return database.DB.Model(&models.Room{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":        name,
		"description": description,
	}).Error
}

func (s *RoomService) Delete(id uint) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("room_id = ?", id).Delete(&models.Message{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.Room{}, id).Error; err != nil {
			return err
		}
		return nil
	})
}
