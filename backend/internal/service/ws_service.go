package service

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/middleware"
	"github.com/aws/aws-lambda-go/events"
)

type ConnectionRepository interface {
	SaveConnection(context.Context, domain.WSConnection) error
	DeleteConnection(context.Context, string) error
	ConsumeRealtimeTicket(context.Context, string, time.Time) (domain.RealtimeTicket, error)
}

type WebSocketService struct {
	repo   ConnectionRepository
	logger *slog.Logger
	now    func() time.Time
}

func NewWebSocket(repo ConnectionRepository, logger *slog.Logger) *WebSocketService {
	return &WebSocketService{repo: repo, logger: logger, now: time.Now}
}

func (s *WebSocketService) Connect(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	ticket, id := strings.TrimSpace(req.QueryStringParameters["ticket"]), strings.TrimSpace(req.RequestContext.ConnectionID)
	if ticket == "" || id == "" {
		return events.APIGatewayProxyResponse{StatusCode: 400, Body: "ticket and connectionId are required"}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	claim, err := s.repo.ConsumeRealtimeTicket(ctx, ticket, s.now())
	if err != nil || !domain.IsValidLinkCode(claim.Code) || claim.OwnerID == "" {
		if err != nil {
			middleware.WithRequestID(ctx, s.logger).WarnContext(ctx, "invalid realtime ticket", "error", err)
		}
		return events.APIGatewayProxyResponse{StatusCode: 401, Body: "realtime ticket is invalid or expired"}, nil
	}
	err = s.repo.SaveConnection(ctx, domain.WSConnection{ConnectionID: id, OwnerID: claim.OwnerID, Code: claim.Code, TTL: s.now().Unix() + 7200})
	return s.response(ctx, "save connection", id, err)
}

func (s *WebSocketService) Disconnect(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	id := req.RequestContext.ConnectionID
	if strings.TrimSpace(id) == "" {
		return events.APIGatewayProxyResponse{StatusCode: 400, Body: "connectionId is required"}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	return s.response(ctx, "delete connection", id, s.repo.DeleteConnection(ctx, id))
}

func (s *WebSocketService) response(ctx context.Context, operation, id string, err error) (events.APIGatewayProxyResponse, error) {
	if err != nil {
		middleware.WithRequestID(ctx, s.logger).ErrorContext(ctx, operation, "connectionId", id, "error", err)
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: "internal server error"}, nil
	}
	return events.APIGatewayProxyResponse{StatusCode: 200, Body: "ok"}, nil
}
