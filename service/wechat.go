package service

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type WeixinUserAskMsg struct {
	ToUserName string `xml:"ToUserName"`
	CreateTime int64  `xml:"CreateTime"`
	MsgType    string `xml:"MsgType"`
	Event      string `xml:"Event"`
	Token      string `xml:"Token"`
	OpenKfId   string `xml:"OpenKfId"`
}

type AccessToken struct {
	Errcode     int    `json:"errcode"`
	Errmsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type MsgRet struct {
	Errcode    int    `json:"errcode"`
	Errmsg     string `json:"errmsg"`
	NextCursor string `json:"next_cursor"`
	MsgList    []Msg  `json:"msg_list"`
}
type Msg struct {
	Msgid    string `json:"msgid"`
	SendTime int64  `json:"send_time"`
	Origin   int    `json:"origin"`
	Msgtype  string `json:"msgtype"`
	Event    struct {
		EventType      string `json:"event_type"`
		Scene          string `json:"scene"`
		OpenKfid       string `json:"open_kfid"`
		ExternalUserid string `json:"external_userid"`
		WelcomeCode    string `json:"welcome_code"`
	} `json:"event"`
	Text struct {
		Content string `json:"content"`
	} `json:"text"`
	OpenKfid       string `json:"open_kfid"`
	ExternalUserid string `json:"external_userid"`
}

type ReplyMsg struct {
	Touser   string `json:"touser,omitempty"`
	OpenKfid string `json:"open_kfid,omitempty"`
	Msgid    string `json:"msgid,omitempty"`
	Msgtype  string `json:"msgtype,omitempty"`
	Text     struct {
		Content string `json:"content,omitempty"`
	} `json:"text,omitempty"`
}

func TalkWeixin(c *gin.Context) {
	token := token
	receiverId := corpid
	encodingAeskey := encodingAesKey
	verifyMsgSign := c.Query("msg_signature")
	verifyTimestamp := c.Query("timestamp")
	verifyNonce := c.Query("nonce")
	crypt := NewWXBizMsgCrypt(token, encodingAeskey, receiverId, 1)
	bodyBytes, _ := ioutil.ReadAll(c.Request.Body)
	data, _ := crypt.DecryptMsg(verifyMsgSign, verifyTimestamp, verifyNonce, bodyBytes)
	var weixinUserAskMsg WeixinUserAskMsg
	err := xml.Unmarshal([]byte(string(data)), &weixinUserAskMsg)
	if err != nil {
		fmt.Println("err:  " + err.Error())
	}
	accessToken, err := accessToken()
	if err != nil {
		c.JSON(500, "ok")
		return
	}
	msgToken := weixinUserAskMsg.Token
	msgRet, err := getMsgs(accessToken, msgToken)
	if err != nil {
		c.JSON(500, "ok")
		return
	}
	if isRetry(verifyMsgSign) {
		c.JSON(200, "ok")
		return
	}
	go handleMsgRet(msgRet)
	c.JSON(200, "ok")
}

func TalkToUser(external_userid, open_kfid, ask, content string) {
	reply := ReplyMsg{
		Touser:   external_userid,
		OpenKfid: open_kfid,
		Msgtype:  "text",
		Text: struct {
			Content string `json:"content,omitempty"`
		}{Content: content},
	}
	atoken, err := accessToken()
	if err != nil {
		return
	}
	callTalk(reply, atoken)
}

func handleMsgRet(msgRet MsgRet) {
	fmt.Println(msgRet)
	size := len(msgRet.MsgList)
	if size < 1 {
		return
	}
	current := msgRet.MsgList[size-1]
	userId := current.ExternalUserid
	kfId := current.OpenKfid
	content := current.Text.Content
	if content == "" {
		return
	}
	ret, err := AskOnConversation(content, userId, weworkConversationSize)
	if err != nil {
		TalkToUser(userId, kfId, content, "服务器火爆")
		return
	}
	TalkToUser(userId, kfId, content, ret)
}

func isRetry(signature string) bool {
	var base = "retry:signature:%s"
	key := fmt.Sprintf(base, signature)
	_, found := retryCache.Get(key)
	if found {
		return true
	}
	retryCache.Set(key, "1", 1*time.Minute)
	return false
}

func getMsgs(accessToken, msgToken string) (MsgRet, error) {
	var msgRet MsgRet
	url := "https://qyapi.weixin.qq.com/cgi-bin/kf/sync_msg?access_token=" + accessToken
	method := "POST"
	payload := strings.NewReader(fmt.Sprintf(`{"token" : "%s"}`, msgToken))
	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		fmt.Println(err)
		return msgRet, err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return msgRet, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return msgRet, err
	}
	json.Unmarshal([]byte(string(body)), &msgRet)
	return msgRet, nil
}

