package dynamodb

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/tester"
)

const dynamoDBLocal = "http://127.0.0.1:8000"

func randSeq(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func newTestCreateTableInput(tableName string) *awsdynamodb.CreateTableInput {
	return &awsdynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String(SessionIDHashKeyName),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String(SessionIDHashKeyName),
				KeyType:       types.KeyTypeHash,
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
		TableName: aws.String(tableName),
	}
}

func prepareDynamoDBTable(ctx context.Context, client *awsdynamodb.Client) string {
	name := randSeq(10)
	_, _ = client.CreateTable(ctx, newTestCreateTableInput(name))
	_ = awsdynamodb.NewTableExistsWaiter(client).Wait(ctx,
		&awsdynamodb.DescribeTableInput{TableName: aws.String(name)},
		2*time.Minute,
	)
	return name
}

var newStore = func(_ *testing.T) sessions.Store {
	ctx := context.Background()
	cfg := aws.Config{
		Region:      "ap-northeast-1",
		Credentials: credentials.NewStaticCredentialsProvider("dummy", "dummy", "dummy"),
	}
	client := awsdynamodb.NewFromConfig(cfg, func(o *awsdynamodb.Options) {
		o.BaseEndpoint = aws.String(dynamoDBLocal)
	})
	name := prepareDynamoDBTable(ctx, client)
	return NewStore(client, name, []byte("secret"))
}

func TestDynamoDB_SessionGetSet(t *testing.T) {
	tester.GetSet(t, newStore)
}

func TestDynamoDB_SessionDeleteKey(t *testing.T) {
	tester.DeleteKey(t, newStore)
}

func TestDynamoDB_SessionFlashes(t *testing.T) {
	tester.Flashes(t, newStore)
}

func TestDynamoDB_SessionClear(t *testing.T) {
	tester.Clear(t, newStore)
}

func TestDynamoDB_SessionOptions(t *testing.T) {
	tester.Options(t, newStore)
}
