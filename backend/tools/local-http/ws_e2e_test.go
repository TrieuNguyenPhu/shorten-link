//go:build integration

package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/service"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestLocalWebSocketSubscriptionsAcrossOneMBPages(t *testing.T) {
	cfg := validConfig()
	cfg.DynamoDBEndpoint = os.Getenv("DYNAMODB_ENDPOINT")
	cfg.TableName = fmt.Sprintf("ws-e2e-%d", time.Now().UnixNano())
	if err := cfg.validate(); err != nil {
		t.Fatal(err)
	}
	ddb, _ := localClients(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := ensureTable(ctx, ddb, cfg.TableName); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := ddb.DeleteTable(cleanup, &dynamodb.DeleteTableInput{TableName: aws.String(cfg.TableName)}); err != nil {
			t.Error(err)
		}
	})
	repo := repository.New(ddb, cfg.TableName)
	ws := service.NewWebSocket(repo, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	before := time.Now().Unix()
	for i := range 6 {
		id := fmt.Sprintf("socket-%d", i)
		res, err := ws.Connect(ctx, events.APIGatewayWebsocketProxyRequest{QueryStringParameters: map[string]string{"code": "Abc1234"}, RequestContext: events.APIGatewayWebsocketProxyRequestContext{ConnectionID: id}})
		if err != nil || res.StatusCode != 200 {
			t.Fatal(res, err)
		}
		// Inflate real stored items to force DynamoDB's one-MB evaluated-page cap.
		_, err = ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{TableName: aws.String(cfg.TableName), Key: map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "CONN#" + id}, "SK": &types.AttributeValueMemberS{Value: "META"}}, UpdateExpression: aws.String("SET padding = :p"), ExpressionAttributeValues: map[string]types.AttributeValue{":p": &types.AttributeValueMemberS{Value: strings.Repeat("a", 220000)}}})
		if err != nil {
			t.Fatal(err)
		}
	}
	out, err := ddb.Query(ctx, &dynamodb.QueryInput{TableName: aws.String(cfg.TableName), IndexName: aws.String("GSI1"), KeyConditionExpression: aws.String("GSI1PK = :c"), ExpressionAttributeValues: map[string]types.AttributeValue{":c": &types.AttributeValueMemberS{Value: "WSCODE#Abc1234"}}})
	if err != nil || len(out.LastEvaluatedKey) == 0 || len(out.Items) >= 6 {
		t.Fatal("fixture did not force pagination", out, err)
	}
	seen := map[string]bool{}
	if err := repo.ForEachConnection(ctx, "Abc1234", func(id string) {
		if seen[id] {
			t.Error("duplicate subscriber", id)
		}
		seen[id] = true
	}); err != nil || len(seen) != 6 {
		t.Fatal("lost subscribers across pages", seen, err)
	}
	for i := range 6 {
		if !seen[fmt.Sprintf("socket-%d", i)] {
			t.Fatal("missing subscriber", i)
		}
	}
	item, err := ddb.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(cfg.TableName), Key: map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "CONN#socket-0"}, "SK": &types.AttributeValueMemberS{Value: "META"}}, ConsistentRead: aws.Bool(true)})
	if err != nil {
		t.Fatal(err)
	}
	var stored struct {
		Code string `dynamodbav:"code"`
		TTL  int64  `dynamodbav:"ttl"`
	}
	if err := attributevalue.UnmarshalMap(item.Item, &stored); err != nil || stored.Code != "Abc1234" || stored.TTL < before+7200 || stored.TTL > time.Now().Unix()+7200 {
		t.Fatal(stored, err)
	}
}
