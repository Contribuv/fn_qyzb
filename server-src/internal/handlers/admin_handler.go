package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"qyzb-server/internal/services"
	"qyzb-server/internal/utils"
	"qyzb-server/internal/websocket"
)

type AdminHandler struct {
	userService    *services.UserService
	roomService    *services.RoomService
	messageService *services.MessageService
	settingService *services.SettingService
}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{
		userService:    services.NewUserService(),
		roomService:    services.NewRoomService(),
		messageService: services.NewMessageService(),
		settingService: services.NewSettingService(),
	}
}

func (h *AdminHandler) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "admin/login.html", gin.H{
		"title":   "管理员登录",
		"appName": h.settingService.GetAppName(),
		"error":   c.Query("error"),
	})
}

func (h *AdminHandler) Login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	user, err := h.userService.Login(username, password)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/login?error="+err.Error())
		return
	}
	if user.Role != "admin" {
		c.Redirect(http.StatusFound, "/admin/login?error=没有管理员权限")
		return
	}

	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Set("username", user.Username)
	session.Set("nickname", user.Nickname)
	session.Set("user_role", user.Role)
	session.Set("avatar", utils.FixAvatarPath(user.Avatar))
	session.Save()

	c.Redirect(http.StatusFound, "/admin")
}

func (h *AdminHandler) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/admin/login")
}

func (h *AdminHandler) currentUser(c *gin.Context) (string, string, string) {
	session := sessions.Default(c)
	username, _ := session.Get("username").(string)
	nickname, _ := session.Get("nickname").(string)
	avatar, _ := session.Get("avatar").(string)
	if nickname == "" {
		nickname = username
	}
	avatar = utils.FixAvatarPath(avatar)
	return username, nickname, avatar
}

func (h *AdminHandler) Dashboard(c *gin.Context) {
	username, nickname, avatar := h.currentUser(c)
	userCount, _ := h.userService.Count()
	roomCount, _ := h.roomService.Count()
	messageCount, _ := h.messageService.Count()
	onlineCount := websocket.GetHub().TotalConnections()

	c.HTML(http.StatusOK, "admin/dashboard.html", gin.H{
		"title":   "仪表盘",
		"appName": h.settingService.GetAppName(),
		"stats": gin.H{
			"userCount":    userCount,
			"roomCount":    roomCount,
			"messageCount": messageCount,
			"activeRooms":  onlineCount,
		},
		"activeTab":  "dashboard",
		"username":   username,
		"nickname":   nickname,
		"userAvatar": avatar,
	})
}

func (h *AdminHandler) Users(c *gin.Context) {
	username, nickname, avatar := h.currentUser(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if pageSize < 5 {
		pageSize = 5
	}
	if pageSize > 100 {
		pageSize = 100
	}
	users, total, totalPages, _ := h.userService.GetPaginated(page, pageSize)
	c.HTML(http.StatusOK, "admin/users.html", gin.H{
		"title":      "用户管理",
		"appName":    h.settingService.GetAppName(),
		"activeTab":  "users",
		"username":   username,
		"nickname":   nickname,
		"userAvatar": avatar,
		"users":      users,
		"page":       page,
		"pageSize":   pageSize,
		"total":      total,
		"totalPages": totalPages,
		"error":      c.Query("error"),
		"success":    c.Query("success"),
	})
}

func (h *AdminHandler) AddUserPage(c *gin.Context) {
	username, nickname, avatar := h.currentUser(c)
	c.HTML(http.StatusOK, "admin/user_form.html", gin.H{
		"title":      "添加用户",
		"appName":    h.settingService.GetAppName(),
		"activeTab":  "users",
		"username":   username,
		"nickname":   nickname,
		"userAvatar": avatar,
		"mode":       "add",
		"error":      c.Query("error"),
	})
}

func (h *AdminHandler) AddUser(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	nickname := c.PostForm("nickname")
	role := c.PostForm("role")

	if role == "" {
		role = "user"
	}
	if nickname == "" {
		nickname = username
	}

	err := h.userService.Create(username, password, nickname, role)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/users/new?error="+err.Error())
		return
	}
	c.Redirect(http.StatusFound, "/admin/users?success=用户添加成功")
}

