package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/Primexz/prefetcharr-go/internal/app"
	"go.uber.org/zap"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to YAML config")
	flag.Parse()

	cfg, err := app.LoadConfig(*configPath)
	if err != nil {
		zap.L().Error("load config", zap.Error(err))
		os.Exit(1)
	}

	logger, err := app.NewLogger(cfg.LogLevel)
	if err != nil {
		zap.L().Error("initialize logger", zap.Error(err))
		os.Exit(1)
	}
	defer logger.Sync() //nolint:errcheck
	logger.Info("starting prefetcharr-go", zap.String("version", version), zap.String("commit", commit), zap.String("date", date))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runner, err := app.New(cfg, logger)
	if err != nil {
		logger.Error("initialize", zap.Error(err))
		os.Exit(1)
	}

	if err := runner.Run(ctx); err != nil && ctx.Err() == nil {
		logger.Error("run", zap.Error(err))
		os.Exit(1)
	}
}
