package chatsession

import (
	"strings"

	"go-zero-voice-agent/app/llm/cmd/api/internal/types"
	"go-zero-voice-agent/app/llm/cmd/rpc/client/chatsessionservice"
)

func toRpcGetChatSessionReq(id int64) *chatsessionservice.GetChatSessionReq {
	return &chatsessionservice.GetChatSessionReq{Id: id}
}

func toRpcDeleteChatSessionReq(id int64) *chatsessionservice.DeleteChatSessionReq {
	return &chatsessionservice.DeleteChatSessionReq{Id: id}
}

func toRpcListChatSessionReq(req *types.ListChatSessionReq) *chatsessionservice.ListChatSessionReq {
	if req == nil {
		return nil
	}

	return &chatsessionservice.ListChatSessionReq{
		PageQuery: toRpcPageQuery(req.PageQuery),
		Filter:    toRpcChatSessionFilter(req.Filter, req.UserId),
	}
}

func toRpcPageQuery(query types.PageQuery) *chatsessionservice.PageQuery {
	return &chatsessionservice.PageQuery{
		Page:     query.Page,
		PageSize: query.PageSize,
		OrderBy:  strings.TrimSpace(query.OrderBy),
	}
}

func toRpcChatSessionFilter(filter types.ChatSessionFilter, userId int64) *chatsessionservice.ListChatSessionFilter {
	return &chatsessionservice.ListChatSessionFilter{
		Id:     filter.Id,
		ConvId: strings.TrimSpace(filter.ConvId),
		UserId: userId,
		Title:  strings.TrimSpace(filter.Title),
	}
}

func toTypesChatSession(session *chatsessionservice.ChatSession) types.ChatSession {
	if session == nil {
		return types.ChatSession{}
	}

	return types.ChatSession{
		Id:         session.Id,
		ConvId:     session.ConvId,
		UserId:     session.UserId,
		Title:      session.Title,
		CreateTime: session.CreateTime,
	}
}
