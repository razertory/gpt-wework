package main

import (
	"gpt-wework/service"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	r := gin.Default()
	r.GET("/ping", Ping)
	r.GET("/wechat/check", service.CheckWeixinSign)
	r.POST("/wechat/check", service.TalkWeixin)
	r.POST("/chat", service.Chat)
	var listenAddr = os.Getenv("LISTEN_ADDR")
	r.Run(listenAddr)
}

func Ping(c *gin.Context) {
	c.Data(200, "text/plain;charset=utf-8", []byte(os.Getenv("WEWORK_CORP_ID")))
}
