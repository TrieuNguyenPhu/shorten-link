package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/repository"
	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/service"
	"github.com/aws/aws-lambda-go/events"
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
	if s.put == nil {
		return &dynamodb.PutItemOutput{}, nil
	}
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

type publishStub func(context.Context, domain.ClickEvent) error

func (p publishStub) Publish(ctx context.Context, e domain.ClickEvent) error { return p(ctx, e) }

func request(body string) events.APIGatewayV2HTTPRequest {
	return events.APIGatewayV2HTTPRequest{Body: body, PathParameters: map[string]string{"code": "Abc1234"}, RequestContext: events.APIGatewayV2HTTPRequestContext{DomainName: "api.example.org", Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{Claims: map[string]string{"sub": "alice"}}}}}
}

func handler(client sdkStub, p publishStub, metric io.Writer) *Handler {
	return New(service.New(repository.New(client, "table"), p), "https://redirect.example.org", slog.New(slog.NewJSONHandler(io.Discard, nil)), metric)
}

func TestPrivateRoutesRequireAuthorizerIdentity(t *testing.T) {
	h := handler(sdkStub{}, nil, io.Discard)
	for _, fn := range []func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error){h.Create, h.List, h.Delete} {
		for _, auth := range []*events.APIGatewayV2HTTPRequestContextAuthorizerDescription{nil, {}, {JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{Claims: map[string]string{}}}} {
			req := request(`{"url":"https://example.org","ownerId":"bob"}`)
			req.RequestContext.Authorizer = auth
			req.Headers = map[string]string{"Authorization": "Bearer unverified", "X-Test-Sub": "alice"}
			res, err := fn(context.Background(), req)
			if err != nil || res.StatusCode != 401 {
				t.Fatal(res, err)
			}
		}
	}
}

func TestCreateValidatesRequestBodyBeforeStorage(t *testing.T) {
	h := handler(sdkStub{}, nil, io.Discard)
	for _, body := range []string{"", `null`, `{}`, `[]`, `{"url":null}`, `{"url":9}`, `{"url":true}`, `{"url":{}}`, `{"url":"javascript:alert(1)"}`, `{"url":"https://example.org"} {}`, strings.Repeat("x", 20000)} {
		res, err := h.Create(context.Background(), request(body))
		if err != nil || res.StatusCode != 400 {
			t.Fatalf("body=%q got %+v %v", body[:min(len(body), 80)], res, err)
		}
	}
	req := request("!")
	req.IsBase64Encoded = true
	res, _ := h.Create(context.Background(), req)
	if res.StatusCode != 400 {
		t.Fatal(res)
	}
}

func TestCreateReturnsAPIDomainAndIgnoresBodyOwner(t *testing.T) {
	var stored map[string]types.AttributeValue
	h := handler(sdkStub{put: func(in *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
		stored = in.Item
		return &dynamodb.PutItemOutput{}, nil
	}}, nil, io.Discard)
	req := request(base64.StdEncoding.EncodeToString([]byte(`{"url":" https://example.org/path ","ownerId":"bob"}`)))
	req.IsBase64Encoded = true
	res, err := h.Create(context.Background(), req)
	if err != nil || res.StatusCode != 201 {
		t.Fatal(res, err)
	}
	var out map[string]string
	if err := json.Unmarshal([]byte(res.Body), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 4 || len(out["code"]) != 7 || out["target"] != "https://example.org/path" || out["shortUrl"] != "https://redirect.example.org/"+out["code"] || out["status"] != "pending" || stored["ownerId"].(*types.AttributeValueMemberS).Value != "alice" {
		t.Fatal(out, stored)
	}
	h.baseURL = ""
	res, err = h.Create(context.Background(), request(`{"url":"https://example.org"}`))
	if err != nil || !strings.Contains(res.Body, "https://api.example.org/") {
		t.Fatal(res, err)
	}
}

func TestCreateCollisionReturns500AfterFiveWrites(t *testing.T) {
	calls := 0
	h := handler(sdkStub{put: func(*dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
		calls++
		return nil, &types.ConditionalCheckFailedException{}
	}}, nil, io.Discard)
	res, err := h.Create(context.Background(), request(`{"url":"https://example.org"}`))
	if err != nil || res.StatusCode != 500 || calls != 5 {
		t.Fatal(res, err, calls)
	}
}

func TestListReturnsOnlyPublicFieldsAndEmptyArray(t *testing.T) {
	for _, empty := range []bool{false, true} {
		h := handler(sdkStub{query: func(in *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
			if in.ExpressionAttributeValues[":u"].(*types.AttributeValueMemberS).Value != "USER#alice" {
				t.Fatal(in)
			}
			if empty {
				return &dynamodb.QueryOutput{}, nil
			}
			return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{{"PK": &types.AttributeValueMemberS{Value: "LINK#Abc1234"}, "ownerId": &types.AttributeValueMemberS{Value: "alice"}, "target": &types.AttributeValueMemberS{Value: "https://example.org"}, "clicks": &types.AttributeValueMemberN{Value: "4"}, "createdAt": &types.AttributeValueMemberS{Value: "2026-11-04T00:00:00.000Z"}, "status": &types.AttributeValueMemberS{Value: "ACTIVE"}}}}, nil
		}}, nil, io.Discard)
		res, err := h.List(context.Background(), request(""))
		if err != nil || res.StatusCode != 200 {
			t.Fatal(res, err)
		}
		var out struct {
			Links []map[string]any `json:"links"`
		}
		if err := json.Unmarshal([]byte(res.Body), &out); err != nil || out.Links == nil {
			t.Fatal(res, err)
		}
		if empty {
			if res.Body != `{"links":[]}` {
				t.Fatal(res.Body)
			}
		} else if len(out.Links) != 1 || len(out.Links[0]) != 7 || out.Links[0]["clicks"] != float64(4) || out.Links[0]["shortUrl"] != "https://redirect.example.org/Abc1234" || out.Links[0]["status"] != "ACTIVE" || out.Links[0]["realtimeTicket"] == "" {
			t.Fatal(out)
		}
	}
}

