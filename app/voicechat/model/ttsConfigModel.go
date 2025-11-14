package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ TtsConfigModel = (*customTtsConfigModel)(nil)

type (
	// TtsConfigModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTtsConfigModel.
	TtsConfigModel interface {
		ttsConfigModel
	}

	customTtsConfigModel struct {
		*defaultTtsConfigModel
	}
)

// NewTtsConfigModel returns a model for the database table.
func NewTtsConfigModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) TtsConfigModel {
	return &customTtsConfigModel{
		defaultTtsConfigModel: newTtsConfigModel(conn, c, opts...),
	}
}
