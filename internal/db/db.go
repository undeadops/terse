package db

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rs/zerolog"
)

type Client struct {
	DebugMode   bool
	Client      aws.Config
	Table       string
	Region      string
	DDBEndpoint string
	DDB         *dynamodb.Client
	Logger      *zerolog.Logger
}

// URLItem represents a shortened URL entry in DynamoDB
type URLItem struct {
	ID          string `dynamodbav:"id"`           // Auto-generated short ID (partition key)
	RedirectURL string `dynamodbav:"redirect_url"` // The full URL to redirect to
	AccessCount int64  `dynamodbav:"access_count"` // Number of times the URL has been accessed
	CreatedAt   int64  `dynamodbav:"created_at"`   // Unix timestamp of creation
}

func SetupDB(ctx context.Context, c *Client) error {
	cfg, err := config.LoadDefaultConfig(ctx, func(o *config.LoadOptions) error {
		o.Region = c.Region

		return nil
	})
	if err != nil {
		c.Logger.Fatal().Err(err)
	}

	c.Client = cfg
	if c.DDBEndpoint != "" {
		// Create DynamoDB client with custom endpoint
		c.DDB = dynamodb.NewFromConfig(c.Client, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String(c.DDBEndpoint)
		})
		c.Client.Credentials = credentials.NewStaticCredentialsProvider("dummy1", "dummy2", "dummy3")
		c.Logger.Info().Str("endpoint", c.DDBEndpoint).Msg("Using custom DynamoDB endpoint")
	} else {
		// Create DynamoDB client
		c.DDB = dynamodb.NewFromConfig(c.Client)
	}

	// Test connection by describing the table
	_, err = c.DDB.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(c.Table),
	})

	if err != nil {
		// Table doesn't exist, create it
		if c.DebugMode {
			log.Printf("Table %s doesn't exist, creating...", c.Table)
		}

		// Create table with schema:
		// - id (string): auto-generated short ID (partition key)
		// - redirect_url (string): the full URL to redirect to
		// - access_count (number): counter for times accessed
		// - created_at (number): timestamp of creation
		// Note: Only key attributes need to be defined in AttributeDefinitions
		_, err = c.DDB.CreateTable(ctx, &dynamodb.CreateTableInput{
			TableName: aws.String(c.Table),
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String("id"),
					KeyType:       types.KeyTypeHash,
				},
			},
			AttributeDefinitions: []types.AttributeDefinition{
				{
					AttributeName: aws.String("id"),
					AttributeType: types.ScalarAttributeTypeS,
				},
			},
			BillingMode: types.BillingModePayPerRequest,
		})

		if err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}

		if c.DebugMode {
			log.Printf("Table %s created successfully", c.Table)
		}
	} else {
		if c.DebugMode {
			log.Printf("Connected to DynamoDB table: %s", c.Table)
		}
	}

	return nil
}
