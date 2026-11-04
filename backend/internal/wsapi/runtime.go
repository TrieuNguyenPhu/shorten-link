package wsapi

import (
	"context"
	"fmt"
	"time"

	appconfig "github.com/TrieuNguyenPhu/shorten-link/backend/internal/config"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/middleware"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/service"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func NewRuntime(ctx context.Context, name string) (*service.WebSocketService, error) {
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
	return service.NewWebSocket(repository.New(dynamodb.NewFromConfig(sdk), cfg.TableName), middleware.NewLogger(name)), nil
}
