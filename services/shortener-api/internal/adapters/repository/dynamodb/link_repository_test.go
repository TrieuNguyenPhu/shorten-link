package dynamodb

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	awstypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/domain"
)

type fakeClient struct {
	put func(context.Context, *awsdynamodb.PutItemInput) (*awsdynamodb.PutItemOutput, error)
	get func(context.Context, *awsdynamodb.GetItemInput) (*awsdynamodb.GetItemOutput, error)
}

func (f *fakeClient) PutItem(
	ctx context.Context,
	input *awsdynamodb.PutItemInput,
	_ ...func(*awsdynamodb.Options),
) (*awsdynamodb.PutItemOutput, error) {
	return f.put(ctx, input)
}

func (f *fakeClient) GetItem(
	ctx context.Context,
	input *awsdynamodb.GetItemInput,
	_ ...func(*awsdynamodb.Options),
) (*awsdynamodb.GetItemOutput, error) {
	return f.get(ctx, input)
}

func TestCreateUsesConditionalWriteAndTTL(t *testing.T) {
	createdAt := time.Date(2026, time.July, 22, 8, 30, 0, 0, time.UTC)
	expiresAt := createdAt.Add(7 * 24 * time.Hour)
	var captured *awsdynamodb.PutItemInput
	client := &fakeClient{
		put: func(_ context.Context, input *awsdynamodb.PutItemInput) (*awsdynamodb.PutItemOutput, error) {
			captured = input
			return &awsdynamodb.PutItemOutput{}, nil
		},
	}
	repository, err := NewLinkRepository(client, "links")
	if err != nil {
		t.Fatalf("NewLinkRepository() error = %v", err)
	}

	err = repository.Create(context.Background(), domain.Link{
		Code:      "abc1234",
		TargetURL: "https://example.com/guide",
		Enabled:   true,
		CreatedAt: createdAt,
		ExpiresAt: &expiresAt,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if captured == nil {
		t.Fatal("PutItem was not called")
	}
	if aws.ToString(captured.TableName) != "links" {
		t.Fatalf("table = %q", aws.ToString(captured.TableName))
	}
	if aws.ToString(captured.ConditionExpression) != "attribute_not_exists(code)" {
		t.Fatalf("condition = %q", aws.ToString(captured.ConditionExpression))
	}
	var item linkItem
	if err := attributevalue.UnmarshalMap(captured.Item, &item); err != nil {
		t.Fatalf("decode item: %v", err)
	}
	if item.Code != "abc1234" || item.TTL != expiresAt.Unix() {
		t.Fatalf("item code/ttl = %q/%d, want abc1234/%d", item.Code, item.TTL, expiresAt.Unix())
	}
}

func TestCreateMapsConditionalConflict(t *testing.T) {
	client := &fakeClient{
		put: func(context.Context, *awsdynamodb.PutItemInput) (*awsdynamodb.PutItemOutput, error) {
			return nil, &awstypes.ConditionalCheckFailedException{Message: aws.String("exists")}
		},
	}
	repository, err := NewLinkRepository(client, "links")
	if err != nil {
		t.Fatalf("NewLinkRepository() error = %v", err)
	}

	err = repository.Create(context.Background(), domain.Link{
		Code: "abc1234", TargetURL: "https://example.com", Enabled: true, CreatedAt: time.Now(),
	})
	if !errors.Is(err, domain.ErrCodeAlreadyExists) {
		t.Fatalf("Create() error = %v, want conflict", err)
	}
}

func TestGetByCodeUsesConsistentRead(t *testing.T) {
	createdAt := time.Date(2026, time.July, 22, 8, 30, 0, 0, time.UTC)
	storedItem, err := attributevalue.MarshalMap(toItem(domain.Link{
		Code: "abc1234", TargetURL: "https://example.com/guide", Enabled: true, CreatedAt: createdAt,
	}))
	if err != nil {
		t.Fatalf("encode test item: %v", err)
	}
	var captured *awsdynamodb.GetItemInput
	client := &fakeClient{
		get: func(_ context.Context, input *awsdynamodb.GetItemInput) (*awsdynamodb.GetItemOutput, error) {
			captured = input
			return &awsdynamodb.GetItemOutput{Item: storedItem}, nil
		},
	}
	repository, err := NewLinkRepository(client, "links")
	if err != nil {
		t.Fatalf("NewLinkRepository() error = %v", err)
	}

	link, err := repository.GetByCode(context.Background(), "abc1234")
	if err != nil {
		t.Fatalf("GetByCode() error = %v", err)
	}
	if captured == nil || !aws.ToBool(captured.ConsistentRead) {
		t.Fatal("GetItem did not request a consistent read")
	}
	var key map[string]string
	if err := attributevalue.UnmarshalMap(captured.Key, &key); err != nil {
		t.Fatalf("decode key: %v", err)
	}
	if key["code"] != "abc1234" {
		t.Fatalf("key code = %q", key["code"])
	}
	if link.TargetURL != "https://example.com/guide" {
		t.Fatalf("target URL = %q", link.TargetURL)
	}
}

func TestGetByCodeMapsMissingItem(t *testing.T) {
	client := &fakeClient{
		get: func(context.Context, *awsdynamodb.GetItemInput) (*awsdynamodb.GetItemOutput, error) {
			return &awsdynamodb.GetItemOutput{}, nil
		},
	}
	repository, err := NewLinkRepository(client, "links")
	if err != nil {
		t.Fatalf("NewLinkRepository() error = %v", err)
	}

	_, err = repository.GetByCode(context.Background(), "missing")
	if !errors.Is(err, domain.ErrLinkNotFound) {
		t.Fatalf("GetByCode() error = %v, want not found", err)
	}
}
