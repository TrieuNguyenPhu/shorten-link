//go:build integration

package repository

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestDynamoDBLocalCRUDIsolationAndPagination(t *testing.T) {
	endpoint := os.Getenv("DYNAMODB_ENDPOINT")
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "http" || net.ParseIP(u.Hostname()) == nil || !net.ParseIP(u.Hostname()).IsLoopback() {
		t.Fatal("DYNAMODB_ENDPOINT must be an explicit loopback HTTP endpoint")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	client := dynamodb.NewFromConfig(aws.Config{Region: "ap-southeast-1", Credentials: credentials.NewStaticCredentialsProvider("local", "local", "")}, func(o *dynamodb.Options) { o.BaseEndpoint = aws.String(endpoint) })
	table := fmt.Sprintf("task21-%d", time.Now().UnixNano())
	_, err = client.CreateTable(ctx, &dynamodb.CreateTableInput{TableName: aws.String(table), BillingMode: types.BillingModePayPerRequest, AttributeDefinitions: []types.AttributeDefinition{{AttributeName: aws.String("PK"), AttributeType: types.ScalarAttributeTypeS}, {AttributeName: aws.String("SK"), AttributeType: types.ScalarAttributeTypeS}, {AttributeName: aws.String("GSI1PK"), AttributeType: types.ScalarAttributeTypeS}, {AttributeName: aws.String("GSI1SK"), AttributeType: types.ScalarAttributeTypeS}}, KeySchema: []types.KeySchemaElement{{AttributeName: aws.String("PK"), KeyType: types.KeyTypeHash}, {AttributeName: aws.String("SK"), KeyType: types.KeyTypeRange}}, GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{{IndexName: aws.String("GSI1"), KeySchema: []types.KeySchemaElement{{AttributeName: aws.String("GSI1PK"), KeyType: types.KeyTypeHash}, {AttributeName: aws.String("GSI1SK"), KeyType: types.KeyTypeRange}}, Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll}}}})
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
	if err := dynamodb.NewTableExistsWaiter(client).Wait(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(table)}, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	repo := New(client, table)
	base := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	// 6 * 220KB exceeds DynamoDB's 1MB query page; this tests real pagination.
	for i := range 6 {
		link := domain.Link{Code: fmt.Sprintf("Code%03d", i), OwnerID: "alice", Target: "https://example.org/" + strings.Repeat("a", 220000), CreatedAt: domain.FormatCreatedAt(base.Add(time.Duration(i) * time.Millisecond))}
		if err := repo.CreateLink(ctx, link); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.CreateLink(ctx, domain.Link{Code: "Bob0001", OwnerID: "bob", Target: "https://example.org", CreatedAt: "2026-11-04T00:00:00.000Z"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateLink(ctx, domain.Link{Code: "Code000", OwnerID: "bob", CreatedAt: "later"}); !errors.Is(err, ErrCollision) {
		t.Fatalf("collision: %v", err)
	}
	if err := repo.DeleteUserLink(ctx, "Code000", "bob"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("IDOR: %v", err)
	}
	link, err := repo.GetLink(ctx, "Code000")
	if err != nil || link.OwnerID != "alice" || link.CreatedAt != "2026-11-04T00:00:00.000Z" {
		t.Fatal(link, err)
	}
	links, err := collectUserLinks(ctx, repo, "alice")
	if err != nil || len(links) != 6 {
		t.Fatalf("pagination len=%d err=%v", len(links), err)
	}
	for i, l := range links {
		if l.OwnerID != "alice" || l.Code != fmt.Sprintf("Code%03d", 5-i) {
			t.Fatalf("order or owner at %d: %s %s", i, l.Code, l.OwnerID)
		}
	}
	bob, err := collectUserLinks(ctx, repo, "bob")
	if err != nil || len(bob) != 1 || bob[0].Code != "Bob0001" {
		t.Fatal(bob, err)
	}
	// Delete must preserve statistics and markers.
	for _, key := range []map[string]types.AttributeValue{{"PK": &types.AttributeValueMemberS{Value: "LINK#Code000"}, "SK": &types.AttributeValueMemberS{Value: "STAT#2026-11-04"}}, {"PK": &types.AttributeValueMemberS{Value: "CLICK#event"}, "SK": &types.AttributeValueMemberS{Value: "CLICK"}}} {
		if _, err := client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(table), Item: key}); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.DeleteUserLink(ctx, "Code000", "alice"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetLink(ctx, "Code000"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if err := repo.DeleteUserLink(ctx, "Code000", "alice"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	for _, key := range []map[string]types.AttributeValue{{"PK": &types.AttributeValueMemberS{Value: "LINK#Code000"}, "SK": &types.AttributeValueMemberS{Value: "STAT#2026-11-04"}}, {"PK": &types.AttributeValueMemberS{Value: "CLICK#event"}, "SK": &types.AttributeValueMemberS{Value: "CLICK"}}} {
		out, err := client.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(table), Key: key, ConsistentRead: aws.Bool(true)})
		if err != nil || len(out.Item) == 0 {
			t.Fatal("deleted unrelated item", err)
		}
	}
}

func collectUserLinks(ctx context.Context, repo *DynamoDB, owner string) ([]domain.Link, error) {
	var links []domain.Link
	cursor := ""
	for {
		page, err := repo.ListUserLinks(ctx, owner, 100, cursor)
		if err != nil {
			return nil, err
		}
		links = append(links, page.Links...)
		cursor = page.NextCursor
		if cursor == "" {
			return links, nil
		}
	}
}
