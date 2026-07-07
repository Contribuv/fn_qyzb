package services

import (
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"qyzb-server/internal/database"
	"qyzb-server/internal/models"
	"qyzb-server/internal/utils"
)

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

func (s *UserService) Login(username, password string) (*models.User, error) {
	var user models.User
	result := database.DB.Where("username = ?", username).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户名或密码错误")
		}
		return nil, result.Error
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	return &user, nil
}

func (s *UserService) Register(username, password, nickname string) (*models.User, error) {
	var count int64
	database.DB.Model(&models.User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		return nil, errors.New("用户名已存在")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := models.User{
		Username: username,
		Password: string(hashed),
		Nickname: nickname,
		Avatar:   "",
		Role:     "user",
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) GetByID(id uint) (*models.User, error) {
	var user models.User
	result := database.DB.First(&user, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

func (s *UserService) GetAll() ([]models.User, error) {
	var users []models.User
	result := database.DB.Order("id desc").Find(&users)
	for i := range users {
		users[i].Avatar = utils.FixAvatarPath(users[i].Avatar)
	}
	return users, result.Error
}

func (s *UserService) GetPaginated(page, pageSize int) ([]models.User, int64, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	var total int64
	database.DB.Model(&models.User{}).Count(&total)
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	var users []models.User
	result := database.DB.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&users)
	for i := range users {
		users[i].Avatar = utils.FixAvatarPath(users[i].Avatar)
	}
	return users, total, totalPages, result.Error
}

func (s *UserService) Count() (int64, error) {
	var count int64
	result := database.DB.Model(&models.User{}).Count(&count)
	return count, result.Error
}

func (s *UserService) CountAdmins() (int64, error) {
	var count int64
	result := database.DB.Model(&models.User{}).Where("role = ?", "admin").Count(&count)
	return count, result.Error
}

func (s *UserService) Create(username, password, nickname, role string) error {
	var count int64
	database.DB.Model(&models.User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		return errors.New("用户名已存在")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := models.User{
		Username:  username,
		Password:  string(hashed),
		Nickname:  nickname,
		Avatar:    "",
		Role:      role,
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	return database.DB.Create(&user).Error
}

func (s *UserService) Update(id uint, nickname, role, password string) error {
	updates := map[string]interface{}{
		"nickname": nickname,
		"role":     role,
	}
	if password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		updates["password"] = string(hashed)
	}
	return database.DB.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error
}

func (s *UserService) UpdateWithAvatar(id uint, nickname, role, password, avatar string) error {
	updates := map[string]interface{}{
		"nickname": nickname,
		"role":     role,
	}
	if password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		updates["password"] = string(hashed)
	}
	if avatar != "" {
		updates["avatar"] = avatar
	}
	return database.DB.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error
}

func (s *UserService) UpdateProfile(id uint, nickname, avatar string) error {
	updates := map[string]interface{}{
		"nickname": nickname,
	}
	if avatar != "" {
		updates["avatar"] = avatar
	}
	return database.DB.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error
}

func (s *UserService) Delete(id uint) error {
	return database.DB.Delete(&models.User{}, id).Error
}

func (s *UserService) VerifyPassword(userID uint, password string) error {
	user, err := s.GetByID(userID)
	if err != nil {
		return err
	}
	return bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
}

func (s *UserService) UpdatePassword(userID uint, newPassword string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return database.DB.Model(&models.User{}).Where("id = ?", userID).Update("password", string(hashed)).Error
}

func (s *UserService) ClearAvatar(id uint) error {
	return database.DB.Model(&models.User{}).Where("id = ?", id).Update("avatar", "").Error
}

func (s *UserService) IsAdmin(userID uint) bool {
	user, err := s.GetByID(userID)
	if err != nil {
		return false
	}
	return user.Role == "admin"
}
