package dynamostore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"shortlink/internal/domain"
	"shortlink/internal/localconfig"
)

func TestDynamoDBIntegration(t *testing.T) {
	if os.Getenv("DYNAMODB_INTEGRATION") != "1" {
		t.Skip("set DYNAMODB_INTEGRATION=1 with DynamoDB Local running")
	}
	cfg, err := localconfig.Load()
	if err != nil {
		t.Fatal(err)
	}
	client := cfg.Client()
	table := fmt.Sprintf("shortlinks-test-%d", time.Now().UnixNano())
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := EnsureTable(ctx, client, table); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if _, err := client.DeleteTable(cleanup, &dynamodb.DeleteTableInput{TableName: aws.String(table)}); err != nil {
			t.Errorf("delete test table: %v", err)
		}
	})
	if err := EnsureTable(ctx, client, table); err != nil {
		t.Fatalf("idempotent initialization: %v", err)
	}
	store := New(client, table)
	created := time.Now().UTC()
	expiry := created.Add(time.Hour)
	link := domain.Link{Code: "shared", URL: "https://example.com", Enabled: true, CreatedAt: created, ExpiresAt: &expiry}
	var wg sync.WaitGroup
	results := make(chan error, 12)
	for range 12 {
		wg.Add(1)
		go func() { defer wg.Done(); results <- store.Create(ctx, link) }()
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, domain.ErrConflict) {
			t.Fatal(err)
		}
	}
	if successes != 1 {
		t.Fatalf("conditional write allowed %d winners", successes)
	}
	stored, err := store.Get(ctx, "shared")
	if err != nil || stored.URL != link.URL || !stored.CreatedAt.Equal(created) || stored.ExpiresAt == nil || !stored.ExpiresAt.Equal(expiry) {
		t.Fatalf("get: %+v %v", stored, err)
	}
	updated, err := store.SetEnabled(ctx, "shared", false)
	if err != nil || updated.Enabled || updated.URL != link.URL {
		t.Fatalf("update: %+v %v", updated, err)
	}
	stored, err = store.Get(ctx, "shared")
	if err != nil || stored.Enabled {
		t.Fatalf("update not persisted: %+v %v", stored, err)
	}
	if _, err := store.Get(ctx, "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing get: %v", err)
	}
	if _, err := store.SetEnabled(ctx, "missing", false); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing update: %v", err)
	}
	for i := range 27 {
		item := domain.Link{Code: fmt.Sprintf("page%03d", i), URL: "https://example.com", Enabled: true, CreatedAt: created.Add(time.Duration(i+1) * time.Second)}
		if err := store.Create(ctx, item); err != nil {
			t.Fatal(err)
		}
	}
	// GSI reads are eventually consistent; poll until all test records become visible.
	var links []domain.Link
	for range 30 {
		links = nil
		cursor := ""
		for {
			page, err := store.List(ctx, cursor)
			if err != nil {
				t.Fatal(err)
			}
			if len(page.Links) > 25 {
				t.Fatal("page limit exceeded")
			}
			links = append(links, page.Links...)
			if page.NextCursor == "" {
				break
			}
			cursor = page.NextCursor
		}
		if len(links) == 28 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if len(links) != 28 {
		t.Fatalf("pagination lost records: got %d", len(links))
	}
	seen := map[string]bool{}
	for i, item := range links {
		if seen[item.Code] {
			t.Fatal("duplicate pagination item")
		}
		seen[item.Code] = true
		if i > 0 && links[i-1].CreatedAt.Before(item.CreatedAt) {
			t.Fatal("not newest first")
		}
	}
	for _, cursor := range []string{"garbage!", "e30", "bnVsbA"} {
		if _, err := store.List(ctx, cursor); !errors.Is(err, domain.ErrInvalid) {
			t.Fatalf("accepted cursor %q: %v", cursor, err)
		}
	}

	t.Run("initialization rejects incompatible attribute types", func(t *testing.T) {
		badTable := table + "-bad-schema"
		_, err := client.CreateTable(ctx, &dynamodb.CreateTableInput{
			TableName: aws.String(badTable), BillingMode: types.BillingModePayPerRequest,
			KeySchema: tableKeys("PK", "SK"),
			AttributeDefinitions: []types.AttributeDefinition{
				{AttributeName: aws.String("PK"), AttributeType: types.ScalarAttributeTypeN},
				{AttributeName: aws.String("SK"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("GSI1PK"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("GSI1SK"), AttributeType: types.ScalarAttributeTypeS},
			},
			GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{{IndexName: aws.String("GSI1"), KeySchema: tableKeys("GSI1PK", "GSI1SK"), Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll}}},
		})
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if _, err := client.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String(badTable)}); err != nil {
				t.Error(err)
			}
		}()
		if err := EnsureTable(ctx, client, badTable); err == nil {
			t.Fatal("accepted incompatible PK attribute type")
		}
		if _, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(badTable)}); err != nil {
			t.Fatalf("initialization removed existing table: %v", err)
		}
	})
}
