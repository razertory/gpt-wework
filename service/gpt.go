package service

import (
	"context"
	"fmt"

	gogpt "github.com/sashabaranov/go-gpt3"
)

// 停顿符，用来阻止 GPT-3 补充联想内容
var stop = "*#06#"

// openai key
var apiKey = "apiKey"

// openai orgId
var orgId = "orgId"

type ChatGPT struct {
	client *gogpt.Client
	ctx    context.Context
	userId string
}

func Ask(question string) (string, error) {
	k, orgId := apiKey, orgId
	chat := NewGPT(k, orgId)
	defer chat.Close()
	answer, err := chat.Chat(question)
	if err != nil {
		fmt.Print(err.Error())
	}
	return answer, err
}

func (c *ChatGPT) Chat(question string) (answer string, err error) {
	var msg = gogpt.ChatCompletionMessage{}
	msg.Content = question
	msg.Role = "system"
	req := gogpt.ChatCompletionRequest{
		Model:    gogpt.GPT3Dot5Turbo,
		Messages: []gogpt.ChatCompletionMessage{msg},
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
