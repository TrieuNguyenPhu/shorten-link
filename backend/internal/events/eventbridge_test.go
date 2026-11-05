package events

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

type putStub func(context.Context, *eventbridge.PutEventsInput, ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error)

func (f putStub) PutEvents(ctx context.Context, in *eventbridge.PutEventsInput, opts ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error) {
	return f(ctx, in, opts...)
}

func success() *eventbridge.PutEventsOutput {
	return &eventbridge.PutEventsOutput{Entries: []types.PutEventsResultEntry{{EventId: aws.String("accepted-id")}}}
}

func TestPublishValidatesEntryAndDeadline(t *testing.T) {
	for _, tc := range []struct {
		name    string
		out     *eventbridge.PutEventsOutput
		wantErr bool
	}{
		{"accepted", success(), false},
		{"internal failure", &eventbridge.PutEventsOutput{FailedEntryCount: 1, Entries: []types.PutEventsResultEntry{{ErrorCode: aws.String("InternalFailure")}}}, true},
		{"denied", &eventbridge.PutEventsOutput{FailedEntryCount: 1, Entries: []types.PutEventsResultEntry{{ErrorCode: aws.String("AccessDeniedException")}}}, true},
		{"missing", &eventbridge.PutEventsOutput{}, true},
		{"nil", nil, true},
		{"no id", &eventbridge.PutEventsOutput{Entries: []types.PutEventsResultEntry{{}}}, true},
		{"count mismatch", &eventbridge.PutEventsOutput{FailedEntryCount: 1, Entries: success().Entries}, true},
		{"both id error", &eventbridge.PutEventsOutput{FailedEntryCount: 1, Entries: []types.PutEventsResultEntry{{EventId: aws.String("id"), ErrorCode: aws.String("InternalFailure")}}}, true},
		{"extra entries", &eventbridge.PutEventsOutput{Entries: append(success().Entries, success().Entries...)}, true},
		{"count missing", &eventbridge.PutEventsOutput{Entries: []types.PutEventsResultEntry{{ErrorCode: aws.String("InternalFailure")}}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			publisher := NewPublisher(putStub(func(ctx context.Context, in *eventbridge.PutEventsInput, _ ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error) {
				calls++
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > 1500*time.Millisecond {
					t.Fatal("unbounded deadline")
				}
				if len(in.Entries) != 1 || aws.ToString(in.Entries[0].EventBusName) != "bus" || aws.ToString(in.Entries[0].Source) != "urlshortener" || aws.ToString(in.Entries[0].DetailType) != "link.clicked" {
					t.Fatal(in)
				}
				var detail domain.ClickEvent
				if err := json.Unmarshal([]byte(aws.ToString(in.Entries[0].Detail)), &detail); err != nil || detail.Code != "Abc1234" || detail.At != "2026-11-04T00:00:00.000Z" {
					t.Fatal(detail, err)
				}
				return tc.out, nil
			}), "bus")
			err := publisher.Publish(context.Background(), domain.ClickEvent{Code: "Abc1234", At: "2026-11-04T00:00:00.000Z"})
			if (err != nil) != tc.wantErr || calls != 1 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
		})
	}
}

func TestPublishHonorsCancellationAndTransportFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancel()
	p := NewPublisher(putStub(func(ctx context.Context, _ *eventbridge.PutEventsInput, _ ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}), "bus")
	start := time.Now()
	if err := p.Publish(ctx, domain.ClickEvent{}); !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > time.Second {
		t.Fatal(err)
	}
	calls := 0
	p = NewPublisher(putStub(func(context.Context, *eventbridge.PutEventsInput, ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error) {
		calls++
		return nil, errors.New("transport unavailable")
	}), "bus")
	if err := p.Publish(context.Background(), domain.ClickEvent{}); err == nil || calls != 1 {
		t.Fatal(err, calls)
	}
}

func TestPublishDoesNotSendWithoutBusOrAfterCancellation(t *testing.T) {
	calls := 0
	client := putStub(func(context.Context, *eventbridge.PutEventsInput, ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error) {
		calls++
		return success(), nil
	})
	if err := NewPublisher(client, "").Publish(context.Background(), domain.ClickEvent{}); err == nil || calls != 0 {
		t.Fatal(err, calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := NewPublisher(client, "bus").Publish(ctx, domain.ClickEvent{}); !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatal(err, calls)
	}
}
