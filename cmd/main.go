package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"durable-links-generator/api"
	"durable-links-generator/config"
	"durable-links-generator/db"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func initLogger(cfg *config.Config) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}
	log.Logger = zerolog.New(consoleWriter).With().Timestamp().Logger()

	level, err := zerolog.ParseLevel(cfg.Server.LogLevel)
	if err != nil {
		log.Error().Err(err).Msg("Invalid log level, using debug")
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)
}

func initDatabase(cfg *config.Config) (*db.DB, error) {
	var database *db.DB
	var err error
	maxRetries := 3
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		database, err = db.New(cfg)
		if err == nil {
			return database, nil
		}
		log.Warn().Err(err).Msgf("Failed to connect to database, attempt %d/%d", i+1, maxRetries)
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}
	return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Warn().Msg("No .env file found, using environment variables")
	}

	cfg := config.New()
	initLogger(cfg)

	database, err := initDatabase(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer database.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	router := api.NewRouter(database.DB, cfg)

	server := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		log.Info().Msgf("Server starting on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited properly")
}
