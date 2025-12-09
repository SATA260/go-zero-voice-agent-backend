package main

import (
	"flag"
	"fmt"

	"go-zero-voice-agent/app/voicechat/cmd/rpc/internal/config"
	asrconfigserviceServer "go-zero-voice-agent/app/voicechat/cmd/rpc/internal/server/asrconfigservice"
	ttsconfigserviceServer "go-zero-voice-agent/app/voicechat/cmd/rpc/internal/server/ttsconfigservice"
	"go-zero-voice-agent/app/voicechat/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/voicechat/cmd/rpc/voicechatpb"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/voicechat.yaml", "the config file")

func main() {
	flag.Parse()

	if err := godotenv.Load(); err != nil {
        logx.Errorf("Error loading .env file, please check if it exists: %v", err)
    }

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		voicechatpb.RegisterAsrConfigServiceServer(grpcServer, asrconfigserviceServer.NewAsrConfigServiceServer(ctx))
		voicechatpb.RegisterTtsConfigServiceServer(grpcServer, ttsconfigserviceServer.NewTtsConfigServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
