# AI Chat 实现详解

> 面向有 Web 开发经验但第一次接触 AI 应用的开发者。所有 AI 相关概念都会用 Web 开发中的对应概念做类比。

---

## 一、整体架构

```
┌─────────────────────────────────────────────────────────────┐
│  模型层（pkg/ai/model_pool.go）—— 全局单例池                  │
│  deepseek │ qwen │ glm │ glm-4v                           │
│  类比：数据库连接池，只负责"推理"，不存任何用户数据              │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│  会话层（services/user_service/aichat_service.go）            │
│  每次请求新建，不驻留内存                                      │
│  流程：读DB历史 → 构建messages → 调模型 → SSE推送 → 存DB       │
│  类比：普通 HTTP 请求处理，每次请求都是独立的                    │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│  存储层（MySQL + Redis + Kafka）                               │
│  所有聊天记录存数据库，绝对不放在结构体里                        │
│  Kafka 异步持久化 AI 响应，避免阻塞 SSE 流                     │
└─────────────────────────────────────────────────────────────┘
```

### 与传统 Web 项目的核心区别

| Web 项目 | AI 聊天项目 |
|----------|------------|
| 请求 → 处理 → 返回完整响应 | 请求 → 流式返回（边生成边推送） |
| 响应时间固定（几十ms~几百ms） | 响应时间不固定（几秒~几十秒） |
| 不需要维护"上下文" | 每次请求需要携带历史对话作为上下文 |
| 无状态，每次请求独立 | 需要"记住"之前的对话 |

---

## 二、核心概念解释

### 2.1 什么是 ChatModel？

**类比**：把 ChatModel 想象成一个"远程 API 调用"。

```go
// 传统 Web：调用外部 API
resp := http.Post("https://api.example.com/translate", body)

// AI 聊天：调用大语言模型
stream := model.Stream(ctx, messages)
```

ChatModel 就是和 AI 模型（DeepSeek/Qwen/GLM）对话的接口。它有两个方法：

- **`Generate()`**：一次性返回完整响应（类似普通 HTTP 请求）
- **`Stream()`**：逐块返回响应（类似 WebSocket 推送，但方向是单向的）

本项目用 `Stream()`，因为 AI 生成内容需要几秒，流式推送可以让用户看到"打字机效果"。

### 2.2 什么是 Message（消息角色）？

AI 对话中的每条消息都有一个"角色"，这是 AI 理解对话的关键：

| 角色 | 含义 | 类比 |
|------|------|------|
| `system` | 系统指令，告诉 AI 它的身份和行为��则 | 服务器的全局配置 |
| `user` | 用户说的是什么 | 客户端请求 |
| `assistant` | AI 的回复（带图片时为多模态） | 服务器响应 |

**为什么需要历史消息？**

AI 模型本身是无状态的（类似 HTTP 协议），它不记得之前说过什么。所以每次调用时，需要把整个对话历史一起发过去：

```
// 第一次对话
messages = [
    UserMessage("你好"),
]
// AI 回复: "你好！有什么可以帮你的？"

// 第二次对话（必须带上第一次的完整历史）
messages = [
    UserMessage("你好"),
    AssistantMessage("你好！有什么可以帮你的？"),
    UserMessage("帮我写个排序算法"),
]
// AI 回复: "好的，这是一个快速排序的实现..."
```

这就是为什么 `SendMessageStream` 中要从 DB 读取历史消息构建上下文。

### 2.3 什么是 SSE（Server-Sent Events）？

**类比**：SSE 是单向的 WebSocket。

| 特性 | WebSocket | SSE |
|------|---------|-----|
| 通信方向 | 双向 | 服务端 → 客户端（单向） |
| 协议 | `ws://` | `http://`（基于 HTTP） |
| 复杂度 | 高（需要心跳、重连等） | 低（浏览器原生支持） |
| 适用场景 | 实时聊天、游戏 | 流式 AI 响应、股票行情 |

SSE 的格式很简单：

```
data: {"content": "你好"}

data: {"content": "，有什么"}

data: {"content": "可以帮你的？"}

data: {"done": true}
```

