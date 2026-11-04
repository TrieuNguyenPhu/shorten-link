package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	ErrCollision = errors.New("short code already exists")
	ErrNotFound  = errors.New("link not found or not owned")
)

type LinkRepository interface {
	CreateLink(context.Context, domain.Link) error
	GetLink(context.Context, string) (domain.Link, error)
	ListUserLinks(context.Context, string) ([]domain.Link, error)
	DeleteUserLink(context.Context, string, string) error
}

type DynamoDBAPI interface {
	PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	DeleteItem(context.Context, *dynamodb.DeleteItemInput, ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
}

type DynamoDB struct {
	client DynamoDBAPI
	table  string
}

func New(client DynamoDBAPI, table string) *DynamoDB { return &DynamoDB{client: client, table: table} }

func linkKey(code string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "LINK#" + code}, "SK": &types.AttributeValueMemberS{Value: "META"}}
}

func (r *DynamoDB) CreateLink(ctx context.Context, link domain.Link) error {
	item, err := attributevalue.MarshalMap(link)
	if err != nil {
		return fmt.Errorf("marshal link: %w", err)
	}
	// The original table derives the code from PK, without a redundant code attribute.
	delete(item, "code")
	for k, v := range linkKey(link.Code) {
		item[k] = v
	}
	item["GSI1PK"] = &types.AttributeValueMemberS{Value: "USER#" + link.OwnerID}
	item["GSI1SK"] = &types.AttributeValueMemberS{Value: "LINK#" + link.CreatedAt}
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(r.table), Item: item, ConditionExpression: aws.String("attribute_not_exists(PK)")})
	var conditional *types.ConditionalCheckFailedException
	if errors.As(err, &conditional) {
		return ErrCollision
	}
	return err
}

func decodeLink(item map[string]types.AttributeValue) (domain.Link, error) {
	var link domain.Link
	if err := attributevalue.UnmarshalMap(item, &link); err != nil {
		return link, fmt.Errorf("decode link: %w", err)
	}
	pk, ok := item["PK"].(*types.AttributeValueMemberS)
	if !ok || !strings.HasPrefix(pk.Value, "LINK#") {
		return link, errors.New("invalid link partition key")
	}
	link.Code = strings.TrimPrefix(pk.Value, "LINK#")
	return link, nil
}

func (r *DynamoDB) GetLink(ctx context.Context, code string) (domain.Link, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(r.table), Key: linkKey(code)})
	if err != nil {
		return domain.Link{}, err
	}
	if out == nil || len(out.Item) == 0 {
		return domain.Link{}, ErrNotFound
	}
	return decodeLink(out.Item)
}

func (r *DynamoDB) ListUserLinks(ctx context.Context, owner string) ([]domain.Link, error) {
	links := make([]domain.Link, 0)
	in := &dynamodb.QueryInput{TableName: aws.String(r.table), IndexName: aws.String("GSI1"), KeyConditionExpression: aws.String("GSI1PK = :u"), ExpressionAttributeValues: map[string]types.AttributeValue{":u": &types.AttributeValueMemberS{Value: "USER#" + owner}}, ScanIndexForward: aws.Bool(false)}
	for {
		out, err := r.client.Query(ctx, in)
		if err != nil {
			return nil, err
		}
		if out == nil {
			return nil, errors.New("empty DynamoDB query response")
		}
		for _, item := range out.Items {
			link, err := decodeLink(item)
			if err != nil {
				return nil, err
			}
			links = append(links, link)
		}
		if len(out.LastEvaluatedKey) == 0 {
			return links, nil
		}
		in.ExclusiveStartKey = out.LastEvaluatedKey
	}
}

func (r *DynamoDB) DeleteUserLink(ctx context.Context, code, owner string) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: aws.String(r.table), Key: linkKey(code), ConditionExpression: aws.String("attribute_exists(PK) AND ownerId = :me"), ExpressionAttributeValues: map[string]types.AttributeValue{":me": &types.AttributeValueMemberS{Value: owner}}})
	var conditional *types.ConditionalCheckFailedException
	if errors.As(err, &conditional) {
		return ErrNotFound
	}
	return err
}
