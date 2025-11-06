// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"time"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	ChatroomRpcConf zrpc.RpcClientConf
	Websocket       WsConfig
}

type WsConfig struct {
	// 最大连接数
	MaxConnections int64
	// 心跳间隔
	HeartbeatInterval time.Duration
	// 连接超时时间
	ConnectionTimeout time.Duration
	// 消息缓冲区大小
	MessageBufferSize int
	// 读缓冲区大小
	ReadBufferSize int
	// 写缓冲区大小
	WriteBufferSize int
	// 最大消息大小
	MaxMessageSize int
	// 是否启用压缩
	EnableCompression bool
	// 是否启用消息队列
	EnableMessageQueue bool
	// 消息队列大小
	MessageQueueSize int
	// 是否启用集群模式
	EnableCluster bool
	// 集群节点ID
	ClusterNodeID string
	// 分片数量
	ShardCount int
	// 广播worker数量
	BroadcastWorkerCount int
	// 发送缓冲区满时是否丢弃
	DropOnFull bool
	// 压缩等级（-2..9）
	CompressionLevel int
	// 慢消费者策略：背压触发时直接断开
	CloseOnBackpressure bool
	// 发送阻塞超时（用于非 DropOnFull 模式）
	SendTimeout time.Duration
	// 启用全局心跳
	EnableGlobalPing bool
	// 全局心跳workers
	PingWorkerCount int
}
