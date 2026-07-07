package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"qyzb-server/internal/services"
	"qyzb-server/internal/utils"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 8192
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type chatMessage struct {
	Content string `json:"content"`
	Type    string `json:"type"`
}

func HandleChat(c *gin.Context) {
	session := sessions.Default(c)
	rawUserID := session.Get("user_id")
	if rawUserID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var userID uint
	switch v := rawUserID.(type) {
	case uint:
		userID = v
	case int:
		userID = uint(v)
	case float64:
		userID = uint(v)
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	roomIDStr := c.Query("room_id")
	roomID, _ := strconv.Atoi(roomIDStr)

	if roomID == 0 || userID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	userService := services.NewUserService()
	user, err := userService.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket 升级失败: %v", err)
		return
	}

	hub := GetHub()
	fixedAvatar := utils.FixAvatarPath(user.Avatar)
	client := &Client{
		ID:       uint(userID),
		RoomID:   uint(roomID),
		Nickname: user.Nickname,
		Username: user.Username,
		Avatar:   fixedAvatar,
		conn:     conn,
		send:     make(chan []byte, 256),
	}

	hub.register <- client

	go writePump(conn, client)
	go readPump(conn, client, hub, user.Nickname, user.Username, fixedAvatar)
}

func readPump(conn *websocket.Conn, client *Client, hub *Hub, nickname, username, avatar string) {
	defer func() {
		hub.unregister <- client
		conn.Close()
	}()

	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket 错误: %v", err)
			}
			break
		}

		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			continue
		}

		if wsMsg.Type == "message" {
			var chatMsg chatMessage
			dataBytes, _ := json.Marshal(wsMsg.Data)
			json.Unmarshal(dataBytes, &chatMsg)

			if chatMsg.Content == "" {
				continue
			}

			msgType := chatMsg.Type
			if msgType == "" {
				msgType = "text"
			}

			msgDTO, err := hub.messageService.Create(client.RoomID, client.ID, chatMsg.Content, msgType)
			if err != nil {
				log.Printf("消息保存失败: %v", err)
				continue
			}

			hub.Broadcast(client.RoomID, "message", msgDTO)
		}
	}
}

func writePump(conn *websocket.Conn, client *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			conn.WriteMessage(websocket.TextMessage, message)

		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
