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
	ctx         context.Context     
	svcCtx      *svc.ServiceContext 
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

	// 2. 从 Redis 读取该会话的所有消息
	cacheKey := publicconsts.ChatCacheKeyPrefix + payload.ConversationID
	values, err := l.svcCtx.ChatCacheRedis.Lrange(cacheKey, 0, -1)
	if err != nil {
		return errors.Wrapf(err, "read conversation cache failed, key: %s", cacheKey)
	}

	// 3. 如果缓存为空，直接返回
	if len(values) == 0 {
		return nil
	}

	// 4. 反序列化缓存中的每条消息
	messages := make([]*pb.ChatMsg, 0, len(values))
	for idx, raw := range values {
		var msg pb.ChatMsg
		if err := json.Unmarshal([]byte(raw), &msg); err != nil {
			return errors.Wrapf(err, "decode cached message failed, key: %s, index: %d", cacheKey, idx)
		}
		messages = append(messages, &msg)
	}

	// 5. 确保会话 session 存在（没有则新建）
	sessionID, err := l.ensureSession(payload.ConversationID)
	if err != nil {
		return err
	}

	// 6. 查询数据库中已存在的消息数量，避免重复写入
	existingCount, err := l.countExistingMessages(sessionID)
	if err != nil {
		return err
	}

	// 7. 如果缓存消息数量 <= 已有数量，说明无新消息
	if int64(len(messages)) <= existingCount {
		return nil
	}

	// 8. 跳过已存在的消息，持久化新增部分
	newMessages := messages[existingCount:]
    if err := l.persistMessages(sessionID, newMessages); err != nil {
        return errors.Wrapf(err, "failed to persist new messages for session %d", sessionID)
    }

	// // 9. 持久化后清理缓存
	// if _, err := l.svcCtx.ChatCacheRedis.Del(cacheKey); err != nil {
	// 	l.Logger.Errorf("delete cache key failed, key: %s, err: %v", cacheKey, err)
	// }

	// 10. 日志记录同步结果
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
			// 插入每条消息
			if _, err := l.svcCtx.ChatMessageModel.Insert(ctx, session, record); err != nil {
				return errors.Wrapf(err, "insert chat message failed, session_id: %d, index: %d", sessionID, idx)
			}
		}
		return nil
	})
}
