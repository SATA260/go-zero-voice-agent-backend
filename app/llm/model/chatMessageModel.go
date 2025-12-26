package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	chatconsts "go-zero-voice-agent/app/llm/pkg/consts"
	"go-zero-voice-agent/pkg/globalkey"

	"github.com/pkg/errors"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ChatMessageModel = (*customChatMessageModel)(nil)

type (
	// ChatMessageModel is an interface to be customized, add more methods here,
	// and implement the added methods in customChatMessageModel.
	ChatMessageModel interface {
		chatMessageModel
		InsertWithId(ctx context.Context, session sqlx.Session, data *ChatMessage) (sql.Result, error)
		UpdateToolCallsById(ctx context.Context, messageID int64, toolCalls string) error
		UpdateLastAssistantToolCalls(ctx context.Context, sessionID int64, toolCalls string) error
	}

	customChatMessageModel struct {
		*defaultChatMessageModel
	}
)

// NewChatMessageModel returns a model for the database table.
func NewChatMessageModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) ChatMessageModel {
	return &customChatMessageModel{
		defaultChatMessageModel: newChatMessageModel(conn, c, opts...),
	}
}

// InsertWithId inserts a chat message with a provided snowflake ID.
func (m *customChatMessageModel) InsertWithId(ctx context.Context, session sqlx.Session, data *ChatMessage) (sql.Result, error) {
	if data == nil || data.Id == 0 {
		return nil, errors.New("missing message id for insert")
	}

	data.DelState = globalkey.DelStateNo
	gzvaLlmserviceChatMessageIdKey := fmt.Sprintf("%s%v", cacheGzvaLlmserviceChatMessageIdPrefix, data.Id)

	return m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		query := fmt.Sprintf("insert into %s (id, del_state, version, session_id, role, content, tool_calls, tool_call_id, extra) values (?, ?, ?, ?, ?, ?, ?, ?, ?)", m.table)
		if session != nil {
			return session.ExecCtx(ctx, query, data.Id, data.DelState, data.Version, data.SessionId, data.Role, data.Content, data.ToolCalls, data.ToolCallId, data.Extra)
		}
		return conn.ExecCtx(ctx, query, data.Id, data.DelState, data.Version, data.SessionId, data.Role, data.Content, data.ToolCalls, data.ToolCallId, data.Extra)
	}, gzvaLlmserviceChatMessageIdKey)
}

// UpdateLastAssistantToolCalls updates the most recent assistant message's tool_calls for a session.
// If no assistant message exists, it is treated as a no-op.
func (m *customChatMessageModel) UpdateLastAssistantToolCalls(ctx context.Context, sessionID int64, toolCalls string) error {
	var msgID int64
	query := fmt.Sprintf("select id from %s where session_id = ? and role = ? and del_state = ? order by id desc limit 1", m.table)
	if err := m.QueryRowNoCacheCtx(ctx, &msgID, query, sessionID, chatconsts.ChatMessageRoleAssistant, globalkey.DelStateNo); err != nil {
		if err == sqlc.ErrNotFound {
			return nil
		}
		return errors.Wrapf(err, "find last assistant message failed, session_id: %d", sessionID)
	}

	gzvaLlmserviceChatMessageIdKey := fmt.Sprintf("%s%v", cacheGzvaLlmserviceChatMessageIdPrefix, msgID)
	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		updateQuery := fmt.Sprintf("update %s set tool_calls = ?, update_time = ? where `id` = ?", m.table)
		return conn.ExecCtx(ctx, updateQuery, toolCalls, time.Now(), msgID)
	}, gzvaLlmserviceChatMessageIdKey)
	if err != nil {
		return errors.Wrapf(err, "update last assistant tool calls failed, session_id: %d, message_id: %d", sessionID, msgID)
	}

	return nil
}

// UpdateToolCallsById updates tool_calls for a specific message id.
func (m *customChatMessageModel) UpdateToolCallsById(ctx context.Context, messageID int64, toolCalls string) error {
	if messageID == 0 {
		return errors.New("message id is required")
	}
	gzvaLlmserviceChatMessageIdKey := fmt.Sprintf("%s%v", cacheGzvaLlmserviceChatMessageIdPrefix, messageID)
	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		updateQuery := fmt.Sprintf("update %s set tool_calls = ?, update_time = ? where `id` = ?", m.table)
		return conn.ExecCtx(ctx, updateQuery, toolCalls, time.Now(), messageID)
	}, gzvaLlmserviceChatMessageIdKey)
	if err != nil {
		return errors.Wrapf(err, "update tool calls by id failed, message_id: %d", messageID)
	}
	return nil
}
