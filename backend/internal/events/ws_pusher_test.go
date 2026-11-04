package events

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/service"
	lambdaevents "github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi/types"
)

type pushRepository struct {
	getErr, queryErr, deleteErr error
	deleted                     []string
}

type committedPushRepository struct {
	pushRepository
	commits int
}

func (s *committedPushRepository) RecordClickTransaction(context.Context, string, string, time.Time, time.Time) (repository.ClickOutcome, error) {
	s.commits++
	return repository.ClickApplied, nil
}

func TestAggregatorAcknowledgesCommittedRecordDespitePusherFailures(t *testing.T) {
	failure := errors.New("dependency unavailable")
	for _, tc := range []struct {
		name string
		repo pushRepository
	}{
		{"get", pushRepository{getErr: failure}},
		{"query", pushRepository{queryErr: failure}},
		{"post", pushRepository{}},
		{"cleanup", pushRepository{deleteErr: failure}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &committedPushRepository{pushRepository: tc.repo}
			var logs bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logs, nil))
			aggregator := service.NewAggregator(repo, NewWSPusher(repo, &managementStub{}, logger), logger, &bytes.Buffer{})
			response, err := aggregator.HandleBatch(context.Background(), lambdaevents.SQSEvent{Records: []lambdaevents.SQSMessage{{MessageId: "committed", Body: `{"id":"click","source":"urlshortener","detail-type":"link.clicked","detail":{"code":"Abc1234","at":"2026-11-04T01:00:00Z"}}`}}})
			if err != nil || len(response.BatchItemFailures) != 0 || repo.commits != 1 || !bytes.Contains(logs.Bytes(), []byte("click notification failed")) {
				t.Fatal(response, err, repo.commits, logs.String())
			}
		})
	}
}

func (s *pushRepository) GetClickCount(context.Context, string) (int64, error) { return 42, s.getErr }
func (s *pushRepository) ForEachConnection(_ context.Context, _ string, visit func(string)) error {
	if s.queryErr != nil {
		return s.queryErr
	}
	for _, id := range []string{"gone", "failed", "live"} {
		visit(id)
	}
	return nil
}
func (s *pushRepository) DeleteConnection(_ context.Context, id string) error {
	s.deleted = append(s.deleted, id)
	return s.deleteErr
}

type managementStub struct{ ids []string }

func (s *managementStub) PostToConnection(_ context.Context, in *apigatewaymanagementapi.PostToConnectionInput, _ ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error) {
	id := aws.ToString(in.ConnectionId)
	s.ids = append(s.ids, id)
	var message struct {
		Code   string `json:"code"`
		Clicks int64  `json:"clicks"`
	}
	if err := json.Unmarshal(in.Data, &message); err != nil || message.Code != "Abc1234" || message.Clicks != 42 {
		return nil, errors.New("wrong payload")
	}
	switch id {
	case "gone":
		return nil, fmt.Errorf("wrapped: %w", &types.GoneException{})
	case "failed":
		return nil, errors.New("network failed")
	}
	return &apigatewaymanagementapi.PostToConnectionOutput{}, nil
}
func TestPusherCleansGoneLogsFailuresAndContinues(t *testing.T) {
	for _, cleanupErr := range []error{nil, errors.New("cleanup denied")} {
		repo := &pushRepository{deleteErr: cleanupErr}
		client := &managementStub{}
		var logs bytes.Buffer
		p := NewWSPusher(repo, client, slog.New(slog.NewJSONHandler(&logs, nil)))
		err := p.NotifyClick(context.Background(), "Abc1234")
		if err == nil || !reflect.DeepEqual(client.ids, []string{"gone", "failed", "live"}) || !reflect.DeepEqual(repo.deleted, []string{"gone"}) {
			t.Fatal(err, client.ids, repo.deleted)
		}
		if !bytes.Contains(logs.Bytes(), []byte("failed")) || cleanupErr != nil && !bytes.Contains(logs.Bytes(), []byte("cleanup denied")) {
			t.Fatal(logs.String())
		}
	}
}
func TestPusherPropagatesReadFailuresBeforeDelivery(t *testing.T) {
	failure := errors.New("read failed")
	for _, repo := range []*pushRepository{{getErr: failure}, {queryErr: failure}} {
		client := &managementStub{}
		var logs bytes.Buffer
		p := NewWSPusher(repo, client, slog.New(slog.NewJSONHandler(&logs, nil)))
		if err := p.NotifyClick(context.Background(), "Abc1234"); !errors.Is(err, failure) || len(client.ids) != 0 {
			t.Fatal(err, client.ids)
		}
	}
}
