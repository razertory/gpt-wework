# gpt-wework
企业微信（客服）能力下的 GPT-3 微信机器人


### 最新更新
仓库已 python 重写，原有的逻辑可能无法兼容。新仓库地址：https://github.com/razertory/gpt-wework-py




目前越来越多的人开始用 GPT-3 相关的产品协助自己的工作和学习，在微信上也有不少接入API的机器人。
不过目前而言，想拥有微信原生的体验，多数是用登陆web微信的方式。这种做法有两个限制

- 用的微信号是能够登陆网页版微信的，这种号会越来越少，阁下可以试试自己的号能不能登陆[微信网页版](https://wx.qq.com/)，我想大概率是不能的
- 随着微信反外挂越来越强烈，这种操作一不注意就有封的风险，在下朋友圈就已经出现号被封的情况了。


在各种尝试和实验下，我上线了一种基于企业微信客服的做法。
- 能够媲美原生的体验：私聊，分享，翻译等等。
- 技术上和微信解耦

微信扫码关注

![](https://raw.githubusercontent.com/razertory/statics/main/staic/wechat_official_qr.jpg)

> 回复「客服」体验最终效果


## 开发要求
1. 一个用于接收请求企业微信、OpenAI 的服务器，支持该项目的 Golang 要求
2. 企业微信管理员账号，用来登陆后台，所有人都可以注册企业微信
3. OpenAI 账号用来获得 key



## 操作流程


#### 0.代码说明
*接口*
```
GET    /ping                     服务健康检查
GET    /wechat/check             企业微信服务器验证地址，在企业微信后台配置 域名+/wechat/check
POST   /wechat/check             企业微信服务器事件推送地址地址
POST   /chat                     OpenAI 聊天接口，用于测试
```

*配置文件*
参考 `.env.example` 文件，创建一个 .env 文件。服务启动的时候会 load `.env`, 如果不存在会 panic！

参数含义
```
# 验证企业微信回调的token
WEWORK_TOKEN=token
# 验证企业微信回调的key
WEWORK_ENCODING_AEK_KEY=encodingAesKey
# 企业微信企业id
WEWORK_CORP_ID=corpid
# 企业微信secret
WEWORK_CROP_SECRET=corpsecret
# openai key
OPENAI_KEY=key
```

#### 1.登陆（注册）你的 OpenAI 账号，拿到对应的 key
参数会用到 [gpt.go](./service/gpt.go) 当中

#### 2.注册并登陆企业微信后台
应用管理 - 微信客服
![](https://raw.githubusercontent.com/razertory/statics/main/staic/2.png)

#### 3.配置应用服务器
host + `/wechat/check`

注意，只有这些参数和企业微信`接收事件服务器`一致的时候，才能验证通过。代码中的 corpsecret 一定是通过企业微信获得的，首次获取一定是`企业微信app发送`
![](https://raw.githubusercontent.com/razertory/statics/main/staic/4.png)
![](https://raw.githubusercontent.com/razertory/statics/main/staic/5.png)

#### 4.配置机器人
让客服机器人被API接管
![](https://raw.githubusercontent.com/razertory/statics/main/staic/6.png)


## 其它
1. 由于 OpenAI 对大陆 ip 的限制，阁下所用的服务器推荐在大陆以外，或者给服务器套代理
2. 企业微信如果没有做企业备案，那么最多服务100人，这意味着阁下需要「拓展业务」，需要想办法做备案
3. 只针对备案后的企业微信：配置的事件接受服务器，需要和企业微信备案的主体一致。
4. 其它问题or商务合作：可以在公众号点击「加我微信」





