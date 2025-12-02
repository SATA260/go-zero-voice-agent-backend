package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ FileUploadModel = (*customFileUploadModel)(nil)

type (
	// FileUploadModel is an interface to be customized, add more methods here,
	// and implement the added methods in customFileUploadModel.
	FileUploadModel interface {
		fileUploadModel
	}

	customFileUploadModel struct {
		*defaultFileUploadModel
	}
)

// NewFileUploadModel returns a model for the database table.
func NewFileUploadModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) FileUploadModel {
	return &customFileUploadModel{
		defaultFileUploadModel: newFileUploadModel(conn, c, opts...),
	}
}
