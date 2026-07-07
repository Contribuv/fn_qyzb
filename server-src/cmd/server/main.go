package main

import (
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	server "qyzb-server"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"qyzb-server/internal/database"
	"qyzb-server/internal/handlers"
	"qyzb-server/internal/middleware"
	"qyzb-server/internal/utils"
	"qyzb-server/internal/websocket"
)

// 编译时可通过 -ldflags 注入版本号
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	log.Printf("========== 千盈助播启动 ==========")
	log.Printf("版本: %s, 编译时间: %s, Git提交: %s", Version, BuildTime, GitCommit)
	log.Printf("环境变量:")
	log.Printf("  PORT=%s", os.Getenv("PORT"))
	log.Printf("  wizard_app_port=%s", os.Getenv("wizard_app_port"))
	log.Printf("  DB_PATH=%s", os.Getenv("DB_PATH"))
	log.Printf("  UPLOAD_DIR=%s", os.Getenv("UPLOAD_DIR"))
	log.Printf("  GATEWAY_PREFIX=%s", os.Getenv("GATEWAY_PREFIX"))
	log.Printf("  TRIM_GATEWAY_PREFIX=%s", os.Getenv("TRIM_GATEWAY_PREFIX"))
	log.Printf("  TRIM_APPDEST=%s", os.Getenv("TRIM_APPDEST"))
	log.Printf("  UNIX_SOCKET=%s", os.Getenv("UNIX_SOCKET"))
	log.Printf("  GATEWAY_SOCKET=%s", os.Getenv("GATEWAY_SOCKET"))
	log.Printf("  TRIM_DATA_SHARE_PATHS=%s", os.Getenv("TRIM_DATA_SHARE_PATHS"))
	log.Printf("  SESSION_SECRET=%s", maskSecret(os.Getenv("SESSION_SECRET")))
	log.Printf("  可执行文件路径: %s", getExeDir())

	if err := database.Init(); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	log.Printf("数据库初始化成功")

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		secret = "qyzb-secret-key-2026"
	}
	store := cookie.NewStore([]byte(secret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   60 * 60 * 24 * 7,
		HttpOnly: true,
	})
	r.Use(sessions.Sessions("qyzb_session", store))

	r.SetFuncMap(utils.TemplateFuncMap())
	r.SetHTMLTemplate(utils.LoadTemplatesFromFS(server.TemplatesFS, "templates"))

	staticFS, _ := fs.Sub(server.StaticFS, "static")
	r.StaticFS("/static", http.FS(staticFS))

	r.GET("/favicon.ico", func(c *gin.Context) {
		c.FileFromFS("images/favicon.ico", http.FS(staticFS))
	})

	r.GET("/avatar.png", func(c *gin.Context) {
		c.FileFromFS("images/avatar.png", http.FS(staticFS))
	})

	exeDir := getExeDir()
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = filepath.Join(exeDir, "uploads")
	}
	r.Static("/uploads", uploadDir)

	setupRoutes(r)

	port := getPort()
	handler := wrapWithGatewayPrefix(r)

	go startUnixSocket(handler)

	log.Printf("HTTP 服务监听 :%s", port)
	log.Printf("========== 启动完成 ==========")
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}

func maskSecret(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}

