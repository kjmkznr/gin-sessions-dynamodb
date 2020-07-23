package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	dynamodbstore "github.com/kjmkznr/gin-sessions-dynamodb"
)

func main() {
	r := gin.Default()
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("ap-northeast-1"),
	})
	if err != nil {
		panic(err)
	}

	ddb := dynamodb.New(sess)
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
		err := sess.Save()
		if err != nil {
			println(err.Error())
		}
		c.JSON(200, gin.H{"count": count})
	})
	r.Run(":8000")
}
