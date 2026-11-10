package repository

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type sdkStub struct {
	put    func(*dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
	get    func(*dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	query  func(*dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
	delete func(*dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error)
}

func (s sdkStub) TransactWriteItems(context.Context, *dynamodb.TransactWriteItemsInput, ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	panic("unexpected transaction")
}

func (s sdkStub) PutItem(_ context.Context, in *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return s.put(in)
}

func (s sdkStub) GetItem(_ context.Context, in *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return s.get(in)
}

func (s sdkStub) Query(_ context.Context, in *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return s.query(in)
}

func (s sdkStub) DeleteItem(_ context.Context, in *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	return s.delete(in)
}

func str(v types.AttributeValue) string { return v.(*types.AttributeValueMemberS).Value }

func TestCreateConditionalAndCollision(t *testing.T) {
	repo := New(sdkStub{put: func(in *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
		if aws.ToString(in.TableName) != "url-shortener" || aws.ToString(in.ConditionExpression) != "attribute_not_exists(PK)" || str(in.Item["PK"]) != "LINK#Abc1234" || str(in.Item["SK"]) != "META" || str(in.Item["GSI1PK"]) != "USER#alice" || str(in.Item["GSI1SK"]) != "LINK#2026-11-04T00:00:00.001Z" || in.Item["ttl"].(*types.AttributeValueMemberN).Value != "1794000000" {
			t.Fatalf("unsafe put: %+v", in)
		}
		return nil, &types.ConditionalCheckFailedException{}
	}}, "url-shortener")
	err := repo.CreateLink(context.Background(), domain.Link{Code: "Abc1234", OwnerID: "alice", Target: "https://example.org", CreatedAt: "2026-11-04T00:00:00.001Z", ExpiresAt: 1794000000})
	if !errors.Is(err, ErrCollision) {
		t.Fatalf("got %v", err)
	}
}

func TestCreateOmitsTTLForPermanentLinks(t *testing.T) {
	repo := New(sdkStub{put: func(in *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
		if _, ok := in.Item["ttl"]; ok {
			t.Fatal("permanent link unexpectedly has ttl")
		}
		return &dynamodb.PutItemOutput{}, nil
	}}, "url-shortener")
	if err := repo.CreateLink(context.Background(), domain.Link{Code: "Abc1234", OwnerID: "alice", Target: "https://example.org", CreatedAt: "2026-11-04T00:00:00.001Z"}); err != nil {
		t.Fatal(err)
	}
}

func TestGetDecodesStoredMetadataWithoutCodeAttribute(t *testing.T) {
	repo := New(sdkStub{get: func(in *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
		if str(in.Key["PK"]) != "LINK#Abc1234" || str(in.Key["SK"]) != "META" {
			t.Fatal(in)
		}
		return &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "LINK#Abc1234"}, "target": &types.AttributeValueMemberS{Value: "https://example.org"}, "clicks": &types.AttributeValueMemberN{Value: "4"}}}, nil
	}}, "url-shortener")
	link, err := repo.GetLink(context.Background(), "Abc1234")
	if err != nil || link.Code != "Abc1234" || link.Clicks != 4 {
		t.Fatalf("%+v %v", link, err)
	}
}

func TestListPaginatesOwnerQueryNewestFirst(t *testing.T) {
	calls := 0
	key := map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "LINK#New0001"}, "SK": &types.AttributeValueMemberS{Value: "META"}, "GSI1PK": &types.AttributeValueMemberS{Value: "USER#alice"}, "GSI1SK": &types.AttributeValueMemberS{Value: "LINK#2026-11-04T00:00:00.001Z"}}
	repo := New(sdkStub{query: func(in *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
		calls++
		if aws.ToString(in.IndexName) != "GSI1" || aws.ToBool(in.ScanIndexForward) || aws.ToString(in.KeyConditionExpression) != "GSI1PK = :u" || str(in.ExpressionAttributeValues[":u"]) != "USER#alice" || aws.ToInt32(in.Limit) != 25 {
			t.Fatalf("unsafe query: %+v", in)
		}
		code := "New0001"
		out := &dynamodb.QueryOutput{}
		if calls == 1 {
			out.LastEvaluatedKey = key
		} else {
			code = "Old0001"
			if str(in.ExclusiveStartKey["PK"]) != "LINK#New0001" {
				t.Fatal("lost cursor")
			}
		}
		out.Items = []map[string]types.AttributeValue{{"PK": &types.AttributeValueMemberS{Value: "LINK#" + code}, "ownerId": &types.AttributeValueMemberS{Value: "alice"}}}
		return out, nil
	}}, "url-shortener")
	first, err := repo.ListUserLinks(context.Background(), "alice", 25, "")
	if err != nil || len(first.Links) != 1 || first.Links[0].Code != "New0001" || first.NextCursor == "" || calls != 1 {
		t.Fatalf("%+v %v calls=%d", first, err, calls)
	}
	second, err := repo.ListUserLinks(context.Background(), "alice", 25, first.NextCursor)
	if err != nil || len(second.Links) != 1 || second.Links[0].Code != "Old0001" || second.NextCursor != "" || calls != 2 {
		t.Fatalf("%+v %v calls=%d", second, err, calls)
	}
}

