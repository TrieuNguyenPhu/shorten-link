package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type transactionStub struct {
	sdkStub
	transact func(*dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error)
}

func (s transactionStub) TransactWriteItems(_ context.Context, in *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	return s.transact(in)
}

func TestRecordClickWritesAtomicCountersAndMarker(t *testing.T) {
	now := time.Date(2026, 11, 4, 1, 0, 0, 0, time.UTC)
	at := time.Date(2026, 11, 4, 1, 0, 0, 0, time.FixedZone("UTC+7", 7*3600))
	r := New(transactionStub{transact: func(in *dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error) {
		if len(in.TransactItems) != 3 {
			t.Fatal(in)
		}
		meta, stat, marker := in.TransactItems[0].Update, in.TransactItems[1].Update, in.TransactItems[2].Put
		if meta == nil || stat == nil || marker == nil {
			t.Fatal("transaction order")
		}
		for _, table := range []*string{meta.TableName, stat.TableName, marker.TableName} {
			if aws.ToString(table) != "url-shortener" {
				t.Fatal("wrong table")
			}
		}
		if str(meta.Key["PK"]) != "LINK#Abc1234" || str(meta.Key["SK"]) != "META" || aws.ToString(meta.ConditionExpression) != "attribute_exists(PK)" || aws.ToString(meta.UpdateExpression) != "ADD clicks :one" || meta.ExpressionAttributeValues[":one"].(*types.AttributeValueMemberN).Value != "1" {
			t.Fatal(meta)
		}
		if str(stat.Key["PK"]) != "LINK#Abc1234" || str(stat.Key["SK"]) != "STAT#2026-11-03" || aws.ToString(stat.UpdateExpression) != "ADD #c :one" || stat.ExpressionAttributeNames["#c"] != "count" || stat.ExpressionAttributeValues[":one"].(*types.AttributeValueMemberN).Value != "1" {
			t.Fatal(stat)
		}
		if str(marker.Item["PK"]) != "CLICK#event-1" || str(marker.Item["SK"]) != "CLICK" || marker.Item["ttl"].(*types.AttributeValueMemberN).Value != "1793840400" || len(marker.Item) != 3 || aws.ToString(marker.ConditionExpression) != "attribute_not_exists(PK)" {
			t.Fatal(marker)
		}
		return &dynamodb.TransactWriteItemsOutput{}, nil
	}}, "url-shortener")
	outcome, err := r.RecordClickTransaction(context.Background(), "event-1", "Abc1234", at, now)
	if err != nil || outcome != ClickApplied {
		t.Fatal(outcome, err)
	}
}

func cancellation(codes ...string) error {
	reasons := make([]types.CancellationReason, len(codes))
	for i, c := range codes {
		if c != "<nil>" {
			reasons[i].Code = aws.String(c)
		}
	}
	return fmt.Errorf("wrapped: %w", &types.TransactionCanceledException{CancellationReasons: reasons})
}

func TestRecordClickClassifiesEveryCancellationReason(t *testing.T) {
	cases := []struct {
		name    string
		codes   []string
		missing bool
		readErr bool
		want    ClickOutcome
		retry   bool
		reads   int
	}{
		{"duplicate", []string{"None", "None", "ConditionalCheckFailed"}, false, false, ClickDuplicate, false, 0},
		{"deleted", []string{"ConditionalCheckFailed", "None", "None"}, true, false, ClickDeleted, false, 1},
		{"deleted duplicate", []string{"ConditionalCheckFailed", "None", "ConditionalCheckFailed"}, true, false, ClickDeleted, false, 1},
		{"meta exists", []string{"ConditionalCheckFailed", "None", "None"}, false, false, 0, true, 1},
		{"get error", []string{"ConditionalCheckFailed", "None", "None"}, true, true, 0, true, 1},
		{"mixed marker", []string{"TransactionConflict", "None", "ConditionalCheckFailed"}, false, false, 0, true, 0},
		{"stat conflict", []string{"None", "TransactionConflict", "ConditionalCheckFailed"}, false, false, 0, true, 0},
		{"deleted mixed", []string{"ConditionalCheckFailed", "TransactionConflict", "None"}, true, false, 0, true, 0},
		{"deleted throttle", []string{"ConditionalCheckFailed", "None", "ThrottlingError"}, true, false, 0, true, 0},
		{"all none", []string{"None", "None", "None"}, false, false, 0, true, 0},
		{"empty", nil, false, false, 0, true, 0},
		{"short", []string{"None", "None"}, false, false, 0, true, 0},
		{"long", []string{"None", "None", "ConditionalCheckFailed", "None"}, false, false, 0, true, 0},
		{"nil code", []string{"<nil>", "None", "ConditionalCheckFailed"}, false, false, 0, true, 0},
		{"empty code", []string{"None", "", "ConditionalCheckFailed"}, false, false, 0, true, 0},
		{"unknown", []string{"FutureReason", "None", "ConditionalCheckFailed"}, false, false, 0, true, 0},
		{"throttle", []string{"ProvisionedThroughputExceeded", "None", "None"}, false, false, 0, true, 0},
		{"stat condition", []string{"None", "ConditionalCheckFailed", "ConditionalCheckFailed"}, false, false, 0, true, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reads := 0
			r := New(transactionStub{sdkStub: sdkStub{get: func(in *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
				reads++
				if !aws.ToBool(in.ConsistentRead) || str(in.Key["PK"]) != "LINK#Abc1234" || str(in.Key["SK"]) != "META" || aws.ToString(in.TableName) != "table" {
					t.Fatal("unsafe deletion read", in)
				}
				if tc.readErr {
					return nil, errors.New("read failed")
				}
				if tc.missing {
					return &dynamodb.GetItemOutput{}, nil
				}
				return &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "LINK#Abc1234"}}}, nil
			}}, transact: func(*dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error) {
				return nil, cancellation(tc.codes...)
			}}, "table")
			outcome, err := r.RecordClickTransaction(context.Background(), "id", "Abc1234", time.Now(), time.Now())
			if (err != nil) != tc.retry || (!tc.retry && outcome != tc.want) || reads != tc.reads {
				t.Fatalf("outcome=%v err=%v reads=%d", outcome, err, reads)
			}
		})
	}
}

func TestRecordClickPropagatesUncertainFailures(t *testing.T) {
	for _, failure := range []error{errors.New("network timeout"), &types.TransactionConflictException{}, &types.ThrottlingException{}} {
		r := New(transactionStub{transact: func(*dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error) {
			return nil, failure
		}}, "table")
		if _, err := r.RecordClickTransaction(context.Background(), "id", "code", time.Now(), time.Now()); !errors.Is(err, failure) {
			t.Fatal(err)
		}
	}
	for _, cancelled := range []bool{false, true} {
		r := New(transactionStub{sdkStub: sdkStub{get: func(*dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) { return nil, nil }}, transact: func(*dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error) {
			if cancelled {
				return nil, cancellation("ConditionalCheckFailed", "None", "None")
			}
			return nil, nil
		}}, "table")
		if _, err := r.RecordClickTransaction(context.Background(), "id", "code", time.Now(), time.Now()); err == nil {
			t.Fatal("nil response acknowledged")
		}
	}
}
