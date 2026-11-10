//go:build integration

package repository

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func clickIntegrationTable(t *testing.T) (context.Context, *dynamodb.Client, string, *DynamoDB) {
	t.Helper()
	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "http" || net.ParseIP(u.Hostname()) == nil || !net.ParseIP(u.Hostname()).IsLoopback() {
		t.Fatal("DYNAMODB_ENDPOINT must be an explicit loopback HTTP endpoint")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	client := dynamodb.NewFromConfig(aws.Config{Region: "ap-southeast-1", Credentials: credentials.NewStaticCredentialsProvider("local", "local", ""), Retryer: func() aws.Retryer { return aws.NopRetryer{} }}, func(o *dynamodb.Options) { o.BaseEndpoint = aws.String(endpoint) })
	table := fmt.Sprintf("click-integration-%d", time.Now().UnixNano())
	_, err = client.CreateTable(ctx, &dynamodb.CreateTableInput{TableName: aws.String(table), BillingMode: types.BillingModePayPerRequest, AttributeDefinitions: []types.AttributeDefinition{{AttributeName: aws.String("PK"), AttributeType: types.ScalarAttributeTypeS}, {AttributeName: aws.String("SK"), AttributeType: types.ScalarAttributeTypeS}}, KeySchema: []types.KeySchemaElement{{AttributeName: aws.String("PK"), KeyType: types.KeyTypeHash}, {AttributeName: aws.String("SK"), KeyType: types.KeyTypeRange}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := client.DeleteTable(cleanup, &dynamodb.DeleteTableInput{TableName: aws.String(table)}); err != nil {
			t.Error(err)
		}
	})
	if err := dynamodb.NewTableExistsWaiter(client).Wait(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(table)}, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	return ctx, client, table, New(client, table)
}
func readClickItem(t *testing.T, ctx context.Context, client *dynamodb.Client, table, pk, sk string) map[string]types.AttributeValue {
	t.Helper()
	out, err := client.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(table), Key: map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: pk}, "SK": &types.AttributeValueMemberS{Value: sk}}, ConsistentRead: aws.Bool(true)})
	if err != nil {
		t.Fatal(err)
	}
	return out.Item
}
func assertClickNumber(t *testing.T, item map[string]types.AttributeValue, name, want string) {
	t.Helper()
	n, ok := item[name].(*types.AttributeValueMemberN)
	if !ok || n.Value != want {
		t.Fatalf("%s want %s got %+v", name, want, item)
	}
}
func TestDynamoDBLocalClickAtomicityTTLAndReplay(t *testing.T) {
	ctx, client, table, repo := clickIntegrationTable(t)
	now := time.Date(2026, 11, 4, 1, 0, 0, 0, time.UTC)
	at := time.Date(2026, 11, 4, 1, 0, 0, 0, time.FixedZone("UTC+7", 7*3600))
	if err := repo.CreateLink(ctx, domain.Link{Code: "Abc1234", OwnerID: "alice", Target: "https://example.org", CreatedAt: domain.FormatCreatedAt(now)}); err != nil {
		t.Fatal(err)
	}
	record := func(id string, want ClickOutcome) {
		t.Helper()
		got, err := repo.RecordClickTransaction(ctx, id, "Abc1234", at, now)
		if err != nil || got != want {
			t.Fatal(got, err)
		}
	}
	record("event-1", ClickApplied)
	record("event-1", ClickDuplicate)
	assertClickNumber(t, readClickItem(t, ctx, client, table, "LINK#Abc1234", "META"), "clicks", "1")
	marker := readClickItem(t, ctx, client, table, "CLICK#event-1", "CLICK")
	assertClickNumber(t, marker, "ttl", "1795050000")
	if marker["GSI1PK"] != nil || marker["GSI1SK"] != nil {
		t.Fatal("marker must stay sparse", marker)
	}
	// TTL is asynchronous: an expired item that is still present must still dedupe.
	_, err := client.UpdateItem(ctx, &dynamodb.UpdateItemInput{TableName: aws.String(table), Key: map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "CLICK#event-1"}, "SK": &types.AttributeValueMemberS{Value: "CLICK"}}, UpdateExpression: aws.String("SET #ttl = :expired"), ExpressionAttributeNames: map[string]string{"#ttl": "ttl"}, ExpressionAttributeValues: map[string]types.AttributeValue{":expired": &types.AttributeValueMemberN{Value: "1"}}})
	if err != nil {
		t.Fatal(err)
	}
	record("event-1", ClickDuplicate)
	_, err = client.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: aws.String(table), Key: map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "CLICK#event-1"}, "SK": &types.AttributeValueMemberS{Value: "CLICK"}}})
	if err != nil {
		t.Fatal(err)
	}
	record("event-1", ClickApplied)
	assertClickNumber(t, readClickItem(t, ctx, client, table, "LINK#Abc1234", "META"), "clicks", "2")
	if err := repo.DeleteUserLink(ctx, "Abc1234", "alice"); err != nil {
		t.Fatal(err)
	}
	record("event-1", ClickDeleted)
	record("late-event", ClickDeleted)
	if len(readClickItem(t, ctx, client, table, "LINK#Abc1234", "META")) != 0 || len(readClickItem(t, ctx, client, table, "CLICK#late-event", "CLICK")) != 0 {
		t.Fatal("late click resurrected metadata/marker")
	}
	// A link that never existed must not create a marker either.
	outcome, err := repo.RecordClickTransaction(ctx, "never-event", "Missing", now, now)
	if err != nil || outcome != ClickDeleted || len(readClickItem(t, ctx, client, table, "CLICK#never-event", "CLICK")) != 0 {
		t.Fatal(outcome, err)
	}
}

