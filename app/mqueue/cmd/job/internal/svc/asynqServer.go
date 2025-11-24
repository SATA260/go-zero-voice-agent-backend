package svc

import (
	"go-zero-voice-agent/app/mqueue/cmd/job/internal/config"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
)

func newAsynqServer(c config.Config) *asynq.Server {
    return asynq.NewServer(
        asynq.RedisClientOpt{
            Addr:     c.Asynq.Host,
            Password: c.Asynq.Pass,
            DB:       c.Asynq.DB,
        },
        asynq.Config{
            IsFailure: func(err error) bool {
                logx.Errorf("asynq server exec task err: %+v", err)
                return true
            },
            Concurrency: 20,
        },
    )
}
