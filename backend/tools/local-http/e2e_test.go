//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	appevents "github.com/TrieuNguyenPhu/shorten-link/backend/internal/events"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/httpapi"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/service"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// This exercises real HTTP, SDK serialization, DynamoDB and business code.
// External EventBridge and Management API services use HTTP boundary fixtures.
func TestLocalHTTPEndToEnd(t *testing.T) {
	var fail atomic.Bool
	var publications atomic.Int64
	recorder := newEventRecorder(io.Discard)
	// Explicitly hand the actual published event to the consumer. This is a local
	// protocol boundary fixture, not evidence of EventBridge/SQS/ESM delivery.
	delivered := make(chan events.SQSMessage, 1)
	eb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			w.Header().Set("Content-Type", "application/x-amz-json-1.1")
			_, _ = io.WriteString(w, `{"FailedEntryCount":1,"Entries":[{"ErrorCode":"AccessDeniedException","ErrorMessage":"test denial"}]}`)
			return
		}
		publications.Add(1)
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			http.Error(w, "read failed", 500)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(data))
		response := httptest.NewRecorder()
		recorder.ServeHTTP(response, r)
		var published struct {
			Entries []struct{ EventBusName, Source, DetailType, Detail string }
		}
		var ack struct{ Entries []struct{ EventID string } }
		if response.Code != 200 || json.Unmarshal(data, &published) != nil || len(published.Entries) != 1 || json.Unmarshal(response.Body.Bytes(), &ack) != nil || len(ack.Entries) != 1 {
			t.Error("invalid publication")
			http.Error(w, "invalid publication", 500)
			return
		}
		entry := published.Entries[0]
		envelope, err := json.Marshal(map[string]any{"id": ack.Entries[0].EventID, "source": entry.Source, "detail-type": entry.DetailType, "detail": json.RawMessage(entry.Detail)})
		if err != nil {
			t.Error(err)
			http.Error(w, "invalid detail", 500)
			return
		}
		delivered <- events.SQSMessage{MessageId: "local-message", Body: string(envelope)}
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.WriteHeader(response.Code)
		_, _ = w.Write(response.Body.Bytes())
	}))
	defer eb.Close()
	cfg := validConfig()
	cfg.DynamoDBEndpoint = os.Getenv("DYNAMODB_ENDPOINT")
	cfg.EventBridgeEndpoint = eb.URL
	cfg.TableName = fmt.Sprintf("http-e2e-%d", time.Now().UnixNano())
	if err := cfg.validate(); err != nil {
		t.Fatal(err)
	}
	ddb, publisher := localClients(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := ensureTable(ctx, ddb, cfg.TableName); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := ddb.DeleteTable(cleanup, &dynamodb.DeleteTableInput{TableName: aws.String(cfg.TableName)}); err != nil {
			t.Error(err)
		}
	})
	handler := httpapi.New(service.New(repository.New(ddb, cfg.TableName), appevents.NewPublisher(publisher, cfg.EventBus)), "http://"+cfg.Listen, slog.New(slog.NewJSONHandler(io.Discard, nil)), io.Discard)
	server := httptest.NewServer(newAdapter(handler, cfg))
	defer server.Close()
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	call := func(method, path, user, body string, status int) string {
		t.Helper()
		req, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("X-Local-User", user)
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != status {
			t.Fatalf("%s %s: got %d want %d body=%s", method, path, res.StatusCode, status, data)
		}
		if status == 302 && (res.Header.Get("Location") != "https://example.org/path" || !strings.Contains(res.Header.Get("Cache-Control"), "no-store") || len(data) != 0) {
			t.Fatal("invalid redirect", res.Header, string(data))
		}
		return string(data)
	}
	call("GET", "/links", "", "", 401)
	for _, body := range []string{`null`, `{"url":8}`, `{"url":"ftp://example.org"}`} {
		call("POST", "/links", "alice", body, 400)
	}
	created := call("POST", "/links", "alice", `{"url":"https://example.org/path","ownerId":"bob"}`, 201)
	var link struct{ Code, ShortURL, Target string }
	if err := json.Unmarshal([]byte(created), &link); err != nil || len(link.Code) != 7 || link.ShortURL != "http://127.0.0.1:8080/"+link.Code {
		t.Fatal(created, err)
	}
	if _, err := ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(cfg.TableName),
		Key:                       map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "LINK#" + link.Code}, "SK": &types.AttributeValueMemberS{Value: "META"}},
		UpdateExpression:          aws.String("SET #status = :active"),
		ExpressionAttributeNames:  map[string]string{"#status": "status"},
		ExpressionAttributeValues: map[string]types.AttributeValue{":active": &types.AttributeValueMemberS{Value: "active"}},
	}); err != nil {
		t.Fatal(err)
	}
	repo := repository.New(ddb, cfg.TableName)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	ws := service.NewWebSocket(repo, logger)
	for id, code := range map[string]string{"a-live": link.Code, "b-gone": link.Code, "c-denied": link.Code, "other-code": "Other01"} {
		ticket := "ticket-" + id
		if err := repo.PutRealtimeTicket(ctx, domain.RealtimeTicket{Ticket: ticket, OwnerID: "alice", Code: code, TTL: time.Now().Unix() + 120}); err != nil {
			t.Fatal(err)
		}
		response, err := ws.Connect(ctx, events.APIGatewayWebsocketProxyRequest{QueryStringParameters: map[string]string{"ticket": ticket}, RequestContext: events.APIGatewayWebsocketProxyRequestContext{ConnectionID: id}})
		if err != nil || response.StatusCode != 200 {
			t.Fatal(response, err)
		}
	}
	var livePushes, gonePushes, deniedPushes atomic.Int64
	management := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Code   string
			Clicks int64
		}
		if r.Method != "POST" || !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") || json.NewDecoder(r.Body).Decode(&payload) != nil || payload.Code != link.Code || payload.Clicks != 1 {
			t.Error("invalid signed push", r.Method, r.URL, payload)
			http.Error(w, "bad request", 400)
			return
		}
		switch r.URL.Path {
		case "/prod/@connections/a-live":
			livePushes.Add(1)
			w.WriteHeader(200)
		case "/prod/@connections/b-gone":
			gonePushes.Add(1)
			w.Header().Set("X-Amzn-Errortype", "GoneException")
			w.WriteHeader(410)
			_, _ = io.WriteString(w, `{"message":"gone"}`)
		case "/prod/@connections/c-denied":
			deniedPushes.Add(1)
			w.Header().Set("X-Amzn-Errortype", "ForbiddenException")
			w.WriteHeader(403)
			_, _ = io.WriteString(w, `{"message":"denied"}`)
		default:
			t.Error("push escaped code subscription", r.URL)
			w.WriteHeader(400)
		}
	}))
	defer management.Close()
	managementClient := apigatewaymanagementapi.NewFromConfig(aws.Config{Region: "ap-southeast-1", Credentials: credentials.NewStaticCredentialsProvider("local", "local", ""), Retryer: func() aws.Retryer { return aws.NopRetryer{} }}, func(o *apigatewaymanagementapi.Options) { o.BaseEndpoint = aws.String(management.URL + "/prod") })
	pusher := appevents.NewWSPusher(repo, managementClient, logger)
	if list := call("GET", "/links", "alice", "", 200); !strings.Contains(list, link.Code) || strings.Contains(list, "ownerId") {
		t.Fatal(list)
	}
	if list := call("GET", "/links", "bob", "", 200); list != `{"links":[]}` {
		t.Fatal(list)
	}
	secondJSON := call("POST", "/links", "alice", `{"url":"https://example.org/path"}`, 201)
	var second struct{ Code string }
	if err := json.Unmarshal([]byte(secondJSON), &second); err != nil {
		t.Fatal(err)
	}
	var firstPage, lastPage struct {
		Links      []struct{ Code string }
		NextCursor string
	}
	if err := json.Unmarshal([]byte(call("GET", "/links?limit=1", "alice", "", 200)), &firstPage); err != nil || len(firstPage.Links) != 1 || firstPage.NextCursor == "" {
		t.Fatal("missing cursor", err)
	}
	if err := json.Unmarshal([]byte(call("GET", "/links?limit=1&cursor="+firstPage.NextCursor, "alice", "", 200)), &lastPage); err != nil || len(lastPage.Links) != 1 || firstPage.Links[0].Code == lastPage.Links[0].Code {
		t.Fatal("invalid next page", err)
	}
	if lastPage.NextCursor != "" {
		var endPage struct {
			Links      []struct{ Code string }
			NextCursor string
		}
		if err := json.Unmarshal([]byte(call("GET", "/links?limit=1&cursor="+lastPage.NextCursor, "alice", "", 200)), &endPage); err != nil || len(endPage.Links) != 0 || endPage.NextCursor != "" {
			t.Fatal("invalid end page", err)
		}
	}
	call("GET", "/links?limit=1&cursor="+firstPage.NextCursor, "bob", "", 400)
	call("DELETE", "/links/"+second.Code, "alice", "", 200)
	denied := call("DELETE", "/links/"+link.Code, "bob", "", 404)
	missing := call("DELETE", "/links/missing", "bob", "", 404)
	if denied != missing {
		t.Fatal("ownership oracle")
	}
	call("GET", "/"+link.Code, "", "", 302)
	if publications.Load() != 1 {
		t.Fatal("click not published")
	}
	message := <-delivered
	aggregator := service.NewAggregator(repo, pusher, logger, io.Discard)
	batchResponse, err := aggregator.HandleBatch(ctx, events.SQSEvent{Records: []events.SQSMessage{message, {MessageId: "duplicate-delivery", Body: message.Body}, {MessageId: "malformed", Body: "{"}}})
	if err != nil || len(batchResponse.BatchItemFailures) != 1 || batchResponse.BatchItemFailures[0].ItemIdentifier != "malformed" {
		t.Fatal(batchResponse, err)
	}
	counted, err := repo.GetLink(ctx, link.Code)
	if err != nil || counted.Clicks != 1 {
		t.Fatal("published click not committed exactly once while marker exists", counted, err)
	}
	if livePushes.Load() != 1 || gonePushes.Load() != 1 || deniedPushes.Load() != 1 {
		t.Fatal("missing broadcast or duplicate push", livePushes.Load(), gonePushes.Load(), deniedPushes.Load())
	}
	for id, present := range map[string]bool{"a-live": true, "b-gone": false, "c-denied": true, "other-code": true} {
		out, err := ddb.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(cfg.TableName), Key: map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "CONN#" + id}, "SK": &types.AttributeValueMemberS{Value: "META"}}, ConsistentRead: aws.Bool(true)})
		if err != nil || (len(out.Item) > 0) != present {
			t.Fatal("incorrect connection cleanup", id, out, err)
		}
	}
	for range 2 {
		response, err := ws.Disconnect(ctx, events.APIGatewayWebsocketProxyRequest{RequestContext: events.APIGatewayWebsocketProxyRequestContext{ConnectionID: "a-live"}})
		if err != nil || response.StatusCode != 200 {
			t.Fatal(response, err)
		}
	}
	var event struct {
		ID     string
		Detail struct {
			ClickID string `json:"clickId"`
		} `json:"detail"`
	}
	if err := json.Unmarshal([]byte(message.Body), &event); err != nil {
		t.Fatal(err)
	}
	clickID := event.Detail.ClickID
	if clickID == "" {
		clickID = event.ID
	}
	item, err := ddb.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(cfg.TableName), Key: map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "CLICK#" + clickID}, "SK": &types.AttributeValueMemberS{Value: "CLICK"}}, ConsistentRead: aws.Bool(true)})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := item.Item["ttl"].(*types.AttributeValueMemberN); !ok {
		t.Fatal(item)
	}
	fail.Store(true)
	call("GET", "/"+link.Code, "", "", 302)
	if publications.Load() != 1 {
		t.Fatal("unexpected publish")
	}
	call("DELETE", "/links/"+link.Code, "alice", "", 200)
	call("GET", "/"+link.Code, "", "", 404)
	lateResponse, err := aggregator.HandleBatch(ctx, events.SQSEvent{Records: []events.SQSMessage{message}})
	if err != nil || len(lateResponse.BatchItemFailures) != 0 {
		t.Fatal(lateResponse, err)
	}
	if _, err := repo.GetLink(ctx, link.Code); err != repository.ErrNotFound {
		t.Fatal("late event resurrected link", err)
	}
	if list := call("GET", "/links", "alice", "", 200); list != `{"links":[]}` {
		t.Fatal(list)
	}
}
