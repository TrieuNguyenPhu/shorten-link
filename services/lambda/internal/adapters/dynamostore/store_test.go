package dynamostore

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"shortlink/internal/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func boolPtr(v bool) *bool { return &v }

func newTestStore(fn roundTripFunc) *Store {
	client := dynamodb.NewFromConfig(aws.Config{
		Region:      "ap-southeast-2",
		Credentials: credentials.NewStaticCredentialsProvider("test", "test", ""),
		HTTPClient:  &http.Client{Transport: fn},
	}, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String("https://dynamodb.ap-southeast-2.amazonaws.com")
	})
	return New(client, "shortly-dev-links")
}

func ddbResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/x-amz-json-1.0"}},
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

func TestStoreUnitOperationsAndOwnership(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 26, 10, 0, 0, 0, time.UTC)

	t.Run("Create encodes USER#ownerId partition and maps conflict", func(t *testing.T) {
		var captured string
		store := newTestStore(func(r *http.Request) (*http.Response, error) {
			raw, _ := io.ReadAll(r.Body)
			captured = string(raw)
			return ddbResponse(400, `{"__type":"com.amazonaws.dynamodb.v20120810#ConditionalCheckFailedException","message":"conditional check failed"}`), nil
		})
		err := store.Create(ctx, domain.Link{
			Code:      "abc12345",
			OwnerID:   "sub-user-1",
			URL:       "https://example.com",
			Enabled:   true,
			Version:   1,
			CreatedAt: now,
		})
		if !errors.Is(err, domain.ErrConflict) {
			t.Fatalf("expected ErrConflict, got %v", err)
		}
		if !strings.Contains(captured, `"USER#sub-user-1"`) || !strings.Contains(captured, `"attribute_not_exists(PK)"`) {
			t.Fatalf("unexpected PutItem payload: %s", captured)
		}
	})

	t.Run("Update distinguishes version conflict for owner from 404 for non-owner", func(t *testing.T) {
		store := newTestStore(func(r *http.Request) (*http.Response, error) {
			return ddbResponse(400, `{
				"__type":"com.amazonaws.dynamodb.v20120810#ConditionalCheckFailedException",
				"message":"The conditional request failed",
				"Item":{
					"PK":{"S":"LINK#abc12345"},
					"SK":{"S":"METADATA"},
					"code":{"S":"abc12345"},
					"ownerId":{"S":"sub-user-1"},
					"url":{"S":"https://example.com"},
					"enabled":{"BOOL":true},
					"version":{"N":"2"},
					"createdAt":{"S":"2026-10-26T10:00:00Z"}
				}
			}`), nil
		})
		// Same owner, stale version -> ErrVersionConflict
		_, err := store.Update(ctx, "sub-user-1", "abc12345", domain.LinkPatch{
			Version:   1,
			Enabled:   boolPtr(false),
			UpdatedAt: now,
		})
		if !errors.Is(err, domain.ErrVersionConflict) {
			t.Fatalf("expected ErrVersionConflict for owner, got %v", err)
		}
		// Different owner -> ErrNotFound
		_, err = store.Update(ctx, "sub-user-2", "abc12345", domain.LinkPatch{
			Version:   2,
			Enabled:   boolPtr(false),
			UpdatedAt: now,
		})
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("expected ErrNotFound for non-owner, got %v", err)
		}
	})

	t.Run("Delete writes minimal tombstone and removes GSI keys", func(t *testing.T) {
		var captured string
		store := newTestStore(func(r *http.Request) (*http.Response, error) {
			raw, _ := io.ReadAll(r.Body)
			captured = string(raw)
			return ddbResponse(200, `{}`), nil
		})
		if err := store.Delete(ctx, "sub-user-1", "abc12345", 2, now); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if !strings.Contains(captured, "REMOVE GSI1PK, GSI1SK, #url, #enabled, #expiresAt, #ownerId, #updatedAt") {
			t.Fatalf("Delete did not remove GSI and destination attributes: %s", captured)
		}
	})

	t.Run("List enforces actor cursor binding and rejects tampering", func(t *testing.T) {
		store := newTestStore(func(r *http.Request) (*http.Response, error) {
			return ddbResponse(200, `{
				"Items":[{
					"PK":{"S":"LINK#abc12345"},
					"SK":{"S":"METADATA"},
					"GSI1PK":{"S":"USER#sub-user-1"},
					"GSI1SK":{"S":"2026-10-26T10:00:00.000000000Z#abc12345"},
					"code":{"S":"abc12345"},
					"ownerId":{"S":"sub-user-1"},
					"url":{"S":"https://example.com"},
					"enabled":{"BOOL":true},
					"version":{"N":"1"},
					"createdAt":{"S":"2026-10-26T10:00:00Z"}
				}],
				"LastEvaluatedKey":{
					"PK":{"S":"LINK#abc12345"},
					"SK":{"S":"METADATA"},
					"GSI1PK":{"S":"USER#sub-user-1"},
					"GSI1SK":{"S":"2026-10-26T10:00:00.000000000Z#abc12345"}
				}
			}`), nil
		})
		page, err := store.List(ctx, "sub-user-1", "")
		if err != nil || len(page.Links) != 1 || page.NextCursor == "" {
			t.Fatalf("List: %+v %v", page, err)
		}
		// User 2 cannot use User 1's cursor
		if _, err := store.List(ctx, "sub-user-2", page.NextCursor); !errors.Is(err, domain.ErrInvalid) {
			t.Fatalf("expected ErrInvalid on cross-user cursor, got %v", err)
		}
		// User 1 can use their own cursor
		if _, err := store.List(ctx, "sub-user-1", page.NextCursor); err != nil {
			t.Fatalf("expected valid cursor for owner, got %v", err)
		}
		// Tampered cursor without ownerId
		raw, _ := json.Marshal(map[string]string{"code": "abc12345", "createdAt": "2026-10-26T10:00:00.000000000Z"})
		if _, err := store.List(ctx, "sub-user-1", base64.RawURLEncoding.EncodeToString(raw)); !errors.Is(err, domain.ErrInvalid) {
			t.Fatalf("expected ErrInvalid for ownerless cursor, got %v", err)
		}
	})
}