**为什么要转义？**

如果 AI 返回的内容包含换行符或引号，需要转义：
- `\n` → `\\n`
- `"` → `\"`

例如 AI 返回 `"你好\nWorld"`，转义后变成：
```
data: {"content": "你好\\nWorld"}
```

每一行以 `data: ` 开头，空行（`\n\n`）表示一条消息结束。前端用 `EventSource` 接收：

```javascript
const evtSource = new EventSource('/user/aichat/sendMessage?session_id=xxx&content=你好&model_type=deepseek');
evtSource.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (data.content) {
        // 逐字追加到页面上
        aiText += data.content;
    }
    if (data.done) {
        evtSource.close();
    }
    if (data.error) {
        console.error('AI 响应错误:', data.error);
        evtSource.close();
    }
};
```

### 2.4 为什么要用 Kafka 异步持久化？

**问题**：AI 响应通过 SSE 流式推送时，如果同步写数据库，会阻塞推送流，导致用户体验卡顿。

**解决方案**：
1. SSE 流式推送（实时，用户体验好）
2. 完整响应写 DB（如果 DB 写入慢或失败，发到 Kafka）
3. Kafka 消费者异步从队列读取并写入 MySQL

```
用户发送消息
    ↓
保存用户消息到 DB（同步，用户立刻看到自己发的消息）
    ↓
调用 AI 模型，流式推送 SSE（不阻塞）
    ↓
AI 响应完成
    ↓
尝试写 DB ──成功──→ 更新会话
    │
    └─失败──→ 发到 Kafka ──→ 消费者异步写 DB
```

**类比**：就像电商下单后，订单立刻确认（同步），但发货通知通过消息队列异步处理。

---

## 三、分层详解

### 3.1 模型层（`pkg/ai/model_pool.go`）

**职责**：管理 AI 模型实例，全局只初始化一次。

```go
type ModelPool struct {
    DeepSeek model.BaseChatModel  // DeepSeek 模型实例
    Qwen     model.BaseChatModel  // 通义千问模型实例
    GLM      model.BaseChatModel  // 智谱 GLM 模型实例
}
```

**关键设计**：

1. **`sync.Once` 保证单例**：和数据库连接池一样，整个应用生命周期只初始化一次
   ```go
   var modelPoolOnce sync.Once
   
   func GetModelPool() *ModelPool {
       modelPoolOnce.Do(func() {
           // 只执行一次
       })
       return modelPool
   }
   ```

2. **配置优先级**：配置文件 → 环境变量
   ```go
   // 优先读 config.toml
   if cfg.DeepseekApiKey != "" {
       // 用配置文件
   } else if apiKey := os.Getenv("DEEPSEEK_API_KEY"); apiKey != "" {
       // 回退到环境变量
   }
   ```

3. **统一接口**：三个模型都通过 `openai.NewChatModel` 创建，因为它们都兼容 OpenAI 协议
   - DeepSeek: `https://api.deepseek.com/v1`
   - Qwen: `https://dashscope.aliyuncs.com/compatible-mode/v1`
   - GLM: `https://open.bigmodel.cn/api/paas/v4/`

4. **按需获取**：`GetModel(modelType)` 根据类型返回对应模型，默认返回 DeepSeek

**多模态模型支持**：

GLM-4V 是多模态模型，支持图片输入。在 `SendMessageStream` 中：
```go
// 判断是否为多模态模型
isMultiModalModel := req.ModelType == "glm-4v"

// 如果是多模态模型，构建多模态消息
if isMultiModalModel && msg.Url != "" {
    multiMsg := &schema.Message{
        Role: schema.User,
        UserInputMultiContent: []schema.MessageInputPart{
            {Type: schema.ChatMessagePartTypeText, Text: msg.Content},
            {Type: schema.ChatMessagePartTypeImageURL, Image: &schema.MessageInputImage{
                MessagePartCommon: schema.MessagePartCommon{URL: &imageURL},
            }},
        },
    }
    chatMessages = append(chatMessages, multiMsg)
}
```

**为什么不存用户数据？**

