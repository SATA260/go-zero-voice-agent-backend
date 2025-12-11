package toolcall

import (
	"context"
	"encoding/json"
	"fmt"

	"go-zero-voice-agent/app/rag/cmd/rpc/client/ragservice"
	"go-zero-voice-agent/app/rag/cmd/rpc/pb" // 引入 RAG 服务的 pb 包
)

type RagTool struct {
	ragClient ragservice.RagService
}

type RagToolParams struct {
	Query   string   `json:"query"`
	UserId  int64    `json:"user_id"`
	FileIds []string `json:"file_ids"`
	TopK    int32    `json:"top_k"`
}

func NewRagTool(ragClient ragservice.RagService) *RagTool {
	return &RagTool{ragClient: ragClient}
}

func (t *RagTool) Name() string {
	return "self_rag"
}

func (t *RagTool) Description() string {
	return "RAG工具,用于基于自己的知识库进行问答"
}

func (t *RagTool) ArgumentsJson() string {
	return `{
		"query": "查询内容，字符串类型",
		"user_id": "用户ID，整数类型",
		"file_ids": "知识库文件ID列表，字符串数组类型",
		"top_k": "返回的相关内容数量，整数类型，默认为3"
	}`
}

func (t *RagTool) Execute(ctx context.Context, argsJson string) (string, error) {
	// 解析参数
	var params RagToolParams
	if err := json.Unmarshal([]byte(argsJson), &params); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	if params.TopK == 0 {
		params.TopK = 3
	}

	// 调用 RAG RPC 服务
	resp, err := t.ragClient.QueryMultiple(ctx, &pb.QueryMultipleReq{
		Query:   params.Query,
		UserId:  params.UserId,
		FileIds: params.FileIds,
		TopK:    params.TopK,
	})
	if err != nil {
		return "", fmt.Errorf("调用 RAG 服务失败: %w", err)
	}

	// 处理返回结果
	if len(resp.Results) == 0 {
		return "未找到相关信息", nil
	}

	resultBytes, err := json.Marshal(resp.Results)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultBytes), nil
}

func (t *RagTool) RequiresConfirmation() bool {
	return false
}
