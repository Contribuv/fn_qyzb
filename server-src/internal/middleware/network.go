package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RequireLocalNetwork() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isLocalRequest(c) {
			c.Next()
			return
		}
		c.HTML(http.StatusForbidden, "403.html", gin.H{
			"title":   "无访问权限",
			"appName": "千盈助播",
			"message": "该页面仅允许内网访问",
		})
		c.Abort()
	}
}

func isLocalRequest(c *gin.Context) bool {
	clientIP := getClientIP(c)
	if clientIP == "" || clientIP == "@" {
		return true
	}
	return isPrivateIP(clientIP)
}

func getClientIP(c *gin.Context) string {
	xForwardedFor := c.GetHeader("X-Forwarded-For")
	if xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[len(ips)-1])
		}
	}
	xRealIP := c.GetHeader("X-Real-IP")
	if xRealIP != "" {
		return xRealIP
	}
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return ip
}

func isPrivateIP(ipStr string) bool {
	if ipStr == "::1" {
		return true
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	privateRanges := []string{
		"127.0.0.0/8",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"fc00::/7",
		"fe80::/10",
	}

	for _, cidr := range privateRanges {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
