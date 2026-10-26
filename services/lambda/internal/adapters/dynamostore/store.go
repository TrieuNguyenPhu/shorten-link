package dynamostore

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"shortlink/internal/application"
	"shortlink/internal/domain"
)

const keyTime = "2006-01-02T15:04:05.000000000Z"

type Store struct {
	client *dynamodb.Client
	table  string
}

func New(client *dynamodb.Client, table string) *Store { return &Store{client: client, table: table} }

func text(value string) types.AttributeValue { return &types.AttributeValueMemberS{Value: value} }

func key(code string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{"PK": text("LINK#" + code), "SK": text("METADATA")}
}

func (s *Store) Create(ctx context.Context, link domain.Link) error {
	item, err := attributevalue.MarshalMap(link)
	if err != nil {
		return fmt.Errorf("encode link: %w", err)
	}
	item["PK"], item["SK"] = text("LINK#"+link.Code), text("METADATA")
	item["GSI1PK"], item["GSI1SK"] = text("LINKS"), text(link.CreatedAt.UTC().Format(keyTime)+"#"+link.Code)
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(s.table), Item: item, ConditionExpression: aws.String("attribute_not_exists(PK)")})
	var conflict *types.ConditionalCheckFailedException
	if errors.As(err, &conflict) {
		return domain.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("create link: %w", err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, code string) (domain.Link, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(s.table), Key: key(code), ConsistentRead: aws.Bool(true)})
	if err != nil {
		return domain.Link{}, fmt.Errorf("get link: %w", err)
	}
	if len(result.Item) == 0 {
		return domain.Link{}, domain.ErrNotFound
	}
	var link domain.Link
	if err := attributevalue.UnmarshalMap(result.Item, &link); err != nil {
		return link, fmt.Errorf("decode link: %w", err)
	}
	return link, nil
}

func (s *Store) SetEnabled(ctx context.Context, code string, enabled bool) (domain.Link, error) {
	result, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.table), Key: key(code),
		ConditionExpression: aws.String("attribute_exists(PK)"), UpdateExpression: aws.String("SET #enabled = :enabled"),
		ExpressionAttributeNames: map[string]string{"#enabled": "enabled"}, ExpressionAttributeValues: map[string]types.AttributeValue{":enabled": &types.AttributeValueMemberBOOL{Value: enabled}},
		ReturnValues: types.ReturnValueAllNew,
	})
	var missing *types.ConditionalCheckFailedException
	if errors.As(err, &missing) {
		return domain.Link{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Link{}, fmt.Errorf("update link: %w", err)
	}
	var link domain.Link
	if err := attributevalue.UnmarshalMap(result.Attributes, &link); err != nil {
		return link, fmt.Errorf("decode updated link: %w", err)
	}
	return link, nil
}

type cursorData struct {
	Code      string `json:"code"`
	CreatedAt string `json:"createdAt"`
}

func (s *Store) List(ctx context.Context, cursor string) (domain.Page, error) {
	input := &dynamodb.QueryInput{TableName: aws.String(s.table), IndexName: aws.String("GSI1"), KeyConditionExpression: aws.String("GSI1PK = :collection"), ExpressionAttributeValues: map[string]types.AttributeValue{":collection": text("LINKS")}, ScanIndexForward: aws.Bool(false), Limit: aws.Int32(25)}
	if cursor != "" {
		var data cursorData
		decoded, err := base64.RawURLEncoding.DecodeString(cursor)
		if len(cursor) > 512 || err != nil || json.Unmarshal(decoded, &data) != nil || !application.ValidCode(data.Code) {
			return domain.Page{}, fmt.Errorf("%w: malformed cursor", domain.ErrInvalid)
		}
		if _, err := time.Parse(keyTime, data.CreatedAt); err != nil {
			return domain.Page{}, fmt.Errorf("%w: malformed cursor", domain.ErrInvalid)
		}
		input.ExclusiveStartKey = key(data.Code)
		input.ExclusiveStartKey["GSI1PK"], input.ExclusiveStartKey["GSI1SK"] = text("LINKS"), text(data.CreatedAt+"#"+data.Code)
	}
	result, err := s.client.Query(ctx, input)
	if err != nil {
		return domain.Page{}, fmt.Errorf("list links: %w", err)
	}
	page := domain.Page{Links: []domain.Link{}}
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &page.Links); err != nil {
		return page, fmt.Errorf("decode links: %w", err)
	}
	if len(result.LastEvaluatedKey) != 0 {
		pk, ok := result.LastEvaluatedKey["PK"].(*types.AttributeValueMemberS)
		sk, okSort := result.LastEvaluatedKey["GSI1SK"].(*types.AttributeValueMemberS)
		if !ok || !okSort {
			return page, fmt.Errorf("list cursor is missing index keys")
		}
		code := strings.TrimPrefix(pk.Value, "LINK#")
		encoded, err := json.Marshal(cursorData{Code: code, CreatedAt: strings.TrimSuffix(sk.Value, "#"+code)})
		if err != nil {
			return page, fmt.Errorf("encode cursor: %w", err)
		}
		page.NextCursor = base64.RawURLEncoding.EncodeToString(encoded)
	}
	return page, nil
}
