package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"qyzb-server/internal/middleware"
	"qyzb-server/internal/services"
	"qyzb-server/internal/utils"
)

type UserHandler struct {
	userService    *services.UserService
	settingService *services.SettingService
}

func NewUserHandler() *UserHandler {
	return &UserHandler{
		userService:    services.NewUserService(),
		settingService: services.NewSettingService(),
	}
}

func (h *UserHandler) ProfilePage(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, _ := h.userService.GetByID(userID)

	nickname := user.Nickname
	if nickname == "" {
		nickname = user.Username
	}
	user.Avatar = utils.FixAvatarPath(user.Avatar)
	avatarInitial := ""
	for _, r := range nickname {
		avatarInitial = string(r)
		break
	}

	c.HTML(http.StatusOK, "user/profile.html", gin.H{
		"title":         "个人资料",
		"appName":       h.settingService.GetAppName(),
		"user":          user,
		"avatarInitial": avatarInitial,
		"success":       c.Query("success"),
		"error":         c.Query("error"),
	})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	nickname := c.PostForm("nickname")

	if nickname == "" {
		c.Redirect(http.StatusFound, "/user/profile")
		return
	}

	h.userService.UpdateProfile(userID, nickname, "")

	session := sessions.Default(c)
	session.Set("nickname", nickname)
	session.Save()

	c.Redirect(http.StatusFound, "/user/profile?success=1")
}

func (h *UserHandler) UploadAvatar(c *gin.Context) {
	userID := middleware.GetUserID(c)

	file, err := c.FormFile("avatar")
	if err != nil {
		c.Redirect(http.StatusFound, "/user/profile?error=请选择要上传的头像")
		return
	}

	ext := filepath.Ext(file.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" {
		c.Redirect(http.StatusFound, "/user/profile?error=只支持 jpg、png、gif 格式的图片")
		return
	}

	if file.Size > 2*1024*1024 {
		c.Redirect(http.StatusFound, "/user/profile?error=头像大小不能超过 2MB")
		return
	}

	uploadDir := filepath.Join(getUploadDir(), "avatars")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.Redirect(http.StatusFound, "/user/profile?error=上传目录创建失败")
		return
	}

	user, _ := h.userService.GetByID(userID)
	oldAvatar := ""
	if user != nil {
		oldAvatar = user.Avatar
	}

	fileName := fmt.Sprintf("avatar_%d_%d%s", userID, time.Now().Unix(), ext)
	filePath := filepath.Join(uploadDir, fileName)

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.Redirect(http.StatusFound, "/user/profile?error=头像保存失败")
		return
	}

	avatarPath := "/uploads/avatars/" + fileName

	if err := h.userService.UpdateProfile(userID, "", avatarPath); err != nil {
		c.Redirect(http.StatusFound, "/user/profile?error=头像更新失败")
		return
	}

	if oldAvatar != "" && oldAvatar != avatarPath {
		oldPath := filepath.Join(getUploadDir(), strings.TrimPrefix(oldAvatar, "/uploads/"))
		if _, err := os.Stat(oldPath); err == nil {
			os.Remove(oldPath)
		}
	}

	session := sessions.Default(c)
	session.Set("avatar", avatarPath)
	session.Save()

	c.Redirect(http.StatusFound, "/user/profile?success=1")
}

func (h *UserHandler) ClearAvatar(c *gin.Context) {
	userID := middleware.GetUserID(c)

	user, _ := h.userService.GetByID(userID)
	oldAvatar := ""
	if user != nil {
		oldAvatar = user.Avatar
	}

	if err := h.userService.ClearAvatar(userID); err != nil {
		c.Redirect(http.StatusFound, "/user/profile?error=清除头像失败")
		return
	}

	if oldAvatar != "" {
		oldPath := filepath.Join(getUploadDir(), strings.TrimPrefix(oldAvatar, "/uploads/"))
		if _, err := os.Stat(oldPath); err == nil {
			os.Remove(oldPath)
		}
	}

	session := sessions.Default(c)
	session.Set("avatar", "")
	session.Save()

	c.Redirect(http.StatusFound, "/user/profile?success=1")
}

func getUploadDir() string {
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

func (h *UserHandler) UpdatePassword(c *gin.Context) {
	userID := middleware.GetUserID(c)
	oldPassword := c.PostForm("oldPassword")
	newPassword := c.PostForm("newPassword")
	confirmPassword := c.PostForm("confirmPassword")

	user, err := h.userService.GetByID(userID)
	if err != nil {
		c.Redirect(http.StatusFound, "/user/profile?error=用户不存在")
		return
	}

	if oldPassword == "" || newPassword == "" || confirmPassword == "" {
		c.Redirect(http.StatusFound, "/user/profile?error=请填写完整的密码信息")
		return
	}

	if len(newPassword) < 6 {
		c.Redirect(http.StatusFound, "/user/profile?error=新密码长度不能少于6位")
		return
	}

	if newPassword != confirmPassword {
		c.Redirect(http.StatusFound, "/user/profile?error=两次输入的新密码不一致")
		return
	}

	if err := h.userService.VerifyPassword(user.ID, oldPassword); err != nil {
		c.Redirect(http.StatusFound, "/user/profile?error=当前密码不正确")
		return
	}

	if err := h.userService.UpdatePassword(userID, newPassword); err != nil {
		c.Redirect(http.StatusFound, "/user/profile?error=密码修改失败")
		return
	}

	c.Redirect(http.StatusFound, "/user/profile?success=1")
}
