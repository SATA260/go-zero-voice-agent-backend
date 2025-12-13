package logic

import (
	"context"
	"database/sql"
	"encoding/json"

	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/model"
	"go-zero-voice-agent/app/mqueue/cmd/job/internal/svc"
	"go-zero-voice-agent/app/mqueue/cmd/job/jobtype"
	publicconsts "go-zero-voice-agent/pkg/consts"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// SyncChatMsgToDbLogic 用于将缓存中的 chat 消息同步到数据库
type SyncChatMsgToDbLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSyncChatMsgToDbLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SyncChatMsgToDbLogic {
	return &SyncChatMsgToDbLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Sync 执行同步逻辑：将 Redis 缓存中的 chat 消息批量写入数据库
func (l *SyncChatMsgToDbLogic) Sync(payload *jobtype.SyncChatMsgPayload) error {
	// 1. 校验任务载荷
	if payload == nil || payload.ConversationID == "" {
		return nil
	}

	// 2. 确保会话 session 存在（没有则新建）
	// 调整顺序：先拿到 sessionID，以便查询 DB 中的消息数量
	sessionID, err := l.ensureSession(payload.ConversationID)
	if err != nil {
		return err
	}

	// 3. 查询数据库中已存在的消息数量
	existingCount, err := l.countExistingMessages(sessionID)
	if err != nil {
		return err
	}

	// 4. 只从 Redis 读取新增的消息 (Offset = existingCount)
	// 优化：避免拉取全量数据，只拉取 DB 中没有的部分
	cacheKey := publicconsts.ChatCacheKeyPrefix + payload.ConversationID
	values, err := l.svcCtx.RedisClient.Lrange(cacheKey, int(existingCount), -1)
	if err != nil {
		return errors.Wrapf(err, "read conversation cache failed, key: %s", cacheKey)
	}

	// 5. 如果没有新消息，直接返回
	if len(values) == 0 {
		return nil
	}

	// 6. 反序列化新增消息
	newMessages := make([]*pb.ChatMsg, 0, len(values))
	for idx, raw := range values {
		var msg pb.ChatMsg
		if err := json.Unmarshal([]byte(raw), &msg); err != nil {
			return errors.Wrapf(err, "decode cached message failed, key: %s, index: %d", cacheKey, int(existingCount)+idx)
		}
		newMessages = append(newMessages, &msg)
	}

	// 7. 批量持久化新增消息
	if err := l.persistMessages(sessionID, newMessages); err != nil {
		return errors.Wrapf(err, "failed to persist new messages for session %d", sessionID)
	}

	// 8. 日志记录同步结果
	logx.Infof("successfully synced %d new messages for conversation %s", len(newMessages), payload.ConversationID)

	return nil
}

// ensureSession 确保数据库中有该会话，没有则新建，返回 sessionID
func (l *SyncChatMsgToDbLogic) ensureSession(conversationID string) (int64, error) {
	session, err := l.svcCtx.ChatSessionModel.FindOneByConvId(l.ctx, conversationID)
	if err != nil {
		if err == model.ErrNotFound {
			// 新建会话
			record := &model.ChatSession{ConvId: conversationID}
			result, err := l.svcCtx.ChatSessionModel.Insert(l.ctx, nil, record)
			if err != nil {
				return 0, errors.Wrapf(err, "create chat session failed, conv_id: %s", conversationID)
			}
			id, err := result.LastInsertId()
			if err != nil {
				return 0, errors.Wrap(err, "obtain chat session id failed")
			}
			record.Id = id
			return record.Id, nil
		}
		return 0, errors.Wrapf(err, "query chat session failed, conv_id: %s", conversationID)
	}

	return session.Id, nil
}

// countExistingMessages 查询数据库中该会话已存在的消息数量
func (l *SyncChatMsgToDbLogic) countExistingMessages(sessionID int64) (int64, error) {
	queryBuilder := l.svcCtx.ChatMessageModel.SelectBuilder().Where("session_id = ?", sessionID)
	count, err := l.svcCtx.ChatMessageModel.FindCount(l.ctx, queryBuilder, "id")
	if err != nil {
		return 0, errors.Wrapf(err, "count chat messages failed, session_id: %d", sessionID)
	}
	return count, nil
}

// persistMessages 批量持久化新增的消息，使用事务保证一致性
func (l *SyncChatMsgToDbLogic) persistMessages(sessionID int64, messages []*pb.ChatMsg) error {
	if len(messages) == 0 {
		return nil
	}

	return l.svcCtx.ChatMessageModel.Trans(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		for idx, msg := range messages {
			record := &model.ChatMessage{
				SessionId: sessionID,
				Role:      msg.Role,
			}
			if msg.Content != "" {
				record.Content = sql.NullString{String: msg.Content, Valid: true}
			}
			if msg.ToolCallId != "" {
				record.Content = sql.NullString{String: msg.ToolCallId, Valid: true}
			}
			if len(msg.ToolCalls) > 0 {
				toolCallsBytes, err := json.Marshal(msg.ToolCalls)
				if err != nil {
					return errors.Wrapf(err, "marshal tool calls failed, session_id: %d, index: %d", sessionID, idx)
				}
				record.ToolCalls = sql.NullString{String: string(toolCallsBytes), Valid: true}
			}
			// 插入每条消息
			if _, err := l.svcCtx.ChatMessageModel.Insert(ctx, session, record); err != nil {
				return errors.Wrapf(err, "insert chat message failed, session_id: %d, index: %d", sessionID, idx)
			}
		}
		return nil
	})
}
