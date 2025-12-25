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

	cacheKey := publicconsts.ChatCacheKeyPrefix + payload.ConversationID
	dirtyKey := cacheKey + ":dirty"

	// 优先处理脏标记：存在表示列表中有更新过的旧消息，需要全量重放
	if dirty, _ := l.svcCtx.RedisClient.Exists(dirtyKey); dirty {
		values, err := l.svcCtx.RedisClient.Lrange(cacheKey, 0, -1)
		if err != nil {
			return errors.Wrapf(err, "read conversation cache failed for dirty resync, key: %s", cacheKey)
		}

		allMessages := make([]*pb.ChatMsg, 0, len(values))
		for idx, raw := range values {
			var msg pb.ChatMsg
			if err := json.Unmarshal([]byte(raw), &msg); err != nil {
				return errors.Wrapf(err, "decode cached message failed (dirty resync), key: %s, index: %d", cacheKey, idx)
			}
			allMessages = append(allMessages, &msg)
		}

		if err := l.persistAll(sessionID, allMessages); err != nil {
			return errors.Wrapf(err, "failed to resync all messages for session %d", sessionID)
		}

		// 清除脏标记
		l.svcCtx.RedisClient.Del(dirtyKey)
		logx.Infof("resynced %d messages for conversation %s due to dirty flag", len(allMessages), payload.ConversationID)
		return nil
	}

	// 正常增量同步
	existingCount, err := l.countExistingMessages(sessionID)
	if err != nil {
		return err
	}
	values, err := l.svcCtx.RedisClient.Lrange(cacheKey, int(existingCount), -1)
	if err != nil {
		return errors.Wrapf(err, "read conversation cache failed, key: %s", cacheKey)
	}

	if len(values) == 0 {
		return nil
	}

	newMessages := make([]*pb.ChatMsg, 0, len(values))
	for idx, raw := range values {
		var msg pb.ChatMsg
		if err := json.Unmarshal([]byte(raw), &msg); err != nil {
			return errors.Wrapf(err, "decode cached message failed, key: %s, index: %d", cacheKey, int(existingCount)+idx)
		}
		newMessages = append(newMessages, &msg)
	}

	if err := l.persistMessages(sessionID, newMessages); err != nil {
		return errors.Wrapf(err, "failed to persist new messages for session %d", sessionID)
	}

	logx.Infof("successfully synced %d new messages for conversation %s", len(newMessages), payload.ConversationID)

	return nil
}

// persistAll 全量重放：先删除该会话已有消息，再按缓存顺序重写
func (l *SyncChatMsgToDbLogic) persistAll(sessionID int64, messages []*pb.ChatMsg) error {
	return l.svcCtx.ChatMessageModel.Trans(l.ctx, func(ctx context.Context, session sqlx.Session) error {
		if _, err := session.ExecCtx(ctx, "delete from chat_message where session_id = ?", sessionID); err != nil {
			return errors.Wrapf(err, "delete existing messages failed, session_id: %d", sessionID)
		}

		for idx, msg := range messages {
			record := &model.ChatMessage{
				SessionId: sessionID,
				Role:      msg.Role,
			}
			if msg.Content != "" {
				record.Content = sql.NullString{String: msg.Content, Valid: true}
			}
			if msg.ToolCallId != "" {
				record.ToolCallId = sql.NullString{String: msg.ToolCallId, Valid: true}
			}
			if len(msg.ToolCalls) > 0 {
				toolCallsBytes, err := json.Marshal(msg.ToolCalls)
				if err != nil {
					return errors.Wrapf(err, "marshal tool calls failed (resync), session_id: %d, index: %d", sessionID, idx)
				}
				record.ToolCalls = sql.NullString{String: string(toolCallsBytes), Valid: true}
			}

			if _, err := l.svcCtx.ChatMessageModel.Insert(ctx, session, record); err != nil {
				return errors.Wrapf(err, "insert chat message failed (resync), session_id: %d, index: %d", sessionID, idx)
			}
		}

		return nil
	})
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
				record.ToolCallId = sql.NullString{String: msg.ToolCallId, Valid: true}
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
