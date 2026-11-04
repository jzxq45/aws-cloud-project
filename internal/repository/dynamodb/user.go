package dynamodb

import (
	"context"
	"errors"
	"fmt"

	"aws-local-test/internal/model"
	"aws-local-test/internal/repository"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoDBAPI defines the subset of the AWS DynamoDB client used by the repository.
type DynamoDBAPI interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
}

// UserRepository implements repository.UserRepository for DynamoDB.
type UserRepository struct {
	client    DynamoDBAPI
	tableName string
}

// NewUserRepository creates a new DynamoDB user repository instance.
func NewUserRepository(client DynamoDBAPI, tableName string) *UserRepository {
	return &UserRepository{
		client:    client,
		tableName: tableName,
	}
}

func (r *UserRepository) Create(ctx context.Context, user model.User) error {
	av, err := attributevalue.MarshalMap(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to put item to dynamodb: %w", err)
	}

	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get item from dynamodb: %w", err)
	}

	if result.Item == nil {
		return nil, repository.ErrUserNotFound
	}

	var user model.User
	if err := attributevalue.UnmarshalMap(result.Item, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, user model.User) error {
	_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{Value: user.ID},
		},
		UpdateExpression: aws.String("SET #name = :name, #email = :email"),
		ExpressionAttributeNames: map[string]string{
			"#name":  "Name",
			"#email": "Email",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":name":  &types.AttributeValueMemberS{Value: user.Name},
			":email": &types.AttributeValueMemberS{Value: user.Email},
		},
		ConditionExpression: aws.String("attribute_exists(ID)"),
	})
	if err != nil {
		var cfe *types.ConditionalCheckFailedException
		if errors.As(err, &cfe) {
			return repository.ErrUserNotFound
		}
		return fmt.Errorf("failed to update item in dynamodb: %w", err)
	}

	return nil
}

func (r *UserRepository) DeleteUser(ctx context.Context, id string) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{Value: id},
		},
		ConditionExpression: aws.String("attribute_exists(ID)"),
	})
	if err != nil {
		var cfe *types.ConditionalCheckFailedException
		if errors.As(err, &cfe) {
			return repository.ErrUserNotFound
		}
		return fmt.Errorf("failed to delete item from dynamodb: %w", err)
	}

	return nil
}

func (r *UserRepository) ListUsers(ctx context.Context) ([]model.User, error) {
	result, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(r.tableName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan dynamodb: %w", err)
	}

	var users []model.User
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &users); err != nil {
		return nil, fmt.Errorf("failed to unmarshal users: %w", err)
	}

	if users == nil {
		users = []model.User{}
	}

	return users, nil
}
