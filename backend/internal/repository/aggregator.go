package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type ClickOutcome int

const (
	ClickApplied ClickOutcome = iota
	ClickDuplicate
	ClickDeleted
)

// RecordClickTransaction keeps META and the finite-lived idempotency marker atomic.
func (r *DynamoDB) RecordClickTransaction(ctx context.Context, eventID, code string, _, now time.Time) (ClickOutcome, error) {
	one := map[string]types.AttributeValue{":one": &types.AttributeValueMemberN{Value: "1"}}
	out, err := r.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: []types.TransactWriteItem{
		{Update: &types.Update{TableName: aws.String(r.table), Key: linkKey(code), UpdateExpression: aws.String("ADD clicks :one"), ConditionExpression: aws.String("attribute_exists(PK)"), ExpressionAttributeValues: one}},
		{Put: &types.Put{TableName: aws.String(r.table), Item: map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "CLICK#" + eventID}, "SK": &types.AttributeValueMemberS{Value: "CLICK"}, "ttl": &types.AttributeValueMemberN{Value: strconv.FormatInt(now.Unix()+15*24*60*60, 10)}}, ConditionExpression: aws.String("attribute_not_exists(PK)")}},
	}})
	if err == nil {
		if out == nil {
			return 0, errors.New("empty DynamoDB transaction response")
		}
		return ClickApplied, nil
	}
	var cancelled *types.TransactionCanceledException
	if !errors.As(err, &cancelled) || cancelled == nil || len(cancelled.CancellationReasons) != 2 {
		return 0, err
	}
	reasons := cancelled.CancellationReasons
	// Nil/empty/unknown codes must never be inferred to mean None.
	meta, marker := aws.ToString(reasons[0].Code), aws.ToString(reasons[1].Code)
	if meta == "None" && marker == "ConditionalCheckFailed" {
		return ClickDuplicate, nil
	}
	if meta == "ConditionalCheckFailed" && (marker == "None" || marker == "ConditionalCheckFailed") {
		existing, readErr := r.client.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(r.table), Key: linkKey(code), ConsistentRead: aws.Bool(true)})
		if readErr != nil {
			return 0, fmt.Errorf("confirm deleted link: %w", readErr)
		}
		if existing == nil {
			return 0, errors.New("empty DynamoDB deletion read response")
		}
		if len(existing.Item) == 0 {
			return ClickDeleted, nil
		}
	}
	return 0, err
}
