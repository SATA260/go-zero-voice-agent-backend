package logic

import (
	"context"
	"encoding/json"

	"go-zero-voice-agent/app/mqueue/cmd/job/internal/svc"
	"go-zero-voice-agent/app/mqueue/cmd/job/jobtype"

	"github.com/hibiken/asynq"
)

type CronJob struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCronJob(ctx context.Context, svcCtx *svc.ServiceContext) *CronJob {
	return &CronJob{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// register job
func (l *CronJob) Register() *asynq.ServeMux {

	mux := asynq.NewServeMux()
	mux.HandleFunc(jobtype.SyncChatMsgToDb, l.handleSyncChatMsgToDb)

	return mux
}

func (l *CronJob) handleSyncChatMsgToDb(ctx context.Context, task *asynq.Task) error {
	var payload jobtype.SyncChatMsgPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}

	return NewSyncChatMsgToDbLogic(ctx, l.svcCtx).Sync(&payload)
}
