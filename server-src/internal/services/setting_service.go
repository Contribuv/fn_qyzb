package services

import (
	"log"
	"time"

	"qyzb-server/internal/database"
	"qyzb-server/internal/models"
)

type SettingService struct{}

func NewSettingService() *SettingService {
	return &SettingService{}
}

func (s *SettingService) GetAll() (map[string]string, error) {
	var settings []models.Setting
	result := database.DB.Find(&settings)
	if result.Error != nil {
		log.Printf("[SettingService] GetAll 错误: %v", result.Error)
		return nil, result.Error
	}

	m := make(map[string]string)
	for _, s := range settings {
		m[s.KeyName] = s.Value
	}
	log.Printf("[SettingService] GetAll 成功, 共 %d 条设置", len(settings))
	return m, nil
}

func (s *SettingService) Get(key string) (string, error) {
	var setting models.Setting
	result := database.DB.Where("key_name = ?", key).First(&setting)
	if result.Error != nil {
		log.Printf("[SettingService] Get(%s) 错误: %v", key, result.Error)
		return "", result.Error
	}
	log.Printf("[SettingService] Get(%s) = %s", key, setting.Value)
	return setting.Value, nil
}

func (s *SettingService) Set(key, value string) error {
	var setting models.Setting
	result := database.DB.Where("key_name = ?", key).First(&setting)
	if result.Error != nil {
		setting = models.Setting{
			KeyName:   key,
			Value:     value,
			UpdatedAt: time.Now().Format(time.RFC3339),
		}
		return database.DB.Create(&setting).Error
	}
	setting.Value = value
	setting.UpdatedAt = time.Now().Format(time.RFC3339)
	return database.DB.Save(&setting).Error
}

func (s *SettingService) GetAppName() string {
	name, _ := s.Get("appName")
	if name == "" {
		return "千盈助播"
	}
	return name
}

func (s *SettingService) GetCopyright() string {
	copyright, _ := s.Get("copyright")
	if copyright == "" {
		return "千盈助播 © 版权所有"
	}
	return copyright
}

func (s *SettingService) IsRegisterAllowed() bool {
	val, _ := s.Get("allow_register")
	return val == "true" || val == ""
}
