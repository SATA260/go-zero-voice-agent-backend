package websocket

import (
	"context"
	"encoding/json"
	"go-zero-voice-agent/app/wsservice/cmd/api/internal/config"
	"go-zero-voice-agent/app/wsservice/cmd/api/internal/types"

	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

// Connection represents the state for a single WebSocket session.
type Connection struct {
	ID           string
	UserID       string
	Conn         *websocket.Conn
	Send         chan []byte
	WsManager    *WsManager
	LastPingTime time.Time
	IsAlive      bool
	mu           sync.RWMutex
	Metadata     map[string]interface{}
}

type broadcastJob struct {
	kind  int
	shard int
	data  []byte
}

// WsManager orchestrates WebSocket connections, fanout, and housekeeping.
type WsManager struct {
	Connections      map[string]*Connection     // active connections keyed by connection ID
	UserConnections  map[string]map[string]bool // user ID -> connection IDs
	GroupConnections map[string]map[string]bool // reserved group mapping
	Broadcast        chan *types.Message        // inbound broadcast messages
	Register         chan *Connection           // queue for new registrations
	Unregister       chan *Connection           // queue for disconnects
	ConnectionCount  int64                      // number of active connections
	Config           *config.WsConfig           // WebSocket runtime config
	Mu               sync.RWMutex               // global state lock
	Ctx              context.Context            // lifecycle context
	Cancel           context.CancelFunc         // cancel function for context

	ShardCount int
	ShardConns []map[string]*Connection
	ShardLocks []sync.RWMutex

	BroadcastJobs chan broadcastJob
	PingJobs      chan int

	logx logx.Logger
}

const broadcastJobAll = 1 // broadcast payload to every shard

// NewWsManager constructs a WebSocket manager with shard workers and background tasks.
func NewWsManager(config *config.WsConfig) *WsManager {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &WsManager{
		Connections:      make(map[string]*Connection),
		UserConnections:  make(map[string]map[string]bool),
		GroupConnections: make(map[string]map[string]bool),
		Broadcast:        make(chan *types.Message, config.MessageQueueSize),
		Register:         make(chan *Connection, 1000),
		Unregister:       make(chan *Connection, 1000),
		Config:           config,
		Ctx:              ctx,
		Cancel:           cancel,
		logx:             logx.WithContext(ctx),
	}

	// init shards
	if manager.Config.ShardCount <= 0 {
		manager.Config.ShardCount = 1
	}
	manager.ShardCount = manager.Config.ShardCount
	manager.ShardConns = make([]map[string]*Connection, manager.ShardCount)
	manager.ShardLocks = make([]sync.RWMutex, manager.ShardCount)
	for i := 0; i < manager.ShardCount; i++ {
		manager.ShardConns[i] = make(map[string]*Connection)
	}

	// init broadcast workers
	if manager.Config.BroadcastWorkerCount <= 0 {
		manager.Config.BroadcastWorkerCount = 1
	}
	manager.BroadcastJobs = make(chan broadcastJob, manager.Config.MessageQueueSize)
	for i := 0; i < manager.Config.BroadcastWorkerCount; i++ {
		go manager.broadcastWorker()
	}

	// init global ping workers
	if manager.Config.EnableGlobalPing {
		if manager.Config.PingWorkerCount <= 0 {
			manager.Config.PingWorkerCount = 1
		}
		manager.PingJobs = make(chan int, manager.ShardCount)
		for i := 0; i < manager.Config.PingWorkerCount; i++ {
			go manager.pingWorker()
		}
	}

	go manager.run()
	return manager
}

// run drives the main event loop for registrations, broadcasts, and housekeeping.
func (m *WsManager) run() {
	ticker := time.NewTicker(m.Config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.Ctx.Done():
			return
		case conn := <-m.Register:
			m.registerConnection(conn)
		case conn := <-m.Unregister:
			m.unregisterConnection(conn)
		case message := <-m.Broadcast:
			// Marshal once to avoid repeated allocations.
			if message.Timestamp == 0 {
				message.Timestamp = time.Now().Unix()
			}
			data, err := json.Marshal(message)
			if err != nil {
				m.logx.Errorf("failed to marshal outbound message: %v", err)
				continue
			}
			switch {
			case message.To != "":
				m.sendToUser(message.To, data)
			default:
				m.enqueueBroadcastAll(data)
			}
		case <-ticker.C:
			if m.Config.EnableGlobalPing {
				// Fire ping jobs per shard to spread the load.
				for i := 0; i < m.ShardCount; i++ {
					select {
					case m.PingJobs <- i:
					default:
					}
				}
			}
			m.checkHeartbeats()
		}
	}
}

// broadcastWorker fans out broadcast jobs for a specific shard.
func (m *WsManager) broadcastWorker() {
	for job := range m.BroadcastJobs {
		m.ShardLocks[job.shard].RLock()
		for _, conn := range m.ShardConns[job.shard] {
			if conn.IsAlive {
				m.trySend(conn, job.data, func() {
					m.logx.Infof("warning: connection %s send buffer is full, applying backpressure policy", conn.ID)
				})
			}
		}
		m.ShardLocks[job.shard].RUnlock()
	}
}