func TestDeleteIsAtomicAndHidesOwnership(t *testing.T) {
	repo := New(sdkStub{delete: func(in *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
		if aws.ToString(in.ConditionExpression) != "attribute_exists(PK) AND ownerId = :me" || str(in.ExpressionAttributeValues[":me"]) != "bob" || str(in.Key["PK"]) != "LINK#Abc1234" || str(in.Key["SK"]) != "META" {
			t.Fatalf("unsafe delete: %+v", in)
		}
		return nil, &types.ConditionalCheckFailedException{}
	}}, "url-shortener")
	if err := repo.DeleteUserLink(context.Background(), "Abc1234", "bob"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}

func TestRepositoryPropagatesFailuresAndEmptyList(t *testing.T) {
	failure := errors.New("unavailable")
	ctx := context.Background()
	repo := New(sdkStub{put: func(*dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) { return nil, failure }, get: func(*dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) { return &dynamodb.GetItemOutput{}, nil }, query: func(*dynamodb.QueryInput) (*dynamodb.QueryOutput, error) { return &dynamodb.QueryOutput{}, nil }, delete: func(*dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) { return nil, failure }}, "table")
	if err := repo.CreateLink(ctx, domain.Link{}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if _, err := repo.GetLink(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	page, err := repo.ListUserLinks(ctx, "alice", 25, "")
	if err != nil || page.Links == nil || len(page.Links) != 0 {
		t.Fatalf("%v %v", page, err)
	}
	if err := repo.DeleteUserLink(ctx, "x", "alice"); !errors.Is(err, failure) {
		t.Fatal(err)
	}
}

func TestRealtimeTicketIsSingleUseAndBoundToOwnerCode(t *testing.T) {
	deleted := false
	repo := New(sdkStub{
		put: func(in *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
			if str(in.Item["PK"]) != "TICKET#ticket-1" || aws.ToString(in.ConditionExpression) != "attribute_not_exists(PK)" {
				t.Fatalf("unsafe ticket put: %+v", in)
			}
			return &dynamodb.PutItemOutput{}, nil
		},
		get: func(in *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
			if str(in.Key["PK"]) != "TICKET#ticket-1" || !aws.ToBool(in.ConsistentRead) {
				t.Fatalf("unsafe ticket get: %+v", in)
			}
			return &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{
				"ownerId": &types.AttributeValueMemberS{Value: "alice"},
				"code":    &types.AttributeValueMemberS{Value: "Abc1234"},
				"ttl":     &types.AttributeValueMemberN{Value: "220"},
			}}, nil
		},
		delete: func(in *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
			deleted = true
			if aws.ToString(in.ConditionExpression) != "attribute_exists(PK)" {
				t.Fatalf("unsafe ticket consume: %+v", in)
			}
			return &dynamodb.DeleteItemOutput{}, nil
		},
	}, "url-shortener")
	ticket := domain.RealtimeTicket{Ticket: "ticket-1", OwnerID: "alice", Code: "Abc1234", TTL: 220}
	if err := repo.PutRealtimeTicket(context.Background(), ticket); err != nil {
		t.Fatal(err)
	}
	got, err := repo.ConsumeRealtimeTicket(context.Background(), "ticket-1", time.Unix(100, 0))
	if err != nil || got != ticket || !deleted {
		t.Fatalf("got=%+v err=%v deleted=%v", got, err, deleted)
	}
}

func TestCursorCannotSelectAnotherOwner(t *testing.T) {
	cursor, err := encodeCursor("bob", map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "LINK#Abc1234"}, "SK": &types.AttributeValueMemberS{Value: "META"}, "GSI1PK": &types.AttributeValueMemberS{Value: "USER#bob"}, "GSI1SK": &types.AttributeValueMemberS{Value: "LINK#2026-11-04T00:00:00.000Z"}})
	if err != nil {
		t.Fatal(err)
	}
	repo := New(sdkStub{}, "table")
	if _, err := repo.ListUserLinks(context.Background(), "alice", 25, cursor); !errors.Is(err, ErrInvalidPagination) {
		t.Fatalf("cross-owner cursor: %v", err)
	}
}

func TestCursorAcceptsCustomAlias(t *testing.T) {
	cursor, err := encodeCursor("alice", map[string]types.AttributeValue{
		"PK":     &types.AttributeValueMemberS{Value: "LINK#my-link"},
		"SK":     &types.AttributeValueMemberS{Value: "META"},
		"GSI1PK": &types.AttributeValueMemberS{Value: "USER#alice"},
		"GSI1SK": &types.AttributeValueMemberS{Value: "LINK#2026-11-04T00:00:00.000Z"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeCursor("alice", cursor); err != nil {
		t.Fatalf("custom alias cursor rejected: %v", err)
	}
}

func TestCursorRejectsForgedKeyShapes(t *testing.T) {
	valid := `{"pk":"LINK#Abc1234","sk":"META","owner":"USER#alice","sort":"LINK#2026-11-04T00:00:00.000Z"}`
	for _, raw := range []string{
		strings.Replace(valid, "META", "STAT#2026-11-04", 1),
		strings.Replace(valid, "Abc1234", "../bad!", 1),
		strings.Replace(valid, ".000Z", ".12Z", 1),
		strings.Replace(valid, "USER#alice", "USER#bob", 1),
		strings.TrimSuffix(valid, "}") + `,"extra":"ignored?"}`,
		valid + ` null`,
	} {
		encoded := base64.RawURLEncoding.EncodeToString([]byte(raw))
		if _, err := New(sdkStub{}, "table").ListUserLinks(context.Background(), "alice", 25, encoded); !errors.Is(err, ErrInvalidPagination) {
			t.Fatalf("accepted cursor %s: %v", raw, err)
		}
	}
	if _, err := New(sdkStub{}, "table").ListUserLinks(context.Background(), "alice", 25, strings.Repeat("a", 2049)); !errors.Is(err, ErrInvalidPagination) {
		t.Fatal(err)
	}
}
