package jobtype

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

const (
	QueueDefault    = "default"
	SyncChatMsgToDb = "task:chat:msg:sync_to_db"
)

type SyncChatMsgPayload struct {
	ConversationID string `json:"conversation_id"`
}

func NewSyncChatMsgTask(conversationID string) (*asynq.Task, error) {
	payload, err := json.Marshal(SyncChatMsgPayload{ConversationID: conversationID})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(SyncChatMsgToDb, payload), nil
}
