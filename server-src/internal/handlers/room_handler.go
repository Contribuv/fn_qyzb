package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"qyzb-server/internal/services"
	"qyzb-server/internal/utils"
)

type RoomHandler struct {
	userService    *services.UserService
	roomService    *services.RoomService
	settingService *services.SettingService
}

func NewRoomHandler() *RoomHandler {
	return &RoomHandler{
		userService:    services.NewUserService(),
		roomService:    services.NewRoomService(),
		settingService: services.NewSettingService(),
	}
}

func (h *RoomHandler) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "room/login.html", gin.H{
		"title":   "登录",
		"appName": h.settingService.GetAppName(),
		"error":   c.Query("error"),
	})
}

func (h *RoomHandler) Login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	user, err := h.userService.Login(username, password)
	if err != nil {
		c.Redirect(http.StatusFound, "/room/login?error="+err.Error())
		return
	}

	session := sessions.Default(c)
	session.Set("user_id", int(user.ID))
	session.Set("username", user.Username)
	session.Set("nickname", user.Nickname)
	session.Set("user_role", user.Role)
	session.Set("avatar", utils.FixAvatarPath(user.Avatar))
	session.Save()

	c.Redirect(http.StatusFound, "/room")
}

func (h *RoomHandler) RegisterPage(c *gin.Context) {
	if !h.settingService.IsRegisterAllowed() {
		c.Redirect(http.StatusFound, "/room/login?error=当前已关闭注册")
		return
	}
	c.HTML(http.StatusOK, "room/register.html", gin.H{
		"title":   "注册",
		"appName": h.settingService.GetAppName(),
		"error":   c.Query("error"),
	})
}

func (h *RoomHandler) Register(c *gin.Context) {
	if !h.settingService.IsRegisterAllowed() {
		c.Redirect(http.StatusFound, "/room/login?error=当前已关闭注册")
		return
	}
	username := c.PostForm("username")
	password := c.PostForm("password")
	nickname := c.PostForm("nickname")

	if username == "" || password == "" {
		c.Redirect(http.StatusFound, "/room/register?error=用户名和密码不能为空")
		return
	}
	if len(password) < 6 {
		c.Redirect(http.StatusFound, "/room/register?error=密码长度不能少于6位")
		return
	}
	if nickname == "" {
		nickname = username
	}

	_, err := h.userService.Register(username, password, nickname)
	if err != nil {
		c.Redirect(http.StatusFound, "/room/register?error="+err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/room/login")
}

func (h *RoomHandler) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/room/login")
}

func (h *RoomHandler) List(c *gin.Context) {
	rooms, _ := h.roomService.GetAll()
	session := sessions.Default(c)
	nickname, _ := session.Get("nickname").(string)
	username, _ := session.Get("username").(string)
	avatar, _ := session.Get("avatar").(string)
	role, _ := session.Get("user_role").(string)
	if nickname == "" {
		nickname = username
	}
	avatar = utils.FixAvatarPath(avatar)

	c.HTML(http.StatusOK, "room/rooms.html", gin.H{
		"title":    "房间列表",
		"appName":  h.settingService.GetAppName(),
		"rooms":    rooms,
		"nickname": nickname,
		"avatar":   avatar,
		"role":     role,
	})
}

func (h *RoomHandler) Chat(c *gin.Context) {
	idStr := c.Param("id")
	roomID, _ := strconv.Atoi(idStr)
	room, err := h.roomService.GetByID(uint(roomID))
	if err != nil {
		c.Redirect(http.StatusFound, "/room")
		return
	}

	session := sessions.Default(c)
	nickname, _ := session.Get("nickname").(string)
	username, _ := session.Get("username").(string)
	avatar, _ := session.Get("avatar").(string)
	role, _ := session.Get("user_role").(string)
	rawUserID := session.Get("user_id")
	userID := uint(0)
	switch v := rawUserID.(type) {
	case uint:
		userID = uint(v)
	case int:
		userID = uint(v)
	case float64:
		userID = uint(v)
	}
	if nickname == "" {
		nickname = username
	}
	avatar = utils.FixAvatarPath(avatar)

	c.HTML(http.StatusOK, "room/chat.html", gin.H{
		"title":    room.Name,
		"appName":  h.settingService.GetAppName(),
		"room":     room,
		"roomId":   roomID,
		"nickname": nickname,
		"username": username,
		"avatar":   avatar,
		"userId":   userID,
		"role":     role,
		"isAdmin":  role == "admin",
	})
}
