package dynamostore

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"shortlink/internal/application"
	"shortlink/internal/domain"
)

const keyTime = "2006-01-02T15:04:05.000000000Z"

type Store struct {
	client *dynamodb.Client
	table  string
}

type linkRecord struct {
	PK        string     `dynamodbav:"PK"`
	SK        string     `dynamodbav:"SK"`
	GSI1PK    string     `dynamodbav:"GSI1PK,omitempty"`
	GSI1SK    string     `dynamodbav:"GSI1SK,omitempty"`
	Code      string     `dynamodbav:"code"`
	OwnerID   string     `dynamodbav:"ownerId,omitempty"`
	URL       string     `dynamodbav:"url,omitempty"`
	Enabled   bool       `dynamodbav:"enabled"`
	Version   int64      `dynamodbav:"version"`
	Deleted   bool       `dynamodbav:"deleted,omitempty"`
	DeletedAt *time.Time `dynamodbav:"deletedAt,omitempty"`
	CreatedAt time.Time  `dynamodbav:"createdAt"`
	UpdatedAt time.Time  `dynamodbav:"updatedAt,omitempty"`
	ExpiresAt *time.Time `dynamodbav:"expiresAt,omitempty"`
}

func (r linkRecord) domain() domain.Link {
	version := r.Version
	if version < 1 {
		version = 1
	}
	updatedAt := r.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = r.CreatedAt
	}
	return domain.Link{
		Code:      r.Code,
		OwnerID:   r.OwnerID,
		URL:       r.URL,
		Enabled:   r.Enabled,
		Version:   version,
		Deleted:   r.Deleted,
		CreatedAt: r.CreatedAt,
		UpdatedAt: updatedAt,
		ExpiresAt: r.ExpiresAt,
	}
}

func New(client *dynamodb.Client, table string) *Store { return &Store{client: client, table: table} }

func text(value string) types.AttributeValue { return &types.AttributeValueMemberS{Value: value} }

func num(value int64) types.AttributeValue {
	return &types.AttributeValueMemberN{Value: strconv.FormatInt(value, 10)}
}

func key(code string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{"PK": text("LINK#" + code), "SK": text("METADATA")}
}

func ownerPartition(ownerID string) string {
	return "USER#" + ownerID
}

func (s *Store) Create(ctx context.Context, link domain.Link) error {
	version := link.Version
	if version < 1 {
		version = 1
	}
	updatedAt := link.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = link.CreatedAt
	}
	item, err := attributevalue.MarshalMap(linkRecord{
		PK:        "LINK#" + link.Code,
		SK:        "METADATA",
		GSI1PK:    ownerPartition(link.OwnerID),
		GSI1SK:    link.CreatedAt.UTC().Format(keyTime) + "#" + link.Code,
		Code:      link.Code,
		OwnerID:   link.OwnerID,
		URL:       link.URL,
		Enabled:   link.Enabled,
		Version:   version,
		CreatedAt: link.CreatedAt,
		UpdatedAt: updatedAt,
		ExpiresAt: link.ExpiresAt,
	})
	if err != nil {
		return fmt.Errorf("encode link: %w", err)
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.table),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(PK)"),
	})
	var conflict *types.ConditionalCheckFailedException
	if errors.As(err, &conflict) {
		return domain.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("create link: %w", err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, code string) (domain.Link, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName:      aws.String(s.table),
		Key:            key(code),
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return domain.Link{}, fmt.Errorf("get link: %w", err)
	}
	if len(result.Item) == 0 {
		return domain.Link{}, domain.ErrNotFound
	}
	var record linkRecord
	if err := attributevalue.UnmarshalMap(result.Item, &record); err != nil {
		return domain.Link{}, fmt.Errorf("decode link: %w", err)
	}
	return record.domain(), nil
}

