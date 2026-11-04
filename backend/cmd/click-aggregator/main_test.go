package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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

func TestRuntimeWiresConfiguredManagementEndpoint(t *testing.T) {
	var pushes atomic.Int64
	management := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.URL.Path != "/prod/@connections/runtime=" || string(body) != `{"code":"Abc1234","clicks":1}` || !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
			t.Error(r.URL, string(body))
		}
		pushes.Add(1)
		w.WriteHeader(200)
	}))
	defer management.Close()
	ddb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		switch r.Header.Get("X-Amz-Target") {
		case "DynamoDB_20120810.TransactWriteItems":
			_, _ = io.WriteString(w, `{}`)
		case "DynamoDB_20120810.GetItem":
			var in struct{ ConsistentRead bool }
			if json.NewDecoder(r.Body).Decode(&in) != nil || !in.ConsistentRead {
				t.Error("runtime read is not consistent")
			}
			_, _ = io.WriteString(w, `{"Item":{"clicks":{"N":"1"}}}`)
		case "DynamoDB_20120810.Query":
			_, _ = io.WriteString(w, `{"Items":[{"PK":{"S":"CONN#runtime="}}]}`)
		default:
			t.Error("unexpected operation", r.Header)
			http.Error(w, "unexpected operation", 400)
		}
	}))
	defer ddb.Close()
	for key, value := range map[string]string{"TABLE_NAME": "table", "AWS_REGION": "ap-southeast-1", "AWS_EC2_METADATA_DISABLED": "true", "AWS_ACCESS_KEY_ID": "local", "AWS_SECRET_ACCESS_KEY": "local", "WS_ENDPOINT": management.URL + "/prod", "AWS_ENDPOINT_URL_DYNAMODB": ddb.URL} {
		t.Setenv(key, value)
	}
	handler, err := newHandler(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	res, err := handler.HandleBatch(context.Background(), events.SQSEvent{Records: []events.SQSMessage{{MessageId: "runtime", Body: `{"id":"runtime","source":"urlshortener","detail-type":"link.clicked","detail":{"code":"Abc1234","at":"2026-11-04T01:00:00Z"}}`}}})
	if err != nil || len(res.BatchItemFailures) != 0 || pushes.Load() != 1 {
		t.Fatal(res, err, pushes.Load())
	}
}
