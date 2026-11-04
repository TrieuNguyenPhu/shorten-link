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

// RecordClickTransaction keeps META, daily STAT and the finite-lived marker atomic.
// The UTC event day determines STAT; processing time determines the marker's TTL.
func (r *DynamoDB) RecordClickTransaction(ctx context.Context, eventID, code string, at, now time.Time) (ClickOutcome, error) {
	one := map[string]types.AttributeValue{":one": &types.AttributeValueMemberN{Value: "1"}}
	out, err := r.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: []types.TransactWriteItem{
		{Update: &types.Update{TableName: aws.String(r.table), Key: linkKey(code), UpdateExpression: aws.String("ADD clicks :one"), ConditionExpression: aws.String("attribute_exists(PK)"), ExpressionAttributeValues: one}},
		{Update: &types.Update{TableName: aws.String(r.table), Key: map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "LINK#" + code}, "SK": &types.AttributeValueMemberS{Value: "STAT#" + at.UTC().Format("2006-01-02")}}, UpdateExpression: aws.String("ADD #c :one"), ExpressionAttributeNames: map[string]string{"#c": "count"}, ExpressionAttributeValues: one}},
		{Put: &types.Put{TableName: aws.String(r.table), Item: map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "CLICK#" + eventID}, "SK": &types.AttributeValueMemberS{Value: "CLICK"}, "ttl": &types.AttributeValueMemberN{Value: strconv.FormatInt(now.Unix()+86400, 10)}}, ConditionExpression: aws.String("attribute_not_exists(PK)")}},
	}})
	if err == nil {
		if out == nil {
			return 0, errors.New("empty DynamoDB transaction response")
		}
		return ClickApplied, nil
	}
	var cancelled *types.TransactionCanceledException
	if !errors.As(err, &cancelled) || cancelled == nil || len(cancelled.CancellationReasons) != 3 {
		return 0, err
	}
	reasons := cancelled.CancellationReasons
	// Nil/empty/unknown codes must never be inferred to mean None.
	meta, stat, marker := aws.ToString(reasons[0].Code), aws.ToString(reasons[1].Code), aws.ToString(reasons[2].Code)
	if meta == "None" && stat == "None" && marker == "ConditionalCheckFailed" {
		return ClickDuplicate, nil
	}
	if meta == "ConditionalCheckFailed" && stat == "None" && (marker == "None" || marker == "ConditionalCheckFailed") {
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
