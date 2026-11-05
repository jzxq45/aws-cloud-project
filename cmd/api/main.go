package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aws-local-test/internal/ai"
	"aws-local-test/internal/handler"
	"aws-local-test/internal/middleware"
	dynamodbRepo "aws-local-test/internal/repository/dynamodb"
	"aws-local-test/web"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithResponseChecksumValidation(aws.ResponseChecksumValidationWhenRequired),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{AccessKeyID: "test", SecretAccessKey: "test"}, nil
		})),
	)
	if err != nil {
		slog.Error("AWS config error", "error", err)
		os.Exit(1)
	}

	endpoint := os.Getenv("AWS_ENDPOINT_URL")
	if endpoint == "" {
		endpoint = "http://localhost:4566"
	}
	cfg.BaseEndpoint = aws.String(endpoint)

	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	dbClient := dynamodb.NewFromConfig(cfg)
	metrics := middleware.NewMetrics()
	authMiddleware := middleware.NewAuthMiddleware()
	authHandler := handler.NewAuthHandler()
	userRepo := dynamodbRepo.NewUserRepository(dbClient, "Users")
	userHandler := handler.NewUserHandler(userRepo)
	ollamaClient := ai.NewOllamaClient(ollamaURL, "qwen2.5:3b", nil)
	aiHandler := handler.NewAIHandler(ollamaClient)
	router := handler.NewRouter(userHandler, aiHandler, authHandler, metrics, authMiddleware, web.Files)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("Server running", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		slog.Error("Server startup error", "error", err)
		os.Exit(1)
	case sig := <-shutdown:
		slog.Info("Initiating graceful shutdown", "signal", sig.String())
		ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctxShutdown); err != nil {
			slog.Error("Graceful shutdown failed", "error", err)
			_ = srv.Close()
		}
		slog.Info("Server stopped cleanly")
	}
}