func (h *AdminHandler) EditUserPage(c *gin.Context) {
	curUser, curNickname, curAvatar := h.currentUser(c)
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	user, err := h.userService.GetByID(uint(id))
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/users")
		return
	}
	c.HTML(http.StatusOK, "admin/user_form.html", gin.H{
		"title":      "编辑用户",
		"appName":    h.settingService.GetAppName(),
		"activeTab":  "users",
		"username":   curUser,
		"nickname":   curNickname,
		"userAvatar": curAvatar,
		"mode":       "edit",
		"user":       user,
	})
}

func (h *AdminHandler) EditUser(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	nickname := c.PostForm("nickname")
	role := c.PostForm("role")
	password := c.PostForm("password")

	var avatarPath string
	file, err := c.FormFile("avatar")
	if err == nil && file != nil {
		ext := filepath.Ext(file.Filename)
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" {
			if file.Size <= 2*1024*1024 {
				uploadDir := filepath.Join(getAdminUploadDir(), "avatars")
				os.MkdirAll(uploadDir, 0755)
				fileName := fmt.Sprintf("avatar_%d_%d%s", id, time.Now().Unix(), ext)
				filePath := filepath.Join(uploadDir, fileName)
				if c.SaveUploadedFile(file, filePath) == nil {
					avatarPath = "/uploads/avatars/" + fileName
				}
			}
		}
	}

	oldUser, _ := h.userService.GetByID(uint(id))
	if oldUser != nil && oldUser.Role == "admin" && role != "admin" {
		adminCount, _ := h.userService.CountAdmins()
		if adminCount <= 1 {
			c.Redirect(http.StatusFound, "/admin/users?error=至少需要保留一个管理员账户")
			return
		}
	}

	oldAvatar := ""
	if oldUser != nil {
		oldAvatar = oldUser.Avatar
	}

	h.userService.UpdateWithAvatar(uint(id), nickname, role, password, avatarPath)

	if avatarPath != "" && oldAvatar != "" && oldAvatar != avatarPath {
		oldPath := filepath.Join(getAdminUploadDir(), strings.TrimPrefix(oldAvatar, "/uploads/"))
		if _, err := os.Stat(oldPath); err == nil {
			os.Remove(oldPath)
		}
	}

	session := sessions.Default(c)
	curUserID := session.Get("user_id")
	var curID uint
	switch v := curUserID.(type) {
	case uint:
		curID = v
	case int:
		curID = uint(v)
	case float64:
		curID = uint(v)
	}
	if curID == uint(id) {
		session.Set("nickname", nickname)
		if avatarPath != "" {
			session.Set("avatar", avatarPath)
		}
		session.Save()
	}

	c.Redirect(http.StatusFound, "/admin/users?success=用户更新成功")
}

func (h *AdminHandler) ClearUserAvatar(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	h.userService.ClearAvatar(uint(id))
	c.Redirect(http.StatusFound, "/admin/users/edit/"+idStr)
}

func getAdminUploadDir() string {
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir != "" {
		return uploadDir
	}
	exe, err := os.Executable()
	if err != nil {
		wd, _ := os.Getwd()
		return filepath.Join(wd, "uploads")
	}
	return filepath.Join(filepath.Dir(exe), "uploads")
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	user, err := h.userService.GetByID(uint(id))
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/users?error=用户不存在")
		return
	}
	if user.Role == "admin" {
		adminCount, _ := h.userService.CountAdmins()
		if adminCount <= 1 {
			c.Redirect(http.StatusFound, "/admin/users?error=至少需要保留一个管理员账户")
			return
		}
	}
	avatar := user.Avatar
	err = h.userService.Delete(uint(id))
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/users?error="+err.Error())
		return
	}

	if avatar != "" {
		avatarPath := filepath.Join(getAdminUploadDir(), strings.TrimPrefix(avatar, "/uploads/"))
		if _, err := os.Stat(avatarPath); err == nil {
			os.Remove(avatarPath)
		}
	}

	c.Redirect(http.StatusFound, "/admin/users?success=用户删除成功")
}

