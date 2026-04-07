# AI Chat 实现详解

> 面向有 Web 开发经验但第一次接触 AI 应用的开发者。所有 AI 相关概念都会用 Web 开发中的对应概念做类比。

---

## 一、整体架构

```
┌─────────────────────────────────────────────────────────────┐
│  模型层（pkg/ai/model_pool.go）—— 全局单例池                  │
│  deepseek │ qwen │ glm                                       │
│  类比：数据库连接池，只负责"推理"，不存任何用户数据              │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│  会话层（services/ai_service/aichat_service.go）               │
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
| `system` | 系统指令，告诉 AI 它的身份和行为准则 | 服务器的全局配置 |
| `user` | 用户说的话 | 客户端请求 |
| `assistant` | AI 的回复 | 服务器响应 |

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
|------|-----------|-----|
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

每一行以 `data: ` 开头，空行表示一条消息结束。前端用 `EventSource` 接收：

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

**为什么不存用户数据？**

模型实例只是一个"调用接口"，类似 HTTP Client。它不知道谁在对话、对话内容是什么。所有用户数据都在数据库里。

---

### 3.2 会话层（`services/ai_service/aichat_service.go`）

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

#### 3.2.2 GetMessageHistory —— 获取历史

```
根据 sessionId 查 message 表 → 按 created_at 正序分页返回
```

**为什么正序？** 聊天界面需要从最早的消息开始显示，所以按时间正序（ASC）。

#### 3.2.3 SendMessageStream —— 核心流式对话

这是整个项目最复杂的方法，分 6 步：

**Step 1：校验会话**
```go
session, err := s.sessionDAO.GetAISessionByUuid(req.SessionId)
// 确认会话存在且是 AI 会话（receive_id 以 'A' 开头）
```

**Step 2：保存用户消息**
```go
userMessage := &models.Message{
    Uuid:       "M" + 随机字符串,
    SessionId:  req.SessionId,
    SendId:     userId,        // 用户 ID（U开头）
    ReceiveId:  session.ReceiveId,  // AI ID（A开头）
    Content:    req.Content,
    Status:     1,  // 已发送
}
s.messageDAO.CreateMessage(userMessage)
```

**Step 3：构建上下文**
```go
// 从 DB 读取最近 100 条消息
messages, _ := s.messageDAO.GetMessagesBySessionId(req.SessionId, 100, 0)

// 转换为 eino 格式
var chatMessages []*schema.Message
for _, msg := range messages {
    if msg.SendId == userId {
        chatMessages = append(chatMessages, schema.UserMessage(msg.Content))
    } else {
        chatMessages = append(chatMessages, schema.AssistantMessage(msg.Content, nil))
    }
}
```

**关键理解**：这里通过 `SendId` 判断角色：
- `SendId == userId` → 这是用户说的 → `UserMessage`
- `SendId != userId`（即 AI 发的）→ `AssistantMessage`

**Step 4：调用模型流式推理**
```go
pool := aipkg.GetModelPool()
model := pool.GetModel(req.ModelType)
stream, _ := model.Stream(ctx, chatMessages)
defer stream.Close()  // 必须关闭，否则连接泄漏
```

**Step 5：逐块推送 SSE**
```go
var fullContent strings.Builder
for {
    chunk, err := stream.Recv()
    if errors.Is(err, io.EOF) { break }
    
    fullContent.WriteString(chunk.Content)
    onChunk(chunk.Content)  // 回调函数，通过 SSE 推送给前端
}
```

`onChunk` 是 Controller 层传入的回调：
```go
onChunk := func(chunk string) error {
    fmt.Fprintf(ctx.Writer, "data: {\"content\": \"%s\"}\n\n", escaped)
    ctx.Writer.Flush()  // 立即推送到客户端
    return nil
}
```

**Step 6：保存 AI 响应**
```go
aiMessage := &models.Message{
    SendId:     "A" + 随机字符串,  // AI 作为发送者
    ReceiveId:  userId,            // 用户作为接收者
    Content:    finalContent,
}

