package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

type EventBridgeAPI interface {
	PutEvents(context.Context, *eventbridge.PutEventsInput, ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error)
}

type Publisher struct {
	client EventBridgeAPI
	bus    string
}

func NewPublisher(client EventBridgeAPI, bus string) *Publisher {
	return &Publisher{client: client, bus: bus}
}

func (p *Publisher) Publish(ctx context.Context, event domain.ClickEvent) error {
	ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	if strings.TrimSpace(p.bus) == "" {
		return errors.New("EVENT_BUS is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	detail, err := json.Marshal(event)
	if err != nil {
		return err
	}
	in := &eventbridge.PutEventsInput{Entries: []types.PutEventsRequestEntry{{EventBusName: aws.String(p.bus), Source: aws.String("urlshortener"), DetailType: aws.String("link.clicked"), Detail: aws.String(string(detail))}}}
	out, err := p.client.PutEvents(ctx, in)
	if err != nil {
		return fmt.Errorf("publish click: %w", err)
	}
	if out == nil || len(out.Entries) != 1 {
		return errors.New("invalid PutEvents response")
	}
	entry := out.Entries[0]
	id := strings.TrimSpace(aws.ToString(entry.EventId))
	code := aws.ToString(entry.ErrorCode)
	if out.FailedEntryCount == 0 && id != "" && code == "" && aws.ToString(entry.ErrorMessage) == "" {
		return nil
	}
	return fmt.Errorf("click publish rejected: failed=%d code=%q", out.FailedEntryCount, code)
}
