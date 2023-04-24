package main

import (
	"gpt-wework/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	r := gin.Default()
	r.GET("/ping", Ping)
	r.GET("/wechat/check", service.CheckWeixinSign)
	r.POST("/wechat/check", service.TalkWeixin)
	r.POST("/chat", service.Chat)
	r.Run(":8888")
}

func Ping(c *gin.Context) {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}
	c.Data(500, "text/plain;charset=utf-8", []byte("ff"))
}