// Retry only transaction contention, as a later SQS delivery would; never hide validation failures.
func recordWithContentionRetry(ctx context.Context, repo *DynamoDB, id, code string) (ClickOutcome, error) {
	for range 50 {
		outcome, err := repo.RecordClickTransaction(ctx, id, code, time.Now(), time.Now())
		if err == nil {
			return outcome, nil
		}
		var cancelled *types.TransactionCanceledException
		var conflict *types.TransactionConflictException
		retry := errors.As(err, &conflict)
		if errors.As(err, &cancelled) {
			for _, reason := range cancelled.CancellationReasons {
				if aws.ToString(reason.Code) == "TransactionConflict" {
					retry = true
				}
			}
		}
		if !retry {
			return 0, err
		}
		timer := time.NewTimer(5 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return 0, ctx.Err()
		case <-timer.C:
		}
	}
	return 0, errors.New("contention retry exhausted")
}

func TestDynamoDBLocalConcurrentClicksAndDeleteRace(t *testing.T) {
	ctx, client, table, repo := clickIntegrationTable(t)
	if err := repo.CreateLink(ctx, domain.Link{Code: "Abc1234", OwnerID: "alice", CreatedAt: domain.FormatCreatedAt(time.Now())}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan ClickOutcome, 12)
	failures := make(chan error, 12)
	for range 12 {
		wg.Go(func() {
			outcome, err := recordWithContentionRetry(ctx, repo, "shared-id", "Abc1234")
			if err != nil {
				failures <- err
				return
			}
			results <- outcome
		})
	}
	wg.Wait()
	close(results)
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	applied := 0
	for outcome := range results {
		if outcome == ClickApplied {
			applied++
		} else if outcome != ClickDuplicate {
			t.Fatal(outcome)
		}
	}
	if applied != 1 {
		t.Fatal("concurrent duplicate count", applied)
	}
	// Distinct events must not lose updates to the shared META/STAT counters.
	failures = make(chan error, 12)
	for i := range 12 {
		wg.Go(func() {
			outcome, err := recordWithContentionRetry(ctx, repo, fmt.Sprintf("unique-%d", i), "Abc1234")
			if err != nil || outcome != ClickApplied {
				failures <- fmt.Errorf("outcome=%v: %v", outcome, err)
			}
		})
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	assertClickNumber(t, readClickItem(t, ctx, client, table, "LINK#Abc1234", "META"), "clicks", "13")
	// Delete and a transaction race. Either the transaction commits entirely first,
	// or it is discarded entirely; a late retry can never resurrect META.
	start := make(chan struct{})
	failures = make(chan error, 2)
	var raced ClickOutcome
	wg.Go(func() {
		<-start
		outcome, err := recordWithContentionRetry(ctx, repo, "raced-id", "Abc1234")
		raced = outcome
		if err != nil {
			failures <- err
		}
	})
	wg.Go(func() {
		<-start
		if err := repo.DeleteUserLink(ctx, "Abc1234", "alice"); err != nil {
			failures <- err
		}
	})
	close(start)
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
	if len(readClickItem(t, ctx, client, table, "LINK#Abc1234", "META")) != 0 {
		t.Fatal("metadata resurrected")
	}
	if raced != ClickApplied && raced != ClickDeleted {
		t.Fatal(raced)
	}
	marker := readClickItem(t, ctx, client, table, "CLICK#raced-id", "CLICK")
	if (len(marker) > 0) != (raced == ClickApplied) {
		t.Fatal("partial transaction", raced, marker)
	}
	outcome, err := repo.RecordClickTransaction(ctx, "after-delete", "Abc1234", time.Now(), time.Now())
	if err != nil || outcome != ClickDeleted || len(readClickItem(t, ctx, client, table, "CLICK#after-delete", "CLICK")) != 0 {
		t.Fatal(outcome, err)
	}
}
