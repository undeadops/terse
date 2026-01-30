package db

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/undeadops/terse/internal/store"
)

// Implements a Store interface
func (client *Client) Get(ctx context.Context, key string) (store.UrlObject, error) {
	// Get item from DynamoDB
	result, err := client.DDB.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(client.Table),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: key},
		},
	})
	if err != nil {
		return store.UrlObject{}, fmt.Errorf("failed to get item: %w", err)
	}

	// Check if item exists
	if result.Item == nil {
		return store.UrlObject{}, nil
	}

	// Unmarshal the item
	var item URLItem
	err = attributevalue.UnmarshalMap(result.Item, &item)
	if err != nil {
		return store.UrlObject{}, fmt.Errorf("failed to unmarshal item: %w", err)
	}

	// Increment access count
	update := expression.Add(expression.Name("access_count"), expression.Value(1))
	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		return store.UrlObject{}, fmt.Errorf("failed to build update expression: %w", err)
	}

	_, err = client.DDB.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(client.Table),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: key},
		},
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		// Log error but don't fail the request
		if client.DebugMode {
			fmt.Printf("failed to increment access count: %v\n", err)
		}
	}

	// Convert to UrlObject
	return store.UrlObject{
		Key:           item.ID,
		URL:           item.RedirectURL,
		RedirectCount: int(item.AccessCount),
	}, nil
}

func (client *Client) Put(ctx context.Context, key string, value string) error {
	// Create item to save
	item := URLItem{
		ID:          key,
		RedirectURL: value,
		AccessCount: 0,
		CreatedAt:   time.Now().Unix(),
	}

	// Marshal the item into DynamoDB attribute values
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	// Put item into DynamoDB
	_, err = client.DDB.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(client.Table),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to put item: %w", err)
	}

	return nil
}

func (client *Client) Delete(ctx context.Context, key string) error {
	// Delete item from DynamoDB
	_, err := client.DDB.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(client.Table),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: key},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	return nil
}

func (client *Client) List(ctx context.Context) ([]store.UrlObject, error) {
	// Scan all items from DynamoDB
	result, err := client.DDB.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(client.Table),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan table: %w", err)
	}

	// Convert items to UrlObjects
	urlObjects := make([]store.UrlObject, 0, len(result.Items))
	for _, item := range result.Items {
		var urlItem URLItem
		err := attributevalue.UnmarshalMap(item, &urlItem)
		if err != nil {
			// Skip items that can't be unmarshaled
			if client.DebugMode {
				fmt.Printf("failed to unmarshal item: %v\n", err)
			}
			continue
		}

		urlObjects = append(urlObjects, store.UrlObject{
			Key:           urlItem.ID,
			URL:           urlItem.RedirectURL,
			RedirectCount: int(urlItem.AccessCount),
		})
	}

	return urlObjects, nil
}
