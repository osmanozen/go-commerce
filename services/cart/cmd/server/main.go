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

	carthttp "github.com/osmanozen/oo-commerce/services/cart/internal/adapters/http"
	"github.com/osmanozen/oo-commerce/services/cart/internal/adapters/persistence"
	"github.com/osmanozen/oo-commerce/services/cart/internal/application/commands"
	"github.com/osmanozen/oo-commerce/services/cart/internal/application/queries"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("cart service starting", slog.String("version", "1.0.0"))

	port := envOrDefault("PORT", "8082")
	kafkaBrokers := []string{envOrDefault("KAFKA_BROKERS", "localhost:9092")}

	kafkaCfg := messaging.DefaultKafkaProducerConfig(kafkaBrokers)
	kafkaProducer := messaging.NewKafkaProducer(kafkaCfg, logger)
	defer kafkaProducer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Background job: cleanup abandoned carts (> 30 days inactive)
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// TODO: cartRepo.CleanupAbandoned(ctx, 720)  // 30 days * 24 hours
				logger.Info("abandoned cart cleanup tick")
			}
		}
	}()

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
		fmt.Fprint(w, `{"status":"healthy","service":"cart"}`)
	})

	// Repository
	cartRepo := persistence.NewCartRepository(nil, logger) // We leave nil for DB pool in this scaffold

	// Handlers
	addToCartHandler := commands.NewAddToCartHandler(cartRepo)
	updateQtyHandler := commands.NewUpdateCartItemQuantityHandler(cartRepo)
	removeItemHandler := commands.NewRemoveFromCartHandler(cartRepo)
	clearCartHandler := commands.NewClearCartHandler(cartRepo)
	mergeCartHandler := commands.NewMergeCartHandler(cartRepo)
	getCartHandler := queries.NewGetCartHandler(cartRepo)

	// API Routes
	httpHandler := carthttp.NewCartHandler(
		addToCartHandler, updateQtyHandler, removeItemHandler,
		clearCartHandler, mergeCartHandler, getCartHandler, logger,
	)
	httpHandler.RegisterRoutes(r)

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
		logger.Info("cart service listening", slog.String("addr", server.Addr))
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

	logger.Info("cart service stopped")
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
