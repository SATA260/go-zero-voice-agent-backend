package model

import (
    "errors"
    "github.com/zeromicro/go-zero/core/stores/sqlx"
)

var ErrNotFound = sqlx.ErrNotFound
var ErrNoRowsUpdate = errors.New("update db no rows change")

var UserPassWordSalt string = "gzva-zhang-salt" //用户密码加盐

var UserAuthTypeSystem string = "system" //平台内部
var UserAuthTypeEmail string = "email"   //微信小程序