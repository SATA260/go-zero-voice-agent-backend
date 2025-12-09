package main

import (
	"flag"
	"fmt"

	"go-zero-voice-agent/app/llm/cmd/rpc/internal/config"
	chatmessageserviceServer "go-zero-voice-agent/app/llm/cmd/rpc/internal/server/chatmessageservice"
	chatsessionserviceServer "go-zero-voice-agent/app/llm/cmd/rpc/internal/server/chatsessionservice"
	llmchatserviceServer "go-zero-voice-agent/app/llm/cmd/rpc/internal/server/llmchatservice"
	llmconfigserviceServer "go-zero-voice-agent/app/llm/cmd/rpc/internal/server/llmconfigservice"
	"go-zero-voice-agent/app/llm/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/llm/cmd/rpc/pb"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/llmservice.yaml", "the config file")

func main() {
	flag.Parse()

	if err := godotenv.Load(); err != nil {
        logx.Errorf("Error loading .env file, please check if it exists: %v", err)
    }

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterLlmChatServiceServer(grpcServer, llmchatserviceServer.NewLlmChatServiceServer(ctx))
		pb.RegisterLlmConfigServiceServer(grpcServer, llmconfigserviceServer.NewLlmConfigServiceServer(ctx))
		pb.RegisterChatSessionServiceServer(grpcServer, chatsessionserviceServer.NewChatSessionServiceServer(ctx))
		pb.RegisterChatMessageServiceServer(grpcServer, chatmessageserviceServer.NewChatMessageServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
