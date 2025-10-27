package logic

import (
	"context"

	"go-zero-voice-agent/app/llmservice/cmd/rpc/internal/consts"
	"go-zero-voice-agent/app/llmservice/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llmservice/cmd/rpc/pb"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateChatLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateChatLogic {
	return &CreateChatLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateChatLogic) CreateChat(in *pb.CreateChatReq) (*pb.CreateChatResp, error) {
	client := openai.NewClient(
		option.WithAPIKey(in.LlmConfig.ApiKey),
		option.WithBaseURL(in.LlmConfig.BaseUrl),
	)

	var chatMsgs []openai.ChatCompletionMessageParamUnion

	for i, msg := range in.Messages {
		switch msg.Role {
		case consts.ChatMessageRole_USER:
			// 用户消息处理
			chatMsgs = append(chatMsgs, openai.UserMessage(msg.Content))
		case consts.ChatMessageRole_ASSISTANT:
			// 助手消息处理
			chatMsgs = append(chatMsgs, openai.AssistantMessage(msg.Content))
		case consts.ChatMessageRole_SYSTEM:
			// 系统消息处理
			if i != 0 {
				logx.Errorf("系统消息只能出现在第一条, 位置: %d", i)
			}
			chatMsgs = append(chatMsgs, openai.SystemMessage(msg.Content))
		default:
			logx.Errorf("未知的消息角色: %v", msg.Role)
		}
	}

	// 进行对话
	chatCompletion, err := client.Chat.Completions.New(
		context.TODO(), openai.ChatCompletionNewParams{
			Messages: openai.F(
				chatMsgs,
			),
			Model: openai.F(in.LlmConfig.Model),
		},
	)

	if err != nil {
		logx.Errorf("创建聊天失败, err: %v", err)
		return nil, err
	}

	return &pb.CreateChatResp{
		Id: chatCompletion.ID,
		RespMsg: &pb.ChatMsg{
			Role: "user",
			Content: chatCompletion.Choices[len(chatCompletion.Choices)-1].Message.Content,
		},
	}, nil
}

