package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"qyzb-server/internal/middleware"
	"qyzb-server/internal/services"
)

type APIHandler struct {
	userService    *services.UserService
	roomService    *services.RoomService
	messageService *services.MessageService
	settingService *services.SettingService
}

func NewAPIHandler() *APIHandler {
	return &APIHandler{
		userService:    services.NewUserService(),
		roomService:    services.NewRoomService(),
		messageService: services.NewMessageService(),
		settingService: services.NewSettingService(),
	}
}

func (h *APIHandler) GetUserInfo(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, err := h.userService.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

func (h *APIHandler) GetRooms(c *gin.Context) {
	rooms, err := h.roomService.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    rooms,
	})
}

func (h *APIHandler) GetMessages(c *gin.Context) {
	idStr := c.Param("id")
	roomID, _ := strconv.Atoi(idStr)
	messages, err := h.messageService.GetByRoomID(uint(roomID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    messages,
	})
}

func (h *APIHandler) GetLatestMessages(c *gin.Context) {
	idStr := c.Param("id")
	roomID, _ := strconv.Atoi(idStr)
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 50
	}
	log.Printf("[API] GetLatestMessages: roomID=%s, limit=%s, path=%s", idStr, limitStr, c.Request.URL.Path)
	messages, err := h.messageService.GetLatest(uint(roomID), limit)
	if err != nil {
		log.Printf("[API] GetLatestMessages 错误: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	log.Printf("[API] GetLatestMessages 成功: %d 条消息", len(messages))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    messages,
	})
}

func (h *APIHandler) GetUsers(c *gin.Context) {
	users, err := h.userService.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    users,
	})
}

func (h *APIHandler) CreateUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *APIHandler) UpdateUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *APIHandler) DeleteUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *APIHandler) CreateRoom(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *APIHandler) UpdateRoom(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *APIHandler) DeleteRoom(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *APIHandler) ClearRoomMessages(c *gin.Context) {
	idStr := c.Param("id")
	roomID, _ := strconv.Atoi(idStr)
	h.messageService.ClearByRoomID(uint(roomID))
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *APIHandler) GetSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"appName":   h.settingService.GetAppName(),
			"copyright": h.settingService.GetCopyright(),
		},
	})
}

func (h *APIHandler) GetPublicSettings(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"appName":   h.settingService.GetAppName(),
			"copyright": h.settingService.GetCopyright(),
		},
	})
}
