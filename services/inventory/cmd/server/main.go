package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/osmanozen/oo-commerce/pkg/buildingblocks/messaging"
	bbmiddleware "github.com/osmanozen/oo-commerce/pkg/buildingblocks/middleware"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("inventory service starting", slog.String("version", "1.0.0"))

	port := envOrDefault("PORT", "8084")
	kafkaBrokers := []string{envOrDefault("KAFKA_BROKERS", "localhost:9092")}

	// ─── Kafka ──────────────────────────────────────────────────────────
	kafkaCfg := messaging.DefaultKafkaProducerConfig(kafkaBrokers)
	kafkaProducer := messaging.NewKafkaProducer(kafkaCfg, logger)
	defer kafkaProducer.Close()

	kafkaConsumer := messaging.NewKafkaConsumer(kafkaBrokers, logger)
	defer kafkaConsumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ─── Ensure Topics ──────────────────────────────────────────────────
	inventoryTopics := []messaging.TopicConfig{
		{Name: "inventory.stock.reserved", NumPartitions: 12, ReplicationFactor: 1},
		{Name: "inventory.stock.released", NumPartitions: 6, ReplicationFactor: 1},
		{Name: "inventory.stock.adjusted", NumPartitions: 6, ReplicationFactor: 1},
		{Name: "inventory.stock.low", NumPartitions: 3, ReplicationFactor: 1},
		{Name: "inventory.stock.reservation-failed", NumPartitions: 6, ReplicationFactor: 1},
	}
	if err := messaging.EnsureTopics(ctx, kafkaBrokers[0], inventoryTopics, logger); err != nil {
		logger.Warn("failed to ensure kafka topics", slog.String("error", err.Error()))
	}

	// ─── Database + Event Consumers ─────────────────────────────────────
	// TODO: Initialize pgxpool, create repositories
	// stockRepo := persistence.NewStockItemRepository(pool)
	// eventConsumer := events.NewInventoryEventConsumer(stockRepo, kafkaConsumer, kafkaProducer, logger)
	// eventConsumer.Start(ctx)

	// Background job: clean up expired reservations
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// TODO: stockRepo.CleanupExpiredReservations(ctx)
				logger.Debug("reservation cleanup tick")
			}
		}
	}()

	// ─── HTTP Router ────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(bbmiddleware.CorrelationID)
	r.Use(bbmiddleware.RequestLogger(logger))
	r.Use(bbmiddleware.Recovery(logger))
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Compress(5))
	r.Use(chimiddleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"healthy","service":"inventory"}`)
	})

	// TODO: Register stock handler routes

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("inventory service listening", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-done
	logger.Info("shutdown signal received")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", slog.String("error", err.Error()))
	}

	logger.Info("inventory service stopped")
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
