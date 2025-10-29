package logic

import (
	"context"
	"database/sql"
	"encoding/json"

	"go-zero-voice-agent/app/llmservice/cmd/rpc/pb"
	"go-zero-voice-agent/app/llmservice/model"
	publicconsts "go-zero-voice-agent/pkg/consts"
	"go-zero-voice-agent/app/mqueue/cmd/job/internal/svc"
	"go-zero-voice-agent/app/mqueue/cmd/job/jobtype"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

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

func (l *SyncChatMsgToDbLogic) Sync(payload *jobtype.SyncChatMsgPayload) error {
	if payload == nil || payload.ConversationID == "" {
		return nil
	}

	cacheKey := publicconsts.ChatCacheKeyPrefix + payload.ConversationID
	values, err := l.svcCtx.ChatCacheRedis.Lrange(cacheKey, 0, -1)
	if err != nil {
		return errors.Wrapf(err, "read conversation cache failed, key: %s", cacheKey)
	}

	if len(values) == 0 {
		return nil
	}

	messages := make([]*pb.ChatMsg, 0, len(values))
	for idx, raw := range values {
		var msg pb.ChatMsg
		if err := json.Unmarshal([]byte(raw), &msg); err != nil {
			return errors.Wrapf(err, "decode cached message failed, key: %s, index: %d", cacheKey, idx)
		}
		messages = append(messages, &msg)
	}

	sessionID, err := l.ensureSession(payload.ConversationID)
	if err != nil {
		return err
	}

	existingCount, err := l.countExistingMessages(sessionID)
	if err != nil {
		return err
	}

	if int64(len(messages)) <= existingCount {
		if _, err := l.svcCtx.ChatCacheRedis.Del(cacheKey); err != nil {
			l.Logger.Errorf("delete cache key failed, key: %s, err: %v", cacheKey, err)
		}
		return nil
	}

	skip := int(existingCount)
	if skip > len(messages) {
		skip = len(messages)
	}
	if err := l.persistMessages(sessionID, messages[skip:]); err != nil {
		return err
	}

	if _, err := l.svcCtx.ChatCacheRedis.Del(cacheKey); err != nil {
		l.Logger.Errorf("delete cache key failed, key: %s, err: %v", cacheKey, err)
	}

	l.Logger.Infof("sync conversation %s to database, new messages: %d", payload.ConversationID, len(messages)-skip)

	return nil
}

func (l *SyncChatMsgToDbLogic) ensureSession(conversationID string) (int64, error) {
	session, err := l.svcCtx.ChatSessionModel.FindOneByConvId(l.ctx, conversationID)
	if err != nil {
		if err == model.ErrNotFound {
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

func (l *SyncChatMsgToDbLogic) countExistingMessages(sessionID int64) (int64, error) {
	queryBuilder := l.svcCtx.ChatMessageModel.SelectBuilder().Where("session_id = ?", sessionID)
	count, err := l.svcCtx.ChatMessageModel.FindCount(l.ctx, queryBuilder, "id")
	if err != nil {
		return 0, errors.Wrapf(err, "count chat messages failed, session_id: %d", sessionID)
	}
	return count, nil
}

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
			if _, err := l.svcCtx.ChatMessageModel.Insert(ctx, session, record); err != nil {
				return errors.Wrapf(err, "insert chat message failed, session_id: %d, index: %d", sessionID, idx)
			}
		}
		return nil
	})
}
