package llmchatservicelogic

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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

type ChatStreamLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// ChatStreamLogic 负责处理 LLM Chat 服务的流式响应
func NewChatStreamLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatStreamLogic {
	return &ChatStreamLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ChatStreamLogic) ChatStream(in *pb.ChatStreamReq, stream pb.LlmChatService_ChatStreamServer) error {
	// 基础参数校验，避免无效请求继续执行
	if in == nil || in.LlmConfig == nil {
		return status.Error(codes.InvalidArgument, "missing llm config")
	}
	if in.LlmConfig.Model == "" {
		return status.Error(codes.InvalidArgument, "model is required")
	}

	ctx := stream.Context()
	if err := ctx.Err(); err != nil {
		return err
	}

	// 如果需要继续会话，则补齐历史上下文，优先读缓存回退数据库
	historyMsgs, err := l.collectHistory(ctx, in)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	if len(in.Messages) > 0 {
		historyMsgs = append(historyMsgs, in.Messages...)
	}

	if len(historyMsgs) == 0 {
		return status.Error(codes.InvalidArgument, "no messages to process")
	}

	// 构建 OpenAI SDK 需要的消息列表
	openaiMsgs := buildOpenAIMessages(historyMsgs)

	client, err := l.newOpenAIClient(in.LlmConfig)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	req := buildChatCompletionRequest(in, openaiMsgs)

	// 发起与模型的流式会话，请求期间逐条中转给客户端
	chatStream, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return status.Errorf(codes.Internal, "create chat stream failed: %v", err)
	}
	defer chatStream.Close()

	conversationID := in.ConversationId
	var assistantBuilder strings.Builder
	toolStates := newToolState()

	var contentCompleted bool
	var usageSent bool

	for {
		// 支持上下文取消，及时结束后台协程
		if err := ctx.Err(); err != nil {
			return err
		}

		resp, err := chatStream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "stream recv failed: %v", err)
		}

		if conversationID == "" && resp.ID != "" {
			conversationID = resp.ID
		}

		if len(resp.Choices) == 0 {
			// 某些实现会单独推送用量信息
			if !usageSent && resp.Usage != nil {
				if err := stream.Send(&pb.ChatStreamResp{
					Id:      conversationID,
					Payload: &pb.ChatStreamResp_Usage{Usage: buildUsage(resp.Usage)},
				}); err != nil {
					return err
				}
				usageSent = true
			}
			continue
		}

		choice := resp.Choices[0]
		delta := choice.Delta

		if delta.Content != "" {
			// 即时下发文本增量，提升前端体验
			assistantBuilder.WriteString(delta.Content)
			if err := stream.Send(&pb.ChatStreamResp{
				Id:      conversationID,
				Payload: &pb.ChatStreamResp_Delta{Delta: &pb.StreamDelta{Content: delta.Content}},
			}); err != nil {
				return err
			}
		}

		if len(delta.ToolCalls) > 0 {
			// 按照工具调用的增量参数输出模板，等待调用方执行
			events := toolStates.Update(delta.ToolCalls)
			for _, ev := range events {
				if err := stream.Send(&pb.ChatStreamResp{
					Id:      conversationID,
					Payload: &pb.ChatStreamResp_ToolCall{ToolCall: ev},
				}); err != nil {
					return err
				}
			}
		}

		switch choice.FinishReason {
		case openai.FinishReasonStop:
			if !contentCompleted {
				// 正文生成结束时推送 completed 标记
				if err := stream.Send(&pb.ChatStreamResp{
					Id:      conversationID,
					Payload: &pb.ChatStreamResp_Delta{Delta: &pb.StreamDelta{Completed: true}},
				}); err != nil {
					return err
				}
				contentCompleted = true
			}
		case openai.FinishReasonToolCalls:
			// 工具调用完成后输出最终参数
			completed := toolStates.CompleteAll()
			for _, ev := range completed {
				if err := stream.Send(&pb.ChatStreamResp{
					Id:      conversationID,
					Payload: &pb.ChatStreamResp_ToolCall{ToolCall: ev},
				}); err != nil {
					return err
				}
			}
		}

		if !usageSent && resp.Usage != nil {
			if err := stream.Send(&pb.ChatStreamResp{
				Id:      conversationID,
				Payload: &pb.ChatStreamResp_Usage{Usage: buildUsage(resp.Usage)},
			}); err != nil {
				return err
			}
			usageSent = true
		}
	}

	if !contentCompleted {
		// 模型被动结束流时，也需要补发 completed
		if err := stream.Send(&pb.ChatStreamResp{
			Id:      conversationID,
			Payload: &pb.ChatStreamResp_Delta{Delta: &pb.StreamDelta{Completed: true}},
		}); err != nil {
			return err
		}
	}

	// 确保所有工具调用最终都有一次 completed 状态
	pending := toolStates.CompleteAll()
	for _, ev := range pending {
		if err := stream.Send(&pb.ChatStreamResp{
			Id:      conversationID,
			Payload: &pb.ChatStreamResp_ToolCall{ToolCall: ev},
		}); err != nil {
			return err
		}
	}

	if conversationID == "" {
		conversationID = generateConversationID()
	}

	assistantMsg := &pb.ChatMsg{
		Role:    chatconsts.ChatMessageRoleAssistant,
		Content: assistantBuilder.String(),
	}

	// 异步写缓存与任务队列，避免阻塞流式返回
	historySnapshot := cloneMessages(historyMsgs)
	go l.svcCtx.CacheConversation(conversationID, historySnapshot, assistantMsg)

	return nil
}