func TestDeleteConditionalFailureUsesSame404(t *testing.T) {
	h := handler(sdkStub{delete: func(in *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
		if in.ExpressionAttributeValues[":me"].(*types.AttributeValueMemberS).Value != "alice" {
			t.Fatal(in)
		}
		return nil, &types.ConditionalCheckFailedException{}
	}}, nil, io.Discard)
	res, err := h.Delete(context.Background(), request(""))
	if err != nil || res.StatusCode != 404 || res.Body != `{"error":"link khong ton tai hoac khong thuoc ve ban"}` {
		t.Fatal(res, err)
	}
	h = handler(sdkStub{delete: func(*dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
		return &dynamodb.DeleteItemOutput{}, nil
	}}, nil, io.Discard)
	res, err = h.Delete(context.Background(), request(""))
	if err != nil || res.StatusCode != 200 || res.Body != `{"deleted":"Abc1234"}` {
		t.Fatal(res, err)
	}
}

func TestResolveRedirectsWith302NoStoreAndBestEffortAnalytics(t *testing.T) {
	for _, tc := range []struct {
		name                           string
		missing, publishFail, readFail bool
		status                         int
		metric                         string
	}{
		{name: "accepted", status: 302, metric: "LinkResolved"},
		{name: "missing", missing: true, status: 404, metric: "LinkNotFound"},
		{name: "publish failed still redirects", publishFail: true, status: 302, metric: "ClickPublishFailed"},
		{name: "read failed", readFail: true, status: 500, metric: "ResolveFailed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var metrics bytes.Buffer
			published := false
			h := handler(sdkStub{get: func(*dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
				if tc.readFail {
					return nil, errors.New("read failed")
				}
				if tc.missing {
					return &dynamodb.GetItemOutput{}, nil
				}
				return &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "LINK#Abc1234"}, "target": &types.AttributeValueMemberS{Value: "https://example.org"}, "status": &types.AttributeValueMemberS{Value: "active"}}}, nil
			}}, func(_ context.Context, e domain.ClickEvent) error {
				published = true
				if e.Code != "Abc1234" {
					t.Fatal(e)
				}
				if tc.publishFail {
					return errors.New("publish denied")
				}
				return nil
			}, &metrics)
			res, err := h.Resolve(context.Background(), request(""))
			if err != nil || res.StatusCode != tc.status {
				t.Fatal(res, err)
			}
			if tc.status == 302 && (!published || res.Headers["Location"] != "https://example.org" || !strings.Contains(res.Headers["Cache-Control"], "no-store") || res.Body != "") {
				t.Fatal(res, published)
			}
			if tc.metric != "" && !strings.Contains(metrics.String(), `"`+tc.metric+`":1`) {
				t.Fatal(metrics.String())
			}
			if tc.readFail && strings.Contains(metrics.String(), "ClickPublishFailed") {
				t.Fatal("read failure counted as publish failure")
			}
		})
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("telemetry unavailable") }

func TestTelemetryFailureDoesNotRepublishOrFailRedirect(t *testing.T) {
	calls := 0
	h := handler(sdkStub{get: func(*dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
		return &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "LINK#Abc1234"}, "target": &types.AttributeValueMemberS{Value: "https://example.org"}, "status": &types.AttributeValueMemberS{Value: "active"}}}, nil
	}}, func(context.Context, domain.ClickEvent) error { calls++; return nil }, brokenWriter{})
	res, err := h.Resolve(context.Background(), request(""))
	if err != nil || res.StatusCode != 302 || calls != 1 {
		t.Fatal(res, err, calls)
	}
}
