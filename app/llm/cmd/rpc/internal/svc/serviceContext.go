package svc

import (
	"context"
	"encoding/json"
	"go-zero-voice-agent/app/llm/cmd/rpc/internal/config"
	"go-zero-voice-agent/app/llm/cmd/rpc/internal/toolcall"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/model"
	"go-zero-voice-agent/app/mqueue/cmd/job/jobtype"
	"go-zero-voice-agent/app/rag/cmd/rpc/client/ragservice"
	"go-zero-voice-agent/pkg/uniqueid"
	"time"

	"github.com/hibiken/asynq"
	"github.com/sashabaranov/go-openai"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"

	chatconsts "go-zero-voice-agent/app/llm/pkg/consts"
	publicconsts "go-zero-voice-agent/pkg/consts"
)

type ServiceContext struct {
	Config config.Config

	RedisClient *redis.Redis
	AsynqClient *asynq.Client

	ChatConfigModel  model.ChatConfigModel
	ChatSessionModel model.ChatSessionModel
	ChatMessageModel model.ChatMessageModel

	RagRpc ragservice.RagService

	ToolRegistry                 map[string]toolcall.Tool
	OpenaiToolList               []openai.Tool
	OpenaiToolListWithoutConfirm []openai.Tool
}

func assignMessageID(msg *pb.ChatMsg) {
	if msg == nil {
		return
	}
	if msg.MessageId == 0 {
		msg.MessageId = uniqueid.GenId()
	}
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewMysql(c.DB.DataSource)
	redisClient := redis.New(c.Redis.Host, func(r *redis.Redis) {
		r.Type = c.Redis.Type
		r.Pass = c.Redis.Pass
	})
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     c.Asynq.Host,
		Password: c.Asynq.Pass,
		DB:       c.Asynq.DB,
	})

	ragRpcClient := ragservice.NewRagService(zrpc.MustNewClient(c.RagRpcConf))

	svcCtx := &ServiceContext{
		Config:           c,
		RedisClient:      redisClient,
		AsynqClient:      asynqClient,
		ChatConfigModel:  model.NewChatConfigModel(sqlConn, c.Cache),
		ChatSessionModel: model.NewChatSessionModel(sqlConn, c.Cache),
		ChatMessageModel: model.NewChatMessageModel(sqlConn, c.Cache),
		RagRpc:           ragRpcClient,
	}

	svcCtx.ToolRegistry = newToolRegistry(svcCtx)
	svcCtx.OpenaiToolList = svcCtx.getOpenaiToolList()
	svcCtx.OpenaiToolListWithoutConfirm = svcCtx.getOpenaiToolListWithoutConfirm()

	return svcCtx
}

func newToolRegistry(svcCtx *ServiceContext) map[string]toolcall.Tool {
	registry := make(map[string]toolcall.Tool)

	// ragTool := toolcall.NewRagTool(svcCtx.RagRpc)
	// registry[ragTool.Name()] = ragTool

	timeTool := toolcall.NewTimeTool()
	registry[timeTool.Name()] = timeTool

	weatherTool := toolcall.NewWeatherTool()
	registry[weatherTool.Name()] = weatherTool

	currencyTool := toolcall.NewCurrencyTool()
	registry[currencyTool.Name()] = currencyTool

	// windowsTool := toolcall.NewWindowsTool()
	// registry[windowsTool.Name()] = windowsTool

	return registry
}

func (svc *ServiceContext) getOpenaiToolList() []openai.Tool {
	toolList := make([]openai.Tool, 0, len(svc.ToolRegistry))
	for _, tool := range svc.ToolRegistry {
		toolList = append(toolList, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  tool.ArgumentsJson(),
			},
		})
	}

	return toolList
}

func (svc *ServiceContext) getOpenaiToolListWithoutConfirm() []openai.Tool {
	toolList := make([]openai.Tool, 0, len(svc.ToolRegistry))
	for _, tool := range svc.ToolRegistry {
		if !tool.RequiresConfirmation() && tool.Scope() == chatconsts.TOOL_CALLING_SCOPE_SERVER {
			toolList = append(toolList, openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        tool.Name(),
					Description: tool.Description(),
					Parameters:  tool.ArgumentsJson(),
				},
			})
		}
	}

	return toolList
}

