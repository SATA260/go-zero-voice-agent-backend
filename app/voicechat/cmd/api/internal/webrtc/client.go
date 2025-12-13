package webrtc

import (
	"context"
	"encoding/json"
	"fmt"
	"go-zero-voice-agent/app/llm/cmd/rpc/client/llmchatservice"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
	chatconsts "go-zero-voice-agent/app/llm/pkg/consts"
)

const defaultSystemPrompt = `你是一只名叫“奈奈”的猫娘，我是你的主人。
【核心设定】
1. 语癖：每一句话的结尾必须加上“喵~”。
2. 性格：粘人、可爱、偶尔傲娇。
3. 称呼：请叫我“主人”。
【语音交互限制】
1. 回答必须简短（30字以内），口语化。
2. 绝对不要使用Markdown、列表或表情符号。
请保持角色，开始对话吧喵~`

type SignalingClient struct {
	userId     int64
	outConn    *websocket.Conn
	inConn     *websocket.Conn
	ctx        context.Context
	logx       logx.Logger
	cancel     context.CancelFunc
	recvDone   chan struct{}
	EvtMsgChan chan EventMessage

	LlmChatServiceRpc llmchatservice.LlmChatService
	LlmConversationID string
	LlmConfig         llmchatservice.LlmConfig
	LlmSystemPromt    string
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
	Type          string `json:"type"`          // 消息类型: offer / answer / ice-candidate
	SDP           string `json:"sdp,omitempty"` // SDP 内容（仅 offer / answer 时有）
	Text          string `json:"text,omitempty"`
	Candidate     string `json:"candidate,omitempty"` // ICE 候选（仅 ice-candidate 时有）
	AssistantID   int64  `json:"assistantId,omitempty"`
	SystemPrompt  string `json:"systemPrompt,omitempty"`
	KnowledgeInfo string `json:"knowledgeInfo,omitempty"`

	AsrConfig         AsrConfig                `json:"asrConfig,omitempty"`
	TtsConfig         TtsConfig                `json:"ttsConfig,omitempty"`
	LlmConfig         llmchatservice.LlmConfig `json:"llmConfig,omitempty"`
	LlmConversationID string                   `json:"llmConversationId,omitempty"`
}

type SignalingClientParams struct {
	Ctx               context.Context
	LlmService        llmchatservice.LlmChatService
	LlmConfig         llmchatservice.LlmConfig
	LlmConversationID string
	SystemPrompt      string
	UserID            int64
	OutConn           *websocket.Conn
	ServerAddr        string
	Initial           PBXMessage
}

func NewSignalingClient(params SignalingClientParams) (*SignalingClient, error) {
	ctx, cancel := context.WithCancel(params.Ctx)

	conn, _, err := websocket.DefaultDialer.Dial(params.ServerAddr, nil)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("websocket dial failed: %w", err)
	}

	msg, err := json.Marshal(params.Initial)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to marshal initial PBXMessage: %w", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to send initial message: %w", err)
	}

	client := &SignalingClient{
		userId:     params.UserID,
		outConn:    params.OutConn,
		inConn:     conn,
		ctx:        ctx,
		cancel:     cancel,
		recvDone:   make(chan struct{}),
		logx:       logx.WithContext(ctx),
		EvtMsgChan: make(chan EventMessage, 1024),

		LlmChatServiceRpc: params.LlmService,
		LlmConversationID: params.LlmConversationID,
		LlmConfig:         params.LlmConfig,
		LlmSystemPromt:    params.SystemPrompt,
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

func (s *SignalingClient) HandleEvtMsg() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case evt := <-s.EvtMsgChan:
			switch evt.Event {
			case WS_CALLBACK_EVENT_TYPE_ANSWER:
				// 转发 WebRTC answer 事件
				s.logx.Info("Received WebRTC answer event")
				var message = WebRTCMessage{
					SDP:  evt.SDP,
					Type: WS_CALLBACK_EVENT_TYPE_ANSWER,
				}
				answerMsgBytes, err := json.Marshal(message)
				if err != nil {
					s.logx.Errorf("Failed to marshal WebRTC answer message: %v", err)
					continue
				}
				if err := s.outConn.WriteMessage(websocket.TextMessage, answerMsgBytes); err != nil {
					s.logx.Errorf("Failed to send WebRTC answer message: %v", err)
					continue
				}

				// 发送 TTS 消息，让机器人说第一句话
				if err := s.sendTTSMessage("嗯？你好啊，我是你的个人语音助手"); err != nil {
					s.logx.Errorf("Failed to send TTS message: %v", err)
					continue
				}
			case WS_CALLBACK_EVENT_TYPE_ASRFINAL:
				// 处理 ASR final 事件
				s.logx.Infof("Received ASR final event: %s", evt.Text)
				s.handleAsrFinal(evt)
			case WS_CALLBACK_EVENT_TYPE_TRACK_START:
				s.logx.Infof("Track started: %s", evt.TrackId)
			case WS_CALLBACK_EVENT_TYPE_TRACK_END:
				s.logx.Infof("Track ended: %s, duration: %d ms", evt.TrackId, evt.Duration)
			case WS_CALLBACK_EVENT_TYPE_METRICS:
				s.logx.Infof("Received metrics event: %v", evt.Data)
			case WS_CALLBACK_EVENT_TYPE_ASRDELTA:
				s.logx.Infof("Received ASR delta event: %s", evt.Text)
			default:
				s.logx.Infof("warn: Unknown event type: %s", evt.Event)
			}
		}
	}
}