模型实例只是一个"调用接口"，类似 HTTP Client。它不知道谁在对话、对话内容是什么。所有用户数据都在数据库里。

---

### 3.2 会话层（`services/user_service/aichat_service.go`）

**职责**：处理 AI 聊天的核心业务逻辑。

#### 3.2.1 CreateSession —— 创建会话

```
用户请求 → 生成 sessionId（S前缀） → 生成 AI receiveId（A前缀） → 存入 session 表
```

**为什么用 'A' 前缀？**

项目用 `receive_id` 的前缀区分聊天对象类型：
- `U` 开头 → 用户（User）
- `G` 开头 → 群组（Group）
- `A` 开头 → AI

这样查询 AI 会话时只需 `WHERE receive_id LIKE 'A%'`，不需要额外字段。

#### 3.2.2 GetSessionList —— 获取会话列表

```
根据 userId 查 session 表 → 筛选 receive_id 以 'A' 开头 → 分页返回
```

#### 3.2.3 GetMessageHistory —— 获取历史

```
根据 sessionId 查 message 表 → 按 created_at 降序分页返回
```

**为什么降序？** 聊天界面需要显示最近的消息，所以按时间降序（DESC），前端再反转为正序显示。

#### 3.2.4 SendMessageStream —— 核心流式对话

这是整个项目最复杂的方法，分 7 步：

**Step 1：校验会话**
```go
session, err := s.sessionDAO.GetAISessionByUuid(req.SessionId)
// 确认会话存在且是 AI 会话（receive_id 以 'A' 开头）
```

**Step 2：获取用户信息**
```go
user, err := s.userInfoDAO.FindUserByUuid(userId)
// 用于保存消息时带上用户昵称和头像
userName := user.Nickname
userAvatar := user.Avatar
```

**Step 3：保存用户消息**
```go
// 处理消息类型：有图片时为文件类型(Type=2)，否则为文本(Type=0)
msgType := int8(0)
if req.ImageURL != "" {
    msgType = 2
}

userMessage := &models.Message{
    Uuid:       "M" + 随机字符串,
    SessionId:  req.SessionId,
    Type:       msgType,
    Content:    content,
    Url:        req.ImageURL,  // 图片URL
    SendId:     userId,        // 用户 ID（U开头）
    SendName:   userName,       // 用户昵称
    SendAvatar: userAvatar,    // 用户头像
    ReceiveId:  session.ReceiveId,  // AI ID（A开头）
    Status:     1,  // 已发送
}
s.messageDAO.CreateMessage(userMessage)

// 如果是第一条消息，更新会话第一条消息
if session.FirstMessage == "" {
    s.sessionDAO.UpdateSessionFirstMessage(req.SessionId, content)
}
```

**Step 4：构建上下文**
```go
// 从 DB 读取最近 100 条消息
messages, _ := s.messageDAO.GetMessagesBySessionId(req.SessionId, 100, 0)

// 判断是否为多模态模型
isMultiModalModel := req.ModelType == "glm-4v"

// 转换为 eino 格式
var chatMessages []*schema.Message
chatMessages = append(chatMessages, schema.SystemMessage("你是一个专业的AI助手..."))

for _, msg := range messages {
    if msg.SendId == userId {
        // 只有多模态模型才处理图片
        if isMultiModalModel && msg.Url != "" {
            // 构建多模态消息
            ...
        } else {
            chatMessages = append(chatMessages, schema.UserMessage(msg.Content))
        }
    } else {
        chatMessages = append(chatMessages, schema.AssistantMessage(msg.Content, nil))
    }
}
```

**关键理解**：这里通过 `SendId` 判断角色：
- `SendId == userId` → 这是用户说的 → `UserMessage`
- `SendId != userId`（即 AI 发的）→ `AssistantMessage`

**Step 5：调用模型流式推理**
```go
pool := aipkg.GetModelPool()
model := pool.GetModel(req.ModelType)
stream, _ := model.Stream(ctx, chatMessages)
defer stream.Close()  // 必须关闭，否则连接泄漏
```

