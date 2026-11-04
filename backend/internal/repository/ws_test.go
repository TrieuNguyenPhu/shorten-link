package repository

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestConnectionPersistenceKeysAndTTL(t *testing.T) {
	failure := errors.New("unavailable")
	repo := New(sdkStub{put: func(in *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
		if aws.ToString(in.TableName) != "table" || str(in.Item["PK"]) != "CONN#socket=" || str(in.Item["SK"]) != "META" || str(in.Item["GSI1PK"]) != "WSCODE#Abc1234" || str(in.Item["GSI1SK"]) != "CONN#socket=" || str(in.Item["code"]) != "Abc1234" || in.Item["ttl"].(*types.AttributeValueMemberN).Value != "7300" {
			t.Fatal(in)
		}
		return nil, failure
	}, delete: func(in *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
		if aws.ToString(in.TableName) != "table" || str(in.Key["PK"]) != "CONN#socket=" || str(in.Key["SK"]) != "META" || in.ConditionExpression != nil {
			t.Fatal(in)
		}
		return nil, failure
	}}, "table")
	if err := repo.SaveConnection(context.Background(), domain.WSConnection{ConnectionID: "socket=", Code: "Abc1234", TTL: 7300}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if err := repo.DeleteConnection(context.Background(), "socket="); !errors.Is(err, failure) {
		t.Fatal(err)
	}
}

func TestReadCommittedCountAndPaginateAllConnections(t *testing.T) {
	calls := 0
	key := map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: "CONN#first"}, "SK": &types.AttributeValueMemberS{Value: "META"}, "GSI1PK": &types.AttributeValueMemberS{Value: "WSCODE#Abc1234"}, "GSI1SK": &types.AttributeValueMemberS{Value: "CONN#first"}}
	repo := New(sdkStub{get: func(in *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
		if !aws.ToBool(in.ConsistentRead) || str(in.Key["PK"]) != "LINK#Abc1234" || str(in.Key["SK"]) != "META" {
			t.Fatal(in)
		}
		return &dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{"clicks": &types.AttributeValueMemberN{Value: "42"}}}, nil
	}, query: func(in *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
		calls++
		if aws.ToString(in.IndexName) != "GSI1" || str(in.ExpressionAttributeValues[":code"]) != "WSCODE#Abc1234" || aws.ToString(in.KeyConditionExpression) != "GSI1PK = :code" || aws.ToBool(in.ConsistentRead) {
			t.Fatal(in)
		}
		if calls == 1 {
			return &dynamodb.QueryOutput{LastEvaluatedKey: key, Items: []map[string]types.AttributeValue{key}}, nil
		}
		if !reflect.DeepEqual(in.ExclusiveStartKey, key) {
			t.Fatal("lost cursor", in)
		}
		return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{{"PK": &types.AttributeValueMemberS{Value: "CONN#last"}}}}, nil
	}}, "table")
	count, err := repo.GetClickCount(context.Background(), "Abc1234")
	if err != nil || count != 42 {
		t.Fatal(count, err)
	}
	var ids []string
	err = repo.ForEachConnection(context.Background(), "Abc1234", func(id string) { ids = append(ids, id) })
	if err != nil || !reflect.DeepEqual(ids, []string{"first", "last"}) || calls != 2 {
		t.Fatal(ids, calls, err)
	}
}

func TestConnectionReadsRejectErrorsAndMalformedResponses(t *testing.T) {
	failure := errors.New("unavailable")
	for _, tc := range []struct {
		out *dynamodb.GetItemOutput
		err error
	}{
		{nil, failure}, {nil, nil}, {&dynamodb.GetItemOutput{}, nil},
		{&dynamodb.GetItemOutput{Item: map[string]types.AttributeValue{"clicks": &types.AttributeValueMemberS{Value: "bad"}}}, nil},
	} {
		repo := New(sdkStub{get: func(*dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) { return tc.out, tc.err }}, "table")
		if _, err := repo.GetClickCount(context.Background(), "Abc1234"); err == nil {
			t.Fatal("invalid count accepted", tc)
		}
	}
	for _, tc := range []struct {
		out *dynamodb.QueryOutput
		err error
	}{
		{nil, failure}, {nil, nil}, {&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{{"PK": &types.AttributeValueMemberS{Value: "LINK#bad"}}}}, nil},
	} {
		repo := New(sdkStub{query: func(*dynamodb.QueryInput) (*dynamodb.QueryOutput, error) { return tc.out, tc.err }}, "table")
		if err := repo.ForEachConnection(context.Background(), "Abc1234", func(string) { t.Fatal("malformed connection emitted") }); err == nil {
			t.Fatal("invalid query accepted")
		}
	}
}