func (l *ChatStreamLogic) collectHistory(ctx context.Context, in *pb.ChatStreamReq) ([]*pb.ChatMsg, error) {
	// 若已有会话 ID 且未显式追加消息，默认补充一次历史上下文
	autoFill := in.AutoFillHistory
	if !autoFill && in.ConversationId != "" && len(in.Messages) == 0 {
		autoFill = true
	}

	if in.ConversationId == "" || !autoFill {
		return []*pb.ChatMsg{}, nil
	}

	cacheKey := publicconsts.ChatCacheKeyPrefix + in.ConversationId
	length := int(in.LlmConfig.ContentLength)
	if length <= 0 {
		length = 50
	}

	// 优先读取 Redis 中的最近消息
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

	// 缓存命中失败后，回退到数据库 session 信息
	session, err := l.svcCtx.ChatSessionModel.FindOneByConvId(ctx, in.ConversationId)
	if err != nil {
		if err == model.ErrNotFound {
			return []*pb.ChatMsg{}, nil
		}
		return nil, err
	}

	queryBuilder := l.svcCtx.ChatMessageModel.SelectBuilder().Where(squirrel.Eq{"session_id": session.Id})
	pageMsgs, err := l.svcCtx.ChatMessageModel.FindPageListByPage(ctx, queryBuilder, 1, int64(length), "id DESC")
	if err != nil {
		return nil, err
	}

	messages := make([]*pb.ChatMsg, 0, len(pageMsgs))
	for i := len(pageMsgs) - 1; i >= 0; i-- {
		msg := pageMsgs[i]
		messages = append(messages, &pb.ChatMsg{
			Role:    msg.Role,
			Content: tool.NullStringToString(msg.Content),
		})
	}
	return messages, nil
}

func (l *ChatStreamLogic) newOpenAIClient(config *pb.LlmConfig) (*openai.Client, error) {
	if config.ApiKey == "" {
		return nil, errors.New("api key is required")
	}

	cfg := openai.DefaultConfig(config.ApiKey)
	if config.BaseUrl != "" {
		cfg.BaseURL = strings.TrimSuffix(config.BaseUrl, "/")
	}
	return openai.NewClientWithConfig(cfg), nil
}

