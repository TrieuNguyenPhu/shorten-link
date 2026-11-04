package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/middleware"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi/types"
)

type SubscriptionRepository interface {
	GetClickCount(context.Context, string) (int64, error)
	ForEachConnection(context.Context, string, func(string)) error
	DeleteConnection(context.Context, string) error
}

type ManagementAPI interface {
	PostToConnection(context.Context, *apigatewaymanagementapi.PostToConnectionInput, ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error)
}

type WSPusher struct {
	repo   SubscriptionRepository
	client ManagementAPI
	logger *slog.Logger
}

func NewWSPusher(repo SubscriptionRepository, client ManagementAPI, logger *slog.Logger) *WSPusher {
	return &WSPusher{repo: repo, client: client, logger: logger}
}

// NotifyClick runs only after commit. Failures are reported for logging, never
// for retrying the committed transaction. One failed connection cannot stop peers.
func (p *WSPusher) NotifyClick(ctx context.Context, code string) error {
	count, err := p.repo.GetClickCount(ctx, code)
	if err != nil {
		return fmt.Errorf("read committed count: %w", err)
	}
	data, err := json.Marshal(struct {
		Code   string `json:"code"`
		Clicks int64  `json:"clicks"`
	}{code, count})
	if err != nil {
		return err
	}
	logger := middleware.WithRequestID(ctx, p.logger)
	var failures error
	err = p.repo.ForEachConnection(ctx, code, func(id string) {
		_, postErr := p.client.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{ConnectionId: aws.String(id), Data: data})
		if postErr == nil {
			return
		}
		var gone *types.GoneException
		if errors.As(postErr, &gone) {
			if cleanupErr := p.repo.DeleteConnection(ctx, id); cleanupErr != nil {
				logger.ErrorContext(ctx, "connection cleanup failed", "connectionId", id, "error", cleanupErr)
				failures = errors.Join(failures, fmt.Errorf("cleanup connection %s: %w", id, cleanupErr))
			}
			return
		}
		logger.ErrorContext(ctx, "connection push failed", "connectionId", id, "error", postErr)
		failures = errors.Join(failures, fmt.Errorf("push connection %s: %w", id, postErr))
	})
	return errors.Join(failures, err)
}
