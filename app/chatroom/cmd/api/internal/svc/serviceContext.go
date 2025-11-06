// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"go-zero-voice-agent/app/chatroom/cmd/api/internal/config"
	"go-zero-voice-agent/app/chatroom/cmd/api/internal/websocket"
)

type ServiceContext struct {
	Config config.Config

	WsManager *websocket.WsManager
}

func NewServiceContext(c config.Config) *ServiceContext {
	wsManager := websocket.NewWsManager(&c.Websocket)

	return &ServiceContext{
		Config:    c,
		WsManager: wsManager,
	}
}
