package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
	"github.com/aws/aws-lambda-go/events"
)

type clickRepoStub func(context.Context, string, string, time.Time, time.Time) (repository.ClickOutcome, error)

func (r clickRepoStub) RecordClickTransaction(ctx context.Context, id, code string, at, now time.Time) (repository.ClickOutcome, error) {
	return r(ctx, id, code, at, now)
}

type notifierStub func(context.Context, string) error

func (n notifierStub) NotifyClick(ctx context.Context, code string) error { return n(ctx, code) }
func clickMessage(t *testing.T, id string) events.SQSMessage {
	t.Helper()
	body, err := json.Marshal(map[string]any{"id": id, "source": "urlshortener", "detail-type": "link.clicked", "time": "2026-11-04T01:00:00Z", "detail": map[string]string{"code": "Abc1234", "at": "2026-11-03T23:59:59.123Z"}})
	if err != nil {
		t.Fatal(err)
	}
	return events.SQSMessage{MessageId: "message-" + id, Body: string(body)}
}
func quietLogger() *slog.Logger { return slog.New(slog.NewJSONHandler(io.Discard, nil)) }

func TestAggregatorPartialBatchRetriesOnlyFailedRecords(t *testing.T) {
	seen := 0
	now := time.Date(2026, 11, 4, 1, 0, 0, 0, time.UTC)
	repo := clickRepoStub(func(ctx context.Context, id, code string, at, actualNow time.Time) (repository.ClickOutcome, error) {
		seen++
		if code != "Abc1234" || at.Format(time.RFC3339Nano) != "2026-11-03T23:59:59.123Z" || !actualNow.Equal(now) {
			t.Fatal(id, code, at, actualNow)
		}
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 4*time.Second {
			t.Fatal("missing batch deadline")
		}
		if id == "3" {
			return 0, errors.New("transaction conflict")
		}
		if id == "7" {
			return 0, errors.New("network failure")
		}
		return repository.ClickApplied, nil
	})
	pushed := 0
	svc := NewAggregator(repo, notifierStub(func(_ context.Context, code string) error {
		pushed++
		if code != "Abc1234" {
			t.Fatal(code)
		}
		return nil
	}), quietLogger(), io.Discard)
	svc.now = func() time.Time { return now }
	batch := events.SQSEvent{}
	for i := range 10 {
		batch.Records = append(batch.Records, clickMessage(t, fmt.Sprint(i)))
	}
	response, err := svc.HandleBatch(context.Background(), batch)
	want := []events.SQSBatchItemFailure{{ItemIdentifier: "message-3"}, {ItemIdentifier: "message-7"}}
	if err != nil || !reflect.DeepEqual(response.BatchItemFailures, want) || seen != 10 || pushed != 1 {
		t.Fatal(response, err, seen, pushed)
	}
}

func TestAggregatorRejectsMalformedEnvelopesBeforeWriting(t *testing.T) {
	valid := clickMessage(t, "id").Body
	invalid := []string{"{", "null", "[]", valid + " {}", strings.Replace(valid, `"id":"id"`, `"id":""`, 1), strings.Replace(valid, `"id":"id"`, `"id":null`, 1), strings.Replace(valid, `"code":"Abc1234"`, `"code":" "`, 1), strings.Replace(valid, `"source":"urlshortener"`, `"source":"other"`, 1), strings.Replace(valid, `"detail-type":"link.clicked"`, `"detail-type":"other"`, 1), strings.Replace(valid, `"at":"2026-11-03T23:59:59.123Z"`, `"at":"invalid"`, 1), strings.Replace(valid, `"at":"2026-11-03T23:59:59.123Z"`, `"at":null`, 1), strings.Replace(valid, `"code":"Abc1234"`, `"code":22`, 1)}
	svc := NewAggregator(clickRepoStub(func(context.Context, string, string, time.Time, time.Time) (repository.ClickOutcome, error) {
		t.Fatal("invalid event wrote data")
		return 0, nil
	}), nil, quietLogger(), io.Discard)
	for _, body := range invalid {
		t.Run(body, func(t *testing.T) {
			res, err := svc.HandleBatch(context.Background(), events.SQSEvent{Records: []events.SQSMessage{{MessageId: "bad", Body: body}}})
			if err != nil || len(res.BatchItemFailures) != 1 || res.BatchItemFailures[0].ItemIdentifier != "bad" {
				t.Fatal(res, err)
			}
		})
	}
}

func TestAggregatorDoesNotNotifyDuplicatesOrDeletedLinks(t *testing.T) {
	for _, outcome := range []repository.ClickOutcome{repository.ClickDuplicate, repository.ClickDeleted} {
		t.Run(fmt.Sprint(outcome), func(t *testing.T) {
			var logs, metrics bytes.Buffer
			svc := NewAggregator(clickRepoStub(func(context.Context, string, string, time.Time, time.Time) (repository.ClickOutcome, error) {
				return outcome, nil
			}), notifierStub(func(context.Context, string) error { t.Fatal("unexpected push"); return nil }), slog.New(slog.NewJSONHandler(&logs, nil)), &metrics)
			res, err := svc.HandleBatch(context.Background(), events.SQSEvent{Records: []events.SQSMessage{clickMessage(t, "id")}})
			if err != nil || len(res.BatchItemFailures) != 0 {
				t.Fatal(res, err)
			}
			if outcome == repository.ClickDeleted && (!strings.Contains(metrics.String(), `"ClickDiscardedDeleted":1`) || !strings.Contains(logs.String(), "click discarded after deletion")) {
				t.Fatal(logs.String(), metrics.String())
			}
		})
	}
}

func TestAggregatorNotificationAndMetricFailuresDoNotRetryCommittedClick(t *testing.T) {
	calls := 0
	var logs bytes.Buffer
	svc := NewAggregator(clickRepoStub(func(context.Context, string, string, time.Time, time.Time) (repository.ClickOutcome, error) {
		calls++
		return repository.ClickApplied, nil
	}), notifierStub(func(ctx context.Context, _ string) error {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 200*time.Millisecond {
			t.Fatal("notifier budget unbounded")
		}
		return errors.New("websocket unavailable")
	}), slog.New(slog.NewJSONHandler(&logs, nil)), brokenWriter{})
	res, err := svc.HandleBatch(context.Background(), events.SQSEvent{Records: []events.SQSMessage{clickMessage(t, "id")}})
	if err != nil || len(res.BatchItemFailures) != 0 || calls != 1 || !strings.Contains(logs.String(), "websocket unavailable") || !strings.Contains(logs.String(), "metric emission failed") {
		t.Fatal(res, err, calls, logs.String())
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("writer failed") }

func TestAggregatorCancelledBatchRetriesEveryUnprocessedRecord(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := NewAggregator(clickRepoStub(func(context.Context, string, string, time.Time, time.Time) (repository.ClickOutcome, error) {
		t.Fatal("cancelled batch wrote data")
		return 0, nil
	}), nil, quietLogger(), io.Discard)
	res, err := svc.HandleBatch(ctx, events.SQSEvent{Records: []events.SQSMessage{clickMessage(t, "a"), clickMessage(t, "b")}})
	if err != nil || !reflect.DeepEqual(res.BatchItemFailures, []events.SQSBatchItemFailure{{ItemIdentifier: "message-a"}, {ItemIdentifier: "message-b"}}) {
		t.Fatal(res, err)
	}
	empty, err := svc.HandleBatch(context.Background(), events.SQSEvent{})
	encoded, _ := json.Marshal(empty)
	if err != nil || string(encoded) != `{"batchItemFailures":[]}` {
		t.Fatal(string(encoded), err)
	}
}
