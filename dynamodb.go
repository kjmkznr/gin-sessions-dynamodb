package dynamodb

import (
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/gin-contrib/sessions"
	"github.com/nabeken/gorilla-sessions-dynamodb/dynamostore"
)

type Store interface {
	sessions.Store
}

// client: DynamoDB client (*dynamodbiface.DynamoDBAPI)
// tableName: Table name used by the DynamoDB store
// keyPairs: Keys are defined in pairs to allow key rotation, but the common case is to set a single
// authentication key and optionally an encryption key.
//
// The first key in a pair is used for authentication and the second for encryption. The
// encryption key can be set to nil or omitted in the last pair, but the authentication key
// is required in all pairs.
//
// It is recommended to use an authentication key with 32 or 64 bytes. The encryption key,
// if set, must be either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256 modes.
func NewStore(client dynamodbiface.DynamoDBAPI, tableName string, keyPairs ...[]byte) Store {
	return &store{dynamostore.New(client, tableName, keyPairs...)}
}

type store struct {
	*dynamostore.Store
}

func (c *store) Options(options sessions.Options) {
	c.Store.Options = options.ToGorillaOptions()

	// If MaxAge is not set, the session will be deleted immediately
	if c.Store.Options.MaxAge == 0 {
		c.Store.Options.MaxAge = dynamostore.DefaultSessionOpts.MaxAge
	} 
}