func buildOpenAIMessages(msgs []*pb.ChatMsg) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, 0, len(msgs))
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		role := convertRole(msg.Role)
		result = append(result, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg.Content,
		})
	}
	return result
}

func convertRole(role string) string {
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

func buildChatCompletionRequest(in *pb.ChatStreamReq, messages []openai.ChatCompletionMessage) openai.ChatCompletionRequest {
	cfg := in.GetLlmConfig()
	req := openai.ChatCompletionRequest{
		Model:    cfg.GetModel(),
		Messages: messages,
		Stream:   true,
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

	if len(in.Tools) > 0 {
		if tools, err := convertTools(in.Tools); err == nil {
			req.Tools = tools
		}
	}

	return req
}

func convertTools(pbTools []*pb.Tool) ([]openai.Tool, error) {
	result := make([]openai.Tool, 0, len(pbTools))
	for _, t := range pbTools {
		if t == nil || t.Function == nil {
			continue
		}

		toolType := openai.ToolTypeFunction
		if t.Type != "" {
			toolType = openai.ToolType(t.Type)
		}

		var params json.RawMessage
		if t.Function.Parameters != nil {
			// Proto Struct 转 JSON Schema，便于透传给 OpenAI
			raw, err := json.Marshal(t.Function.Parameters.AsMap())
			if err != nil {
				return nil, err
			}
			params = raw
		}

		result = append(result, openai.Tool{
			Type: toolType,
			Function: &openai.FunctionDefinition{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  params,
			},
		})
	}
	return result, nil
}

func buildUsage(usage *openai.Usage) *pb.UsageData {
	if usage == nil {
		return nil
	}
	return &pb.UsageData{
		PromptTokens:     int64(usage.PromptTokens),
		CompletionTokens: int64(usage.CompletionTokens),
		TotalTokens:      int64(usage.TotalTokens),
	}
}

func cloneMessages(msgs []*pb.ChatMsg) []*pb.ChatMsg {
	clones := make([]*pb.ChatMsg, 0, len(msgs))
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		clones = append(clones, &pb.ChatMsg{Role: msg.Role, Content: msg.Content})
	}
	return clones
}

type toolCallState struct {
	id        string
	name      string
	arguments strings.Builder
	completed bool
}

type toolStateTracker struct {
	calls map[int]*toolCallState
}

func newToolState() *toolStateTracker {
	return &toolStateTracker{calls: make(map[int]*toolCallState)}
}

func (t *toolStateTracker) Update(updates []openai.ToolCall) []*pb.ToolCallDelta {
	events := make([]*pb.ToolCallDelta, 0, len(updates))
	for _, update := range updates {
		index := 0
		if update.Index != nil {
			index = *update.Index
		}

		state := t.calls[index]
		if state == nil {
			state = &toolCallState{}
			t.calls[index] = state
		}
		if update.ID != "" {
			state.id = update.ID
		}
		if update.Function.Name != "" {
			state.name = update.Function.Name
		}
		if update.Function.Arguments != "" {
			// OpenAI 会分片返回参数，需要自行拼接成完整 JSON
			state.arguments.WriteString(update.Function.Arguments)
		}

		events = append(events, &pb.ToolCallDelta{
			Id:            state.id,
			Name:          state.name,
			ArgumentsJson: state.arguments.String(),
			Completed:     false,
			Status:        "pending",
		})
	}
	return events
}

func (t *toolStateTracker) CompleteAll() []*pb.ToolCallDelta {
	events := make([]*pb.ToolCallDelta, 0, len(t.calls))
	for _, state := range t.calls {
		if state.completed {
			continue
		}
		state.completed = true
		events = append(events, &pb.ToolCallDelta{
			Id:            state.id,
			Name:          state.name,
			ArgumentsJson: state.arguments.String(),
			Completed:     true,
			Status:        "requires_execution",
		})
	}
	return events
}

func generateConversationID() string {
	return "conv-" + strings.ReplaceAll(uuid.NewString(), "-", "")
}
