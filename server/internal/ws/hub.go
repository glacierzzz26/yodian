// Package ws 实时推送 Hub（设计 3.6 / 7.4 /ws/kitchen）。
// 全局单例 ws.Default：服务层任意处可 Broadcast（新单/支付/沽清/呼叫），
// 连接端用员工 token 鉴权，仅 kitchen|owner 可订阅。
package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/yodian/server/internal/auth"
	"github.com/yodian/server/internal/config"
	"github.com/yodian/server/internal/pkg/respond"
)

// Message 推送信封 {event, data}
type Message struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

type conn struct {
	ws   *websocket.Conn
	send chan []byte
	role string
}

type Hub struct {
	mu    sync.Mutex
	conns map[*conn]bool
}

// Default 全局事件总线（路由 / 服务层共享）
var Default = NewHub()

func NewHub() *Hub { return &Hub{conns: make(map[*conn]bool)} }

// Broadcast 向所有订阅连接推送事件（单条写入失败不影响其他连接）
func (h *Hub) Broadcast(event string, payload any) {
	raw, err := json.Marshal(Message{Event: event, Data: payload})
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.conns {
		select {
		case c.send <- raw:
		default:
			slog.Warn("ws send buffer full, drop", "event", event)
		}
	}
}

// HandleKitchen GET /api/ws/kitchen?token=<staff JWT> 升级并订阅
func (h *Hub) HandleKitchen(cfg *config.Config) gin.HandlerFunc {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true }, // 2 核 2G 内网部署，端口绑定收口
	}
	return func(c *gin.Context) {
		claims, err := auth.ParseStaff(cfg.JWTStaffSecret, c.Query("token"))
		if err != nil || (claims.Role != "kitchen" && claims.Role != "owner") {
			respond.Err(c, respond.ErrUnauthorized)
			return
		}
		wsc, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			slog.Warn("ws upgrade failed", "err", err)
			return
		}
		cl := &conn{ws: wsc, send: make(chan []byte, 32), role: claims.Role}
		h.mu.Lock()
		h.conns[cl] = true
		h.mu.Unlock()

		go h.writeLoop(cl)
		go h.pingLoop(cl)
		h.readLoop(cl) // 阻塞至连接关闭
	}
}

// writeLoop 出站推送（含握手完成包）
func (h *Hub) writeLoop(cl *conn) {
	cl.send <- mustMarshal(Message{Event: "connected", Data: gin.H{"role": cl.role}})
	for raw := range cl.send {
		if err := cl.ws.WriteMessage(websocket.TextMessage, raw); err != nil {
			h.unregister(cl)
			return
		}
	}
}

// pingLoop 心跳 30s，保活 + 探测死连
func (h *Hub) pingLoop(cl *conn) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for range t.C {
		if err := cl.ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
			h.unregister(cl)
			return
		}
	}
}

// readLoop 读循环：客户端异常关闭时清理（消息体不做业务处理）
func (h *Hub) readLoop(cl *conn) {
	defer h.unregister(cl)
	for {
		if _, _, err := cl.ws.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Hub) unregister(cl *conn) {
	h.mu.Lock()
	if _, ok := h.conns[cl]; ok {
		delete(h.conns, cl)
		close(cl.send)
		cl.ws.Close()
	}
	h.mu.Unlock()
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
