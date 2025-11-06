package websocket

const (
	// 系统预定义的消息类型
	MessageTypeText = "text"
	MessageTypeBinary = "binary"
	MessageTypeClose = "close"
	MessageTypePing = "ping"
	MessageTypePong = "pong"
	MessageTypeStatus = "status"
	MessageTypeStatusUpdated = "status_updated"
)