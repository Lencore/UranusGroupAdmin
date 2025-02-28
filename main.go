package main

import (
	"app/dto"
	"app/endpoint/cron"
	"app/endpoint/polling"
	"app/gateway/database"
	"app/gateway/redis"
	tele "app/gateway/telegram"
	"app/usecase/telegram"
	"app/util/logger"
	"context"
	"fmt"
	"github.com/jinzhu/configor"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var config dto.Config
	err := configor.Load(&config, "./config.yml")
	if err != nil {
		fmt.Printf("can't load config: %v", err)
		panic("Cant load config")
	}

	log, err := logger.NewLogger("local", config.LogLevel)
	if err != nil {
		panic(err)
	}

	log.Info("Config loaded")

	dbClient, err := database.New(config)
	if err != nil {
		log.Fatal("Can't init db", zap.Error(err))
	}

	log.Info("DB loaded")

	redisClient, err := redis.NewClient(config)
	if err != nil {
		log.Fatal("Can't init redis", zap.Error(err))
	}

	log.Info("Redis loaded")

	tgClient, err := tele.NewBot(config)
	if err != nil {
		log.Fatal("Can't init bot", zap.Error(err))
	}

	tg, err := telegram.NewTelegram(config, dbClient, redisClient, tgClient)
	if err != nil {
		log.Fatal("Can't init telegram", zap.Error(err))
	}

	cronEndpoint, err := cron.NewCron(config, tg)
	if err != nil {
		log.Fatal("Can't init cron", zap.Error(err))
	}

	pollingEndpoint, err := polling.New(tgClient)
	if err != nil {
		log.Fatal("Can't init polling", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g, eCtx := errgroup.WithContext(ctx)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	g.Go(func() error {
		log.Info("Running telegram...")

		return tg.Run(eCtx)
	})

	g.Go(func() error {
		log.Info("Running cron...")

		return cronEndpoint.Run(eCtx)
	})

	g.Go(func() error {
		log.Info("Running polling...")

		return pollingEndpoint.Run(eCtx)
	})

	if err := g.Wait(); err != nil {
		log.Info("App terminated", zap.Error(err))
	}

}
