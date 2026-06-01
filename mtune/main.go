package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/QQGoblin/damonctl/pkg/mtune"
)

const defaultConfigPath = "/etc/mtune/config.json"

func main() {
	configPath := flag.String("config", defaultConfigPath, "path to mtune config file")
	flag.Parse()

	if err := mtune.InitLogger(); err != nil {
		logrus.WithError(err).Error("init logger failed")
		os.Exit(1)
	}

	cfg, err := mtune.LoadConfig(*configPath)
	if err != nil {
		logrus.WithError(err).Error("load config failed")
		os.Exit(1)
	}

	ctl, err := mtune.NewController(cfg)
	if err != nil {
		logrus.WithError(err).Error("create controller failed")
		os.Exit(1)
	}

	if err = ctl.Initialize(cfg.Reclaim); err != nil {
		logrus.WithError(err).Error("initialize damon_reclaim failed")
		os.Exit(1)
	}

	logrus.Info("mtune started")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err = ctl.Run(ctx); err != nil {
		logrus.WithError(err).Error("execute controller failed")
		os.Exit(1)
	}

	logrus.Info("mtune stopped")
}
