package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
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
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if strings.TrimSpace(p.bus) == "" {
		return errors.New("EVENT_BUS is required")
	}
	detail, err := json.Marshal(event)
	if err != nil {
		return err
	}
	in := &eventbridge.PutEventsInput{Entries: []types.PutEventsRequestEntry{{EventBusName: aws.String(p.bus), Source: aws.String("urlshortener"), DetailType: aws.String("link.clicked"), Detail: aws.String(string(detail))}}}
	for attempt := 0; attempt < 3; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		out, err := p.client.PutEvents(ctx, in, func(o *eventbridge.Options) { o.Retryer = aws.NopRetryer{} })
		// Transport failures are ambiguous: fail closed without additional publication.
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
		if out.FailedEntryCount != 1 || id != "" || (code != "InternalFailure" && code != "ThrottlingException") {
			return fmt.Errorf("click publish rejected: failed=%d code=%q", out.FailedEntryCount, code)
		}
		if attempt == 2 {
			return fmt.Errorf("click publish retry budget exhausted: %s", code)
		}
		// Full jitter, 0..50ms then 0..100ms, within the shared deadline.
		delay := time.Duration(rand.Int64N(int64(50*time.Millisecond<<attempt) + 1))
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return errors.New("click publish failed")
}
