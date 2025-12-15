package chatmessage

import (
	"strings"

	"go-zero-voice-agent/app/llm/cmd/api/internal/types"
	"go-zero-voice-agent/app/llm/cmd/rpc/client/chatmessageservice"
)

func toRpcGetChatMessageReq(id int64) *chatmessageservice.GetChatMessageReq {
	return &chatmessageservice.GetChatMessageReq{Id: id}
}

func toRpcDeleteChatMessageReq(id int64) *chatmessageservice.DeleteChatMessageReq {
	return &chatmessageservice.DeleteChatMessageReq{Id: id}
}

func toRpcListChatMessageReq(req *types.ListChatMessageBySessionReq) *chatmessageservice.ListChatMessageReq {
	if req == nil {
		return nil
	}

	return &chatmessageservice.ListChatMessageReq{
		PageQuery: toRpcPageQuery(req.PageQuery),
		Filter: &chatmessageservice.ListChatMessageFilter{
			SessionId: req.SessionId,
		},
	}
}

func toRpcPageQuery(query types.PageQuery) *chatmessageservice.PageQuery {
	return &chatmessageservice.PageQuery{
		Page:     query.Page,
		PageSize: query.PageSize,
		OrderBy:  strings.TrimSpace(query.OrderBy),
	}
}

func toTypesChatMessage(message *chatmessageservice.ChatMessage) types.ChatMessage {
	if message == nil {
		return types.ChatMessage{}
	}

	return types.ChatMessage{
		Id:         message.Id,
		SessionId:  message.SessionId,
		Role:       message.Role,
		Content:    message.Content,
		ToolCalls:  toTypesToolCalls(message.ToolCalls),
		ToolCallId: message.ToolCallId,
		Extra:      message.Extra,
		CreateTime: message.CreateTime,
	}
}

func toTypesToolCalls(toolCalls []*chatmessageservice.ToolCall) []types.ToolCall {
	res := make([]types.ToolCall, 0, len(toolCalls))
	for _, tc := range toolCalls {
		if tc == nil {
			continue
		}
		res = append(res, types.ToolCall{
			Info:   toTypesToolCallInfo(tc.Info),
			Status: tc.Status,
			Result: tc.Result,
			Error:  tc.Error,
		})
	}
	return res
}

func toTypesToolCallInfo(info *chatmessageservice.ToolCallInfo) types.ToolCallInfo {
	if info == nil {
		return types.ToolCallInfo{}
	}
	return types.ToolCallInfo{
		Id:                   info.Id,
		Name:                 info.Name,
		ArgumentsJson:        info.ArgumentsJson,
		Scope:                info.Scope,
		RequiresConfirmation: info.RequiresConfirmation,
	}
}
