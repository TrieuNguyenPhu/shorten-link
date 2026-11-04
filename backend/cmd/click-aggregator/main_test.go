package main

import (
	"context"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestNewHandlerRequiresTableAndHandlesEmptyBatchWithoutAWS(t *testing.T) {
	t.Setenv("TABLE_NAME", "")
	if _, err := newHandler(context.Background()); err == nil {
		t.Fatal("missing table accepted")
	}
	t.Setenv("TABLE_NAME", "table")
	t.Setenv("AWS_REGION", "ap-southeast-1")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_ACCESS_KEY_ID", "local")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "local")
	t.Setenv("WS_ENDPOINT", "")
	handler, err := newHandler(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	res, err := handler.HandleBatch(context.Background(), events.SQSEvent{})
	if err != nil || res.BatchItemFailures == nil || len(res.BatchItemFailures) != 0 {
		t.Fatal(res, err)
	}
}