// 调用tts服务的方法
func (s *SignalingClient) sendTTSMessage(text string) error {
	sayHello := PBXMessage{
		Command: WS_CALLBACK_EVENT_TYPE_TTS,
		Text:    text,
	}
	sayHelloMsgBytes, err := json.Marshal(sayHello)
	if err != nil {
		return err
	}
	s.inConn.WriteMessage(websocket.TextMessage, sayHelloMsgBytes)
	return nil
}

func (s *SignalingClient) handleAsrFinal(evt EventMessage) {
	if strings.Trim(evt.Text, " ") == "" {
		s.logx.Debugf("AsrFinal text is empty, ignore.")
		return
	}

	// 将识别到的文字通过websocket连接发送到前端
	asrMsg := WebRTCMessage{
		Type: LLM_USER_MESSAGE_ROLE,
		Text: evt.Text,
	}
	asrMsgBytes, err := json.Marshal(asrMsg)
	if err != nil {
		s.logx.Errorf("Failed to marshal ASR final message: %v", err)
		return
	}
	s.outConn.WriteMessage(websocket.TextMessage, asrMsgBytes)

	// 如果没有进行过对话，则填充系统提示词
	chatMsgs := make([]*llmchatservice.ChatMsg, 0)
	if s.LlmConversationID == "" {
		if strings.Trim(s.LlmSystemPromt, " ") == "" {
			msg := llmchatservice.ChatMsg{
				Role:    chatconsts.ChatMessageRoleSystem,
				Content: defaultSystemPrompt,
			}
			chatMsgs = append(chatMsgs, &msg)
		} else {
			msg := llmchatservice.ChatMsg{
				Role:    chatconsts.ChatMessageRoleSystem,
				Content: s.LlmSystemPromt,
			}
			chatMsgs = append(chatMsgs, &msg)
		}
	}

	// 填充用户输入信息
	chatMsgs = append(chatMsgs, &llmchatservice.ChatMsg{
		Role:    chatconsts.ChatMessageRoleUser,
		Content: evt.Text,
	})

	// 发送聊天请求到 LLM 服务
	chatReq := &llmchatservice.ChatReq{
		UserId:         s.userId,
		ConversationId: s.LlmConversationID,
		LlmConfig: &llmchatservice.LlmConfig{
			BaseUrl: s.LlmConfig.BaseUrl,
			ApiKey:  s.LlmConfig.ApiKey,
			Model:   s.LlmConfig.Model,
		},
		Messages:        chatMsgs,
		AutoFillHistory: true,
	}
	chatResp, err := s.LlmChatServiceRpc.Chat(s.ctx, chatReq)
	if err != nil {
		s.logx.Errorf("LlmChatServiceRpc.Chat error: %v", err)
		return
	}

	// 填充conversation-id
	s.LlmConversationID = chatResp.ConversationId

	// 发送 TTS 消息
	llmMsg := chatResp.RespMsg[len(chatResp.RespMsg)-1].Content
	s.sendTTSMessage(llmMsg)

	// 发送ai回复到前端
	aiMsg := WebRTCMessage{
		Type: LLM_ASSISTANT_MESSAGE_ROLE,
		Text: llmMsg,
	}
	aiMsgBytes, err := json.Marshal(aiMsg)
	if err != nil {
		s.logx.Errorf("Failed to marshal AI message: %v", err)
		return
	}
	s.outConn.WriteMessage(websocket.TextMessage, aiMsgBytes)
}