// CacheConversation 缓存对话记录并异步同步到数据库
// 采用增量追加到 Redis + 延迟任务同步数据库的策略
func (svc *ServiceContext) CacheConversation(conversationId string, userMsgs []*pb.ChatMsg, aiRespMsg *pb.ChatMsg) {
	defer func() {
		if r := recover(); r != nil {
			logx.Errorf("panic recovered in cacheConversation, err: %v", r)
		}
	}()

	cacheKey := publicconsts.ChatCacheKeyPrefix + conversationId

	// 1. 构造需要追加的新消息列表（增量更新，保留 Redis 中的历史上下文）
	msgsToAppend := make([]*pb.ChatMsg, 0, len(userMsgs)+1)
	msgsToAppend = append(msgsToAppend, userMsgs...)
	if aiRespMsg != nil {
		msgsToAppend = append(msgsToAppend, aiRespMsg)
	}

	for _, msg := range msgsToAppend {
		assignMessageID(msg)
	}

	for _, msg := range msgsToAppend {
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			logx.Errorf("failed to marshal message, err: %v", err)
			continue
		}

		// 使用 Rpush 将新消息追加到列表末尾
		if _, err = svc.RedisClient.Rpush(cacheKey, string(msgBytes)); err != nil {
			logx.Errorf("fail to push message to Redis, key: %s, err: %v", cacheKey, err)
		}
	}

	// 刷新过期时间，保证活跃会话不丢失
	svc.RedisClient.Expire(cacheKey, chatconsts.ChatCacheExpireSeconds)

	// 2. 异步同步任务 (防抖)
	svc.enqueueSyncChatTask(conversationId)
}

// UpdateAssistantToolCalls 更新缓存中已有的 assistant 消息（按 toolCallId 匹配），避免重复新增消息
func (svc *ServiceContext) UpdateAssistantToolCalls(conversationId string, updatedAssistant *pb.ChatMsg) {
	if updatedAssistant == nil || len(updatedAssistant.ToolCalls) == 0 {
		return
	}
	assignMessageID(updatedAssistant)

	cacheKey := publicconsts.ChatCacheKeyPrefix + conversationId
	updateKey := publicconsts.ChatToolCallUpdateKeyPrefix + conversationId

	// 尝试找到包含相同 toolCallId 的最近一条 assistant 消息并原地更新
	values, err := svc.RedisClient.Lrange(cacheKey, 0, -1)
	if err != nil {
		logx.Errorf("failed to read cached conversation for update, key: %s, err: %v", cacheKey, err)
		return
	}

	targetIdx := int64(-1)
	for idx := len(values) - 1; idx >= 0; idx-- {
		var cached pb.ChatMsg
		if err := json.Unmarshal([]byte(values[idx]), &cached); err != nil {
			logx.Errorf("decode cached message failed during update, key: %s, index: %d, err: %v", cacheKey, idx, err)
			continue
		}
		if cached.GetRole() != chatconsts.ChatMessageRoleAssistant || len(cached.ToolCalls) == 0 {
			continue
		}
		if cached.MessageId == updatedAssistant.MessageId {
			targetIdx = int64(idx)
			break
		}
		// 只更新最近一条 assistant 消息
		break
	}

	payload, err := json.Marshal(updatedAssistant)
	if err != nil {
		logx.Errorf("marshal updated assistant failed, conv: %s, err: %v", conversationId, err)
		return
	}

	if targetIdx >= 0 {
		// go-zero redis 没有 LSET 封装，使用 Lua 调用原生命令
		script := "return redis.call('LSET', KEYS[1], ARGV[1], ARGV[2])"
		if _, err := svc.RedisClient.EvalCtx(context.Background(), script, []string{cacheKey}, targetIdx, string(payload)); err != nil {
			logx.Errorf("failed to update cached assistant at index %d, key: %s, err: %v", targetIdx, cacheKey, err)
			return
		}
	} else {
		// 没找到则降级为追加，避免状态丢失
		if _, err := svc.RedisClient.Rpush(cacheKey, string(payload)); err != nil {
			logx.Errorf("failed to append assistant message as fallback, key: %s, err: %v", cacheKey, err)
			return
		}
	}

	// 刷新过期时间并单独记录 toolCalls 更新，便于后续增量更新 DB
	svc.RedisClient.Expire(cacheKey, chatconsts.ChatCacheExpireSeconds)
	if err := svc.RedisClient.Setex(updateKey, string(payload), chatconsts.ChatCacheExpireSeconds); err != nil {
		logx.Errorf("failed to cache toolcall update, key: %s, err: %v", updateKey, err)
	}

	svc.enqueueSyncChatTask(conversationId)
}

func (svc *ServiceContext) enqueueSyncChatTask(conversationId string) {
	task, err := jobtype.NewSyncChatMsgTask(conversationId)
	if err != nil {
		logx.Errorf("failed to create sync task for conversation %s, err: %v", conversationId, err)
		return
	}

	// 使用 TaskID 确保同一会话在短时间内只有一个同步任务在队列中
	// 延迟 5 秒执行，让 Redis 积累几条消息后，由消费者一次性批量同步到 MySQL
	taskID := "sync:chat:" + conversationId
	if _, err = svc.AsynqClient.Enqueue(
		task,
		asynq.TaskID(taskID),
		asynq.ProcessIn(5*time.Second),
	); err != nil && err != asynq.ErrTaskIDConflict {
		logx.Infof("failed to enqueue sync task for conversation %s, err: %v", conversationId, err)
	}
}
