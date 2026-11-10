package httpapi

import (
	"context"
	"errors"
	"os"
	"time"

	appconfig "github.com/TrieuNguyenPhu/shorten-link/backend/internal/config"
	appevents "github.com/TrieuNguyenPhu/shorten-link/backend/internal/events"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/middleware"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/service"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
)

type LambdaHandler func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error)

// NewRuntime initializes shared SDK clients once per execution environment.
func NewRuntime(ctx context.Context, name string) (*Handler, error) {
	cfg, err := appconfig.Load()
	if err != nil {
		return nil, err
	}
	if name == "resolve-link" && cfg.EventBusName == "" {
		return nil, errors.New("EVENT_BUS is required for resolve-link")
	}
	if name == "create-link" && cfg.ModerationStateMachineARN == "" {
		return nil, errors.New("MODERATION_STATE_MACHINE_ARN is required for create-link")
	}
	sdk, err := middleware.LoadAWSConfig(ctx)
	if err != nil {
		return nil, err
	}
	repo := repository.New(dynamodb.NewFromConfig(sdk), cfg.TableName)
	var publisher service.ClickPublisher
	if name == "resolve-link" {
		publisher = appevents.NewPublisher(eventbridge.NewFromConfig(sdk), cfg.EventBusName)
	}
	linkService := service.New(repo, publisher)
	if name == "create-link" {
		linkService.SetModerationStarter(appevents.NewModerationStarter(sfn.NewFromConfig(sdk), cfg.ModerationStateMachineARN))
	}
	return New(linkService, cfg.PublicRedirectBaseURL, middleware.NewLogger(name), os.Stdout), nil
}

// WithDeadline leaves headroom below the 5-second Lambda timeout.
func WithDeadline(next LambdaHandler) LambdaHandler {
	return func(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
		defer cancel()
		return next(ctx, req)
	}
}
