# gin-session-dynamodb

[![PkgGoDev](https://pkg.go.dev/badge/github.com/kjmkznr/gin-sessions-dynamodb/v2)](https://pkg.go.dev/github.com/kjmkznr/gin-sessions-dynamodb/v2)

A session store backend for [gin-contrib/sessions](https://github.com/gin-contrib/sessions/) using AWS DynamoDB.

## Requirements

- Go 1.21+
- AWS SDK for Go v2

## Usage

```go
import dynamodbstore "github.com/kjmkznr/gin-sessions-dynamodb/v2"
```

## Basic Example

```go
package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	dynamodbstore "github.com/kjmkznr/gin-sessions-dynamodb/v2"
)

func main() {
	r := gin.Default()

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("ap-northeast-1"),
	)
	if err != nil {
		panic(err)
	}
	ddb := awsdynamodb.NewFromConfig(cfg)
	store := dynamodbstore.NewStore(ddb, "SessionTable", []byte("secret"))
	r.Use(sessions.Sessions("mysession", store))

	r.GET("/incr", func(c *gin.Context) {
		sess := sessions.Default(c)
		var count int
		v := sess.Get("count")
		if v == nil {
			count = 0
		} else {
			count = v.(int)
			count++
		}
		sess.Set("count", count)
		if err := sess.Save(); err != nil {
			println(err.Error())
		}
		c.JSON(200, gin.H{"count": count})
	})
	r.Run(":8000")
}
```

## DynamoDB Table Schema

| Attribute | Type | Role |
|---|---|---|
| `SessionID` | String (S) | Hash key |
| `Data` | String (S) | gob-encoded session values (base64) |
| `ExpiresAt` | Number (N) | Unix epoch seconds (DynamoDB TTL) |

## Testing

```shell
$ docker run -p 8000:8000 -d amazon/dynamodb-local:latest
$ go test ./...
```
