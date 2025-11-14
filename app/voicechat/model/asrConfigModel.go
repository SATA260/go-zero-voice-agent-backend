package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AsrConfigModel = (*customAsrConfigModel)(nil)

type (
	// AsrConfigModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAsrConfigModel.
	AsrConfigModel interface {
		asrConfigModel
	}

	customAsrConfigModel struct {
		*defaultAsrConfigModel
	}
)

// NewAsrConfigModel returns a model for the database table.
func NewAsrConfigModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AsrConfigModel {
	return &customAsrConfigModel{
		defaultAsrConfigModel: newAsrConfigModel(conn, c, opts...),
	}
}
