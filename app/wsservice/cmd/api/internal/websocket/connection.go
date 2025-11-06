package websocket

import (
	"encoding/json"
	"go-zero-voice-agent/app/wsservice/cmd/api/internal/types"
	"go-zero-voice-agent/pkg/uniqueid"
	"net/http"
	"time"

	wsTool "github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

type UpgraderConfig struct {
	ReadBufferSize    int
	WriteBufferSize   int
	EnableCompression bool
}

func newUpgrader(config UpgraderConfig) *wsTool.Upgrader {
	// 创建 WebSocket 升级器
	upgrader := wsTool.Upgrader{
		ReadBufferSize:    config.ReadBufferSize,
		WriteBufferSize:   config.WriteBufferSize,
		EnableCompression: config.EnableCompression,
		CheckOrigin: func(r *http.Request) bool {
			// 在生产环境需检查跨域
			return true
		},
	}

	return &upgrader
}

func NewConnection(wsManager *WsManager, w http.ResponseWriter, r *http.Request, userId string) error {
	upgrader := newUpgrader(UpgraderConfig{
		ReadBufferSize:    wsManager.Config.ReadBufferSize,
		WriteBufferSize:   wsManager.Config.WriteBufferSize,
		EnableCompression: wsManager.Config.EnableCompression,
	})

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return errors.Errorf("Fail to upgrade http to websocket, %v", err)
	}

	if wsManager.Config.EnableCompression {
		conn.EnableWriteCompression(true)
		if wsManager.Config.CompressionLevel != 0 {
			_ = conn.SetCompressionLevel(wsManager.Config.CompressionLevel)
		}
	}

	connection := &Connection{
		ID:        uniqueid.GenSn(uniqueid.SN_PREFIX_WEBSOCKET),
		UserID:    userId,
		Conn:      conn,
		Send:      make(chan []byte, wsManager.Config.MessageBufferSize),
		WsManager: wsManager,
		LastPingTime: time.Now(),
		IsAlive:   true,
		Metadata:  make(map[string]interface{}),
	}

	wsManager.Register <- connection
	go connection.readPump()
	go connection.writePump()

	return nil
}

func (c *Connection) readPump() {
	defer func() {
		c.WsManager.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(int64(c.WsManager.Config.MaxMessageSize))
	c.Conn.SetReadDeadline(time.Now().Add(c.WsManager.Config.ConnectionTimeout))
	c.Conn.SetPongHandler(func(string) error {
		c.mu.Lock()
		c.LastPingTime = time.Now()
		c.mu.Unlock()
		c.Conn.SetReadDeadline(time.Now().Add(c.WsManager.Config.ConnectionTimeout))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if wsTool.IsUnexpectedCloseError(err, wsTool.CloseGoingAway, wsTool.CloseAbnormalClosure) {
				c.WsManager.logx.Errorf("error: unexpected websocket close error: %v", err)
			}
			break
		}

		// 处理接收到的消息
		c.handleMessage(message)
	}
}

func (c *Connection) handleMessage(message []byte) {
	var msg types.Message
	if err := json.Unmarshal(message, &msg); err != nil {
		c.WsManager.logx.Errorf("消息解析失败: %v", err)
		return
	}

	// 设置发送者ID
	msg.From = c.UserID

	// 根据消息类型处理
	switch msg.Type {
	case "ping":
		c.handlePing()
	case "chat":
		c.handleChat(msg)
	case "notification":
		c.handleNotification(msg)
	case "status":
		c.handleStatus(msg)
	default:
		c.WsManager.logx.Infof("warn: unknown message type: %s", msg.Type)
	}
}

func (c *Connection) handlePing() {
	c.mu.Lock()
	c.LastPingTime = time.Now()
	c.mu.Unlock()

	// 发送pong响应
	response := types.Message{
		Type:      MessageTypePong,
		Timestamp: time.Now().Unix(),
	}

	data, _ := json.Marshal(response)
	select {
	case c.Send <- data:
	default:
		c.WsManager.logx.Infof("warning: connection %s send buffer is full, applying backpressure policy", c.ID)
	}
}

func (c *Connection) handleChat(msg types.Message) {
	// 验证消息数据
	if _, ok := msg.Data.(map[string]interface{}); !ok {
		c.WsManager.logx.Errorf("Invalid chat message data: %v", msg.Data)
		return
	}

	// 检查是否有目标用户或组
	if msg.To == "" {
		c.WsManager.logx.Infof("warn: Chat message is missing target")
		return
	}

	// 广播消息
	c.WsManager.Broadcast <- &msg
}

func (c *Connection) handleNotification(msg types.Message) {
	// 验证通知数据
	if _, ok := msg.Data.(map[string]interface{}); !ok {
		c.WsManager.logx.Infof("warn: Invalid notification data: %v", msg.Data)
		return
	}

	// 广播通知
	c.WsManager.Broadcast <- &msg
}

func (c *Connection) handleStatus(msg types.Message) {
	// 更新连接状态
	if statusData, ok := msg.Data.(map[string]interface{}); ok {
		c.mu.Lock()
		for key, value := range statusData {
			c.Metadata[key] = value
		}
		c.mu.Unlock()
	}

	// 发送状态确认
	response := types.Message{
		Type:      MessageTypeStatusUpdated,
		Timestamp: time.Now().Unix(),
	}

	data, _ := json.Marshal(response)
	select {
	case c.Send <- data:
	default:
		c.WsManager.logx.Infof("warn: Connection %s send buffer is full", c.ID)
	}
}

// writePump 发送消息的协程
func (c *Connection) writePump() {
	var ticker *time.Ticker
	if !c.WsManager.Config.EnableGlobalPing {
		interval := c.WsManager.Config.HeartbeatInterval
		if interval <= 0 {
			interval = 30 * time.Second
		}
		pingEvery := time.Duration(float64(interval) * 0.9)
		ticker = time.NewTicker(pingEvery)
	}
	defer func() {
		if ticker != nil {
			ticker.Stop()
		}
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(wsTool.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(wsTool.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// 将队列中的其他消息也一起发送
			n := len(c.Send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-func() <-chan time.Time {
			if ticker != nil {
				return ticker.C
			}
			return make(chan time.Time)
		}():
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(wsTool.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendMessage 发送消息给当前连接
func (c *Connection) SendMessage(message *types.Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	select {
	case c.Send <- data:
		return nil
	default:
		return errors.New("send buffer is full")
	}
}