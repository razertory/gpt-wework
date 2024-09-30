package service

import (
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/patrickmn/go-cache"
)

var token = os.Getenv("WEWORK_TOKEN")

var encodingAesKey = os.Getenv("WEWORK_ENCODING_AEK_KEY")

// 企业微信企业id
var corpid = os.Getenv("WEWORK_CORP_ID")

// 企业微信secret
var corpsecret = os.Getenv("WEWORK_CROP_SECRET")

// openai key
var openAiKey = os.Getenv("OPENAI_KEY")

// 企业微信的重试缓存，如果服务器延迟低，可以去掉该变量以及 isRetry 逻辑
var retryCache = cache.New(60*time.Minute, 10*time.Minute)

// 企业微信 token 缓存，请求频次过高可能有一些额外的问题
var tokenCache = cache.New(5*time.Minute, 5*time.Minute)

// 上下文对话能力，默认是 3, 可以根据需要修改对话长度
var weworkConversationSize = 3
