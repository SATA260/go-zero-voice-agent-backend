package model

import (
    "errors"
    "github.com/zeromicro/go-zero/core/stores/sqlx"
)

var ErrNotFound = sqlx.ErrNotFound
var ErrNoRowsUpdate = errors.New("update db no rows change")

var UserAuthTypeEmail = "email"
var UserPassWordSalt = "go-zero-voice-agent-usercenter-salt"
var UserAuthTypeSystem = "system"