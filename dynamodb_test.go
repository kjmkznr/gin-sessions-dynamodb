package dynamodb

import (
	"math/rand"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/tester"
	"github.com/nabeken/gorilla-sessions-dynamodb/dynamostore"
)

const dynamoDBLocal = "http://127.0.0.1:8000"

func randSeq(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func newTestCreateTableInput(tableName string) *dynamodb.CreateTableInput {
	attributeName := aws.String(dynamostore.SessionIdHashKeyName)
	return &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: attributeName,
				AttributeType: aws.String(dynamodb.ScalarAttributeTypeS),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: attributeName,
				KeyType:       aws.String(dynamodb.KeyTypeHash),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
		TableName: aws.String(tableName),
	}
}

// prepareDynamoDBTable prepares DynamoDB table and it returns table name.
func prepareDynamoDBTable(dynamodbClient *dynamodb.DynamoDB) string {
	dummyTableName := randSeq(10)

	input := newTestCreateTableInput(dummyTableName)
	dynamodbClient.CreateTable(input)

	dynamodbClient.WaitUntilTableExists(&dynamodb.DescribeTableInput{
		TableName: aws.String(dummyTableName),
	})

	return dummyTableName
}

var newStore = func(_ *testing.T) sessions.Store {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("ap-northeast-1"),
		Endpoint:    aws.String(dynamoDBLocal),
		Credentials: credentials.NewStaticCredentials("dummy", "dummy", "dummy"),
	})
	if err != nil {
		panic(err)
	}

	db := dynamodb.New(sess)
	dummyTableName := prepareDynamoDBTable(db)
	store := NewStore(db, dummyTableName, []byte("secret"))
	return store
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