func (s *Store) classifyConditionFailure(ctx context.Context, ownerID, code string, expectedVersion int64, condErr *types.ConditionalCheckFailedException) error {
	if len(condErr.Item) > 0 {
		var existing linkRecord
		if err := attributevalue.UnmarshalMap(condErr.Item, &existing); err == nil {
			if !existing.Deleted && existing.OwnerID != "" && existing.OwnerID == ownerID {
				return domain.ErrVersionConflict
			}
			return domain.ErrNotFound
		}
	}
	if expectedVersion > 0 {
		if existing, err := s.Get(ctx, code); err == nil && !existing.Deleted && existing.OwnerID != "" && existing.OwnerID == ownerID && existing.Version != expectedVersion {
			return domain.ErrVersionConflict
		}
	}
	return domain.ErrNotFound
}

func (s *Store) Update(ctx context.Context, ownerID, code string, patch domain.LinkPatch) (domain.Link, error) {
	names := map[string]string{
		"#deleted":   "deleted",
		"#ownerId":   "ownerId",
		"#version":   "version",
		"#updatedAt": "updatedAt",
	}
	values := map[string]types.AttributeValue{
		":actor":     text(ownerID),
		":version":   num(patch.Version),
		":one":       num(1),
		":updatedAt": text(patch.UpdatedAt.UTC().Format(time.RFC3339Nano)),
	}
	setParts := []string{"#version = #version + :one", "#updatedAt = :updatedAt"}
	var removeParts []string

	if patch.URL != nil {
		names["#url"] = "url"
		values[":url"] = text(*patch.URL)
		setParts = append(setParts, "#url = :url")
	}
	if patch.Enabled != nil {
		names["#enabled"] = "enabled"
		values[":enabled"] = &types.AttributeValueMemberBOOL{Value: *patch.Enabled}
		setParts = append(setParts, "#enabled = :enabled")
	}
	if patch.ExpiresAtSet {
		names["#expiresAt"] = "expiresAt"
		if patch.ExpiresAt != nil {
			values[":expiresAt"] = text(patch.ExpiresAt.UTC().Format(time.RFC3339Nano))
			setParts = append(setParts, "#expiresAt = :expiresAt")
		} else {
			removeParts = append(removeParts, "#expiresAt")
		}
	}
	updateExpr := "SET " + strings.Join(setParts, ", ")
	if len(removeParts) > 0 {
		updateExpr += " REMOVE " + strings.Join(removeParts, ", ")
	}
	result, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                           aws.String(s.table),
		Key:                                 key(code),
		ConditionExpression:                 aws.String("attribute_exists(PK) AND attribute_not_exists(#deleted) AND #ownerId = :actor AND #version = :version"),
		UpdateExpression:                    aws.String(updateExpr),
		ExpressionAttributeNames:            names,
		ExpressionAttributeValues:           values,
		ReturnValues:                        types.ReturnValueAllNew,
		ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailureAllOld,
	})
	var condErr *types.ConditionalCheckFailedException
	if errors.As(err, &condErr) {
		return domain.Link{}, s.classifyConditionFailure(ctx, ownerID, code, patch.Version, condErr)
	}
	if err != nil {
		return domain.Link{}, fmt.Errorf("update link: %w", err)
	}
	var record linkRecord
	if err := attributevalue.UnmarshalMap(result.Attributes, &record); err != nil {
		return domain.Link{}, fmt.Errorf("decode updated link: %w", err)
	}
	return record.domain(), nil
}

