package wsapi

import (
	"context"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestRuntimeRequiresTableAndUsesWebSocketEnvelope(t *testing.T) {
	t.Setenv("TABLE_NAME", "")
	if _, err := NewRuntime(context.Background(), "ws-connect"); err == nil {
		t.Fatal("missing table accepted")
	}
	t.Setenv("TABLE_NAME", "table")
	t.Setenv("WS_ENDPOINT", "")
	t.Setenv("AWS_REGION", "ap-southeast-1")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_ACCESS_KEY_ID", "local")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "local")
	svc, err := NewRuntime(context.Background(), "ws-connect")
	if err != nil {
		t.Fatal(err)
	}
	res, err := svc.Connect(context.Background(), events.APIGatewayWebsocketProxyRequest{})
	if err != nil || res.StatusCode != 400 {
		t.Fatal(res, err)
	}
}
