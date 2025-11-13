package webrtc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

type SignalingClient struct {
	outConn    *websocket.Conn
	inConn       *websocket.Conn
	ctx        context.Context
	cancel     context.CancelFunc
	recvDone   chan struct{}
	EvtMsgChan chan EventMessage
	logx       logx.Logger
}

type PBXMessage struct {
	Command string       `json:"command"` // 操作类型，如 'invite', 'tts'
	Option  *CallOptions `json:"option,omitempty"`
	Text    string       `json:"text,omitempty"`
	PlayId  string       `json:"playId,omitempty"`
}

// CallOptions 包含呼叫配置的详细信息
type CallOptions struct {
	Asr   *AsrConfig `json:"asr,omitempty"`   // 自动语音识别配置
	Tts   *TtsConfig `json:"tts,omitempty"`   // 语音合成配置
	Offer string     `json:"offer,omitempty"` // SDP Offer 信息
}

// AsrConfig 自动语音识别配置
type AsrConfig struct {
	Provider  string `json:"provider"`
	AppId     string `json:"appId"`
	SecretId  string `json:"secretId"`
	SecretKey string `json:"secretKey"`
	Language  string `json:"language"`
}

// TtsConfig 文本转语音配置
type TtsConfig struct {
	Provider  string  `json:"provider"`
	Speaker   string  `json:"speaker"`
	AppId     string  `json:"appId"`
	SecretId  string  `json:"secretId"`
	SecretKey string  `json:"secretKey"`
	Speed     float32 `json:"speed"`
	Volume    int     `json:"volume"`
}

// EventMessage 表示服务端发送的事件通知
type EventMessage struct {
	Event     string                 `json:"event"`
	TrackId   string                 `json:"trackId,omitempty"`
	Timestamp *uint64                `json:"timestamp,omitempty"`
	Key       string                 `json:"key,omitempty"`
	Duration  uint32                 `json:"duration,omitempty"`
	SDP       string                 `json:"sdp,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Text      string                 `json:"text,omitempty"`
}

type WebRTCMessage struct {
	Type         string  `json:"type"`          // 消息类型: offer / answer / ice-candidate
	SDP          string  `json:"sdp,omitempty"` // SDP 内容（仅 offer / answer 时有）
	Text         string  `json:"text,omitempty"`
	AssistantID  int64   `json:"assistantId" binding:"required"`
	SystemPrompt string  `json:"systemPrompt"`
	Instruction  string  `json:"instruction"`
	Speaker      string  `json:"speaker"`
	Language     string  `json:"language"`
	ApiKey       string  `json:"apiKey"`
	ApiSecret    string  `json:"apiSecret"`
	Speed        float32 `json:"speed"`
	Volume       int     `json:"volume"`
	PersonaTag   string  `json:"personaTag"`
	Temperature  float32 `json:"temperature"`
	MaxTokens    int     `json:"maxTokens"`
}

func NewSignalingClient(Conn *websocket.Conn, ctx context.Context, serverAddr string, initial PBXMessage) (*SignalingClient, error) {
	ctx, cancel := context.WithCancel(ctx)

	conn, _, err := websocket.DefaultDialer.Dial(serverAddr, nil)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("websocket dial failed: %w", err)
	}

	msg, err := json.Marshal(initial)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to marshal initial PBXMessage: %w", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to send initial message: %w", err)
	}

	client := &SignalingClient{
		outConn:  Conn,
		inConn:     conn,
		ctx:      ctx,
		cancel:   cancel,
		recvDone: make(chan struct{}),
		logx:     logx.WithContext(ctx),
		EvtMsgChan: make(chan EventMessage, 1024),
	}
	client.logx.Info("Send invite command to RustPBX....")

	return client, nil
}

func (s *SignalingClient) Listen(knowledgeInfo string) {
	defer close(s.recvDone)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			typeVal, data, err := s.inConn.ReadMessage()
			if err != nil {
				return
			}
			if typeVal != websocket.TextMessage {
				continue
			}
			s.logx.Info("received:", string(data))
			var evt EventMessage
			if err := json.Unmarshal(data, &evt); err != nil {
				continue
			}
			s.EvtMsgChan <- evt
		}
	}
}
