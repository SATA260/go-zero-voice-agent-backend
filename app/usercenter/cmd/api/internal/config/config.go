// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.1

package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	JwtAuth struct {
		AccessSecret string
		AccessExpire int64
	}
}
