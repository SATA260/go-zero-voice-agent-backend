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
	"github.com/google/uuid"
	openai "github.com/sashabaranov/go-openai"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CollectHistory 通用的历史记录收集逻辑
func CollectHistory(ctx context.Context, svcCtx *svc.ServiceContext, log logx.Logger, conversationId string, autoFill bool, config *pb.LlmConfig, sessionId int64) ([]*pb.ChatMsg, error) {
	if conversationId == "" || !autoFill {
		return []*pb.ChatMsg{}, nil
	}

	cacheKey := publicconsts.ChatCacheKeyPrefix + conversationId
	length := int(config.GetContentLength())
	if length <= 0 {
		length = 20
	}

	// 1. 尝试从 Redis 获取
	rawMsgs, err := svcCtx.RedisClient.Lrange(cacheKey, -length, -1)
	if err != nil {
		log.Errorf("failed to get history from redis: %v", err)
	}

	if len(rawMsgs) > 0 {
		messages := make([]*pb.ChatMsg, 0, len(rawMsgs))
		for idx, raw := range rawMsgs {
			var decoded pb.ChatMsg
			if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
				log.Errorf("decode cached message failed, key: %s, index: %d, err: %v", cacheKey, idx, err)
				continue
			}
			messages = append(messages, &decoded)
		}
		if len(messages) > 0 {
			return messages, nil
		}
	}

	// 2. Redis 未命中，从 DB 获取
	log.Infof("no cached history found in redis for key: %s", cacheKey)
	queryBuilder := svcCtx.ChatMessageModel.SelectBuilder().Where(squirrel.Eq{"session_id": sessionId})
	pageMsgs, err := svcCtx.ChatMessageModel.FindPageListByPage(ctx, queryBuilder, 1, int64(length), "id DESC")
	if err != nil {
		log.Errorf("failed to get history from db: %v", err)
		return nil, err
	}

	messages := make([]*pb.ChatMsg, 0, len(pageMsgs))
	for i := len(pageMsgs) - 1; i >= 0; i-- {
		msg := pageMsgs[i]

		var toolcalls []*pb.ToolCall
		if msg.ToolCalls.Valid && msg.ToolCalls.String != "" {
			var tc []*pb.ToolCall
			if err := json.Unmarshal([]byte(msg.ToolCalls.String), &tc); err != nil {
				log.Errorf("decode db tool_calls failed, session_id: %d, msg_id: %d, err: %v", msg.SessionId, msg.Id, err)
			} else {
				toolcalls = tc
			}
		}

		messages = append(messages, &pb.ChatMsg{
			Role:       msg.Role,
			Content:    tool.NullStringToString(msg.Content),
			ToolCalls:  toolcalls,
			ToolCallId: tool.NullStringToString(msg.ToolCallId),
		})
	}

	// 3. 回填 Redis
	if len(messages) > 0 {
		// 异步回填，避免阻塞主流程
		go repopulateCache(svcCtx, log, cacheKey, messages)
	}

	return messages, nil
}

func repopulateCache(svcCtx *svc.ServiceContext, log logx.Logger, cacheKey string, messages []*pb.ChatMsg) {
	cachePayload := make([]string, 0, len(messages))
	for _, message := range messages {
		if message == nil {
			continue
		}
		encoded, _ := json.Marshal(message)
		cachePayload = append(cachePayload, string(encoded))
	}

	if len(cachePayload) > 0 {
		svcCtx.RedisClient.Del(cacheKey)
		values := make([]any, 0, len(cachePayload))
		for _, payload := range cachePayload {
			values = append(values, payload)
		}
		svcCtx.RedisClient.Rpush(cacheKey, values...)
		svcCtx.RedisClient.Expire(cacheKey, chatconsts.ChatCacheExpireSeconds)
	}
}

// NewOpenAIClient 创建 OpenAI 客户端
func NewOpenAIClient(cfg *pb.LlmConfig) (*openai.Client, error) {
	if cfg.GetApiKey() == "" {
		return nil, errors.New("api key is required")
	}
	clientCfg := openai.DefaultConfig(cfg.GetApiKey())
	if base := cfg.GetBaseUrl(); base != "" {
		clientCfg.BaseURL = strings.TrimSuffix(base, "/")
	}
	return openai.NewClientWithConfig(clientCfg), nil
}

// BuildOpenAIMessages 转换消息格式
func BuildOpenAIMessages(msgs []*pb.ChatMsg) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, 0, len(msgs))
	for _, msg := range msgs {
		if msg == nil {
			continue
		}

		openaiMsg := openai.ChatCompletionMessage{
			Role:    MapRoleToOpenAI(msg.Role),
			Content: msg.Content,
		}

		if openaiMsg.Role == openai.ChatMessageRoleTool && msg.ToolCallId != "" {
			openaiMsg.ToolCallID = msg.ToolCallId
		}

		if len(msg.ToolCalls) > 0 {
			openaiToolCalls := make([]openai.ToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				if tc.Info == nil {
					continue
				}
				openaiToolCalls = append(openaiToolCalls, openai.ToolCall{
					ID:   tc.Info.Id,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						tc.Info.Name,
						tc.Info.ArgumentsJson,
					},
				})
			}
			openaiMsg.ToolCalls = openaiToolCalls
		}

		result = append(result, openaiMsg)
	}
	return result
}

// MapRoleToOpenAI 角色映射
func MapRoleToOpenAI(role string) string {
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

// BuildChatCompletionRequest 构建通用请求参数
func BuildChatCompletionRequest(cfg *pb.LlmConfig, messages []openai.ChatCompletionMessage, stream bool, tools []openai.Tool) openai.ChatCompletionRequest {
	req := openai.ChatCompletionRequest{
		Model:    cfg.GetModel(),
		Messages: messages,
		Stream:   stream,
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

	if len(tools) > 0 {
		req.Tools = tools
		req.ToolChoice = "auto"
	}

	return req
}

// GetOrCreateSession 获取或创建会话
func GetOrCreateSession(ctx context.Context, svcCtx *svc.ServiceContext, conversationId string, userId int64, messages []*pb.ChatMsg) (*model.ChatSession, error) {
	var chatSession model.ChatSession

	if conversationId != "" {
		// 如果提供了会话 ID，则查找现有会话
		session, err := svcCtx.ChatSessionModel.FindOneByConvId(ctx, conversationId)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				return nil, status.Errorf(codes.NotFound, "conversation ID %s not found", conversationId)
			}
			return nil, status.Errorf(codes.Internal, "failed to fetch conversation ID %s: %v", conversationId, err)
		}

		// 验证会话归属
		if session.UserId.Int64 != userId {
			return nil, status.Errorf(codes.PermissionDenied, "conversation ID %s does not belong to user %d", conversationId, userId)
		}

		chatSession = *session
	} else {
		// 如果未提供会话 ID，则创建新会话
		// 截取输入消息的前 10 字符作为标题
		title := ""
		if len(messages) > 0 {
			content := messages[0].GetContent()
			if len(content) > 10 {
				title = content[:10]
			} else {
				title = content
			}
		}

		newSession := &model.ChatSession{
			ConvId: uuid.New().String(),
			UserId: sql.NullInt64{Int64: userId, Valid: true},
			Title:  title,
		}
		_, err := svcCtx.ChatSessionModel.Insert(ctx, nil, newSession)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create new conversation: %v", err)
		}
		chatSession = *newSession
	}
	return &chatSession, nil
}