func (h *AdminHandler) Rooms(c *gin.Context) {
	username, nickname, avatar := h.currentUser(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if pageSize < 5 {
		pageSize = 5
	}
	if pageSize > 100 {
		pageSize = 100
	}
	rooms, total, totalPages, _ := h.roomService.GetPaginated(page, pageSize)
	c.HTML(http.StatusOK, "admin/rooms.html", gin.H{
		"title":      "房间管理",
		"appName":    h.settingService.GetAppName(),
		"activeTab":  "rooms",
		"username":   username,
		"nickname":   nickname,
		"userAvatar": avatar,
		"rooms":      rooms,
		"page":       page,
		"pageSize":   pageSize,
		"total":      total,
		"totalPages": totalPages,
		"error":      c.Query("error"),
		"success":    c.Query("success"),
	})
}

func (h *AdminHandler) AddRoomPage(c *gin.Context) {
	username, nickname, avatar := h.currentUser(c)
	c.HTML(http.StatusOK, "admin/room_form.html", gin.H{
		"title":      "添加房间",
		"appName":    h.settingService.GetAppName(),
		"activeTab":  "rooms",
		"username":   username,
		"nickname":   nickname,
		"userAvatar": avatar,
		"mode":       "add",
		"error":      c.Query("error"),
	})
}

func (h *AdminHandler) AddRoom(c *gin.Context) {
	name := c.PostForm("name")
	description := c.PostForm("description")
	err := h.roomService.Create(name, description)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/rooms/new?error="+err.Error())
		return
	}
	c.Redirect(http.StatusFound, "/admin/rooms?success=房间添加成功")
}

func (h *AdminHandler) EditRoomPage(c *gin.Context) {
	curUser, curNickname, curAvatar := h.currentUser(c)
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	room, err := h.roomService.GetByID(uint(id))
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/rooms")
		return
	}
	c.HTML(http.StatusOK, "admin/room_form.html", gin.H{
		"title":      "编辑房间",
		"appName":    h.settingService.GetAppName(),
		"activeTab":  "rooms",
		"username":   curUser,
		"nickname":   curNickname,
		"userAvatar": curAvatar,
		"mode":       "edit",
		"room":       room,
	})
}

func (h *AdminHandler) EditRoom(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	name := c.PostForm("name")
	description := c.PostForm("description")
	h.roomService.Update(uint(id), name, description)
	c.Redirect(http.StatusFound, "/admin/rooms?success=房间更新成功")
}

func (h *AdminHandler) DeleteRoom(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	err := h.roomService.Delete(uint(id))
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/rooms?error="+err.Error())
		return
	}
	c.Redirect(http.StatusFound, "/admin/rooms?success=房间删除成功")
}

func (h *AdminHandler) ClearMessages(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	h.messageService.ClearByRoomID(uint(id))
	c.Redirect(http.StatusFound, "/admin/rooms")
}

func (h *AdminHandler) SettingsPage(c *gin.Context) {
	username, nickname, avatar := h.currentUser(c)
	settingMap, _ := h.settingService.GetAll()
	appName := h.settingService.GetAppName()
	copyright := h.settingService.GetCopyright()
	allowRegister := h.settingService.IsRegisterAllowed()
	if _, ok := settingMap["appName"]; !ok || settingMap["appName"] == "" {
		settingMap["appName"] = appName
	}
	if _, ok := settingMap["copyright"]; !ok || settingMap["copyright"] == "" {
		settingMap["copyright"] = copyright
	}
	c.HTML(http.StatusOK, "admin/settings.html", gin.H{
		"title":         "系统设置",
		"appName":       appName,
		"activeTab":     "settings",
		"username":      username,
		"nickname":      nickname,
		"userAvatar":    avatar,
		"settings":      settingMap,
		"copyright":     copyright,
		"allowRegister": allowRegister,
		"success":       c.Query("success") == "1",
	})
}

func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	appName := c.PostForm("appName")
	copyright := c.PostForm("copyright")
	allowRegister := c.PostForm("allow_register")

	h.settingService.Set("appName", appName)
	h.settingService.Set("copyright", copyright)
	h.settingService.Set("allow_register", allowRegister)

	c.Redirect(http.StatusFound, "/admin/settings?success=1")
}

func (h *AdminHandler) APIDoc(c *gin.Context) {
	username, nickname, avatar := h.currentUser(c)
	c.HTML(http.StatusOK, "admin/api.html", gin.H{
		"title":      "接口文档",
		"appName":    h.settingService.GetAppName(),
		"activeTab":  "api",
		"username":   username,
		"nickname":   nickname,
		"userAvatar": avatar,
	})
}
