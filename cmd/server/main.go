package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/config"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/handler"
	applogger "github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/logger"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/repository"
	"github.com/Phoenix-Event-Ticketing/phoenix-online-ticket-inventory-service/internal/service"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}

	log, err := applogger.New(cfg.LogLevel, cfg.ServiceName, cfg.Environment)
	if err != nil {
		fmt.Fprintln(os.Stderr, "logger:", err)
		os.Exit(1)
	}
	defer func() { _ = log.Sync() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	cancel()
	if err != nil {
		log.Fatal("mongo connect failed", zap.Error(err))
	}
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	db := client.Database(cfg.MongoDatabase)
	repo := repository.NewInventoryRepository(db)
	if err := repo.Ping(context.Background()); err != nil {
		log.Fatal("mongo ping failed", zap.Error(err))
	}
	if err := repo.EnsureIndexes(context.Background()); err != nil {
		log.Fatal("mongo indexes", zap.Error(err))
	}

	svc := service.NewInventoryService(repo, cfg.HoldTTL())
	invHandler := handler.NewInventoryHandler(svc)
	router := handler.NewRouter(log, invHandler)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("server listening",
			zap.String("operation", "listen"),
			zap.String("addr", srv.Addr),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown", zap.Error(err))
	}
	log.Info("server stopped", zap.String("operation", "shutdown"))
}