func setupRoutes(r *gin.Engine) {
	roomHandler := handlers.NewRoomHandler()
	userHandler := handlers.NewUserHandler()
	adminHandler := handlers.NewAdminHandler()
	apiHandler := handlers.NewAPIHandler()
	gatewayHandler := handlers.NewGatewayHandler()

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/room")
	})

	r.GET("/room/login", roomHandler.LoginPage)
	r.POST("/room/login", roomHandler.Login)
	r.GET("/room/register", roomHandler.RegisterPage)
	r.POST("/room/register", roomHandler.Register)
	r.GET("/room/logout", roomHandler.Logout)

	roomGroup := r.Group("/room")
	roomGroup.Use(middleware.AuthRequired())
	{
		roomGroup.GET("", roomHandler.List)
		roomGroup.GET("/:id", roomHandler.Chat)
	}

	userGroup := r.Group("/user")
	userGroup.Use(middleware.AuthRequired())
	{
		userGroup.GET("/profile", userHandler.ProfilePage)
		userGroup.POST("/profile", userHandler.UpdateProfile)
		userGroup.POST("/avatar", userHandler.UploadAvatar)
		userGroup.POST("/avatar/clear", userHandler.ClearAvatar)
		userGroup.POST("/password", userHandler.UpdatePassword)
	}

	r.GET("/admin/login", adminHandler.LoginPage)
	r.POST("/admin/login", adminHandler.Login)
	r.GET("/admin/logout", adminHandler.Logout)

	adminGroup := r.Group("/admin")
	adminGroup.Use(middleware.AdminRequired())
	{
		adminGroup.GET("", adminHandler.Dashboard)
		adminGroup.GET("/users", adminHandler.Users)
		adminGroup.GET("/users/new", adminHandler.AddUserPage)
		adminGroup.POST("/users/new", adminHandler.AddUser)
		adminGroup.GET("/users/:id/edit", adminHandler.EditUserPage)
		adminGroup.POST("/users/:id/edit", adminHandler.EditUser)
		adminGroup.POST("/users/:id/avatar/clear", adminHandler.ClearUserAvatar)
		adminGroup.GET("/users/:id/delete", adminHandler.DeleteUser)

		adminGroup.GET("/rooms", adminHandler.Rooms)
		adminGroup.GET("/rooms/new", adminHandler.AddRoomPage)
		adminGroup.POST("/rooms/new", adminHandler.AddRoom)
		adminGroup.GET("/rooms/:id/edit", adminHandler.EditRoomPage)
		adminGroup.POST("/rooms/:id/edit", adminHandler.EditRoom)
		adminGroup.GET("/rooms/:id/delete", adminHandler.DeleteRoom)
		adminGroup.GET("/rooms/:id/clear", adminHandler.ClearMessages)

		adminGroup.GET("/settings", adminHandler.SettingsPage)
		adminGroup.POST("/settings", adminHandler.UpdateSettings)

		adminGroup.GET("/api", adminHandler.APIDoc)
	}

	r.GET("/api/public/settings", apiHandler.GetPublicSettings)

	apiGroup := r.Group("/api")
	apiGroup.Use(middleware.AuthRequired())
	{
		apiGroup.GET("/user/info", apiHandler.GetUserInfo)
		apiGroup.GET("/rooms", apiHandler.GetRooms)
		apiGroup.GET("/room/:id/messages", apiHandler.GetMessages)
		apiGroup.GET("/rooms/:id/messages", apiHandler.GetMessages)
		apiGroup.GET("/room/:id/latest-messages", apiHandler.GetLatestMessages)
		apiGroup.GET("/settings", apiHandler.GetSettings)
	}

	adminAPI := r.Group("/api")
	adminAPI.Use(middleware.AdminRequired())
	{
		adminAPI.GET("/users", apiHandler.GetUsers)
		adminAPI.POST("/users", apiHandler.CreateUser)
		adminAPI.PUT("/users/:id", apiHandler.UpdateUser)
		adminAPI.DELETE("/users/:id", apiHandler.DeleteUser)

		adminAPI.POST("/rooms", apiHandler.CreateRoom)
		adminAPI.PUT("/rooms/:id", apiHandler.UpdateRoom)
		adminAPI.DELETE("/rooms/:id", apiHandler.DeleteRoom)
		adminAPI.DELETE("/room/:id/messages", apiHandler.ClearRoomMessages)
	}

	gatewayGroup := r.Group("/gateway")
	gatewayGroup.Use(middleware.RequireLocalNetwork())
	{
		gatewayGroup.GET("", gatewayHandler.Index)
		gatewayGroup.GET("/api/status", gatewayHandler.GetStatus)
		gatewayGroup.GET("/api/check-port", gatewayHandler.CheckPort)
		gatewayGroup.POST("/api/start", gatewayHandler.StartProxy)
		gatewayGroup.POST("/api/stop", gatewayHandler.StopProxy)
		gatewayGroup.POST("/api/config", gatewayHandler.UpdateConfig)
		gatewayGroup.GET("/api/logs", gatewayHandler.GetLogs)
		gatewayGroup.GET("/api/certs", gatewayHandler.GetCerts)
	}

	r.GET("/ws/chat", websocket.HandleChat)

	r.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "404.html", gin.H{
			"title":   "页面未找到",
			"appName": "千盈助播",
		})
	})
}

