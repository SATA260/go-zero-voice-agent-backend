package main

import (
	"flag"
	"fmt"

	"go-zero-voice-agent/app/rag/cmd/rpc/internal/config"
	docserviceServer "go-zero-voice-agent/app/rag/cmd/rpc/internal/server/docservice"
	"go-zero-voice-agent/app/rag/cmd/rpc/internal/svc"
	"go-zero-voice-agent/app/rag/cmd/rpc/pb"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/rag.yaml", "the config file")

func main() {
	flag.Parse()

	if err := godotenv.Load(); err != nil {
        logx.Errorf("Error loading .env file, please check if it exists: %v", err)
    }

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterDocServiceServer(grpcServer, docserviceServer.NewDocServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
