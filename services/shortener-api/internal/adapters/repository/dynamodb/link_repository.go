package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	awstypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/TrieuNguyenPhu/shorten-link/services/shortener-api/internal/domain"
)

type client interface {
	PutItem(
		ctx context.Context,
		params *awsdynamodb.PutItemInput,
		optFns ...func(*awsdynamodb.Options),
	) (*awsdynamodb.PutItemOutput, error)
	GetItem(
		ctx context.Context,
		params *awsdynamodb.GetItemInput,
		optFns ...func(*awsdynamodb.Options),
	) (*awsdynamodb.GetItemOutput, error)
}

type LinkRepository struct {
	client    client
	tableName string
}

type linkItem struct {
	Code      string `dynamodbav:"code"`
	TargetURL string `dynamodbav:"target_url"`
	Enabled   bool   `dynamodbav:"enabled"`
	CreatedAt string `dynamodbav:"created_at"`
	ExpiresAt string `dynamodbav:"expires_at,omitempty"`
	TTL       int64  `dynamodbav:"ttl,omitempty"`
}

func NewLinkRepository(client client, tableName string) (*LinkRepository, error) {
	if client == nil {
		return nil, errors.New("DynamoDB client is required")
	}
	if tableName == "" {
		return nil, errors.New("DynamoDB table name is required")
	}
	return &LinkRepository{client: client, tableName: tableName}, nil
}

func (r *LinkRepository) Create(ctx context.Context, link domain.Link) error {
	item, err := attributevalue.MarshalMap(toItem(link))
	if err != nil {
		return fmt.Errorf("marshal link: %w", err)
	}

	_, err = r.client.PutItem(ctx, &awsdynamodb.PutItemInput{
		TableName:           aws.String(r.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(code)"),
	})
	if err == nil {
		return nil
	}

	var conflict *awstypes.ConditionalCheckFailedException
	if errors.As(err, &conflict) {
		return domain.ErrCodeAlreadyExists
	}
	return fmt.Errorf("put link item: %w", err)
}

func (r *LinkRepository) GetByCode(ctx context.Context, code string) (domain.Link, error) {
	key, err := attributevalue.MarshalMap(map[string]string{
		"code": code,
	})
	if err != nil {
		return domain.Link{}, fmt.Errorf("marshal link key: %w", err)
	}

	result, err := r.client.GetItem(ctx, &awsdynamodb.GetItemInput{
		TableName:      aws.String(r.tableName),
		Key:            key,
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return domain.Link{}, fmt.Errorf("get link item: %w", err)
	}
	if result == nil || len(result.Item) == 0 {
		return domain.Link{}, domain.ErrLinkNotFound
	}

	var item linkItem
	if err := attributevalue.UnmarshalMap(result.Item, &item); err != nil {
		return domain.Link{}, fmt.Errorf("unmarshal link: %w", err)
	}
	return fromItem(item)
}

func toItem(link domain.Link) linkItem {
	item := linkItem{
		Code:      link.Code,
		TargetURL: link.TargetURL,
		Enabled:   link.Enabled,
		CreatedAt: link.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if link.ExpiresAt != nil {
		item.ExpiresAt = link.ExpiresAt.UTC().Format(time.RFC3339Nano)
		item.TTL = link.ExpiresAt.Unix()
	}
	return item
}

func fromItem(item linkItem) (domain.Link, error) {
	createdAt, err := time.Parse(time.RFC3339Nano, item.CreatedAt)
	if err != nil {
		return domain.Link{}, fmt.Errorf("parse created_at: %w", err)
	}

	var expiresAt *time.Time
	if item.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339Nano, item.ExpiresAt)
		if err != nil {
			return domain.Link{}, fmt.Errorf("parse expires_at: %w", err)
		}
		expiresAt = &parsed
	}

	return domain.Link{
		Code:      item.Code,
		TargetURL: item.TargetURL,
		Enabled:   item.Enabled,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}, nil
}
