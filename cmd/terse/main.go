package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/httplog"

	"github.com/undeadops/terse/internal/api"
	"github.com/undeadops/terse/internal/db"
)

const (
	appName = "terse"
)

var (
	port        string
	region      string
	table       string
	ddbEndpoint string
	debug       bool
	version     string
)

func main() {
	flag.StringVar(&port, "port", getEnv("PORT", "5000"), "port to listen on")
	flag.StringVar(&region, "region", getEnv("AWS_REGION", "us-east-1"), "AWS region")
	flag.StringVar(&table, "table", getEnv("DYNAMODB_TABLE", "terse"), "DynamoDB table name")
	flag.StringVar(&ddbEndpoint, "ddb-endpoint", getEnv("DYNAMODB_ENDPOINT", ""), "DynamoDB endpoint URL")
	flag.BoolVar(&debug, "debug", getEnvBool("DEBUG", false), "Enable debug mode")

	// Parse flags
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := httplog.NewLogger(appName, httplog.Options{
		JSON:    true,
		Concise: true,
		Tags: map[string]string{
			"version": version,
			"app":     appName,
		},
	})

	logger.Info().Str("version", version).Msgf("Starting %s version %s", appName, version)

	logger.Info().Msg("Setting up database connection...")
	client := &db.Client{
		Region:      region,
		Table:       table,
		DDBEndpoint: ddbEndpoint,
		DebugMode:   debug,
		Logger:      &logger,
	}

	err := db.SetupDB(ctx, client)
	if err != nil {
		panic(err)
	}
	//defer client.Close()

	router := api.Router(ctx, client, logger)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	logger.Info().Msgf("Starting %s server on port %s", appName, port)
	// Run server in the background
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Err(err).Msg("Server error")
		}
	}()

	// Listen for the interrupt signal
	<-ctx.Done()

	// Create shutdown context with 30-second timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Trigger graceful shutdown
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Err(err)
	}
	logger.Printf("Shutting down %s server", appName)
}

// Helper functions to get environment variables with default values
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		return value == "true" || value == "1" || value == "yes"
	}

	return defaultVal
}
