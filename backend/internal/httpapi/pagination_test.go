package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestListPaginationKeepsOversizedOwnerDataBelowLambdaLimit(t *testing.T) {
	calls := 0
	client := sdkStub{query: func(in *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
		calls++
		if in.Limit == nil || *in.Limit < 1 || *in.Limit > 100 {
			t.Fatalf("unbounded query: %v", in.Limit)
		}
		start := 0
		if key, ok := in.ExclusiveStartKey["PK"].(*types.AttributeValueMemberS); ok {
			if _, err := fmt.Sscanf(key.Value, "LINK#%07d", &start); err != nil {
				t.Fatal(err)
			}
			start++
		}
		end := min(start+min(int(*in.Limit), 60), 450)
		items := make([]map[string]types.AttributeValue, 0, end-start)
		for n := start; n < end; n++ {
			items = append(items, map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("LINK#%07d", n)}, "SK": &types.AttributeValueMemberS{Value: "META"}, "ownerId": &types.AttributeValueMemberS{Value: "alice"}, "target": &types.AttributeValueMemberS{Value: "https://example.org/?q=" + strings.Repeat("&", 14900)}, "createdAt": &types.AttributeValueMemberS{Value: "2026-11-04T00:00:00.000Z"}})
		}
		out := &dynamodb.QueryOutput{Items: items}
		if end < 450 {
			out.LastEvaluatedKey = map[string]types.AttributeValue{"PK": items[len(items)-1]["PK"], "SK": &types.AttributeValueMemberS{Value: "META"}, "GSI1PK": &types.AttributeValueMemberS{Value: "USER#alice"}, "GSI1SK": &types.AttributeValueMemberS{Value: "LINK#2026-11-04T00:00:00.000Z"}}
		}
		return out, nil
	}}
	h := handler(client, nil, io.Discard)
	cursor := ""
	seen := map[string]bool{}
	for pages := 0; pages < 100; pages++ {
		req := request("")
		req.QueryStringParameters = map[string]string{"limit": "100", "cursor": cursor}
		res, err := h.List(context.Background(), req)
		if err != nil || res.StatusCode != 200 {
			t.Fatal(res.StatusCode, err)
		}
		encoded, err := json.Marshal(res)
		if err != nil || len(encoded) >= 6*1024*1024 {
			t.Fatalf("oversized Lambda response: %d %v", len(encoded), err)
		}
		var page struct {
			Links      []struct{ Code string }
			NextCursor string
		}
		if err := json.Unmarshal([]byte(res.Body), &page); err != nil {
			t.Fatal(err)
		}
		for _, link := range page.Links {
			if seen[link.Code] {
				t.Fatalf("duplicate %s", link.Code)
			}
			seen[link.Code] = true
		}
		cursor = page.NextCursor
		if cursor == "" {
			break
		}
	}
	if len(seen) != 450 || calls < 5 {
		t.Fatalf("lost owner data: %d links, %d queries", len(seen), calls)
	}
}

func TestListRejectsInvalidPaginationBeforeQuery(t *testing.T) {
	h := handler(sdkStub{}, nil, io.Discard)
	for _, query := range []map[string]string{{"limit": "0"}, {"limit": "101"}, {"limit": "wat"}, {"cursor": "not-base64!"}} {
		req := request("")
		req.QueryStringParameters = query
		res, err := h.List(context.Background(), req)
		if err != nil || res.StatusCode != 400 {
			t.Fatal(res, err)
		}
	}
}
