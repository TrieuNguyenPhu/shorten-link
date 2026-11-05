package repository

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/domain"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var ErrInvalidPagination = errors.New("invalid pagination")

type linkCursor struct {
	PK       string `json:"pk" dynamodbav:"PK"`
	SK       string `json:"sk" dynamodbav:"SK"`
	OwnerKey string `json:"owner" dynamodbav:"GSI1PK"`
	SortKey  string `json:"sort" dynamodbav:"GSI1SK"`
}

func (c linkCursor) valid(owner string) bool {
	code := strings.TrimPrefix(c.PK, "LINK#")
	if c.OwnerKey != "USER#"+owner || c.SK != "META" || !strings.HasPrefix(c.PK, "LINK#") || !domain.IsValidShortCode(code) {
		return false
	}
	if !strings.HasPrefix(c.SortKey, "LINK#") {
		return false
	}
	stamp := strings.TrimPrefix(c.SortKey, "LINK#")
	at, err := time.Parse("2006-01-02T15:04:05.000Z", stamp)
	return err == nil && domain.FormatCreatedAt(at) == stamp
}
func decodeCursor(owner, encoded string) (map[string]types.AttributeValue, error) {
	if encoded == "" {
		return nil, nil
	}
	if len(encoded) > 2048 {
		return nil, ErrInvalidPagination
	}
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, ErrInvalidPagination
	}
	var c linkCursor
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&c); err != nil || !c.valid(owner) {
		return nil, ErrInvalidPagination
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, ErrInvalidPagination
	}
	return attributevalue.MarshalMap(c)
}
func encodeCursor(owner string, key map[string]types.AttributeValue) (string, error) {
	var c linkCursor
	if err := attributevalue.UnmarshalMap(key, &c); err != nil || !c.valid(owner) {
		return "", errors.New("invalid DynamoDB pagination key")
	}
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}