**Step 6：逐块推送 SSE**
```go
var fullContent strings.Builder
for {
    chunk, err := stream.Recv()
    if errors.Is(err, io.EOF) { break }
    if err != nil { break }
    
    if chunk != nil && len(chunk.Content) > 0 {
        fullContent.WriteString(chunk.Content)
        onChunk(chunk.Content)  // 回调函数，通过 SSE 推送给前端
    }
}

// 如果 AI 没有返回任何内容，发送默认回复
if finalContent := fullContent.String(); finalContent == "" {
    finalContent = "抱歉，我暂时无法回答这个问题。"
    onChunk(finalContent)
}
```

**为什么要转义？**
- AI 可能返回换行符 `\n`，会破坏 SSE 消息边界
- AI 可能返回引号 `"`，会破坏 JSON 格式

**Step 7：保存 AI 响应**
```go
// 发送 AI 响应到 Kafka 异步持久化
aiSendId := "A" + 随机字符串
aipkg.SendAIMessage(aipkg.AIMessagePayload{
    SessionId:  req.SessionId,
    SendId:     aiSendId,      // AI 作为发送者
    SendName:   "AI助手",
    ReceiveId:  userId,        // 用户作为接收者
    Content:    finalContent,
    ModelType:  req.ModelType,
})

// 更新会话最后一条消息
s.sessionDAO.UpdateSessionLastMessage(req.SessionId, finalContent, userMessage.CreatedAt)

// 发送完成信号
if onComplete != nil {
    onComplete(finalContent)
}
```

**为什么要用 Kafka 异步持久化？**

- AI 响应已经通过 SSE 实时推送给用户了
- 如果同步写 DB 慢，会阻塞后续的 AI 调用
- Kafka 保证消息不丢失，最终一致性由消费者保障

#### 3.2.5 DeleteSession —— 删除会话

删除会话时级联删除所有相关消息：
```go
// 先删除所有消息
s.messageDAO.DeleteMessagesBySessionId(sessionId)
// 再删除会话
s.sessionDAO.DeleteAISession(sessionId)
```

---

### 3.3 存储层

#### 3.3.1 数据库表复用

**不新建表**，复用现有的 `session` 和 `message` 表：

**session 表**：
| 字段 | 说明 |
|------|------|
| uuid | 会话 ID（S 前缀） |
| send_id | 创建者（用户 ID） |
| receive_id | AI ID（A 前缀，如 `A20260403123456`） |
| first_message | 第一条消息（用于会话列表展示） |
| last_message | 最后一条消息内容 |
| last_message_at | 最后消息时间 |

**message 表**：
| 字段 | 说明 |
|------|------|
| uuid | 消息 ID（M 前缀） |
| session_id | 所属会话 ID |
| send_id | 发送者（用户 U 开头 / AI A 开头） |
| receive_id | 接收者 |
| content | 消息内容 |
| url | 图片 URL（多模态） |
| type | 0 = 文本, 2 = 文件（图片） |

**角色推断规则**：
- `send_id` 以 `U` 开头 → 用户消息
- `send_id` 以 `A` 开头 → AI 回复

#### 3.3.2 Kafka

**topic 配置**（`config.toml`）：
```toml
[kafkaConfig]
chatTopic = "chat_message"     # 人聊
aiChatTopic = "ai_chat_message" # AI 聊
```

**消费者**（`pkg/ai/ai_kafka.go`）：
```go
func StartAIConsumer() {
    go func() {
        for {
            msg := kafka.KafkaServiceInstance.AIChatReader.ReadMessage(ctx)
            payload := 解析 msg.Value
            db.GormDB.Create(&models.Message{...})
        }
    }()
}
```

---

### 3.4 Controller 层（`controllers/user/aichat_controller.go`）

#### 普通接口（JSON 响应）

```go
func (c *AIChatController) CreateSession(ctx *gin.Context) {
    var req userreq.CreateAISessionRequest
    ctx.ShouldBindJSON(&req)
    
    userId := ctx.GetString("Uuid")  // 从 JWT 中间件获取
    result, _ := c.aiChatService.CreateSession(userId, req)
    
    resp.Success(ctx, "创建AI会话成功", result)
}
```

