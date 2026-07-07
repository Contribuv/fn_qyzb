package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"qyzb-server/internal/services"
)

type Hub struct {
	clients    map[uint]map[*Client]bool
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex

	messageService *services.MessageService
	userService    *services.UserService
}

type Client struct {
	ID       uint
	RoomID   uint
	Nickname string
	Username string
	Avatar   string
	conn     interface{}
	send     chan []byte
}

type Message struct {
	RoomID uint
	Data   []byte
}

type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

var hub *Hub
var once sync.Once

func GetHub() *Hub {
	once.Do(func() {
		hub = &Hub{
			clients:        make(map[uint]map[*Client]bool),
			broadcast:      make(chan *Message, 256),
			register:       make(chan *Client),
			unregister:     make(chan *Client),
			messageService: services.NewMessageService(),
			userService:    services.NewUserService(),
		}
		go hub.run()
	})
	return hub
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.RoomID] == nil {
				h.clients[client.RoomID] = make(map[*Client]bool)
			}
			h.clients[client.RoomID][client] = true
			h.mu.Unlock()
			log.Printf("客户端加入房间 %d，当前人数: %d", client.RoomID, len(h.clients[client.RoomID]))
			go h.BroadcastOnlineUsers(client.RoomID)

		case client := <-h.unregister:
			h.mu.Lock()
			roomID := client.RoomID
			if room, ok := h.clients[roomID]; ok {
				if _, ok := room[client]; ok {
					delete(room, client)
					close(client.send)
					if len(room) == 0 {
						delete(h.clients, roomID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("客户端离开房间 %d", roomID)
			go h.BroadcastOnlineUsers(roomID)

		case message := <-h.broadcast:
			h.mu.RLock()
			room := h.clients[message.RoomID]
			h.mu.RUnlock()
			for client := range room {
				select {
				case client.send <- message.Data:
				default:
					close(client.send)
					h.mu.Lock()
					if r, ok := h.clients[message.RoomID]; ok {
						delete(r, client)
					}
					h.mu.Unlock()
				}
			}
		}
	}
}

func (h *Hub) Broadcast(roomID uint, msgType string, data interface{}) {
	wsMsg := WSMessage{Type: msgType, Data: data}
	jsonData, err := json.Marshal(wsMsg)
	if err != nil {
		log.Printf("消息序列化失败: %v", err)
		return
	}
	h.broadcast <- &Message{
		RoomID: roomID,
		Data:   jsonData,
	}
}

func (h *Hub) RoomCount(roomID uint) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if room, ok := h.clients[roomID]; ok {
		return len(room)
	}
	return 0
}

func (h *Hub) TotalConnections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	count := 0
	for _, room := range h.clients {
		count += len(room)
	}
	return count
}

type OnlineUser struct {
	ID       uint   `json:"id"`
	Nickname string `json:"nickname"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

func (h *Hub) GetOnlineUsers(roomID uint) []OnlineUser {
	h.mu.RLock()
	defer h.mu.RUnlock()
	users := make([]OnlineUser, 0)
	seen := make(map[uint]bool)
	if room, ok := h.clients[roomID]; ok {
		for client := range room {
			if !seen[client.ID] {
				seen[client.ID] = true
				users = append(users, OnlineUser{
					ID:       client.ID,
					Nickname: client.Nickname,
					Username: client.Username,
					Avatar:   client.Avatar,
				})
			}
		}
	}
	return users
}

func (h *Hub) BroadcastOnlineUsers(roomID uint) {
	users := h.GetOnlineUsers(roomID)
	h.Broadcast(roomID, "onlineUsers", users)
}
