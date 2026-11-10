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
	ticket  domain.RealtimeTicket
}

func (s *connectionStub) SaveConnection(_ context.Context, c domain.WSConnection) error {
	s.saved = c
	return s.err
}
func (s *connectionStub) DeleteConnection(_ context.Context, id string) error {
	s.deleted = id
	return s.err
}
func (s *connectionStub) ConsumeRealtimeTicket(_ context.Context, token string, _ time.Time) (domain.RealtimeTicket, error) {
	if token != s.ticket.Ticket || s.err != nil {
		return domain.RealtimeTicket{}, s.err
	}
	return s.ticket, nil
}

func TestWebSocketConnectAndDisconnect(t *testing.T) {
	repo := &connectionStub{ticket: domain.RealtimeTicket{Ticket: "ticket-1", OwnerID: "alice", Code: "Abc1234", TTL: 7300}}
	s := NewWebSocket(repo, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	s.now = func() time.Time { return time.Unix(100, 0) }
	req := events.APIGatewayWebsocketProxyRequest{RequestContext: events.APIGatewayWebsocketProxyRequestContext{ConnectionID: "socket="}}
	for _, ticket := range []string{"", "   "} {
		req.QueryStringParameters = map[string]string{"ticket": ticket}
		res, err := s.Connect(context.Background(), req)
		if err != nil || res.StatusCode != 400 || repo.saved.ConnectionID != "" {
			t.Fatal(res, err, repo.saved)
		}
	}
	req.QueryStringParameters = map[string]string{"ticket": "ticket-1"}
	res, err := s.Connect(context.Background(), req)
	if err != nil || res.StatusCode != 200 || repo.saved != (domain.WSConnection{ConnectionID: "socket=", OwnerID: "alice", Code: "Abc1234", TTL: 7300}) {
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
	req.QueryStringParameters = map[string]string{"ticket": "ticket-1"}
	for i, handle := range []func(context.Context, events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error){s.Connect, s.Disconnect} {
		res, err = handle(context.Background(), req)
		want := 500
		if i == 0 {
			want = 401
		}
		if err != nil || res.StatusCode != want {
			t.Fatal(res, err)
		}
	}
	res, err = s.Connect(context.Background(), events.APIGatewayWebsocketProxyRequest{QueryStringParameters: map[string]string{"ticket": "ticket-1"}})
	if err != nil || res.StatusCode != 400 {
		t.Fatal(res, err)
	}
	res, err = s.Disconnect(context.Background(), events.APIGatewayWebsocketProxyRequest{})
	if err != nil || res.StatusCode != 400 {
		t.Fatal(res, err)
	}
}

func TestWebSocketConnectAcceptsCustomAlias(t *testing.T) {
	repo := &connectionStub{ticket: domain.RealtimeTicket{Ticket: "ticket-1", OwnerID: "alice", Code: "my-link", TTL: 7300}}
	s := NewWebSocket(repo, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	res, err := s.Connect(context.Background(), events.APIGatewayWebsocketProxyRequest{QueryStringParameters: map[string]string{"ticket": "ticket-1"}, RequestContext: events.APIGatewayWebsocketProxyRequestContext{ConnectionID: "socket="}})
	if err != nil || res.StatusCode != 200 || repo.saved.Code != "my-link" {
		t.Fatalf("res=%+v saved=%+v err=%v", res, repo.saved, err)
	}
}