func (s *Store) Delete(ctx context.Context, ownerID, code string, version int64, deletedAt time.Time) error {
	cond := "attribute_exists(PK) AND attribute_not_exists(#deleted) AND #ownerId = :actor"
	names := map[string]string{
		"#deleted":   "deleted",
		"#deletedAt": "deletedAt",
		"#version":   "version",
		"#ownerId":   "ownerId",
		"#url":       "url",
		"#enabled":   "enabled",
		"#expiresAt": "expiresAt",
		"#updatedAt": "updatedAt",
	}
	values := map[string]types.AttributeValue{
		":actor":     text(ownerID),
		":true":      &types.AttributeValueMemberBOOL{Value: true},
		":deletedAt": text(deletedAt.UTC().Format(time.RFC3339Nano)),
		":zero":      num(0),
		":one":       num(1),
	}
	if version > 0 {
		cond += " AND #version = :version"
		values[":version"] = num(version)
	}
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                           aws.String(s.table),
		Key:                                 key(code),
		ConditionExpression:                 aws.String(cond),
		UpdateExpression:                    aws.String("SET #deleted = :true, #deletedAt = :deletedAt, #version = if_not_exists(#version, :zero) + :one REMOVE GSI1PK, GSI1SK, #url, #enabled, #expiresAt, #ownerId, #updatedAt"),
		ExpressionAttributeNames:            names,
		ExpressionAttributeValues:           values,
		ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailureAllOld,
	})
	var condErr *types.ConditionalCheckFailedException
	if errors.As(err, &condErr) {
		return s.classifyConditionFailure(ctx, ownerID, code, version, condErr)
	}
	if err != nil {
		return fmt.Errorf("delete link: %w", err)
	}
	return nil
}

type cursorData struct {
	OwnerID   string `json:"ownerId"`
	Code      string `json:"code"`
	CreatedAt string `json:"createdAt"`
}

func (s *Store) List(ctx context.Context, ownerID, cursor string) (domain.Page, error) {
	partition := ownerPartition(ownerID)
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(s.table),
		IndexName:                 aws.String("GSI1"),
		KeyConditionExpression:    aws.String("GSI1PK = :collection"),
		ExpressionAttributeValues: map[string]types.AttributeValue{":collection": text(partition)},
		ScanIndexForward:          aws.Bool(false),
		Limit:                     aws.Int32(25),
	}
	if cursor != "" {
		var data cursorData
		decoded, err := base64.RawURLEncoding.DecodeString(cursor)
		if len(cursor) > 512 || err != nil || json.Unmarshal(decoded, &data) != nil || !application.ValidCode(data.Code) || data.OwnerID == "" || data.OwnerID != ownerID {
			return domain.Page{}, fmt.Errorf("%w: malformed cursor", domain.ErrInvalid)
		}
		if _, err := time.Parse(keyTime, data.CreatedAt); err != nil {
			return domain.Page{}, fmt.Errorf("%w: malformed cursor", domain.ErrInvalid)
		}
		input.ExclusiveStartKey = key(data.Code)
		input.ExclusiveStartKey["GSI1PK"], input.ExclusiveStartKey["GSI1SK"] = text(partition), text(data.CreatedAt+"#"+data.Code)
	}
	result, err := s.client.Query(ctx, input)
	if err != nil {
		return domain.Page{}, fmt.Errorf("list links: %w", err)
	}
	var records []linkRecord
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &records); err != nil {
		return domain.Page{}, fmt.Errorf("decode links: %w", err)
	}
	links := make([]domain.Link, 0, len(records))
	for _, record := range records {
		if !record.Deleted && record.OwnerID == ownerID {
			links = append(links, record.domain())
		}
	}
	page := domain.Page{Links: links}
	if len(result.LastEvaluatedKey) != 0 {
		pk, ok := result.LastEvaluatedKey["PK"].(*types.AttributeValueMemberS)
		sk, okSort := result.LastEvaluatedKey["GSI1SK"].(*types.AttributeValueMemberS)
		if !ok || !okSort {
			return page, fmt.Errorf("list cursor is missing index keys")
		}
		code := strings.TrimPrefix(pk.Value, "LINK#")
		encoded, err := json.Marshal(cursorData{OwnerID: ownerID, Code: code, CreatedAt: strings.TrimSuffix(sk.Value, "#"+code)})
		if err != nil {
			return page, fmt.Errorf("encode cursor: %w", err)
		}
		page.NextCursor = base64.RawURLEncoding.EncodeToString(encoded)
	}
	return page, nil
}
