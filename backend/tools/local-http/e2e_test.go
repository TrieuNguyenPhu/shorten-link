//go:build integration

package main

import (
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

	appevents "github.com/TrieuNguyenPhu/shorten-link/backend/internal/events"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/httpapi"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/service"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// This exercises real HTTP, SDK serialization, DynamoDB and business code.
// Only the external EventBridge service is replaced at its HTTP boundary.
func TestLocalHTTPEndToEnd(t *testing.T) {
	var fail atomic.Bool
	var publications atomic.Int64
	recorder := newEventRecorder(io.Discard)
	eb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			w.Header().Set("Content-Type", "application/x-amz-json-1.1")
			_, _ = io.WriteString(w, `{"FailedEntryCount":1,"Entries":[{"ErrorCode":"AccessDeniedException","ErrorMessage":"test denial"}]}`)
			return
		}
		publications.Add(1)
		recorder.ServeHTTP(w, r)
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
		if status == 301 && (res.Header.Get("Location") != "https://example.org/path" || len(data) != 0) {
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
	if list := call("GET", "/links", "alice", "", 200); !strings.Contains(list, link.Code) || strings.Contains(list, "ownerId") {
		t.Fatal(list)
	}
	if list := call("GET", "/links", "bob", "", 200); list != `{"links":[]}` {
		t.Fatal(list)
	}
	denied := call("DELETE", "/links/"+link.Code, "bob", "", 404)
	missing := call("DELETE", "/links/missing", "bob", "", 404)
	if denied != missing {
		t.Fatal("ownership oracle")
	}
	call("GET", "/"+link.Code, "", "", 301)
	if publications.Load() != 1 {
		t.Fatal("click not published")
	}
	fail.Store(true)
	call("GET", "/"+link.Code, "", "", 502)
	if publications.Load() != 1 {
		t.Fatal("unexpected publish")
	}
	call("DELETE", "/links/"+link.Code, "alice", "", 200)
	call("GET", "/"+link.Code, "", "", 404)
	if list := call("GET", "/links", "alice", "", 200); list != `{"links":[]}` {
		t.Fatal(list)
	}
}