func getPort() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	if p := os.Getenv("wizard_app_port"); p != "" {
		return p
	}
	return "3008"
}

func startUnixSocket(h http.Handler) {
	appDest := os.Getenv("TRIM_APPDEST")
	unixSocket := os.Getenv("UNIX_SOCKET")
	if unixSocket == "" {
		unixSocket = os.Getenv("GATEWAY_SOCKET")
	}

	log.Printf("[UnixSocket] 开始初始化: UNIX_SOCKET=%s, GATEWAY_SOCKET=%s, TRIM_APPDEST=%s",
		os.Getenv("UNIX_SOCKET"), os.Getenv("GATEWAY_SOCKET"), appDest)

	if unixSocket == "" {
		log.Println("[UnixSocket] 未配置 Unix Socket，跳过 Unix Socket 监听")
		return
	}

	var sockPath string
	if filepath.IsAbs(unixSocket) {
		sockPath = unixSocket
	} else if appDest != "" {
		sockPath = filepath.Join(appDest, unixSocket)
	} else {
		log.Println("[UnixSocket] Unix Socket 路径无效，跳过 Unix Socket 监听")
		return
	}

	log.Printf("[UnixSocket] Socket 路径: %s", sockPath)
	os.Remove(sockPath)

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		log.Printf("[UnixSocket] 启动失败: %v", err)
		return
	}
	defer listener.Close()

	if err := os.Chmod(sockPath, 0666); err != nil {
		log.Printf("[UnixSocket] 权限设置失败: %v", err)
	}

	if info, statErr := os.Stat(sockPath); statErr == nil {
		log.Printf("[UnixSocket] Socket 文件状态: mode=%v, size=%d", info.Mode(), info.Size())
	} else {
		log.Printf("[UnixSocket] Socket 文件 stat 失败: %v", statErr)
	}

	wrappedListener := &loggingListener{listener: listener, sockPath: sockPath}
	log.Printf("[UnixSocket] 监听成功: %s", sockPath)
	if err := http.Serve(wrappedListener, h); err != nil {
		log.Printf("[UnixSocket] 服务结束: %v", err)
	}
}

type loggingListener struct {
	listener net.Listener
	sockPath string
}

func (l *loggingListener) Accept() (net.Conn, error) {
	conn, err := l.listener.Accept()
	if err != nil {
		log.Printf("[UnixSocket] Accept 失败: %v", err)
		return nil, err
	}
	log.Printf("[UnixSocket] 新连接: remote=%v, local=%v", conn.RemoteAddr(), conn.LocalAddr())
	return conn, nil
}

func (l *loggingListener) Close() error {
	return l.listener.Close()
}

func (l *loggingListener) Addr() net.Addr {
	return l.listener.Addr()
}

func wrapWithGatewayPrefix(h http.Handler) http.Handler {
	prefix := os.Getenv("GATEWAY_PREFIX")
	if prefix == "" {
		prefix = os.Getenv("TRIM_GATEWAY_PREFIX")
	}
	log.Printf("[Gateway] GATEWAY_PREFIX=%s, TRIM_GATEWAY_PREFIX=%s",
		os.Getenv("GATEWAY_PREFIX"), os.Getenv("TRIM_GATEWAY_PREFIX"))

	if prefix == "" || prefix == "/" {
		log.Printf("[Gateway] 无前缀，直接返回原 Handler")
		return h
	}
	prefix = strings.TrimSuffix(prefix, "/")
	log.Printf("[Gateway] 启用前缀剥离: prefix=%s", prefix)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origPath := r.URL.Path
		if strings.HasPrefix(r.URL.Path, prefix+"/") || r.URL.Path == prefix {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
			log.Printf("[Gateway] 剥离前缀: %s -> %s", origPath, r.URL.Path)
		}
		h.ServeHTTP(w, r)
	})
}

func init() {
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
}

func getExeDir() string {
	exe, err := os.Executable()
	if err != nil {
		wd, _ := os.Getwd()
		return wd
	}
	return filepath.Dir(exe)
}
