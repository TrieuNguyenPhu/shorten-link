package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	appconfig "github.com/TrieuNguyenPhu/shorten-link/backend/internal/config"
	appevents "github.com/TrieuNguyenPhu/shorten-link/backend/internal/events"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/middleware"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/service"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
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
	logger := middleware.NewLogger("click-aggregator")
	var notifier service.ClickNotifier
	// Keep existing deployments usable until Task 4.2 supplies the management URL.
	if cfg.WSEndpoint != "" {
		client := apigatewaymanagementapi.NewFromConfig(sdk, func(o *apigatewaymanagementapi.Options) { o.BaseEndpoint = aws.String(cfg.WSEndpoint) })
		notifier = appevents.NewWSPusher(repo, client, logger)
	}
	return service.NewAggregator(repo, notifier, logger, os.Stdout), nil
}

func main() {
	handler, err := newHandler(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	lambda.Start(handler.HandleBatch)
}
