package database

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"qyzb-server/internal/models"

	"github.com/glebarez/sqlite"
)

var DB *gorm.DB

func Init() error {
	dbPath := getDBPath()
	log.Printf("[Database] 数据库路径: %s", dbPath)
	dataDir := filepath.Dir(dbPath)

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("创建数据目录失败: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("数据库连接失败: %v", err)
	}
	log.Printf("[Database] 连接成功")

	DB = db

	if err := autoMigrate(); err != nil {
		backupPath := dbPath + ".broken." + time.Now().Format("20060102150405")
		log.Printf("数据库迁移失败，备份旧数据库到: %s", backupPath)
		if copyErr := copyFile(dbPath, backupPath); copyErr != nil {
			log.Printf("备份旧数据库失败: %v", copyErr)
		}
		DB = nil
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		if rmErr := os.Remove(dbPath); rmErr != nil {
			return fmt.Errorf("删除损坏数据库失败: %v", rmErr)
		}
		db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err != nil {
			return fmt.Errorf("重建数据库连接失败: %v", err)
		}
		DB = db
		if err := autoMigrate(); err != nil {
			return fmt.Errorf("重建数据库迁移失败: %v", err)
		}
	}

	if err := initDefaultData(); err != nil {
		return fmt.Errorf("初始化默认数据失败: %v", err)
	}

	log.Printf("数据库初始化成功，路径: %s", dbPath)
	return nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func getDBPath() string {
	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		log.Printf("[Database] 使用 DB_PATH=%s", dbPath)
		return dbPath
	}
	if dataDir := os.Getenv("DATA_DIR"); dataDir != "" {
		p := filepath.Join(dataDir, "qyzb.db")
		log.Printf("[Database] 使用 DATA_DIR, 路径=%s", p)
		return p
	}
	if pkgVar := os.Getenv("TRIM_PKGVAR"); pkgVar != "" {
		p := filepath.Join(pkgVar, "data", "qyzb.db")
		log.Printf("[Database] 使用 TRIM_PKGVAR, 路径=%s", p)
		return p
	}
	p := filepath.Join(".", "data", "qyzb.db")
	log.Printf("[Database] 使用默认路径=%s", p)
	return p
}

func GetDataDir() string {
	return filepath.Dir(getDBPath())
}

func autoMigrate() error {
	usersSQL := `CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username VARCHAR(50) NOT NULL UNIQUE,
		password VARCHAR(255) NOT NULL,
		nickname VARCHAR(50) DEFAULT '',
		avatar VARCHAR(255) DEFAULT '',
		role VARCHAR(20) DEFAULT 'user',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);`

	roomsSQL := `CREATE TABLE IF NOT EXISTS rooms (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(100) NOT NULL,
		description VARCHAR(255) DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	messagesSQL := `CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		room_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		content TEXT NOT NULL,
		type VARCHAR(20) DEFAULT 'text',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_messages_room_id ON messages(room_id);
	CREATE INDEX IF NOT EXISTS idx_messages_user_id ON messages(user_id);
	CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);`

	settingsSQL := `CREATE TABLE IF NOT EXISTS settings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key_name VARCHAR(100) NOT NULL UNIQUE,
		value TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	for _, sql := range []string{usersSQL, roomsSQL, messagesSQL, settingsSQL} {
		if err := DB.Exec(sql).Error; err != nil {
			return err
		}
	}
	return nil
}

func initDefaultData() error {
	if err := initDefaultAdmin(); err != nil {
		return err
	}
	if err := initDefaultSettings(); err != nil {
		return err
	}
	if err := initDefaultRooms(); err != nil {
		return err
	}
	return nil
}

func initDefaultAdmin() error {
	adminUser := os.Getenv("ADMIN_USERNAME")
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPass := os.Getenv("ADMIN_PASSWORD")
	if adminPass == "" {
		adminPass = "admin123"
	}

	var existing models.User
	result := DB.Where("role = ?", "admin").First(&existing)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			hashed, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			admin := models.User{
				Username:  adminUser,
				Password:  string(hashed),
				Nickname:  "管理员",
				Avatar:    "/static/images/avatar.png",
				Role:      "admin",
				CreatedAt: time.Now().Format(time.RFC3339),
			}
			return DB.Create(&admin).Error
		}
		return result.Error
	}

	if existing.Username != adminUser {
		existing.Username = adminUser
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	existing.Password = string(hashed)
	return DB.Save(&existing).Error
}

func initDefaultSettings() error {
	defaults := map[string]string{
		"appName":   "千盈助播",
		"copyright": "千盈助播 © 版权所有",
	}

	for key, value := range defaults {
		var setting models.Setting
		result := DB.Where("key_name = ?", key).First(&setting)
		if result.Error == gorm.ErrRecordNotFound {
			setting = models.Setting{
				KeyName:   key,
				Value:     value,
				UpdatedAt: time.Now().Format(time.RFC3339),
			}
			if err := DB.Create(&setting).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func initDefaultRooms() error {
	var count int64
	DB.Model(&models.Room{}).Count(&count)
	if count > 0 {
		return nil
	}

	rooms := []models.Room{
		{Name: "主助播室", Description: "主要直播助播房间", CreatedAt: time.Now().Format(time.RFC3339)},
		{Name: "备用助播室", Description: "备用直播助播房间", CreatedAt: time.Now().Format(time.RFC3339)},
	}
	return DB.Create(&rooms).Error
}