func accessToken() (string, error) {
	var tokenCacheKey = "tokenCache"
	data, found := tokenCache.Get(tokenCacheKey)
	if found {
		return fmt.Sprintf("%v", data), nil
	}
	urlBase := "https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s"
	url := fmt.Sprintf(urlBase, corpid, corpsecret)
	method := "GET"
	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	s := string(body)
	var accessToken AccessToken
	json.Unmarshal([]byte(s), &accessToken)
	token := accessToken.AccessToken
	tokenCache.Set(tokenCacheKey, token, 5*time.Minute)
	return token, nil
}

func CheckWeixinSign(c *gin.Context) {
	token := token
	receiverId := corpid
	encodingAeskey := encodingAesKey
	wxcpt := NewWXBizMsgCrypt(token, encodingAeskey, receiverId, 1)
	/*
	   	------------使用示例一：验证回调URL---------------
	   	*企业开启回调模式时，企业微信会向验证url发送一个get请求
	   	假设点击验证时，企业收到类似请求：
	   	* GET /cgi-bin/wxpush?msg_signature=5c45ff5e21c57e6ad56bac8758b79b1d9ac89fd3&timestamp=1409659589&nonce=263014780&echostr=P9nAzCzyDtyTWESHep1vC5X9xho%2FqYX3Zpb4yKa9SKld1DsH3Iyt3tP3zNdtp%2B4RPcs8TgAE7OaBO%2BFZXvnaqQ%3D%3D
	   	* HTTP/1.1 Host: qy.weixin.qq.com

	   	接收到该请求时，企业应
	        1.解析出Get请求的参数，包括消息体签名(msg_signature)，时间戳(timestamp)，随机数字串(nonce)以及企业微信推送过来的随机加密字符串(echostr),
	        这一步注意作URL解码。
	        2.验证消息体签名的正确性
	        3. 解密出echostr原文，将原文当作Get请求的response，返回给企业微信
	        第2，3步可以用企业微信提供的库函数VerifyURL来实现。

	*/
	// 解析出url上的参数值如下：
	// verifyMsgSign := HttpUtils.ParseUrl("msg_signature")
	verifyMsgSign := c.Query("msg_signature")
	// verifyTimestamp := HttpUtils.ParseUrl("timestamp")
	verifyTimestamp := c.Query("timestamp")
	// verifyNonce := HttpUtils.ParseUrl("nonce")
	verifyNonce := c.Query("nonce")
	// verifyEchoStr := HttpUtils.ParseUrl("echoStr")
	verifyEchoStr := c.Query("echostr")
	echoStr, cryptErr := wxcpt.VerifyURL(verifyMsgSign, verifyTimestamp, verifyNonce, verifyEchoStr)
	if nil != cryptErr {
		panic(111)
	}
	c.Data(200, "text/plain;charset=utf-8", []byte(echoStr))
}

func callTalk(reply ReplyMsg, accessToken string) error {
	url := "https://qyapi.weixin.qq.com/cgi-bin/kf/send_msg?access_token=" + accessToken
	method := "POST"
	data, err := json.Marshal(reply)
	if err != nil {
		return err
	}
	reqBody := string(data)
	fmt.Println(reqBody)
	payload := strings.NewReader(reqBody)
	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}
	s := string(body)
	fmt.Println(s)
	return nil
}
