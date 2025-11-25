package chatmessageservicelogic

import (
	"database/sql"
	"strings"
	"time"

	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/model"
	"go-zero-voice-agent/pkg/tool"
)

func chatMessageToPb(message *model.ChatMessage) *pb.ChatMessage {
	if message == nil {
		return nil
	}

	return &pb.ChatMessage{
		Id:         message.Id,
		SessionId:  message.SessionId,
		ConfigId:   tool.NullInt64ToInt64(message.ConfigId),
		Role:       message.Role,
		Content:    tool.NullStringToString(message.Content),
		Extra:      tool.NullStringToString(message.Extra),
		Version:    message.Version,
		DelState:   message.DelState,
		CreateTime: timeToUnix(message.CreateTime),
		UpdateTime: timeToUnix(message.UpdateTime),
		DeleteTime: nullTimeToUnix(message.DeleteTime),
	}
}

func timeToUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func nullTimeToUnix(t sql.NullTime) int64 {
	if !t.Valid {
		return 0
	}
	return t.Time.Unix()
}

func toNullString(value string) sql.NullString {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: trimmed, Valid: true}
}

func toNullInt64(value int64) sql.NullInt64 {
	if value <= 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: value, Valid: true}
}
