package llmchatservicelogic

import (
	"context"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"
	"go-zero-voice-agent/app/llm/pkg/consts"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type ExecuteToolLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewExecuteToolLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ExecuteToolLogic {
	return &ExecuteToolLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ExecuteToolLogic) ExecuteTool(in *pb.ExecuteToolReq) (*pb.ExecuteToolResp, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if in.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "tool name is required")
	}

	if in.Scope != "" && in.Scope != consts.TOOL_CALLING_SCOPE_SERVER {
		return nil, status.Error(codes.InvalidArgument, "executeTool only supports server scope")
	}

	toolIns, ok := l.svcCtx.ToolRegistry[in.Name]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported tool: %s", in.Name)
	}

	if toolIns.RequiresConfirmation() && !in.Approved {
		return nil, status.Error(codes.PermissionDenied, "tool execution requires explicit approval")
	}

	if err := l.ctx.Err(); err != nil {
		return nil, err
	}

	toolCall := &pb.ToolCallDelta{
		Id:                   in.Id,
		Name:                 in.Name,
		ArgumentsJson:        in.ArgumentsJson,
		Scope:                consts.TOOL_CALLING_SCOPE_SERVER,
		Status:               consts.TOOL_CALLING_EXECUTING,
		RequiresConfirmation: toolIns.RequiresConfirmation(),
	}

	l.Logger.Infof("executing backend tool: %s", in.Name)
	result, err := toolIns.Execute(l.ctx, in.ArgumentsJson)
	toolCall.Status = consts.TOOL_CALLING_FINISHED
	if err != nil {
		toolCall.Error = err.Error()
		l.Logger.Errorf("tool %s execution failed: %v", in.Name, err)
		return &pb.ExecuteToolResp{ToolCall: toolCall}, nil
	}

	toolCall.Result = result
	return &pb.ExecuteToolResp{ToolCall: toolCall}, nil
}