和普通 Web 接口完全一样。

#### SSE 流式接口

```go
func (c *AIChatController) SendMessage(ctx *gin.Context) {
    // 1. 解析 Query 参数（SSE 必须用 GET）
    var req userreq.SendAIMessageRequest
    if err := ctx.ShouldBind(&req); err != nil {
        resp.Error(ctx, "参数错误", http.StatusBadRequest)
        return
    }

    // 2. 如果有图片文件，上传到 OSS（支持 form-data 上传）
    if file, err := ctx.FormFile("image"); err == nil {
        result, err := c.fileService.UploadFile(ctx.Request.Context(), file, oss.MessageImage)
        req.ImageURL = result.URL
    }

    // 3. 设置 SSE 响应头
    ctx.Header("Content-Type", "text/event-stream")
    ctx.Header("Cache-Control", "no-cache")
    ctx.Header("Connection", "keep-alive")
    ctx.Status(http.StatusOK)
    ctx.Writer.Flush()

    // 4. 定义推��回��
    onChunk := func(chunk string) error {
        // 转义特殊字符，防止破坏 JSON 格式
        escaped := strings.ReplaceAll(chunk, "\n", "\\n")
        escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
        
        fmt.Fprintf(ctx.Writer, "data: {\"content\": \"%s\"}\n\n", escaped)
        ctx.Writer.Flush()  // 关键：立即推送
        return nil
    }

    onComplete := func(fullContent string) error {
        fmt.Fprintf(ctx.Writer, "data: {\"done\": true}\n\n")
        ctx.Writer.Flush()
        return nil
    }

    // 5. 调用 Service（阻塞直到流结束）
    err := c.aiChatService.SendMessageStream(ctx.Request.Context(), userId, req, onChunk, onComplete)
    if err != nil {
        // 发送错误信号给前端
        fmt.Fprintf(ctx.Writer, "data: {\"error\": \"%s\"}\n\n", err.Error())
        ctx.Writer.Flush()
    }
}
```

**与普通接口的区别**：
- 不返回 JSON，而是持续写入 `ctx.Writer`
- 每次写入后必须 `Flush()`，否则数据会缓存在缓冲区
- 连接保持打开，直到流结束或客户端断开
- 错误时发送 `data: {"error": "xxx"}` 信号
- 支持 `FormFile` 上传图片文件

---

## 四、完整请求流程

以用户发送一条消息为例：

```
1. 前端发起 GET 请求
   GET /user/aichat/sendMessage?session_id=Sxxx&content=你好&model_type=deepseek
   （带 JWT token，经过 Auth 中间件校验）

2. Controller 层
   - 解析 query 参数（支持 form-data 上传图片）
   - 设置 SSE 响应头
   - 定义 onChunk 回调（写入 HTTP 响应流）
   - 调用 Service.SendMessageStream()

3. Service 层
   a. 校验会话存在（sessionDAO.GetAISessionByUuid）
   b. 获取用户信息（userInfoDAO.FindUserByUuid）
   c. 保存用户消息到 DB（messageDAO.CreateMessage）
   d. 更新会话第一/最后消息（sessionDAO）
   e. 读取历史消息（messageDAO.GetMessagesBySessionId，最近100条）
   f. 构建 eino messages 数组（支持多模态）
   g. 获取模型实例（ai.GetModelPool().GetModel("glm-4v")）
   h. 调用 model.Stream() 开始流式推理

4. 流式处理循环
   for {
       chunk = stream.Recv()        // 等待 AI 生成下一个片段
       onChunk(chunk.Content)       // 通过 SSE 推送给前端
       fullContent += chunk.Content  // 累积完整内容
   }

5. 流结束后
   a. 发送 AI 响应到 Kafka 异步持久化
   b. 更新会话最后消息
   c. 发送 done 信号给前端

6. 前端接收
   data: {"content": "你"}
   
   data: {"content": "好"}
   
   data: {"content": "！有什么可以帮"}
   
   data: {"content": "你的？"}
   
   data: {"done": true}
   
   // 或错误情况
   data: {"error": "AI响应失败"}
```

