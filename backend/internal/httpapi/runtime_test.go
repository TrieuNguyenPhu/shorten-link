package httpapi

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

func TestRuntimeFailsClosedOnMissingConfig(t *testing.T) {
	t.Setenv("TABLE_NAME", "")
	if _, err := NewRuntime(context.Background(), "create-link"); err == nil {
		t.Fatal("missing table accepted")
	}
	t.Setenv("TABLE_NAME", "table")
	t.Setenv("EVENT_BUS", "")
	if _, err := NewRuntime(context.Background(), "resolve-link"); err == nil {
		t.Fatal("missing bus accepted")
	}
}

func TestHandlerDeadlinePreservesEarlierParent(t *testing.T) {
	for _, parentBudget := range []time.Duration{time.Minute, 20 * time.Millisecond} {
		ctx, cancel := context.WithTimeout(context.Background(), parentBudget)
		wrapped := WithDeadline(func(ctx context.Context, _ events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) > min(parentBudget, 4*time.Second) {
				t.Fatal("handler lost deadline")
			}
			return events.APIGatewayV2HTTPResponse{StatusCode: 204}, nil
		})
		res, err := wrapped(ctx, events.APIGatewayV2HTTPRequest{})
		cancel()
		if err != nil || res.StatusCode != 204 {
			t.Fatal(res, err)
		}
	}
}
