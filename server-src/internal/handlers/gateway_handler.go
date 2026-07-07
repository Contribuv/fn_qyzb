package handlers

import (
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"

	"qyzb-server/internal/services"
	"qyzb-server/internal/utils"
)

type GatewayHandler struct {
	rproxy         *utils.RProxyManager
	certs          *utils.CertManager
	settingService *services.SettingService
}

func NewGatewayHandler() *GatewayHandler {
	return &GatewayHandler{
		rproxy:         utils.GetRProxyManager(),
		certs:          utils.GetCertManager(),
		settingService: services.NewSettingService(),
	}
}

func (h *GatewayHandler) Index(c *gin.Context) {
	prefix := ""
	if c.GetHeader("X-Trim-Userid") != "" {
		prefix = os.Getenv("GATEWAY_PREFIX")
	}
	c.HTML(http.StatusOK, "gateway/index.html", gin.H{
		"title":   "公网访问配置",
		"appName": h.settingService.GetAppName(),
		"prefix":  prefix,
	})
}

func (h *GatewayHandler) GetStatus(c *gin.Context) {
	status, _ := h.rproxy.GetStatus()
	c.JSON(http.StatusOK, status)
}

type startRequest struct {
	Domain string `json:"domain"`
	Port   int    `json:"port"`
}

func (h *GatewayHandler) CheckPort(c *gin.Context) {
	portStr := c.Query("port")
	if portStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"available": false,
			"message":   "缺少 port 参数",
		})
		return
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{
			"available": false,
			"message":   "端口号无效",
		})
		return
	}

	available, err := utils.CheckPortAvailable(port)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"available": false,
			"port":      port,
			"message":   "端口检测失败",
		})
		return
	}

	result := gin.H{
		"available": available,
		"port":      port,
	}

	if available {
		result["message"] = "端口可用"
	} else {
		suggested := utils.SuggestPort(port)
		result["message"] = "端口已被占用"
		if suggested > 0 {
			result["suggested_port"] = suggested
		}
	}

	c.JSON(http.StatusOK, result)
}

func (h *GatewayHandler) StartProxy(c *gin.Context) {
	var req startRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}

	if req.Domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请选择域名"})
		return
	}

	if req.Port == 0 {
		req.Port = 7786
	}

	cert := h.certs.GetCertByDomain(req.Domain)
	if cert == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "未找到对应证书"})
		return
	}

	err := h.rproxy.Start(req.Domain, req.Port, cert.CertPath, cert.KeyPath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *GatewayHandler) StopProxy(c *gin.Context) {
	err := h.rproxy.Stop()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *GatewayHandler) UpdateConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *GatewayHandler) GetLogs(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "100")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 100
	}

	logs, err := h.rproxy.GetLogs(limit)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"logs": []interface{}{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

func (h *GatewayHandler) GetCerts(c *gin.Context) {
	certs := h.certs.ListCerts()
	domains := make([]string, 0, len(certs))
	for _, cert := range certs {
		domains = append(domains, cert.Domain)
	}
	c.JSON(http.StatusOK, gin.H{"certs": domains})
}
