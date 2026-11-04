package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/httpapi"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/service"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type noSDK struct{}

func (noSDK) PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	panic("unexpected storage call")
}

func (noSDK) GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	panic("unexpected storage call")
}

func (noSDK) Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	panic("unexpected storage call")
}

func (noSDK) DeleteItem(context.Context, *dynamodb.DeleteItemInput, ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	panic("unexpected storage call")
}

func validConfig() localConfig {
	return localConfig{Listen: "127.0.0.1:8080", DynamoDBEndpoint: "http://127.0.0.1:8000", EventBridgeEndpoint: "http://127.0.0.1:8010", TableName: "url-shortener-local", EventBus: "local-events", Users: []string{"alice", "bob"}}
}

func TestConfigRejectsNonlocalSurfaces(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*localConfig)
	}{
		{"listen public", func(c *localConfig) { c.Listen = "0.0.0.0:8080" }},
		{"ddb remote", func(c *localConfig) { c.DynamoDBEndpoint = "https://dynamodb.ap-southeast-1.amazonaws.com" }},
		{"event remote", func(c *localConfig) { c.EventBridgeEndpoint = "https://events.ap-southeast-1.amazonaws.com" }},
		{"endpoint credentials", func(c *localConfig) { c.EventBridgeEndpoint = "http://user:pass@127.0.0.1:8010" }},
		{"endpoint query", func(c *localConfig) { c.DynamoDBEndpoint += "?query=x" }},
		{"no identities", func(c *localConfig) { c.Users = nil }},
		{"no bus", func(c *localConfig) { c.EventBus = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := validConfig()
			tc.mutate(&c)
			if err := c.validate(); err == nil {
				t.Fatal("unsafe config accepted")
			}
		})
	}
	if err := validConfig().validate(); err != nil {
		t.Fatal(err)
	}
}

func TestAdapterRejectsUntrustedIdentityAndInvalidRoutes(t *testing.T) {
	h := httpapi.New(service.New(repository.New(noSDK{}, "local"), nil), "http://127.0.0.1:8080", slog.New(slog.NewJSONHandler(io.Discard, nil)), io.Discard)
	adapter := newAdapter(h, validConfig())
	for _, tc := range []struct {
		method, path, user, body string
		status                   int
	}{
		{"GET", "/links", "", "", 401},
		{"GET", "/links?limit=0", "alice", "", 400}, {"GET", "/links", "mallory", "", 401},
		{"POST", "/links", "alice", `null`, 400}, {"PATCH", "/links", "alice", "", 404},
		{"GET", "/unknown/path", "alice", "", 404}, {"POST", "/links", "alice", strings.Repeat("x", 20000), 400},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		req.Header.Set("X-Local-User", tc.user)
		req.Header.Set("Authorization", "Bearer unverified")
		res := httptest.NewRecorder()
		adapter.ServeHTTP(res, req)
		if res.Code != tc.status {
			t.Fatalf("%s %s: %d %s", tc.method, tc.path, res.Code, res.Body)
		}
	}
}

func TestRecorderAcknowledgesOnlyValidPutEvents(t *testing.T) {
	var captured strings.Builder
	recorder := newEventRecorder(&captured)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"Entries":[{"EventBusName":"local-events","Source":"urlshortener","DetailType":"link.clicked","Detail":"{\"code\":\"Abc1234\",\"at\":\"2026-11-04T00:00:00.000Z\"}"}]}`))
	req.Header.Set("X-Amz-Target", "AWSEvents.PutEvents")
	res := httptest.NewRecorder()
	recorder.ServeHTTP(res, req)
	var out struct {
		Entries []struct {
			EventID string `json:"EventId"`
		}
		FailedEntryCount int
	}
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil || res.Code != 200 || len(out.Entries) != 1 || out.Entries[0].EventID == "" || out.FailedEntryCount != 0 || !strings.Contains(captured.String(), "Abc1234") {
		t.Fatal(res.Body, err)
	}
	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	res = httptest.NewRecorder()
	recorder.ServeHTTP(res, req)
	if res.Code < 400 {
		t.Fatal("invalid operation accepted")
	}
}
