package wsapi

import (
	"context"

	appconfig "github.com/TrieuNguyenPhu/shorten-link/backend/internal/config"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/middleware"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/service"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func NewRuntime(ctx context.Context, name string) (*service.WebSocketService, error) {
	cfg, err := appconfig.Load()
	if err != nil {
		return nil, err
	}
	sdk, err := middleware.LoadAWSConfig(ctx)
	if err != nil {
		return nil, err
	}
	return service.NewWebSocket(repository.New(dynamodb.NewFromConfig(sdk), cfg.TableName), middleware.NewLogger(name)), nil
}