---

## 五、API 接口文档

### 5.1 创建 AI 会话

```
POST /user/aichat/createSession
Authorization: Bearer <token>

Request:
{
    "model_type": "deepseek"   // deepseek | qwen | glm | glm-4v
}

Response:
{
    "code": 200,
    "msg": "创建AI会话成功",
    "data": {
        "session_id": "S20260403123456",
        "receive_id": "A20260403123457",
        "model_type": "deepseek"
    }
}
```

### 5.2 获取 AI 会话列表

```
POST /user/aichat/getSessionList
Authorization: Bearer <token>

Request:
{
    "page": 1,
    "page_size": 20
}

Response:
{
    "code": 200,
    "data": {
        "list": [
            {
                "session_id": "S20260403123456",
                "first_message": "你好",
                "created_at": "2026-04-03 12:34:56"
            }
        ],
        "total": 1
    }
}
```

### 5.3 获取消息历史

```
POST /user/aichat/getMessageHistory
Authorization: Bearer <token>

Request:
{
    "session_id": "S20260403123456",
    "page": 1,
    "page_size": 20
}

Response:
{
    "code": 200,
    "data": {
        "list": [
            {
                "session_id": "S20260403123456",
                "send_id": "U20260403111111",
                "send_name": "张三",
                "content": "你好",
                "type": 0,
                "url": "",
                "created_at": "2026-04-03 12:35:00"
            },
            {
                "session_id": "S20260403123456",
                "send_id": "A20260403123457",
                "send_name": "AI助手",
                "content": "你好！有什么可以帮你的？",
                "type": 0,
                "url": "",
                "created_at": "2026-04-03 12:35:02"
            }
        ],
        "total": 2
    }
}
```

### 5.4 发送消息（SSE 流式）

**支持 Query 参数：**
```
GET /user/aichat/sendMessage?session_id=Sxxx&content=你好&model_type=deepseek
Authorization: Bearer <token>
Accept: text/event-stream
```

**支持 Form-Data 上传图片：**
```
POST /user/aichat/sendMessage
Authorization: Bearer <token>
Content-Type: multipart/form-data

session_id: Sxxx
content: 分析这张图片
model_type: glm-4v
image: <图片文件>
```

**Response (SSE stream):**
```
data: {"content": "你"}

data: {"content": "��"}

data: {"content": "！"}

data: {"done": true}

或者错误情况：
data: {"error": "会话不存在"}
```

### 5.5 删除 AI 会话

```
POST /user/aichat/deleteSession
Authorization: Bearer <token>

Request:
{
    "session_id": "S20260403123456"
}

Response:
{
    "code": 200,
    "msg": "删除会话成功",
    "data": null
}
```

---

## 六、配置说明

### 6.1 config.toml

```toml
[kafkaConfig]
aiChatTopic = "ai_chat_message"   # AI 消息 Kafka topic

[aiModelConfig]
# DeepSeek
deepseekApiKey = "sk-xxx"
deepseekModel = "deepseek-chat"
deepseekBaseUrl = "https://api.deepseek.com/v1"

# 通义千问
qwenApiKey = "sk-xxx"
qwenModel = "qwen-plus"
qwenBaseUrl = "https://dashscope.aliyuncs.com/compatible-mode/v1"

# 智谱 GLM
glmApiKey = "xxx"
glmModel = "glm-4"
glmBaseUrl = "https://open.bigmodel.cn/api/paas/v4/"

# 智谱 GLM-4V（多模态）
glm4vApiKey = "xxx"
glm4vModel = "glm-4v"
glm4vBaseUrl = "https://open.bigmodel.cn/api/paas/v4/"
```

### 6.2 环境变量（备选）

如果配置文件中 API Key 为空，会自动从环境变量读取：

