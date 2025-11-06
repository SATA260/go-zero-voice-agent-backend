package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ChatConfigModel = (*customChatConfigModel)(nil)

type (
	// ChatConfigModel is an interface to be customized, add more methods here,
	// and implement the added methods in customChatConfigModel.
	ChatConfigModel interface {
		chatConfigModel
	}

	customChatConfigModel struct {
		*defaultChatConfigModel
	}
)

// NewChatConfigModel returns a model for the database table.
func NewChatConfigModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) ChatConfigModel {
	return &customChatConfigModel{
		defaultChatConfigModel: newChatConfigModel(conn, c, opts...),
	}
}