if err := s.messageDAO.CreateMessage(aiMessage); err != nil {
    // DB 写入失败，降级为 Kafka 异步持久化
    aipkg.SendAIMessage(aipkg.AIMessagePayload{...})
}
```

**为什么不把 AI 响应也通过 Kafka 异步化？**

- AI 响应已经通过 SSE 实时推送给用户了，用户不需要等 DB 写入
- 但 DB 写入是"最终一致性"需求，即使失败也可以通过 Kafka 补救
- 如果同步等待 Kafka，反而增加复杂度

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
| last_message | 最后一条消息内容 / 模型类型 |
| last_message_at | 最后消息时间 |

**message 表**：
| 字段 | 说明 |
|------|------|
| uuid | 消息 ID（M 前缀） |
| session_id | 所属会话 ID |
| send_id | 发送者（用户 U 开头 / AI A 开头） |
| receive_id | 接收者（用户 U 开头 / AI A 开头） |
| content | 消息内容 |
| type | 0 = 文本 |

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

**Writer/Reader 初始化**（`pkg/kafka/kafka.go`）：
```go
func (k *KafkaService) Init() {
    // 人聊
    k.ChatWriter = &kafka.Writer{...}
    k.ChatReader = kafka.NewReader{...}
    
    // AI 聊
    k.AIChatWriter = &kafka.Writer{...}
    k.AIChatReader = kafka.NewReader{...}
}
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
    
    userId := ctx.GetString("uuid")  // 从 JWT 中间件获取
    result, _ := c.aiChatService.CreateSession(userId, req)
    
    resp.Success(ctx, "创建AI会话成功", result)
}
```

和普通 Web 接口完全一样。

#### SSE 流式接口

```go
func (c *AIChatController) SendMessage(ctx *gin.Context) {
    // 1. 设置 SSE 响应头
    ctx.Header("Content-Type", "text/event-stream")
    ctx.Header("Cache-Control", "no-cache")
    ctx.Header("Connection", "keep-alive")
    
    // 2. 定义推送回调
    onChunk := func(chunk string) error {
        fmt.Fprintf(ctx.Writer, "data: {\"content\": \"%s\"}\n\n", escaped)
        ctx.Writer.Flush()  // 关键：立即推送
        return nil
    }
    
    // 3. 调用 Service（阻塞直到流结束）
    c.aiChatService.SendMessageStream(ctx.Request.Context(), userId, req, onChunk, onComplete)
}
```

**与普通接口的区别**：
- 不返回 JSON，而是持续写入 `ctx.Writer`
- 每次写入后必须 `Flush()`，否则数据会缓存在缓冲区
- 连接保持打开，直到流结束或客户端断开

---

## 四、完整请求流程

以用户发送一条消息为例：

```
1. 前端发起 GET 请求
   GET /user/aichat/sendMessage?session_id=Sxxx&content=你好&model_type=deepseek
   （带 JWT token，经过 Auth 中间件校验）

2. Controller 层
   - 解析 query 参数
   - 设置 SSE 响应头
   - 定义 onChunk 回调（写入 HTTP 响应流）
   - 调用 Service.SendMessageStream()

3. Service 层
   a. 校验会话存在（sessionDAO.GetAISessionByUuid）
   b. 获取用户信息（userInfoDAO.FindUserByUuid）
   c. 保存用户消息到 DB（messageDAO.CreateMessage）
   d. 更新会话最后消息（sessionDAO.UpdateSessionLastMessage）
   e. 读取历史消息（messageDAO.GetMessagesBySessionId，最近100条）
   f. 构建 eino messages 数组
   g. 获取模型实例（ai.GetModelPool().GetModel("deepseek")）
   h. 调用 model.Stream() 开始流式推理

4. 流式处理循环
   for {
       chunk = stream.Recv()        // 等待 AI 生成下一个片段
       onChunk(chunk.Content)       // 通过 SSE 推送给前端
       fullContent += chunk.Content  // 累积完整内容
   }

5. 流结束后
   a. 保存 AI 响应到 DB
   b. 如果 DB 写入失败 → 发到 Kafka 异步持久化
   c. 更新会话最后消息
   d. 发送 done 信号给前端

6. 前端接收
   data: {"content": "你"}
   data: {"content": "好"}
   data: {"content": "！有什么可以帮"}
   data: {"content": "你的？"}
   data: {"done": true}