```bash
export DEEPSEEK_API_KEY="sk-xxx"
export DEEPSEEK_MODEL="deepseek-chat"
export DEEPSEEK_BASE_URL="https://api.deepseek.com/v1"

export QWEN_API_KEY="sk-xxx"
export QWEN_MODEL="qwen-plus"
export QWEN_BASE_URL="https://dashscope.aliyuncs.com/compatible-mode/v1"

export GLM_API_KEY="xxx"
export GLM_MODEL="glm-4"
export GLM_BASE_URL="https://open.bigmodel.cn/api/paas/v4/"

export GLM4V_API_KEY="xxx"
export GLM4V_MODEL="glm-4v"
export GLM4V_BASE_URL="https://open.bigmodel.cn/api/paas/v4/"
```

---

## 七、常见问题

### Q1: 为什么 sendMessage 用 GET 而不是 POST？

因为 SSE 通过 `EventSource` 发起，浏览器 `EventSource` API 只支持 GET 请求。如果要用 POST，前端需要手写 `fetch` + `ReadableStream`，复杂度更高。

如果需要传图片，可以用 Form-Data 上传，Controller 支持 `ctx.FormFile("image")`。

### Q2: 为什么需要转义？

AI 返回的内容可能包含换行符和引号：
- 换行符 `\n` 会破坏 SSE 消息边界
- 引号 `"` 会破坏 JSON 格式

所以要先转义再发送：`\n` → `\\n`，`"` → `\"`

### Q3: 上下文窗口有限制吗？

有。当前限制读取最近 100 条消息。如果对话很长，可以：
1. 增加条数限制
2. 使用消息摘要/总结（把早期对话压缩成一段摘要）
3. 使用 Token 计数，按 Token 数量截断

### Q4: 支持图片输入吗？

支持。满足以下条件即可：
1. `model_type` 设为 `glm-4v`（多模态模型）
2. 通过 Form-Data 上传图片文件，或提供 `image_url`

### Q5: 如果 AI 模型调用超时怎么办？

`model.Stream()` 会使用 `ctx`（请求上下文），如果客户端断开连接，context 会被取消，流式调用会自动终止。

### Q6: 多个用户同时调用会冲突吗？

不会。模型实例是无状态的（类似 HTTP Client），多个 goroutine 可以安全并发调用。每个请求的会话数据都从 DB 独立读取。

### Q7: 和 WebSocket 人聊的区别？

| 特性 | 人聊（WebSocket） | AI 聊（SSE） |
|------|-------------------|-------------|
| 实时性 | 双向实时 | 服务端单向推送 |
| 消息来源 | 另一个用户 | AI 模型推理 |
| 上下文 | 不需要 | 需要历史对话 |
| 响应时间 | 即时 | 几秒~几十秒 |
| 持久化 | 同步写 DB | 同步 + Kafka 降级 |
| 多模态 | 不支持 | 支持图片输入 |

---

## 八、文件结构

```
seekF-backend/
├── config/config.toml                    # AI 模���配��� + Kafka topic
├── internal/
│   ├── configs/configs.go                # AIModelConfig 结构体
│   ├── dao/user_dao/
│   │   ├── session_dao.go                # + AI 会话方法（5个）
│   │   └── message_dao.go               # + AI 消息方法
│   ├── dto/user/
│   │   ├── user_req/
│   │   │   ├── create_ai_session_request.go
│   │   │   ├── get_ai_session_list_request.go
│   │   │   ├── get_ai_message_history_request.go
│   │   │   └── send_ai_message_request.go   # + image_url
│   │   └── user_resp/
│   │       ├── create_ai_session_respond.go
│   │       ├── get_ai_session_list_respond.go
│   │       └── get_ai_message_history_respond.go
│   ├── services/user_service/
│   │   └── aichat_service.go            # 核心业务逻辑
│   ├── controllers/user/
│   │   └── aichat_controller.go         # HTTP 端点 + SSE
│   ├── pkg/
│   │   ├── ai/
│   │   │   ├── model_pool.go            # 模型单例池
│   │   │   └── ai_kafka.go              # AI Kafka 消费者
│   │   └── kafka/kafka.go               # + AI Writer/Reader 初始化
│   ├── router/router.go                 # + /user/aichat/* 路由
│   └── main.go                          # 初始化 AI 组件
└── docs/aichat.md                       # 本文档
```