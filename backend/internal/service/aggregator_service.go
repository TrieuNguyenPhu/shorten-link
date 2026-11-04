package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/middleware"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
	"github.com/aws/aws-lambda-go/events"
)

type ClickRepository interface {
	RecordClickTransaction(context.Context, string, string, time.Time, time.Time) (repository.ClickOutcome, error)
}

// ClickNotifier reads the committed count before broadcasting it (Phase 4).
type ClickNotifier interface {
	NotifyClick(context.Context, string) error
}

type AggregatorService struct {
	repo     ClickRepository
	notifier ClickNotifier
	logger   *slog.Logger
	metrics  io.Writer
	now      func() time.Time
}

func NewAggregator(repo ClickRepository, notifier ClickNotifier, logger *slog.Logger, metrics io.Writer) *AggregatorService {
	return &AggregatorService{repo: repo, notifier: notifier, logger: logger, metrics: metrics, now: time.Now}
}

// HandleBatch returns record-level failures, leaving retry/redrive to the SQS ESM.
func (s *AggregatorService) HandleBatch(ctx context.Context, batch events.SQSEvent) (events.SQSEventResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	response := events.SQSEventResponse{BatchItemFailures: make([]events.SQSBatchItemFailure, 0)}
	logger := middleware.WithRequestID(ctx, s.logger)
	for _, record := range batch.Records {
		if err := s.process(ctx, record.Body, logger); err != nil {
			logger.ErrorContext(ctx, "click processing failed", "messageId", record.MessageId, "error", err)
			response.BatchItemFailures = append(response.BatchItemFailures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
		}
	}
	return response, nil
}

func (s *AggregatorService) process(ctx context.Context, body string, logger *slog.Logger) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var envelope struct {
		ID         string            `json:"id"`
		Source     string            `json:"source"`
		DetailType string            `json:"detail-type"`
		Detail     domain.ClickEvent `json:"detail"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		return fmt.Errorf("decode click envelope: %w", err)
	}
	if strings.TrimSpace(envelope.ID) == "" || strings.TrimSpace(envelope.Detail.Code) == "" || envelope.Source != "urlshortener" || envelope.DetailType != "link.clicked" {
		return errors.New("invalid click envelope")
	}
	at, err := time.Parse(time.RFC3339Nano, envelope.Detail.At)
	if err != nil {
		return fmt.Errorf("invalid click timestamp: %w", err)
	}
	outcome, err := s.repo.RecordClickTransaction(ctx, envelope.ID, envelope.Detail.Code, at, s.now())
	if err != nil {
		return err
	}
	switch outcome {
	case repository.ClickDuplicate:
		logger.InfoContext(ctx, "duplicate click skipped", "eventId", envelope.ID)
	case repository.ClickDeleted:
		logger.InfoContext(ctx, "click discarded after deletion", "eventId", envelope.ID, "code", envelope.Detail.Code)
		s.metric(ctx, logger, "ClickDiscardedDeleted")
	case repository.ClickApplied:
		s.metric(ctx, logger, "ClickAggregated")
		if s.notifier != nil {
			notifyCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
			err := s.notifier.NotifyClick(notifyCtx, envelope.Detail.Code)
			cancel()
			if err != nil {
				logger.ErrorContext(ctx, "click notification failed", "code", envelope.Detail.Code, "error", err)
			}
		}
	default:
		return errors.New("unknown click transaction outcome")
	}
	return nil
}

func (s *AggregatorService) metric(ctx context.Context, logger *slog.Logger, name string) {
	if err := middleware.EmitCountMetric(s.metrics, "UrlShortener", "click-aggregator", name, 1); err != nil {
		logger.ErrorContext(ctx, "metric emission failed", "error", err)
	}
}
