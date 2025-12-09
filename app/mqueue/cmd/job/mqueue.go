package main

import (
	"context"
	"flag"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/logx"

	"os"

	"go-zero-voice-agent/app/mqueue/cmd/job/internal/config"
	"go-zero-voice-agent/app/mqueue/cmd/job/internal/logic"
	"go-zero-voice-agent/app/mqueue/cmd/job/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
)

var configFile = flag.String("f", "etc/mqueue.yaml", "Specify the config file")

func main() {
	flag.Parse()

	if err := godotenv.Load(); err != nil {
        logx.Errorf("Error loading .env file, please check if it exists: %v", err)
    }

	var c config.Config

	conf.MustLoad(*configFile, &c, conf.UseEnv())

	// log、prometheus、trace、metricsUrl
	if err := c.SetUp(); err != nil {
		panic(err)
	}

	//logx.DisableStat()

	svcContext := svc.NewServiceContext(c)
	ctx := context.Background()
	cronJob := logic.NewCronJob(ctx, svcContext)
	mux := cronJob.Register()

	if err := svcContext.AsynqServer.Run(mux); err != nil {
		logx.WithContext(ctx).Errorf("!!!CronJobErr!!! run err:%+v", err)
		os.Exit(1)
	}
}
