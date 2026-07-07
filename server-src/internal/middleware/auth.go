package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get("user_id")
		if userID == nil {
			if isAPIRequest(c) {
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"message": "未登录或登录已过期",
				})
				c.Abort()
				return
			}
			c.Redirect(http.StatusFound, "/room/login")
			c.Abort()
			return
		}
		c.Set("user_id", userID)
		c.Next()
	}
}

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userRole := session.Get("user_role")
		if userRole != "admin" {
			if isAPIRequest(c) {
				c.JSON(http.StatusForbidden, gin.H{
					"success": false,
					"message": "需要管理员权限",
				})
				c.Abort()
				return
			}
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

func isAPIRequest(c *gin.Context) bool {
	return strings.HasPrefix(c.Request.URL.Path, "/api/")
}

func GetUserID(c *gin.Context) uint {
	val, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	if id, ok := val.(uint); ok {
		return id
	}
	if id, ok := val.(int); ok {
		return uint(id)
	}
	return 0
}
