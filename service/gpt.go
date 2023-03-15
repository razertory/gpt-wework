package service

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	gogpt "github.com/sashabaranov/go-gpt3"
)

// openai key
var apiKey = ""

// 这是一个可以自定义的 id，用默认值不会有问题
var userId = "orgId"

// 企业微信 token 缓存，请求频次过高可能有一些额外的问题
var conversationCache = cache.New(5*time.Minute, 5*time.Minute)

type ChatGPT struct {
	client *gogpt.Client
	ctx    context.Context
	userId string
}

func Chat(c *gin.Context) {
	question := c.Query("question")
	conversationId := c.Query("conversationId")
	ret, err := AskOnConversation(question, conversationId, weworkConversationSize)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, ret)
}

func AskOnConversation(question, conversationId string, size int) (string, error) {
	var messages = []gogpt.ChatCompletionMessage{}
	key := fmt.Sprintf("cache:conversation:%s", conversationId)
	data, found := conversationCache.Get(key)
	if found {
		messages = data.([]gogpt.ChatCompletionMessage)
	}
	messages = append(messages, gogpt.ChatCompletionMessage{
		Role:    "system",
		Content: question,
	})
	fmt.Println(messages)
	pivot := size
	if pivot > len(messages) {
		pivot = len(messages)
	}
	messages = messages[len(messages)-pivot:]
	conversationCache.Set(key, messages, 12*time.Hour)
	k, userId := apiKey, userId
	chat := NewGPT(k, userId)
	defer chat.Close()
	answer, err := chat.Chat(messages)
	if err != nil {
		fmt.Print(err.Error())
	}
	return answer, err
}

func (c *ChatGPT) Chat(messages []gogpt.ChatCompletionMessage) (answer string, err error) {
	var msg = gogpt.ChatCompletionMessage{}
	msg.Role = "system"
	req := gogpt.ChatCompletionRequest{
		Model:    gogpt.GPT3Dot5Turbo,
		Messages: messages,
	}
	resp, err := c.client.CreateChatCompletion(c.ctx, req)
	if err != nil {
		return "", err
	}
	answer = resp.Choices[0].Message.Content
	for len(answer) > 0 {
		if answer[0] == '\n' {
			answer = answer[1:]
		} else {
			break
		}
	}
	return answer, err
}

func NewGPT(ApiKey, UserId string) *ChatGPT {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-ctx.Done()
		cancel()
	}()
	return &ChatGPT{
		client: gogpt.NewClient(ApiKey),
		ctx:    ctx,
		userId: UserId,
	}
}
func (c *ChatGPT) Close() {
	c.ctx.Done()
}
