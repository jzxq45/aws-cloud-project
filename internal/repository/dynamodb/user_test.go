package dynamodb

import (
	"context"
	"errors"
	"testing"

	"aws-local-test/internal/model"
	"aws-local-test/internal/repository"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockDynamoClient struct {
	PutItemFunc    func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	GetItemFunc    func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	UpdateItemFunc func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	DeleteItemFunc func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	ScanFunc       func(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
}

func (m *MockDynamoClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	if m.PutItemFunc != nil {
		return m.PutItemFunc(ctx, params, optFns...)
	}
	return &dynamodb.PutItemOutput{}, nil
}

func (m *MockDynamoClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	if m.GetItemFunc != nil {
		return m.GetItemFunc(ctx, params, optFns...)
	}
	return &dynamodb.GetItemOutput{}, nil
}

func (m *MockDynamoClient) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	if m.UpdateItemFunc != nil {
		return m.UpdateItemFunc(ctx, params, optFns...)
	}
	return &dynamodb.UpdateItemOutput{}, nil
}

func (m *MockDynamoClient) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	if m.DeleteItemFunc != nil {
		return m.DeleteItemFunc(ctx, params, optFns...)
	}
	return &dynamodb.DeleteItemOutput{}, nil
}

func (m *MockDynamoClient) Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	if m.ScanFunc != nil {
		return m.ScanFunc(ctx, params, optFns...)
	}
	return &dynamodb.ScanOutput{}, nil
}

func TestDynamoUserRepository_Create_Success(t *testing.T) {
	called := false
	mockClient := &MockDynamoClient{
		PutItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			assert.Equal(t, "Users", *params.TableName)
			called = true
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	repo := NewUserRepository(mockClient, "Users")
	err := repo.Create(context.Background(), model.User{
		ID:    "u-1",
		Name:  "Alice",
		Email: "alice@example.com",
	})

	require.NoError(t, err)
	assert.True(t, called)
}

func TestDynamoUserRepository_Create_DBError(t *testing.T) {
	mockClient := &MockDynamoClient{
		PutItemFunc: func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, errors.New("dynamo timeout")
		},
	}

	repo := NewUserRepository(mockClient, "Users")
	err := repo.Create(context.Background(), model.User{
		ID:    "u-1",
		Name:  "Alice",
		Email: "alice@example.com",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "dynamo timeout")
}

func TestDynamoUserRepository_GetByID_Success(t *testing.T) {
	expected := model.User{
		ID:    "u-2",
		Name:  "Bob",
		Email: "bob@example.com",
	}
	av, err := attributevalue.MarshalMap(expected)
	require.NoError(t, err)

	mockClient := &MockDynamoClient{
		GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			assert.Equal(t, "Users", *params.TableName)
			return &dynamodb.GetItemOutput{Item: av}, nil
		},
	}

	repo := NewUserRepository(mockClient, "Users")
	user, err := repo.GetByID(context.Background(), "u-2")

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, expected, *user)
}

func TestDynamoUserRepository_GetByID_NotFound(t *testing.T) {
	mockClient := &MockDynamoClient{
		GetItemFunc: func(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	}

	repo := NewUserRepository(mockClient, "Users")
	user, err := repo.GetByID(context.Background(), "non-existent")

	require.ErrorIs(t, err, repository.ErrUserNotFound)
	assert.Nil(t, user)
}

func TestDynamoUserRepository_UpdateUser_Success(t *testing.T) {
	called := false
	mockClient := &MockDynamoClient{
		UpdateItemFunc: func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			assert.Equal(t, "Users", *params.TableName)
			assert.Equal(t, "u-update", params.Key["ID"].(*types.AttributeValueMemberS).Value)
			called = true
			return &dynamodb.UpdateItemOutput{}, nil
		},
	}

	repo := NewUserRepository(mockClient, "Users")
	err := repo.UpdateUser(context.Background(), model.User{
		ID:    "u-update",
		Name:  "Alice Cooper",
		Email: "alice.c@example.com",
	})

	require.NoError(t, err)
	assert.True(t, called)
}

func TestDynamoUserRepository_UpdateUser_NotFound(t *testing.T) {
	mockClient := &MockDynamoClient{
		UpdateItemFunc: func(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
			return nil, &types.ConditionalCheckFailedException{}
		},
	}

	repo := NewUserRepository(mockClient, "Users")
	err := repo.UpdateUser(context.Background(), model.User{
		ID:    "u-not-found",
		Name:  "Ghost",
		Email: "ghost@example.com",
	})

	require.ErrorIs(t, err, repository.ErrUserNotFound)
}

func TestDynamoUserRepository_DeleteUser_Success(t *testing.T) {
	called := false
	mockClient := &MockDynamoClient{
		DeleteItemFunc: func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			assert.Equal(t, "Users", *params.TableName)
			assert.Equal(t, "u-del", params.Key["ID"].(*types.AttributeValueMemberS).Value)
			called = true
			return &dynamodb.DeleteItemOutput{}, nil
		},
	}

	repo := NewUserRepository(mockClient, "Users")
	err := repo.DeleteUser(context.Background(), "u-del")

	require.NoError(t, err)
	assert.True(t, called)
}

func TestDynamoUserRepository_DeleteUser_NotFound(t *testing.T) {
	mockClient := &MockDynamoClient{
		DeleteItemFunc: func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			return nil, &types.ConditionalCheckFailedException{}
		},
	}

	repo := NewUserRepository(mockClient, "Users")
	err := repo.DeleteUser(context.Background(), "u-missing")

	require.ErrorIs(t, err, repository.ErrUserNotFound)
}

func TestDynamoUserRepository_ListUsers_Success(t *testing.T) {
	items := []map[string]types.AttributeValue{
		{
			"ID":    &types.AttributeValueMemberS{Value: "1"},
			"Name":  &types.AttributeValueMemberS{Value: "One"},
			"Email": &types.AttributeValueMemberS{Value: "one@test.com"},
		},
	}
	mockClient := &MockDynamoClient{
		ScanFunc: func(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
			assert.Equal(t, "Users", *params.TableName)
			return &dynamodb.ScanOutput{Items: items}, nil
		},
	}

	repo := NewUserRepository(mockClient, "Users")
	users, err := repo.ListUsers(context.Background())

	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "1", users[0].ID)
}
