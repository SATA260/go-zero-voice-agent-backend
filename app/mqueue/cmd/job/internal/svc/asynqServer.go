package svc

import (
	"go-zero-voice-agent/app/mqueue/cmd/job/internal/config"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
)

func newAsynqServer(c config.Config) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{Addr: c.Redis.Host, Password: c.Redis.Pass},
		asynq.Config{
			IsFailure: func(err error) bool {
				logx.Errorf("asynq server exec task IsFailure ======== >>>>>>>>>>>  err : %+v \n", err)
				return true
			},
			Concurrency: 20, //max concurrent process job task num
		},
	)
}
