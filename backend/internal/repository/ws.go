package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func connectionKey(id string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "CONN#" + id}, "SK": &types.AttributeValueMemberS{Value: "META"}}
}

func (r *DynamoDB) SaveConnection(ctx context.Context, c domain.WSConnection) error {
	item := connectionKey(c.ConnectionID)
	item["ownerId"] = &types.AttributeValueMemberS{Value: c.OwnerID}
	item["code"] = &types.AttributeValueMemberS{Value: c.Code}
	item["GSI1PK"] = &types.AttributeValueMemberS{Value: "WSCODE#" + c.Code}
	item["GSI1SK"] = &types.AttributeValueMemberS{Value: "CONN#" + c.ConnectionID}
	item["ttl"] = &types.AttributeValueMemberN{Value: strconv.FormatInt(c.TTL, 10)}
	_, err := r.client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(r.table), Item: item})
	return err
}

func (r *DynamoDB) DeleteConnection(ctx context.Context, id string) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: aws.String(r.table), Key: connectionKey(id)})
	return err
}

func (r *DynamoDB) GetClickCount(ctx context.Context, code string) (int64, error) {
	out, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(r.table), Key: linkKey(code), ConsistentRead: aws.Bool(true), ProjectionExpression: aws.String("clicks")})
	if err != nil {
		return 0, err
	}
	if out == nil {
		return 0, errors.New("empty DynamoDB count response")
	}
	if len(out.Item) == 0 {
		return 0, ErrNotFound
	}
	n, ok := out.Item["clicks"].(*types.AttributeValueMemberN)
	if !ok {
		return 0, errors.New("missing or invalid click count")
	}
	count, err := strconv.ParseInt(n.Value, 10, 64)
	if err != nil || count < 0 {
		return 0, fmt.Errorf("invalid click count %q", n.Value)
	}
	return count, nil
}

// ForEachConnection streams every eventual-consistent GSI page without loading
// all subscribers into memory. The caller's context bounds the entire broadcast.
func (r *DynamoDB) ForEachConnection(ctx context.Context, code string, visit func(string)) error {
	in := &dynamodb.QueryInput{TableName: aws.String(r.table), IndexName: aws.String("GSI1"), KeyConditionExpression: aws.String("GSI1PK = :code"), ExpressionAttributeValues: map[string]types.AttributeValue{":code": &types.AttributeValueMemberS{Value: "WSCODE#" + code}}, ProjectionExpression: aws.String("PK")}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		out, err := r.client.Query(ctx, in)
		if err != nil {
			return err
		}
		if out == nil {
			return errors.New("empty DynamoDB connection query response")
		}
		for _, item := range out.Items {
			if err := ctx.Err(); err != nil {
				return err
			}
			pk, ok := item["PK"].(*types.AttributeValueMemberS)
			if !ok || !strings.HasPrefix(pk.Value, "CONN#") || len(pk.Value) == len("CONN#") {
				return errors.New("invalid connection partition key")
			}
			visit(strings.TrimPrefix(pk.Value, "CONN#"))
		}
		if len(out.LastEvaluatedKey) == 0 {
			return nil
		}
		in.ExclusiveStartKey = out.LastEvaluatedKey
	}
}
