package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	appconfig "github.com/TrieuNguyenPhu/shorten-link/backend/internal/config"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/middleware"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/service"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func newHandler(ctx context.Context) (*service.AggregatorService, error) {
	cfg, err := appconfig.Load()
	if err != nil {
		return nil, err
	}
	sdk, err := config.LoadDefaultConfig(ctx, config.WithRetryer(func() aws.Retryer {
		return retry.NewStandard(func(o *retry.StandardOptions) { o.MaxAttempts = 2; o.MaxBackoff = 100 * time.Millisecond })
	}))
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	middleware.InstrumentAWSConfig(&sdk)
	repo := repository.New(dynamodb.NewFromConfig(sdk), cfg.TableName)
	// WebSocket delivery is wired in Phase 4; no synthetic notifier in production.
	return service.NewAggregator(repo, nil, middleware.NewLogger("click-aggregator"), os.Stdout), nil
}

func main() {
	handler, err := newHandler(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	lambda.Start(handler.HandleBatch)
}
