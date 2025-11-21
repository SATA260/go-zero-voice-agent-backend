package llmchatservicelogic

import (
	"context"
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

type ChatLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatLogic {
	return &ChatLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ChatLogic) Chat(in *pb.ChatReq) (*pb.ChatResp, error) {
	if in == nil || in.LlmConfig == nil {
		return nil, status.Error(codes.InvalidArgument, "missing llm config")
	}
	if in.LlmConfig.Model == "" {
		return nil, status.Error(codes.InvalidArgument, "model is required")
	}

	if err := l.ctx.Err(); err != nil {
		return nil, err
	}

	historyMsgs, err := l.collectHistory(in)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if len(in.Messages) > 0 {
		historyMsgs = append(historyMsgs, in.Messages...)
	}

	if len(historyMsgs) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no messages to process")
	}

	openaiMsgs := buildSyncMessages(historyMsgs)

	client, err := l.newSyncOpenAIClient(in.LlmConfig)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	req := buildSyncChatCompletionRequest(in, openaiMsgs)

	completion, err := client.CreateChatCompletion(l.ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create chat completion failed: %v", err)
	}
	if len(completion.Choices) == 0 {
		return nil, status.Error(codes.Internal, "empty response from llm")
	}

	choice := completion.Choices[0]
	if len(choice.Message.ToolCalls) > 0 {
		return nil, status.Error(codes.Unimplemented, "tool calls are not supported in sync chat mode")
	}

	conversationID := in.GetConversationId()
	if conversationID == "" {
		conversationID = completion.ID
	}
	if conversationID == "" {
		conversationID = generateConversationID()
	}

	assistantMsg := &pb.ChatMsg{
		Role:    chatconsts.ChatMessageRoleAssistant,
		Content: choice.Message.Content,
	}

	historySnapshot := cloneSyncMessages(historyMsgs)
	go l.svcCtx.CacheConversation(conversationID, historySnapshot, assistantMsg)

	respMsgs := make([]*pb.ChatMsg, 0)
	respMsgs = append(respMsgs, assistantMsg)

	return &pb.ChatResp{
		Id:      conversationID,
		RespMsg: respMsgs,
	}, nil
}

func (l *ChatLogic) collectHistory(in *pb.ChatReq) ([]*pb.ChatMsg, error) {
	autoFill := in.GetAutoFillHistory()
	if !autoFill && in.GetConversationId() != "" && len(in.GetMessages()) == 0 {
		autoFill = true
	}

	if in.GetConversationId() == "" || !autoFill {
		return []*pb.ChatMsg{}, nil
	}

	cacheKey := publicconsts.ChatCacheKeyPrefix + in.GetConversationId()
	length := int(in.GetLlmConfig().GetContentLength())
	if length <= 0 {
		length = 50
	}

	rawMsgs, err := l.svcCtx.RedisClient.Lrange(cacheKey, -length, -1)
	if err == nil {
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

	session, err := l.svcCtx.ChatSessionModel.FindOneByConvId(l.ctx, in.GetConversationId())
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return []*pb.ChatMsg{}, nil
		}
		return nil, err
	}

	queryBuilder := l.svcCtx.ChatMessageModel.SelectBuilder().Where(squirrel.Eq{"session_id": session.Id})
	pageMsgs, err := l.svcCtx.ChatMessageModel.FindPageListByPage(l.ctx, queryBuilder, 1, int64(length), "id DESC")
	if err != nil {
		return nil, err
	}

	messages := make([]*pb.ChatMsg, 0, len(pageMsgs))
	for i := len(pageMsgs) - 1; i >= 0; i-- {
		msg := pageMsgs[i]
		messages = append(messages, &pb.ChatMsg{Role: msg.Role, Content: tool.NullStringToString(msg.Content)})
	}
	return messages, nil
}

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
