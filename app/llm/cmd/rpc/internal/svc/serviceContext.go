package svc

import (
	"encoding/json"
	"go-zero-voice-agent/app/llm/cmd/rpc/internal/config"
	"go-zero-voice-agent/app/llm/cmd/rpc/internal/toolcall"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/model"
	"go-zero-voice-agent/app/mqueue/cmd/job/jobtype"
	"go-zero-voice-agent/app/rag/cmd/rpc/client/ragservice"
	"time"

	"github.com/hibiken/asynq"
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

	ToolRegistry map[string]toolcall.Tool
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

	return svcCtx
}

func newToolRegistry(svcCtx *ServiceContext) map[string]toolcall.Tool {
	registry := make(map[string]toolcall.Tool)

	ragTool := toolcall.NewRagTool(svcCtx.RagRpc)
	registry[ragTool.Name()] = ragTool

	timeTool := toolcall.NewTimeTool()
	registry[timeTool.Name()] = timeTool

	emailCfg := svcCtx.Config.Toolcall.Email
	if emailCfg.Host != "" && emailCfg.Port != "" && emailCfg.Username != "" && emailCfg.Password != "" {
		emailTool := toolcall.NewEmailTool(emailCfg.Host, emailCfg.Port, emailCfg.Username, emailCfg.Password)
		registry[emailTool.Name()] = emailTool
	} else {
		logx.Infof("email tool is not registered because SMTP config is incomplete")
	}

	return registry
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
	); err != nil {
		// 如果错误是 TaskID 冲突，说明已有任务在排队，这是预期的，忽略即可
		if err != asynq.ErrTaskIDConflict {
			logx.Errorf("failed to enqueue sync task for conversation %s, err: %v", conversationId, err)
		}
	}
}
