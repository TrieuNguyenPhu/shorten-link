package main

import (
	"context"
	"errors"
	"log"
	"os"

	appconfig "github.com/TrieuNguyenPhu/shorten-link/backend/internal/config"
	appevents "github.com/TrieuNguyenPhu/shorten-link/backend/internal/events"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/middleware"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/service"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func newHandler(ctx context.Context) (*service.AggregatorService, error) {
	cfg, err := appconfig.Load()
	if err != nil {
		return nil, err
	}
	if cfg.WSEndpoint == "" {
		return nil, errors.New("WS_ENDPOINT is required for click-aggregator")
	}
	sdk, err := middleware.LoadAWSConfig(ctx)
	if err != nil {
		return nil, err
	}
	repo := repository.New(dynamodb.NewFromConfig(sdk), cfg.TableName)
	logger := middleware.NewLogger("click-aggregator")
	client := apigatewaymanagementapi.NewFromConfig(sdk, func(o *apigatewaymanagementapi.Options) { o.BaseEndpoint = aws.String(cfg.WSEndpoint) })
	notifier := appevents.NewWSPusher(repo, client, logger)
	return service.NewAggregator(repo, notifier, logger, os.Stdout), nil
}

func main() {
	handler, err := newHandler(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	lambda.Start(handler.HandleBatch)
}
