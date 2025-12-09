// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"flag"
	"fmt"

	"go-zero-voice-agent/app/rag/cmd/api/internal/config"
	"go-zero-voice-agent/app/rag/cmd/api/internal/handler"
	"go-zero-voice-agent/app/rag/cmd/api/internal/svc"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/rag.yaml", "the config file")

func main() {
	flag.Parse()

	if err := godotenv.Load(); err != nil {
        logx.Errorf("Error loading .env file, please check if it exists: %v", err)
    }

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