// pingWorker issues control ping frames for connections in a shard.
func (m *WsManager) pingWorker() {
	for shard := range m.PingJobs {
		m.ShardLocks[shard].RLock()
		for _, conn := range m.ShardConns[shard] {
			if conn.IsAlive {
				_ = conn.Conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second))
			}
		}
		m.ShardLocks[shard].RUnlock()
	}
}

// registerConnection attaches a new connection to the manager.
func (m *WsManager) registerConnection(conn *Connection) {
	m.Mu.Lock()
	defer m.Mu.Unlock()

	// Enforce the maximum connection limit.
	if atomic.LoadInt64(&m.ConnectionCount) >= m.Config.MaxConnections {
		conn.Conn.Close()
		m.logx.Infof("warning: max connection limit reached (%d)", m.Config.MaxConnections)
		return
	}

	m.Connections[conn.ID] = conn
	atomic.AddInt64(&m.ConnectionCount, 1)

	// Attach the connection to its shard bucket.
	sh := m.shardIndex(conn.ID)
	m.ShardLocks[sh].Lock()
	m.ShardConns[sh][conn.ID] = conn
	m.ShardLocks[sh].Unlock()

	// Track the connection under the user mapping for fanout.
	if conn.UserID != "" {
		if m.UserConnections[conn.UserID] == nil {
			m.UserConnections[conn.UserID] = make(map[string]bool)
		}
		m.UserConnections[conn.UserID][conn.ID] = true
	}

	m.logx.Infof("registered websocket connection %s for user %s, active connections: %d",
		conn.ID, conn.UserID, atomic.LoadInt64(&m.ConnectionCount))
}

// unregisterConnection removes the connection from tracking structures.
func (m *WsManager) unregisterConnection(conn *Connection) {
	m.Mu.Lock()
	defer m.Mu.Unlock()

	if _, exists := m.Connections[conn.ID]; exists {
		delete(m.Connections, conn.ID)
		atomic.AddInt64(&m.ConnectionCount, -1)

		// Remove the connection from the shard bucket.
		sh := m.shardIndex(conn.ID)
		m.ShardLocks[sh].Lock()
		delete(m.ShardConns[sh], conn.ID)
		m.ShardLocks[sh].Unlock()

		// Remove the connection from the user mapping.
		if conn.UserID != "" && m.UserConnections[conn.UserID] != nil {
			delete(m.UserConnections[conn.UserID], conn.ID)
			if len(m.UserConnections[conn.UserID]) == 0 {
				delete(m.UserConnections, conn.UserID)
			}
		}

		close(conn.Send)
		m.logx.Infof("unregistered websocket connection %s, active connections: %d",
			conn.ID, atomic.LoadInt64(&m.ConnectionCount))
	}
}

// shardIndex deterministically maps an identifier to a shard slot.
func (m *WsManager) shardIndex(id string) int {
	if m.ShardCount <= 1 {
		return 0
	}
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(id))
	return int(hasher.Sum32() % uint32(m.ShardCount))
}

// trySend attempts to enqueue data for the connection while applying backpressure policies.
func (m *WsManager) trySend(conn *Connection, data []byte, onDrop func()) {
	if m.Config.DropOnFull {
		select {
		case conn.Send <- data:
		default:
			onDrop()
			if m.Config.CloseOnBackpressure {
				conn.Conn.Close()
			}
		}
		return
	}
	// In blocking mode, wait at most the configured duration before giving up.
	timeout := m.Config.SendTimeout
	if timeout <= 0 {
		timeout = 50 * time.Millisecond
	}
	select {
	case conn.Send <- data:
	case <-time.After(timeout):
		onDrop()
		if m.Config.CloseOnBackpressure {
			conn.Conn.Close()
		}
	}
}

// sendToUser delivers a payload to all connections attached to a user ID.
func (m *WsManager) sendToUser(userID string, data []byte) {
	if connections, exists := m.UserConnections[userID]; exists {
		for connID := range connections {
			if conn, ok := m.Connections[connID]; ok && conn.IsAlive {
				m.trySend(conn, data, func() { m.logx.Infof("warning: send buffer is full for user %s connection %s", userID, conn.ID) })
			}
		}
	}
}

// enqueueBroadcastAll schedules a broadcast job for every shard.
func (m *WsManager) enqueueBroadcastAll(data []byte) {
	for i := 0; i < m.ShardCount; i++ {
		select {
		case m.BroadcastJobs <- broadcastJob{kind: broadcastJobAll, shard: i, data: data}:
		default:
			m.logx.Infof("warning: broadcast job queue is full, dropping message")
		}
	}
}

// checkHeartbeats closes connections that missed the heartbeat deadline.
func (m *WsManager) checkHeartbeats() {
	m.Mu.RLock()
	defer m.Mu.RUnlock()

	now := time.Now()
	for _, conn := range m.Connections {
		if now.Sub(conn.LastPingTime) > m.Config.ConnectionTimeout {
			m.logx.Infof("warning: connection %s exceeded heartbeat timeout, closing", conn.ID)
			conn.IsAlive = false
			conn.Conn.Close()
		}
	}
}
