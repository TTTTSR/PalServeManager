package services

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSMessage WebSocket 推送的消息结构。
type WSMessage struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub 管理所有 WebSocket 客户端连接。
type Hub struct {
	mu      sync.RWMutex
	clients map[*wsClient]bool
}

// wsClient 表示一个 WebSocket 连接。
type wsClient struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

var (
	wsHub     *Hub
	wsHubOnce sync.Once
)

// GetWSHub 返回全局 WebSocket Hub 单例。
func GetWSHub() *Hub {
	wsHubOnce.Do(func() {
		wsHub = &Hub{
			clients: make(map[*wsClient]bool),
		}
	})
	return wsHub
}

// BroadcastStatus 向所有已连接客户端广播服务器状态。
func (h *Hub) BroadcastStatus(status string) {
	msg := WSMessage{Type: "status", Status: status}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.send <- data:
		default:
			// 客户端发送缓冲区已满，跳过
		}
	}
}

// HandleWS 处理 WebSocket 升级和连接生命周期。
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	c := &wsClient{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 16),
	}

	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()

	go c.writePump()
	go c.readPump()
}

// readPump 从 WebSocket 读取消息（仅用于检测断开）。
func (c *wsClient) readPump() {
	defer func() {
		c.hub.mu.Lock()
		delete(c.hub.clients, c)
		c.hub.mu.Unlock()
		close(c.send)
		c.conn.Close()
	}()
	c.conn.SetReadLimit(256)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// writePump 将消息从 send 通道写入 WebSocket。
func (c *wsClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// BroadcastStatusGlobal 便捷函数：向全局 Hub 广播状态。
func BroadcastStatusGlobal(status string) {
	if wsHub != nil {
		wsHub.BroadcastStatus(status)
	}
}

