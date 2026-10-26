package dynamostore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func tableKeys(partition, sort string) []types.KeySchemaElement {
	return []types.KeySchemaElement{{AttributeName: aws.String(partition), KeyType: types.KeyTypeHash}, {AttributeName: aws.String(sort), KeyType: types.KeyTypeRange}}
}

// EnsureTable waits for DynamoDB Local, preserving existing tables and rejecting incompatible schemas.
func EnsureTable(ctx context.Context, client *dynamodb.Client, table string) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var lastErr error
	for {
		result, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(table)})
		lastErr = err
		if err == nil && result.Table.TableStatus == types.TableStatusActive {
			for _, name := range []string{"PK", "SK", "GSI1PK", "GSI1SK"} {
				found := false
				for _, attribute := range result.Table.AttributeDefinitions {
					if aws.ToString(attribute.AttributeName) == name && attribute.AttributeType == types.ScalarAttributeTypeS {
						found = true
					}
				}
				if !found {
					return fmt.Errorf("existing table %s requires string attribute %s; no data was changed", table, name)
				}
			}
			if !sameKeys(result.Table.KeySchema, tableKeys("PK", "SK")) {
				return fmt.Errorf("existing table %s has an incompatible primary key; no data was changed", table)
			}
			for _, index := range result.Table.GlobalSecondaryIndexes {
				if aws.ToString(index.IndexName) == "GSI1" && sameKeys(index.KeySchema, tableKeys("GSI1PK", "GSI1SK")) && index.Projection != nil && index.Projection.ProjectionType == types.ProjectionTypeAll && index.IndexStatus == types.IndexStatusActive {
					return nil
				}
			}
			return fmt.Errorf("existing table %s requires active GSI1 (GSI1PK/GSI1SK, projection ALL); no data was changed", table)
		}
		var missing *types.ResourceNotFoundException
		if errors.As(err, &missing) {
			attributes := []types.AttributeDefinition{}
			for _, name := range []string{"PK", "SK", "GSI1PK", "GSI1SK"} {
				attributes = append(attributes, types.AttributeDefinition{AttributeName: aws.String(name), AttributeType: types.ScalarAttributeTypeS})
			}
			_, lastErr = client.CreateTable(ctx, &dynamodb.CreateTableInput{TableName: aws.String(table), BillingMode: types.BillingModePayPerRequest, KeySchema: tableKeys("PK", "SK"), AttributeDefinitions: attributes, GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{{IndexName: aws.String("GSI1"), KeySchema: tableKeys("GSI1PK", "GSI1SK"), Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll}}}})
			var inUse *types.ResourceInUseException
			if lastErr != nil && !errors.As(lastErr, &inUse) {
				return fmt.Errorf("create local table: %w", lastErr)
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("initialize local table: %w (last database error: %v)", ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func sameKeys(actual, expected []types.KeySchemaElement) bool {
	if len(actual) != len(expected) {
		return false
	}
	for _, want := range expected {
		found := false
		for _, got := range actual {
			if aws.ToString(got.AttributeName) == aws.ToString(want.AttributeName) && got.KeyType == want.KeyType {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}
