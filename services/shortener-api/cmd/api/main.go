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

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"

	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/adapters/generator"
	ginhttp "github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/adapters/http/gin"
	dynamorepository "github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/adapters/repository/dynamodb"
	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/adapters/repository/memory"
	clockadapter "github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/adapters/time"
	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/application/ports"
	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/application/service"
	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error("shortener API stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	runningInLambda := os.Getenv("AWS_LAMBDA_RUNTIME_API") != ""
	appConfig, err := config.Load(runningInLambda)
	if err != nil {
		return err
	}

	repository, err := buildRepository(context.Background(), appConfig)
	if err != nil {
		return err
	}
	codeGenerator, err := generator.NewCryptoRandom(7)
	if err != nil {
		return err
	}

	linkService := service.NewLinkService(repository, codeGenerator, clockadapter.SystemClock{})
	handler := ginhttp.NewHandler(linkService, appConfig.PublicBaseURL)
	router := ginhttp.NewRouter(handler, appConfig.CORSAllowedOrigins)

	if runningInLambda {
		adapter := ginadapter.NewV2(router)
		lambda.Start(adapter.ProxyWithContext)
		return nil
	}

	server := &http.Server{
		Addr:              appConfig.Address,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("shortener API listening", "address", appConfig.Address)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-shutdownContext.Done():
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	}
}

func buildRepository(ctx context.Context, appConfig config.Config) (ports.LinkRepository, error) {
	if appConfig.StorageDriver == config.StorageMemory {
		return memory.NewLinkRepository(), nil
	}

	awsConfig, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return dynamorepository.NewLinkRepository(awsdynamodb.NewFromConfig(awsConfig), appConfig.LinksTable)
}
