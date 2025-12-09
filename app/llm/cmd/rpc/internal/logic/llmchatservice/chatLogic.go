package llmchatservicelogic

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/model"
	chatconsts "go-zero-voice-agent/app/llm/pkg/consts"
	publicconsts "go-zero-voice-agent/pkg/consts"
	"go-zero-voice-agent/pkg/tool"

	"github.com/Masterminds/squirrel"
	openai "github.com/sashabaranov/go-openai"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ChatLogic 处理聊天请求的逻辑结构体
type ChatLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// NewChatLogic 创建一个新的 ChatLogic 实例
func NewChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatLogic {
	return &ChatLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Chat 处理聊天请求，与 LLM 进行交互
func (l *ChatLogic) Chat(in *pb.ChatReq) (*pb.ChatResp, error) {
	l.Logger.Infof("Chat request: %+v", in)

	// 1. 校验请求参数
	if in == nil || in.LlmConfig == nil {
		return nil, status.Error(codes.InvalidArgument, "missing llm config")
	}
	if in.LlmConfig.Model == "" {
		return nil, status.Error(codes.InvalidArgument, "model is required")
	}

	if err := l.ctx.Err(); err != nil {
		return nil, err
	}

	var chatSession model.ChatSession

	// 2. 获取或创建会话
	if in.ConversationId != "" {
		// 如果提供了会话 ID，则查找现有会话
		session, err := l.svcCtx.ChatSessionModel.FindOneByConvId(l.ctx, in.ConversationId)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return nil, status.Errorf(codes.NotFound, "conversation ID %s not found", in.ConversationId)
			}
			return nil, status.Errorf(codes.Internal, "failed to fetch conversation ID %s: %v", in.ConversationId, err)
		}

		// 验证会话归属
		if session.UserId.Int64 != in.UserId {
			return nil, status.Errorf(codes.PermissionDenied, "conversation ID %s does not belong to user %d", in.ConversationId, in.UserId)
		}

		chatSession = *session
	} else {
		// 如果未提供会话 ID，则创建新会话
		// 截取输入消息的前 10 字符作为标题
		title := ""
		if len(in.Messages) > 0 {
			content := in.Messages[0].GetContent()
			if len(content) > 10 {
				title = content[:10]
			} else {
				title = content
			}
		}

		newSession := &model.ChatSession{
			ConvId: generateConversationID(),
			UserId: sql.NullInt64{Int64: in.UserId, Valid: true},
			Title:  title,
		}
		_, err := l.svcCtx.ChatSessionModel.Insert(l.ctx, nil, newSession)
		if err != nil {
			l.Logger.Errorf("failed to create new conversation: %v", err)
			return nil, status.Errorf(codes.Internal, "failed to create new conversation: %v", err)
		}
		chatSession = *newSession
	}

	// 3. 收集历史消息
	historyMsgs, err := l.collectHistory(in, &chatSession)
	if err != nil {
		l.Logger.Errorf("collectHistory error: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	l.Logger.Infof("Collected %d history messages for conversation %s", len(historyMsgs), chatSession.ConvId)
	l.Logger.Debugf("History messages: %+v", historyMsgs)

	// 将当前请求的消息追加到历史消息中
	if len(in.Messages) > 0 {
		historyMsgs = append(historyMsgs, in.Messages...)
	}

	// 构建 OpenAI 格式的消息列表
	openaiMsgs := buildSyncMessages(historyMsgs)

	// 4. 创建 OpenAI 客户端
	client, err := l.newSyncOpenAIClient(in.LlmConfig)
	if err != nil {
		l.Logger.Errorf("newSyncOpenAIClient error: %v", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// 5. 构建并发送聊天完成请求
	req := buildSyncChatCompletionRequest(in, openaiMsgs)
	l.Logger.Infof("OpenAI request: %+v", req)

	completion, err := client.CreateChatCompletion(l.ctx, req)
	if err != nil {
		l.Logger.Errorf("CreateChatCompletion error: %v", err)
		return nil, status.Errorf(codes.Internal, "create chat completion failed: %v", err)
	}
	if len(completion.Choices) == 0 {
		return nil, status.Error(codes.Internal, "empty response from llm")
	}

	choice := completion.Choices[0]
	if len(choice.Message.ToolCalls) > 0 {
		return nil, status.Error(codes.Unimplemented, "tool calls are not supported in sync chat mode")
	}

	// 6. 处理响应
	assistantMsg := &pb.ChatMsg{
		Role:    chatconsts.ChatMessageRoleAssistant,
		Content: choice.Message.Content,
	}
	l.Logger.Infof("LLM response content: %s", choice.Message.Content)

	// 异步缓存新消息（用户输入 + 助手响应）以避免重复
	go l.svcCtx.CacheConversation(chatSession.ConvId, in.Messages, assistantMsg)

	respMsgs := make([]*pb.ChatMsg, 0)
	respMsgs = append(respMsgs, assistantMsg)

	return &pb.ChatResp{
		Id:      chatSession.ConvId,
		RespMsg: respMsgs,
	}, nil
}

// collectHistory 收集聊天历史记录，优先从 Redis 缓存获取，如果缓存未命中则从数据库获取
func (l *ChatLogic) collectHistory(in *pb.ChatReq, session *model.ChatSession) ([]*pb.ChatMsg, error) {
	if in.ConversationId == "" {
		return []*pb.ChatMsg{}, nil
	}

	autoFill := in.GetAutoFillHistory()
	if autoFill == false {
		return []*pb.ChatMsg{}, nil
	}

	cacheKey := publicconsts.ChatCacheKeyPrefix + in.GetConversationId()
	length := int(in.GetLlmConfig().GetContentLength())
	if length <= 0 {
		length = 20
	}

	rawMsgs, err := l.svcCtx.RedisClient.Lrange(cacheKey, -length, -1)
	if err != nil {
		l.Logger.Errorf("failed to get history from redis: %v", err)
	}

	// 优先使用缓存中的历史记录
	if len(rawMsgs) > 0 {
		messages := make([]*pb.ChatMsg, 0, len(rawMsgs))
		for idx, raw := range rawMsgs {
			var decoded pb.ChatMsg
			if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
				l.Logger.Errorf("decode cached message failed, key: %s, index: %d, err: %v", cacheKey, idx, err)
				continue
			}
			messages = append(messages, &pb.ChatMsg{Role: decoded.Role, Content: decoded.Content})
		}
		if len(messages) > 0 {
			return messages, nil
		}
	}

	// 当缓存中没有数据时，从数据库中获取历史记录
	l.Logger.Infof("no cached history found in redis for key: %s", cacheKey)
	queryBuilder := l.svcCtx.ChatMessageModel.SelectBuilder().Where(squirrel.Eq{"session_id": session.Id})
	pageMsgs, err := l.svcCtx.ChatMessageModel.FindPageListByPage(l.ctx, queryBuilder, 1, int64(length), "id DESC")
	if err != nil {
		l.Logger.Errorf("failed to get history from db: %v", err)
		return nil, err
	}
	messages := make([]*pb.ChatMsg, 0, len(pageMsgs))
	for i := len(pageMsgs) - 1; i >= 0; i-- {
		msg := pageMsgs[i]
		messages = append(messages, &pb.ChatMsg{Role: msg.Role, Content: tool.NullStringToString(msg.Content)})
	}

	// 将历史记录重新缓存到 Redis 中
	if len(messages) > 0 {
		cachePayload := make([]string, 0, len(messages))
		for idx, message := range messages {
			if message == nil {
				continue
			}
			encoded, marshalErr := json.Marshal(message)
			if marshalErr != nil {
				l.Logger.Errorf("marshal db message failed, key: %s, index: %d, err: %v", cacheKey, idx, marshalErr)
				continue
			}
			cachePayload = append(cachePayload, string(encoded))
		}

		if len(cachePayload) > 0 {
			if _, delErr := l.svcCtx.RedisClient.Del(cacheKey); delErr != nil {
				l.Logger.Errorf("failed to clear redis history cache for key: %s, err: %v", cacheKey, delErr)
			}

			values := make([]any, 0, len(cachePayload))
			for _, payload := range cachePayload {
				values = append(values, payload)
			}

			if _, pushErr := l.svcCtx.RedisClient.Rpush(cacheKey, values...); pushErr != nil {
				l.Logger.Errorf("failed to repopulate redis history cache for key: %s, err: %v", cacheKey, pushErr)
			} else {
				l.svcCtx.RedisClient.Expire(cacheKey, chatconsts.ChatCacheExpireSeconds)
			}
		}
	}

	return messages, nil
}

// newSyncOpenAIClient 创建一个新的同步 OpenAI 客户端
func (l *ChatLogic) newSyncOpenAIClient(cfg *pb.LlmConfig) (*openai.Client, error) {
	if cfg.GetApiKey() == "" {
		return nil, errors.New("api key is required")
	}

	clientCfg := openai.DefaultConfig(cfg.GetApiKey())
	if base := cfg.GetBaseUrl(); base != "" {
		clientCfg.BaseURL = strings.TrimSuffix(base, "/")
	}
	return openai.NewClientWithConfig(clientCfg), nil
}

// buildSyncMessages 将 pb.ChatMsg 转换为 openai.ChatCompletionMessage
func buildSyncMessages(msgs []*pb.ChatMsg) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, 0, len(msgs))
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		result = append(result, openai.ChatCompletionMessage{
			Role:    mapSyncRole(msg.Role),
			Content: msg.Content,
		})
	}
	return result
}

