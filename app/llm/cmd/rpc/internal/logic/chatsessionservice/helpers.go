package chatsessionservicelogic

import (
	"database/sql"
	"strings"
	"time"

	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/model"
	"go-zero-voice-agent/pkg/tool"

	"github.com/google/uuid"
)

func chatSessionToPb(session *model.ChatSession) *pb.ChatSession {
	if session == nil {
		return nil
	}

	return &pb.ChatSession{
		Id:         session.Id,
		ConvId:     session.ConvId,
		UserId:     tool.NullInt64ToInt64(session.UserId),
		Title:      session.Title,
		Version:    session.Version,
		DelState:   session.DelState,
		CreateTime: timeToUnix(session.CreateTime),
		UpdateTime: timeToUnix(session.UpdateTime),
		DeleteTime: nullTimeToUnix(session.DeleteTime),
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

func toNullInt64(value int64) sql.NullInt64 {
	if value <= 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: value, Valid: true}
}

func normalizeConvID(convID string) string {
	convID = strings.TrimSpace(convID)
	if convID == "" {
		return generateConversationID()
	}
	return convID
}

func generateConversationID() string {
	return "conv-" + strings.ReplaceAll(uuid.NewString(), "-", "")
}
