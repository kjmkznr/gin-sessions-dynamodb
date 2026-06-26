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
