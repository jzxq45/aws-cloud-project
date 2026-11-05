#!/bin/bash
echo "Initializing DynamoDB Users table..."
awslocal dynamodb create-table \
    --table-name Users \
    --attribute-definitions AttributeName=ID,AttributeType=S \
    --key-schema AttributeName=ID,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST || true
echo "DynamoDB Users table ready!"
