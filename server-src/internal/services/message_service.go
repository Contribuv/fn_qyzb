package services

import (
	"log"
	"time"

	"qyzb-server/internal/database"
	"qyzb-server/internal/models"
	"qyzb-server/internal/utils"
)

type MessageService struct{}

func NewMessageService() *MessageService {
	return &MessageService{}
}

func (s *MessageService) Create(roomID, userID uint, content, msgType string) (*models.MessageDTO, error) {
	log.Printf("[MessageService] Create: roomID=%d, userID=%d, type=%s, content=%s", roomID, userID, msgType, content)
	msg := models.Message{
		RoomID:    roomID,
		UserID:    userID,
		Content:   content,
		Type:      msgType,
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	if err := database.DB.Create(&msg).Error; err != nil {
		log.Printf("[MessageService] Create 失败: %v", err)
		return nil, err
	}
	log.Printf("[MessageService] Create 成功: msgID=%d", msg.ID)
	return s.GetByID(msg.ID)
}

func (s *MessageService) GetByID(id uint) (*models.MessageDTO, error) {
	var msg models.Message
	if err := database.DB.First(&msg, id).Error; err != nil {
		log.Printf("[MessageService] GetByID(%d) 失败: %v", id, err)
		return nil, err
	}
	return s.toDTO(&msg), nil
}

func (s *MessageService) GetByRoomID(roomID uint) ([]models.MessageDTO, error) {
	var messages []models.Message
	result := database.DB.
		Where("room_id = ?", roomID).
		Order("created_at asc").
		Find(&messages)
	if result.Error != nil {
		log.Printf("[MessageService] GetByRoomID(%d) 失败: %v", roomID, result.Error)
		return nil, result.Error
	}
	return s.toDTOList(messages), nil
}

func (s *MessageService) GetLatest(roomID uint, limit int) ([]models.MessageDTO, error) {
	log.Printf("[MessageService] GetLatest: roomID=%d, limit=%d", roomID, limit)
	var messages []models.Message
	result := database.DB.
		Where("room_id = ?", roomID).
		Order("created_at desc").
		Limit(limit).
		Find(&messages)
	if result.Error != nil {
		log.Printf("[MessageService] GetLatest(%d, %d) 失败: %v", roomID, limit, result.Error)
		return nil, result.Error
	}
	log.Printf("[MessageService] GetLatest 成功: %d 条消息", len(messages))

	dtos := s.toDTOList(messages)
	for i, j := 0, len(dtos)-1; i < j; i, j = i+1, j-1 {
		dtos[i], dtos[j] = dtos[j], dtos[i]
	}
	return dtos, nil
}

func (s *MessageService) ClearByRoomID(roomID uint) error {
	return database.DB.Where("room_id = ?", roomID).Delete(&models.Message{}).Error
}

func (s *MessageService) Count() (int64, error) {
	var count int64
	result := database.DB.Model(&models.Message{}).Count(&count)
	return count, result.Error
}

func (s *MessageService) GetRecentAll(limit int) ([]models.MessageDTO, error) {
	var messages []models.Message
	result := database.DB.
		Order("created_at desc").
		Limit(limit).
		Find(&messages)
	if result.Error != nil {
		return nil, result.Error
	}
	return s.toDTOList(messages), nil
}

func (s *MessageService) GetAllPaginated(page, pageSize int) ([]models.MessageDTO, int64, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 5 {
		pageSize = 5
	}
	if pageSize > 100 {
		pageSize = 100
	}
	var total int64
	database.DB.Model(&models.Message{}).Count(&total)
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	var messages []models.Message
	result := database.DB.
		Order("created_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&messages)
	if result.Error != nil {
		return nil, 0, 0, result.Error
	}
	return s.toDTOList(messages), total, totalPages, nil
}

func (s *MessageService) toDTOList(messages []models.Message) []models.MessageDTO {
	if len(messages) == 0 {
		return []models.MessageDTO{}
	}

	userIDs := make([]uint, 0, len(messages))
	roomIDs := make([]uint, 0, len(messages))
	userIDSet := make(map[uint]bool)
	roomIDSet := make(map[uint]bool)

	for _, msg := range messages {
		if !userIDSet[msg.UserID] {
			userIDs = append(userIDs, msg.UserID)
			userIDSet[msg.UserID] = true
		}
		if !roomIDSet[msg.RoomID] {
			roomIDs = append(roomIDs, msg.RoomID)
			roomIDSet[msg.RoomID] = true
		}
	}

	userMap := make(map[uint]models.User)
	if len(userIDs) > 0 {
		var users []models.User
		database.DB.Where("id IN ?", userIDs).Find(&users)
		for _, u := range users {
			userMap[u.ID] = u
		}
	}

	roomMap := make(map[uint]models.Room)
	if len(roomIDs) > 0 {
		var rooms []models.Room
		database.DB.Where("id IN ?", roomIDs).Find(&rooms)
		for _, r := range rooms {
			roomMap[r.ID] = r
		}
	}

	dtos := make([]models.MessageDTO, len(messages))
	for i, msg := range messages {
		user := userMap[msg.UserID]
		room := roomMap[msg.RoomID]
		dtos[i] = s.buildDTO(&msg, &user, &room)
	}
	return dtos
}

func (s *MessageService) toDTO(msg *models.Message) *models.MessageDTO {
	var user models.User
	var room models.Room
	database.DB.First(&user, msg.UserID)
	database.DB.First(&room, msg.RoomID)
	dto := s.buildDTO(msg, &user, &room)
	return &dto
}

func (s *MessageService) buildDTO(msg *models.Message, user *models.User, room *models.Room) models.MessageDTO {
	timeStr, timeText := utils.FormatTimeString(msg.CreatedAt)

	nickname := user.Nickname
	if nickname == "" {
		nickname = user.Username
	}
	avatar := utils.FixAvatarPath(user.Avatar)

	return models.MessageDTO{
		ID:             msg.ID,
		RoomID:         msg.RoomID,
		RoomIDSnake:    msg.RoomID,
		UserID:         msg.UserID,
		UserIDSnake:    msg.UserID,
		Content:        msg.Content,
		Type:           msg.Type,
		Nickname:       nickname,
		Username:       user.Username,
		Avatar:         avatar,
		RoomName:       room.Name,
		CreatedAt:      timeStr,
		CreatedAtSnake: timeStr,
		CreatedAtText:  timeText,
	}
}
