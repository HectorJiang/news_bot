package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zelenin/go-tdlib/client"
)

// Message 消息结构体
type Message struct {
	Platform    string    `json:"platform"`
	SourceID    string    `json:"source_id"`
	SourceName  string    `json:"source_name"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	PublishedAt time.Time `json:"published_at"`
}

// Hub 管理所有 WebSocket 连接
type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan Message
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mutex      sync.RWMutex
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan Message, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *Hub) run() {
	for {
		select {
		case conn := <-h.register:
			h.mutex.Lock()
			h.clients[conn] = true
			h.mutex.Unlock()
			log.Printf("New WebSocket client connected. Total clients: %d", len(h.clients))

		case conn := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
			h.mutex.Unlock()
			log.Printf("WebSocket client disconnected. Total clients: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mutex.RLock()
			for conn := range h.clients {
				err := conn.WriteJSON(message)
				if err != nil {
					log.Printf("Error writing to WebSocket: %v", err)
					conn.Close()
					delete(h.clients, conn)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，不需要鉴权
	},
}

func (h *Hub) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	h.register <- conn

	// 保持连接，处理客户端断开
	go func() {
		defer func() {
			h.unregister <- conn
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

func main() {
	// 创建 WebSocket Hub
	hub := newHub()
	go hub.run()

	// 启动 WebSocket 服务器
	http.HandleFunc("/ws/trading/original_news", hub.handleWebSocket)

	// 启动 HTTP 服务器
	go func() {
		port := ":3001"
		log.Printf("WebSocket server starting on port %s", port)
		log.Printf("WebSocket endpoint: ws://localhost%s/ws/trading/original_news", port)
		if err := http.ListenAndServe(port, nil); err != nil {
			log.Fatal("HTTP server error:", err)
		}
	}()

	// 设置日志级别为 0，禁用 TDLib 调试日志
	client.SetLogVerbosityLevel(&client.SetLogVerbosityLevelRequest{
		NewVerbosityLevel: 0,
	})

	// 初始化 TDLib 授权器
	authorizer := client.ClientAuthorizer(&client.SetTdlibParametersRequest{
		UseTestDc:           false,
		DatabaseDirectory:   filepath.Join("./tdlib-db", "database"),
		FilesDirectory:      filepath.Join("./tdlib-db", "files"),
		UseFileDatabase:     true,
		UseChatInfoDatabase: true,
		UseMessageDatabase:  true,
		UseSecretChats:      false,
		ApiId:               22067161,
		ApiHash:             "4318f60e99526473d075240b3de8a6ee",
		SystemLanguageCode:  "en",
		DeviceModel:         "Server",
		SystemVersion:       "1.0.0",
		ApplicationVersion:  "0.1.0",
	})
	go client.CliInteractor(authorizer)

	tdlibClient, err := client.NewClient(authorizer)
	if err != nil {
		log.Fatal(err)
	}

	listener := tdlibClient.GetListener()
	defer listener.Close()

	for update := range listener.Updates {
		switch u := update.(type) {
		case *client.UpdateNewMessage:
			msg := u.Message

			// 只处理文本消息
			content, ok := msg.Content.(*client.MessageText)
			if !ok {
				continue // 非文本消息，跳过
			}

			text := content.Text.Text

			// 跳过空消息
			if text == "" {
				continue
			}

			// 获取群聊信息
			chat, err := tdlibClient.GetChat(&client.GetChatRequest{ChatId: msg.ChatId})
			if err != nil {
				log.Printf("Failed to get chat info for chat %d: %v", msg.ChatId, err)
				continue
			}

			// 构造消息
			// 设置时区为 UTC+8
			loc := time.FixedZone("UTC+8", 8*60*60)
			message := Message{
				Platform:    "telegram",
				SourceID:    fmt.Sprintf("%d", msg.ChatId),
				SourceName:  chat.Title,
				Title:       "",
				Content:     text,
				PublishedAt: time.Unix(int64(msg.Date), 0).In(loc),
			}

			// 广播消息
			hub.broadcast <- message

			// 打印日志
			msgJSON, _ := json.MarshalIndent(message, "", "  ")
			log.Printf("Broadcasting message:\n%s", string(msgJSON))
		}
	}
}
