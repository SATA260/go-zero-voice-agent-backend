package chatmessage

import (
	"strings"

	"go-zero-voice-agent/app/llm/cmd/api/internal/types"
	"go-zero-voice-agent/app/llm/cmd/rpc/client/chatmessageservice"
)

func toRpcGetChatMessageReq(id int64) *chatmessageservice.GetChatMessageReq {
	return &chatmessageservice.GetChatMessageReq{Id: id}
}

func toRpcDeleteChatMessageReq(id, version int64) *chatmessageservice.DeleteChatMessageReq {
	return &chatmessageservice.DeleteChatMessageReq{Id: id, Version: version}
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
		ConfigId:   message.ConfigId,
		Role:       message.Role,
		Content:    message.Content,
		Extra:      message.Extra,
		Version:    message.Version,
		DelState:   message.DelState,
		CreateTime: message.CreateTime,
		UpdateTime: message.UpdateTime,
		DeleteTime: message.DeleteTime,
	}
}
