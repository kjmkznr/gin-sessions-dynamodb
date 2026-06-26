package dynamodb

import (
	"bytes"
	"context"
	"encoding/base32"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	ginsessions "github.com/gin-contrib/sessions"
	"github.com/gorilla/securecookie"
	gsessions "github.com/gorilla/sessions"
)

const SessionIDHashKeyName = "SessionID"

var DefaultSessionOptions = &gsessions.Options{
	Path:   "/",
	MaxAge: 86400 * 30,
}

// DynamoDBAPI is the minimal interface satisfied by *dynamodb.Client from aws-sdk-go-v2.
type DynamoDBAPI interface {
	GetItem(ctx context.Context, in *awsdynamodb.GetItemInput, opts ...func(*awsdynamodb.Options)) (*awsdynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, in *awsdynamodb.PutItemInput, opts ...func(*awsdynamodb.Options)) (*awsdynamodb.PutItemOutput, error)
	DeleteItem(ctx context.Context, in *awsdynamodb.DeleteItemInput, opts ...func(*awsdynamodb.Options)) (*awsdynamodb.DeleteItemOutput, error)
}

var _ DynamoDBAPI = (*awsdynamodb.Client)(nil)

// Store is the public interface returned by NewStore.
type Store interface {
	ginsessions.Store
}

type dynamoStore struct {
	client    DynamoDBAPI
	tableName string
	codecs    []securecookie.Codec
	options   *gsessions.Options
}

// NewStore creates a new DynamoDB-backed session store.
// client must satisfy DynamoDBAPI (e.g. *dynamodb.Client from aws-sdk-go-v2).
// keyPairs follow the same convention as gorilla/securecookie: pairs of
// authentication key (required) and optional encryption key.
func NewStore(client DynamoDBAPI, tableName string, keyPairs ...[]byte) Store {
	opts := *DefaultSessionOptions
	return &dynamoStore{
		client:    client,
		tableName: tableName,
		codecs:    securecookie.CodecsFromPairs(keyPairs...),
		options:   &opts,
	}
}

func (s *dynamoStore) Options(options ginsessions.Options) {
	gopts := options.ToGorillaOptions()
	if gopts.MaxAge == 0 {
		gopts.MaxAge = DefaultSessionOptions.MaxAge
	}
	s.options = gopts
}

func (s *dynamoStore) Get(r *http.Request, name string) (*gsessions.Session, error) {
	return gsessions.GetRegistry(r).Get(s, name)
}

func (s *dynamoStore) New(r *http.Request, name string) (*gsessions.Session, error) {
	sess := gsessions.NewSession(s, name)
	opts := *s.options
	sess.Options = &opts
	sess.IsNew = true

	c, err := r.Cookie(name)
	if err != nil {
		return sess, nil
	}
	if err := securecookie.DecodeMulti(name, c.Value, &sess.ID, s.codecs...); err != nil {
		return sess, nil
	}
	if err := s.load(r.Context(), sess); err != nil {
		return sess, nil
	}
	sess.IsNew = false
	return sess, nil
}

func (s *dynamoStore) Save(r *http.Request, w http.ResponseWriter, sess *gsessions.Session) error {
	if sess.Options.MaxAge < 0 {
		if sess.ID != "" {
			if err := s.delete(r.Context(), sess.ID); err != nil {
				return err
			}
		}
		http.SetCookie(w, gsessions.NewCookie(sess.Name(), "", sess.Options))
		return nil
	}
	if sess.ID == "" {
		sess.ID = strings.TrimRight(
			base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32)), "=")
	}
	if err := s.save(r.Context(), sess); err != nil {
		return err
	}
	encoded, err := securecookie.EncodeMulti(sess.Name(), sess.ID, s.codecs...)
	if err != nil {
		return err
	}
	http.SetCookie(w, gsessions.NewCookie(sess.Name(), encoded, sess.Options))
	return nil
}

func (s *dynamoStore) load(ctx context.Context, sess *gsessions.Session) error {
	out, err := s.client.GetItem(ctx, &awsdynamodb.GetItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			SessionIDHashKeyName: &types.AttributeValueMemberS{Value: sess.ID},
		},
		ConsistentRead: ptrBool(true),
	})
	if err != nil {
		return err
	}
	if len(out.Item) == 0 {
		return errors.New("session not found")
	}
	if expAV, ok := out.Item["ExpiresAt"].(*types.AttributeValueMemberN); ok {
		if isExpired(expAV.Value) {
			return errors.New("session expired")
		}
	}
	dataAV, ok := out.Item["Data"].(*types.AttributeValueMemberS)
	if !ok {
		return errors.New("session data missing")
	}
	return decodeValues(dataAV.Value, &sess.Values)
}

func (s *dynamoStore) save(ctx context.Context, sess *gsessions.Session) error {
	encoded, err := encodeValues(sess.Values)
	if err != nil {
		return err
	}
	exp := time.Now().Add(time.Duration(sess.Options.MaxAge) * time.Second).Unix()
	_, err = s.client.PutItem(ctx, &awsdynamodb.PutItemInput{
		TableName: &s.tableName,
		Item: map[string]types.AttributeValue{
			SessionIDHashKeyName: &types.AttributeValueMemberS{Value: sess.ID},
			"Data":               &types.AttributeValueMemberS{Value: encoded},
			"ExpiresAt":          &types.AttributeValueMemberN{Value: strconv.FormatInt(exp, 10)},
		},
	})
	return err
}

func (s *dynamoStore) delete(ctx context.Context, id string) error {
	_, err := s.client.DeleteItem(ctx, &awsdynamodb.DeleteItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			SessionIDHashKeyName: &types.AttributeValueMemberS{Value: id},
		},
	})
	return err
}

func encodeValues(v map[interface{}]interface{}) (string, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func decodeValues(s string, v *map[interface{}]interface{}) error {
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewReader(raw)).Decode(v)
}

func ptrBool(b bool) *bool { return &b }

func isExpired(s string) bool {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return false
	}
	return time.Now().Unix() > n
}
