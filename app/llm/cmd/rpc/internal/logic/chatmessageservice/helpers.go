package chatmessageservicelogic

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/model"
	"go-zero-voice-agent/pkg/tool"

	"github.com/zeromicro/go-zero/core/logx"
)

func chatMessageToPb(message *model.ChatMessage) *pb.ChatMessage {
	if message == nil {
		return nil
	}

	return &pb.ChatMessage{
		Id:         message.Id,
		SessionId:  message.SessionId,
		Role:       message.Role,
		Content:    tool.NullStringToString(message.Content),
		Extra:      tool.NullStringToString(message.Extra),
		ToolCalls:  toolCallsToPb(message.ToolCalls),
		ToolCallId: tool.NullStringToString(message.ToolCallId),
		CreateTime: timeToUnix(message.CreateTime),
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

func toolCallsToModel(toolCalls []*pb.ToolCall) sql.NullString {
	if len(toolCalls) == 0 {
		return sql.NullString{}
	}
	bytes, err := json.Marshal(toolCalls)
	if err != nil {
		logx.Errorf("marshal tool calls failed: %v", err)
		return sql.NullString{}
	}
	return sql.NullString{String: string(bytes), Valid: true}
}

func toolCallsToPb(ns sql.NullString) []*pb.ToolCall {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	var toolCalls []*pb.ToolCall
	if err := json.Unmarshal([]byte(ns.String), &toolCalls); err != nil {
		logx.Errorf("unmarshal tool calls failed: %v, data: %s", err, ns.String)
		return nil
	}
	return toolCalls
}
