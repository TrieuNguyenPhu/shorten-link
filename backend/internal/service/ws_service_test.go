package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/aws/aws-lambda-go/events"
)

type connectionStub struct {
	saved   domain.WSConnection
	deleted string
	err     error
}

func (s *connectionStub) SaveConnection(_ context.Context, c domain.WSConnection) error {
	s.saved = c
	return s.err
}
func (s *connectionStub) DeleteConnection(_ context.Context, id string) error {
	s.deleted = id
	return s.err
}

func TestWebSocketConnectAndDisconnect(t *testing.T) {
	repo := &connectionStub{}
	s := NewWebSocket(repo, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	s.now = func() time.Time { return time.Unix(100, 0) }
	req := events.APIGatewayWebsocketProxyRequest{RequestContext: events.APIGatewayWebsocketProxyRequestContext{ConnectionID: "socket="}}
	for _, code := range []string{"", "   "} {
		req.QueryStringParameters = map[string]string{"code": code}
		res, err := s.Connect(context.Background(), req)
		if err != nil || res.StatusCode != 400 || repo.saved.ConnectionID != "" {
			t.Fatal(res, err, repo.saved)
		}
	}
	req.QueryStringParameters["code"] = "Abc1234"
	res, err := s.Connect(context.Background(), req)
	if err != nil || res.StatusCode != 200 || repo.saved != (domain.WSConnection{ConnectionID: "socket=", Code: "Abc1234", TTL: 7300}) {
		t.Fatal(res, err, repo.saved)
	}
	// Disconnect only has a connection ID; it must not require the subscription code.
	req.QueryStringParameters = nil
	res, err = s.Disconnect(context.Background(), req)
	if err != nil || res.StatusCode != 200 || repo.deleted != "socket=" {
		t.Fatal(res, err, repo.deleted)
	}
	// Repeated disconnect is safe, and persistence errors do not report success.
	res, err = s.Disconnect(context.Background(), req)
	if err != nil || res.StatusCode != 200 {
		t.Fatal(res, err)
	}
	repo.err = errors.New("unavailable")
	req.QueryStringParameters = map[string]string{"code": "Abc1234"}
	for _, handle := range []func(context.Context, events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error){s.Connect, s.Disconnect} {
		res, err = handle(context.Background(), req)
		if err != nil || res.StatusCode != 500 {
			t.Fatal(res, err)
		}
	}
	res, err = s.Connect(context.Background(), events.APIGatewayWebsocketProxyRequest{QueryStringParameters: map[string]string{"code": "Abc1234"}})
	if err != nil || res.StatusCode != 400 {
		t.Fatal(res, err)
	}
	res, err = s.Disconnect(context.Background(), events.APIGatewayWebsocketProxyRequest{})
	if err != nil || res.StatusCode != 400 {
		t.Fatal(res, err)
	}
}