```

---

## 五、API 接口文档

### 5.1 创建 AI 会话

```
POST /user/aichat/createSession
Authorization: Bearer <token>

Request:
{
    "model_type": "deepseek"   // deepseek | qwen | glm
}

Response:
{
    "code": 0,
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
    "code": 0,
    "data": {
        "list": [
            {
                "session_id": "S20260403123456",
                "model_type": "deepseek",
                "last_message": "deepseek",
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
    "code": 0,
    "data": {
        "list": [
            {
                "session_id": "S20260403123456",
                "send_id": "U20260403111111",
                "send_name": "张三",
                "content": "你好",
                "type": 0,
                "created_at": "2026-04-03 12:35:00"
            },
            {
                "session_id": "S20260403123456",
                "send_id": "A20260403123457",
                "send_name": "deepseek",
                "content": "你好！有什么可以帮你的？",
                "type": 0,
                "created_at": "2026-04-03 12:35:02"
            }
        ],
        "total": 2
    }
}
```

### 5.4 发送消息（SSE 流式）

```
GET /user/aichat/sendMessage?session_id=Sxxx&content=你好&model_type=deepseek
Authorization: Bearer <token>
Accept: text/event-stream

Response (SSE stream):
data: {"content": "你"}

data: {"content": "好"}

data: {"content": "！"}

data: {"done": true}
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
```

---

## 七、常见问题

### Q1: 为什么 sendMessage 用 GET 而不是 POST？

因为 SSE 通过 `EventSource` 发起，浏览器 `EventSource` API 只支持 GET 请求。如果要用 POST，前端需要手写 `fetch` + `ReadableStream`，复杂度更高。

### Q2: 上下文窗口有限制吗？

有。当前限制读取最近 100 条消息。如果对话很长，可以：
1. 增加条数限制
2. 使用消息摘要/总结（把早期对话压缩成一段摘要）
3. 使用 Token 计数，按 Token 数量截断

### Q3: 如果 AI 模型调用超时怎么办？

`model.Stream()` 会使用 `ctx`（请求上下文），如果客户端断开连接，context 会被取消，流式调用会自动终止。

### Q4: 多个用户同时调用会冲突吗？

不会。模型实例是无状态的（类似 HTTP Client），多个 goroutine 可以安全并发调用。每个请求的会话数据都从 DB 独立读取。

### Q5: 和 WebSocket 人聊的区别？

| 特性 | 人聊（WebSocket） | AI 聊（SSE） |
|------|-------------------|-------------|
| 实时性 | 双向实时 | 服务端单向推送 |
| 消息来源 | 另一个用户 | AI 模型推理 |
| 上下文 | 不需要 | 需要历史对话 |
| 响应时间 | 即时 | 几秒~几十秒 |
| 持久化 | 同步写 DB | 同步 + Kafka 降级 |

---

## 八、文件结构

```
seekF-backend/
├── config/config.toml                    # AI 模型配置 + Kafka topic
├── internal/
│   ├── configs/configs.go                # AIModelConfig 结构体
│   ├── dao/user_dao/
│   │   ├── session_dao.go                # + AI 会话方法（4个）
│   │   └── message_dao.go               # + AI 消息方法（2个）
│   ├── dto/user/
│   │   ├── user_req/
│   │   │   ├── create_ai_session_request.go
│   │   │   ├── get_ai_session_list_request.go
│   │   │   ├── get_ai_message_history_request.go
│   │   │   └── send_ai_message_request.go
│   │   └── user_resp/
│   │       ├── create_ai_session_respond.go
│   │       ├── get_ai_session_list_respond.go
│   │       └── get_ai_message_history_respond.go
│   ├── services/ai_service/
│   │   └── aichat_service.go            # 核心业务逻辑
│   ├── controllers/user/
│   │   └── aichat_controller.go         # HTTP 端点
│   ├── pkg/
│   │   ├── ai/
│   │   │   ├── model_pool.go            # 模型单例池
│   │   │   └── ai_kafka.go              # AI Kafka 消费者
│   │   └── kafka/kafka.go               # + AI Writer/Reader 初始化
│   ├── router/router.go                 # + /user/aichat/* 路由
│   └── main.go                          # 初始化 AI 组件
└── docs/aichat.md                       # 本文档
```
