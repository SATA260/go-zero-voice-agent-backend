package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ChatSessionModel = (*customChatSessionModel)(nil)

type (
	// ChatSessionModel is an interface to be customized, add more methods here,
	// and implement the added methods in customChatSessionModel.
	ChatSessionModel interface {
		chatSessionModel
	}

	customChatSessionModel struct {
		*defaultChatSessionModel
	}
)

// NewChatSessionModel returns a model for the database table.
func NewChatSessionModel(conn sqlx.SqlConn, c cache.CacheConf) ChatSessionModel {
	return &customChatSessionModel{
		defaultChatSessionModel: newChatSessionModel(conn, c),
	}
}
