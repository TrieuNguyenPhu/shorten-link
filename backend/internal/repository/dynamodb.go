package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	ErrCollision      = errors.New("short code already exists")
	ErrNotFound       = errors.New("link not found or not owned")
	ErrTicketNotFound = errors.New("realtime ticket not found or expired")
)

type LinkRepository interface {
	CreateLink(context.Context, domain.Link) error
	GetLink(context.Context, string) (domain.Link, error)
	ListUserLinks(context.Context, string, int, string) (domain.LinkPage, error)
	DeleteUserLink(context.Context, string, string) error
}

type DynamoDBAPI interface {
	TransactWriteItems(context.Context, *dynamodb.TransactWriteItemsInput, ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
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

func realtimeTicketKey(ticket string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "TICKET#" + ticket}, "SK": &types.AttributeValueMemberS{Value: "META"}}
}

func (r *DynamoDB) PutRealtimeTicket(ctx context.Context, ticket domain.RealtimeTicket) error {
	item := realtimeTicketKey(ticket.Ticket)
	item["ownerId"] = &types.AttributeValueMemberS{Value: ticket.OwnerID}
	item["code"] = &types.AttributeValueMemberS{Value: ticket.Code}
	item["ttl"] = &types.AttributeValueMemberN{Value: strconv.FormatInt(ticket.TTL, 10)}
	_, err := r.client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(r.table), Item: item, ConditionExpression: aws.String("attribute_not_exists(PK)")})
	var conditional *types.ConditionalCheckFailedException
	if errors.As(err, &conditional) {
		return ErrTicketNotFound
	}
	return err
}

func (r *DynamoDB) ConsumeRealtimeTicket(ctx context.Context, ticket string, now time.Time) (domain.RealtimeTicket, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(r.table), Key: realtimeTicketKey(ticket), ConsistentRead: aws.Bool(true)})
	if err != nil || out == nil || len(out.Item) == 0 {
		if err != nil {
			return domain.RealtimeTicket{}, err
		}
		return domain.RealtimeTicket{}, ErrTicketNotFound
	}
	owner, ownerOK := out.Item["ownerId"].(*types.AttributeValueMemberS)
	code, codeOK := out.Item["code"].(*types.AttributeValueMemberS)
	ttl, ttlOK := out.Item["ttl"].(*types.AttributeValueMemberN)
	if !ownerOK || !codeOK || !ttlOK {
		return domain.RealtimeTicket{}, ErrTicketNotFound
	}
	expires, parseErr := strconv.ParseInt(ttl.Value, 10, 64)
	if parseErr != nil || expires < now.Unix() {
		return domain.RealtimeTicket{}, ErrTicketNotFound
	}
	_, err = r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: aws.String(r.table), Key: realtimeTicketKey(ticket), ConditionExpression: aws.String("attribute_exists(PK)")})
	var conditional *types.ConditionalCheckFailedException
	if errors.As(err, &conditional) {
		return domain.RealtimeTicket{}, ErrTicketNotFound
	}
	if err != nil {
		return domain.RealtimeTicket{}, err
	}
	return domain.RealtimeTicket{Ticket: ticket, OwnerID: owner.Value, Code: code.Value, TTL: expires}, nil
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

func (r *DynamoDB) ListUserLinks(ctx context.Context, owner string, limit int, cursor string) (domain.LinkPage, error) {
	if limit < 1 || limit > 100 {
		return domain.LinkPage{}, ErrInvalidPagination
	}
	key, err := decodeCursor(owner, cursor)
	if err != nil {
		return domain.LinkPage{}, err
	}
	in := &dynamodb.QueryInput{TableName: aws.String(r.table), IndexName: aws.String("GSI1"), KeyConditionExpression: aws.String("GSI1PK = :u"), ExpressionAttributeValues: map[string]types.AttributeValue{":u": &types.AttributeValueMemberS{Value: "USER#" + owner}}, ScanIndexForward: aws.Bool(false), Limit: aws.Int32(int32(limit)), ExclusiveStartKey: key}
	out, err := r.client.Query(ctx, in)
	if err != nil {
		return domain.LinkPage{}, err
	}
	if out == nil {
		return domain.LinkPage{}, errors.New("empty DynamoDB query response")
	}
	page := domain.LinkPage{Links: make([]domain.Link, 0, len(out.Items))}
	for _, item := range out.Items {
		link, err := decodeLink(item)
		if err != nil {
			return domain.LinkPage{}, err
		}
		page.Links = append(page.Links, link)
	}
	if len(out.LastEvaluatedKey) > 0 {
		page.NextCursor, err = encodeCursor(owner, out.LastEvaluatedKey)
		if err != nil {
			return domain.LinkPage{}, err
		}
	}
	return page, nil
}

func (r *DynamoDB) DeleteUserLink(ctx context.Context, code, owner string) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: aws.String(r.table), Key: linkKey(code), ConditionExpression: aws.String("attribute_exists(PK) AND ownerId = :me"), ExpressionAttributeValues: map[string]types.AttributeValue{":me": &types.AttributeValueMemberS{Value: owner}}})
	var conditional *types.ConditionalCheckFailedException
	if errors.As(err, &conditional) {
		return ErrNotFound
	}
	return err
}