// buildSyncChatCompletionRequest 构建 OpenAI 聊天完成请求
func buildSyncChatCompletionRequest(in *pb.ChatReq, messages []openai.ChatCompletionMessage) openai.ChatCompletionRequest {
	cfg := in.GetLlmConfig()
	req := openai.ChatCompletionRequest{
		Model:    cfg.GetModel(),
		Messages: messages,
		Stream:   false,
	}

	if temp := cfg.GetTemperature(); temp > 0 {
		req.Temperature = float32(temp)
	}
	if topP := cfg.GetTopP(); topP > 0 {
		req.TopP = float32(topP)
	}
	if presence := cfg.GetPresencePenalty(); presence != 0 {
		req.PresencePenalty = float32(presence)
	}
	if repetition := cfg.GetRepetitionPenalty(); repetition != 0 {
		req.FrequencyPenalty = float32(repetition)
	}
	if cfg.GetMaxTokens() > 0 {
		req.MaxTokens = int(cfg.GetMaxTokens())
	}
	if cfg.GetResponseFormat() != "" {
		req.ResponseFormat = &openai.ChatCompletionResponseFormat{Type: openai.ChatCompletionResponseFormatType(cfg.GetResponseFormat())}
	}
	if seed := cfg.GetSeed(); seed != 0 {
		s := int(seed)
		req.Seed = &s
	}

	return req
}

// cloneSyncMessages 克隆消息列表
func cloneSyncMessages(msgs []*pb.ChatMsg) []*pb.ChatMsg {
	clones := make([]*pb.ChatMsg, 0, len(msgs))
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		clones = append(clones, &pb.ChatMsg{Role: msg.Role, Content: msg.Content})
	}
	return clones
}

// mapSyncRole 将内部角色映射到 OpenAI 角色
func mapSyncRole(role string) string {
	switch role {
	case chatconsts.ChatMessageRoleAssistant:
		return openai.ChatMessageRoleAssistant
	case chatconsts.ChatMessageRoleSystem:
		return openai.ChatMessageRoleSystem
	case chatconsts.ChatMessageRoleTool:
		return openai.ChatMessageRoleTool
	default:
		return openai.ChatMessageRoleUser
	}
}
