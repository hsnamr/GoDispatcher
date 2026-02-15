package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/halamri/go-dispatcher/internal/config"
	"github.com/halamri/go-dispatcher/internal/controller"
	"github.com/halamri/go-dispatcher/internal/consumer"
	"github.com/halamri/go-dispatcher/internal/dispatcher"
	"github.com/halamri/go-dispatcher/internal/kafka"
	"github.com/halamri/go-dispatcher/internal/redis"
	"github.com/halamri/go-dispatcher/internal/strategies"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	initLogger(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var broker *redis.Broker
	if !cfg.UseKafka() {
		var err error
		broker, err = redis.NewBroker(cfg)
		if err != nil {
			slog.Error("redis broker failed", "error", err)
			os.Exit(1)
		}
		defer broker.Close()
	} else {
		broker, _ = redis.NewBroker(cfg)
		if broker != nil {
			defer broker.Close()
		}
	}

	backend, err := redis.NewBackendRedis(cfg)
	if err != nil {
		slog.Error("backend redis failed", "error", err)
		os.Exit(1)
	}
	defer backend.Close()

	registry := controller.NewRegistry()
	strategies.RegisterStubStrategies(registry)
	ctrl := controller.NewController(registry)

	cons := consumer.NewConsumer(cfg, broker, backend, ctrl, slog.Default())
	pubWorker := dispatcher.NewWorker(cfg, broker, backend, cons.SenderQueue(), slog.Default())

	if cfg.UseKafka() {
		kafkaClient, err := kafka.NewClient(cfg)
		if err != nil {
			slog.Error("kafka client failed", "error", err)
			os.Exit(1)
		}
		defer kafkaClient.Close()
		cons.SetCalcExportProducer(kafkaClient)
		pubWorker.SetKafkaOutput(kafkaClient)
		go pubWorker.Run(ctx)
		go func() {
			if err := kafkaClient.RunConsumer(ctx, slog.Default(), cons.ProcessMessage); err != nil && ctx.Err() == nil {
				slog.Error("kafka consumer exited", "error", err)
			}
		}()
	} else {
		go pubWorker.Run(ctx)
		go func() {
			if err := cons.Run(ctx); err != nil && ctx.Err() == nil {
				slog.Error("consumer exited", "error", err)
			}
		}()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh
	slog.Info("shutting down...")
	cancel()
}

func initLogger(cfg *config.Config) {
	opts := &slog.HandlerOptions{}
	var handler slog.Handler
	if cfg.LogFmt {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}
