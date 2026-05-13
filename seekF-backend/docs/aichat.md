# AI Chat 实现详解

> 面向有 Web 开发经验但第一次接触 AI 应用的开发者。所有 AI 相关概念都会用 Web 开发中的对应概念做类比。

---

## 一、整体架构

```
┌─────────────────────────────────────────────────────────────┐
│  模型层（pkg/ai/model_pool.go）—— 全局单例池                  │
│  deepseek │ qwen │ glm │ glm-4v                            │
│  类比：数据库连接池，只负责"推理"，不存任何用户数据              │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│  MCP 工具层（pkg/ai/mcp/）—— AI 可调用的外部能力              │
│  天气查询 │ 汇率查询 │ 联网搜索（Tavily）                     │
│  两阶段 Agent：Generate(工具决策) → 执行工具 → Stream(总结)   │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│  会话层（services/user_service/aichat_service.go）            │
│  每次请求新建，不驻留内存                                      │
│  流程：读DB历史 → 构建messages → MCP/调模型 → SSE推送 → 存DB  │
│  类比：普通 HTTP 请求处理，每次请求都是独立的                    │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│  存储层（MySQL + Redis + Kafka + Qdrant）                     │
│  所有聊天记录存数据库，绝对不放在结构体里                        │
│  Kafka 异步持久化 AI 响应，避免阻塞 SSE 流                     │
│  Qdrant 向量数据库，支持知识库 RAG 检索                        │
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

ChatModel 就是和 AI 模型（DeepSeek/Qwen/GLM）对话的接口。本项目使用 Eino 框架的 `model.ToolCallingChatModel` 接口，它有三个关键方法：

- **`Generate()`**：一次性返回完整响应（用于 MCP 工具决策阶段）
- **`Stream()`**：逐块返回响应（用于普通对话和 MCP 最终总结，实现"打字机效果"）
- **`WithTools(toolInfos)`**：绑定工具列表，返回带工具调用能力的模型实例

本项目主要用 `Stream()`，因为 AI 生成内容需要几秒，流式推送可以让用户看到逐字输出效果。

### 2.2 什么是 Message（消息角色）？

AI 对话中的每条消息都有一个"角色"，这是 AI 理解对话的关键：

| 角色 | 含义 | 类比 |
|------|------|------|
| `system` | 系统指令，告诉 AI 它的行为规则 | 服务器的全局配置 |
| `user` | 用户说的是什么 | 客户端请求 |
| `assistant` | AI 的回复（可携带 tool_calls） | 服务器响应 |
| `tool` | 工具执行结果 | 第三方 API 回调 |

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

**本项目的 SSE 协议**：

| 消息类型 | 格式 | 说明 |
|---------|------|------|
| 内容块 | `data: {"content": "..."}` | AI 生成的文本片段，可多次发送 |
| 搜索来源 | `data: {"sources": [...]}` | 联网搜索结果，发送一次（在内容之前） |
| 完成信号 | `data: {"done": true}` | 流式响应结束 |
| 错误信号 | `data: {"error": "..."}` | 发生错误 |

**为什么转义？**

如果 AI 返回的内容包含换行符或引号，需要转义：
- `\n` → `\\n`
- `"` → `\"`

例如 AI 返回 `"你好\nWorld"`，转义后变成：
```
data: {"content": "你好\\nWorld"}
```

**前端接收方式**：

本项目使用 `fetch` + `ReadableStream` 手动解析 SSE（而非浏览器原生 `EventSource`），因为需要支持 POST 请求 + FormData 上传图片：

```javascript
fetch(url, { method: 'POST', body: formData, credentials: 'include' })
    .then(response => {
        const reader = response.body.getReader()
        const decoder = new TextDecoder()
        const readStream = () => {
            reader.read().then(({ done, value }) => {
                if (done) { onComplete?.(); return }
                // 解析 data: {...} 行
                for (const line of lines) {
                    if (line.startsWith('data: ')) {
                        const data = JSON.parse(line.slice(6))
                        if (data.sources) onSources?.(data.sources)
                        if (data.content) onChunk?.(data.content)
                        if (data.done) onComplete?.()
                        if (data.error) onError?.(data.error)
                    }
                }
                readStream()
            })
        }
        readStream()
    })
```

### 2.4 什么是 MCP（Model Context Protocol）？

**类比**：MCP 就像给 AI 装上了"手脚"，让它能调用外部工具。

普通 AI 只能根据训练数据回答问题。有了 MCP，AI 可以：
- 查询实时天气（`get_weather`）
- 查询汇率（`get_exchange_rate`）
- 联网搜索（`web_search`）

**两阶段 Agent 流程**：

```
用户提问 → 第1次调用（Generate，非流式）
                ↓
         AI 决定是否需要工具
                ↓
     ┌─── 不需要 → 回退普通 Stream（流式回答）
     │
     └─── 需要工具 → 执行工具 → 第2次调用（Stream，流式总结）
                                    ↓
                             把工具结果 + 用户问题一起给 AI
                                    ↓
                             AI 基于真实数据流式回答
```

### 2.5 什么是 RAG（检索增强生成）？

**类比**：RAG 就像给 AI 配了一个"参考书架"。

用户上传文档到知识库后，系统会：
1. 把文档切分成小块（500 字符/块）
2. 用 GLM Embedding 模型把每块转为向量
3. 存入 Qdrant 向量数据库

当用户提问时：
1. 把问题向量化
2. 在 Qdrant 中搜索最相似的 3 个知识块
3. 把知识块注入到 system prompt 中
4. AI 基于知识库内容回答

### 2.6 为什么要用 Kafka 异步持久化？

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

---

## 三、分层详解

### 3.1 模型层（`pkg/ai/model_pool.go`）

**职责**：管理 AI 模型实例，全局只初始化一次。

```go
type ModelPool struct {
    DeepSeek model.ToolCallingChatModel
    Qwen     model.ToolCallingChatModel
    GLM      model.ToolCallingChatModel
    GLM4V    model.ToolCallingChatModel
}
```

**关键设计**：

1. **`sync.Once` 保证单例**：和数据库连接池一样，整个应用生命周期只初始化一次

2. **配置优先级**：配置文件 → 环境变量

3. **统一接口**：所有模型都通过 `openai.NewChatModel` 创建（Eino 框架的 OpenAI 兼容适配器），因为它们都兼容 OpenAI 协议
   - DeepSeek: `https://api.deepseek.com/v1`
   - Qwen: `https://dashscope.aliyuncs.com/compatible-mode/v1`
   - GLM/GLM-4V: `https://open.bigmodel.cn/api/paas/v4/`

4. **按需获取**：`GetModel(modelType)` 根据类型返回对应模型，默认返回 DeepSeek

**多模态模型支持**：

GLM-4V 是多模态模型，支持图片输入。在 `SendMessageStream` 中构建多模态消息：
```go
multiMsg := &schema.Message{
    Role: schema.User,
    UserInputMultiContent: []schema.MessageInputPart{
        {Type: schema.ChatMessagePartTypeText, Text: content},
        {Type: schema.ChatMessagePartTypeImageURL, Image: &schema.MessageInputImage{
            MessagePartCommon: schema.MessagePartCommon{URL: &imageURL},
        }},
    },
}
```

---

### 3.2 MCP 工具层（`pkg/ai/mcp/`）

**职责**：为 AI 提供可调用的外部工具。

#### 3.2.1 工具注册（`server.go`）

使用 `mcp-go` 库创建 MCP Server，注册所有工具：

```go
mcpServer = server.NewMCPServer("seekF-weather", "1.0.0", ...)
mcpServer.AddTool(weatherTool.GetWeatherTool(), weatherTool.HandleWeatherRequest)
mcpServer.AddTool(exchangeRateTool.GetExchangeRateTool(), exchangeRateTool.HandleExchangeRateRequest)
mcpServer.AddTool(webSearchTool.GetWebSearchTool(), webSearchTool.HandleWebSearchRequest)
```

#### 3.2.2 进程内连接（`client.go`）

使用 `client.NewInProcessClient` 实现进程内 MCP 连接，不需要网络通信。工具列表通过 `mcpp.GetTools` 转换为 Eino 的 `tool.BaseTool` 接口。

#### 3.2.3 工具实现

每个工具都实现两个方法：
- `GetXxxTool() mcp.Tool` — 定义工具名、描述、参数
- `HandleXxxRequest(ctx, request) (*mcp.CallToolResult, error)` — 处理请求

**天气工具**（`tool/weather.go`）：
- 工具名：`get_weather`
- 参数：`location`（城市名）
- 调用心知天气 API

**汇率工具**（`tool/exchange_rate.go`）：
- 工具名：`get_exchange_rate`
- 参数：`base_currency`、`target_currency`
- 调用 ExchangeRate-API

**联网搜索工具**（`tool/web_search.go`）：
- 工具名：`web_search`
- 参数：`query`（搜索关键词）
- 调用 Tavily Search API
- 返回文本摘要 + `__SOURCES_JSON__` 标记的结构化来源数据

#### 3.2.4 Agent 流程（`aichat_service.go` 的 `runMCPAgentFlow`）

```go
func runMCPAgentFlow(ctx, chatModel, chatMessages, onChunk, enableWebSearch) (
    finalContent string, handled bool, sources []tool.SearchSource, err error) {

    // 1. 获取 MCP 工具列表
    tools, _ := mcppkg.GetMCPTools(ctx)

    // 2. 根据开关过滤工具（enableWebSearch=false 时移除 web_search）
    for _, t := range tools {
        info, _ := t.Info(ctx)
        if !enableWebSearch && info.Name == "web_search" { continue }
        toolInfos = append(toolInfos, info)
    }

    // 3. 第1次调用：非流式 Generate（工具决策）
    modelWithTools, _ := chatModel.WithTools(toolInfos)
    first, _ := modelWithTools.Generate(ctx, chatMessages)

    // 4. 如果 AI 不需要工具，回退到普通流式
    if len(first.ToolCalls) == 0 {
        return "", false, nil, nil
    }

    // 5. 执行工具
    for _, tc := range first.ToolCalls {
        runOut, _ := inv.InvokableRun(ctx, tc.Function.Arguments)
        out := toolCallResultToText(runOut)

        // 从 web_search 结果提取来源数据
        if name == "web_search" {
            sources = extractSources(out)
            out = stripSourcesSentinel(out)
        }
        msgs2 = append(msgs2, schema.ToolMessage(out, tc.ID))
    }

    // 6. 第2次调用：流式 Stream（最终总结）
    finalContent, _ = streamChatModelToSSE(ctx, chatModel, msgs2, onChunk)
    return finalContent, true, sources, nil
}
```

---

### 3.3 会话层（`services/user_service/aichat_service.go`）

**职责**：处理 AI 聊天的核心业务逻辑。

#### 3.3.1 SendMessageStream —— 核心流式对话

这是整个项目最复杂的方法，流程如下：

**Step 1：校验会话** → 确认 `receive_id` 以 `A` 开头

**Step 2：获取用户信息** → 用于保存消息时带上昵称和头像

**Step 3：保存用户消息到 DB** → 同步写入 `message` 表

**Step 4：更新会话元数据** → 更新 `last_message`，如果是首条消息则更新 `first_message`

**Step 5：构建上下文**
```go
// 从 DB 读取最近 100 条消息
messages, _ := s.messageDAO.GetMessagesBySessionId(req.SessionId, 100, 0)

// 系统提示（注入当前时间）
currentTime := time.Now().Format("2006-01-02 15:04:05")
systemPrompt := "你是一个专业的AI助手，当前使用的模型是" + req.ModelType + "。当前时间是" + currentTime + "。"

// 如果启用知识库，注入 RAG 检索结果
if req.UseKnowledge {
    knowledgeResults, _ := rag.Search(ctx, "knowledge_"+userId, content, 3)
    systemPrompt = "知识库内容:\n" + 格式化(knowledgeResults) + "\n\n" + systemPrompt
}

// 如果启用联网搜索，在系统提示中引导 AI 使用 web_search 工具
if req.UseWebSearch {
    systemPrompt += "你拥有联网搜索能力（web_search 工具）。当用户询问新闻、时事、最新数据时必须优先使用。"
}

// 转换历史消息为 Eino 格式
chatMessages = [SystemMessage(systemPrompt), ...历史消息]
```

**Step 6：MCP Agent 流程** → `runMCPAgentFlow()` 尝试工具调用

**Step 7：普通流式推理** → 如果 MCP 未处理，调用 `streamChatModelToSSE()`

**Step 8：推送搜索来源** → 如果有 `sources`，通过 `onSources` 回调发送 SSE `sources` 事件

**Step 9：持久化 AI 响应** → 通过 Kafka 异步写 DB，触发 `onComplete` 回调

#### 3.3.2 其他方法

- **CreateSession**：创建会话，生成 AI receiveId（A 前缀）
- **GetSessionList**：根据 userId 查 session 表，筛选 `receive_id LIKE 'A%'`
- **GetMessageHistory**：按 `created_at` 降序分页返回（前端再反转为正序）
- **DeleteSession**：级联删除消息和会话

---

### 3.4 Controller 层（`controllers/user/aichat_controller.go`）

#### 普通接口（JSON 响应）

CreateSession / GetSessionList / GetMessageHistory / DeleteSession —— 和普通 Web 接口完全一样。

#### SSE 流式接口（SendMessage）

```go
func (c *AIChatController) SendMessage(ctx *gin.Context) {
    // 1. 解析 form-data 参数（支持图片上传）
    var req userreq.SendAIMessageRequest
    ctx.ShouldBind(&req)

    // 2. 如果有图片文件，上传到 OSS
    if file, err := ctx.FormFile("image"); err == nil {
        result, _ := c.fileService.UploadFile(ctx, file, oss.MessageImage)
        req.ImageURL = result.URL
    }

    // 3. 设置 SSE 响应头
    ctx.Header("Content-Type", "text/event-stream")
    ctx.Header("Cache-Control", "no-cache")
    ctx.Header("Connection", "keep-alive")

    // 4. 定义三个回调
    onChunk := func(chunk string) error {
        // 转义 → 写入 data: {"content": "..."} → Flush
    }
    onSources := func(sources []tool.SearchSource) error {
        // 序列化 → 写入 data: {"sources": [...]} → Flush
    }
    onComplete := func(fullContent string) error {
        // 写入 data: {"done": true} → Flush
    }

    // 5. 调用 Service（阻塞直到流结束）
    c.aiChatService.SendMessageStream(ctx, userId, req, onChunk, onSources, onComplete)
}
```

---

### 3.5 存储层

#### 数据库表复用

**不新建表**，复用现有的 `session` 和 `message` 表，通过前缀区分类型：

| 前缀 | 含义 | 表字段 |
|------|------|--------|
| `S` | Session ID | `session.uuid` |
| `U` | 用户 ID | `user.uuid` / `message.send_id` |
| `G` | 群组 ID | `session.receive_id` |
| `A` | AI ID | `session.receive_id` / `message.send_id` |
| `M` | 消息 ID | `message.uuid` |

#### Kafka

```toml
[kafkaConfig]
aiChatTopic = "ai_chat_message"    # AI 消息持久化
aiCommentTopic = "ai_comment"      # AI 评论回复
```

#### Qdrant 向量数据库

- 集合维度：2048（GLM Embedding 模型输出维度）
- 相似度：余弦相似度（Cosine）
- 每个向量携带 `text`（原文）和 `doc_uuid`（文档标识）
- 用户知识库集合名：`knowledge_{userId}`

---

## 四、完整请求流程

### 4.1 普通 AI 对话

```
1. 前端发起 POST 请求（fetch + FormData）
   POST /user/aichat/sendMessage
   Content-Type: multipart/form-data
   session_id=Sxxx&content=你好&model_type=deepseek

2. Controller 层
   - 解析 form-data 参数
   - 设置 SSE 响应头
   - 定义 onChunk / onSources / onComplete 回调
   - 调用 Service.SendMessageStream()

3. Service 层
   a. 校验会话 → 保存用户消息 → 更新会话
   b. 读取历史消息（最近 100 条）
   c. 构建系统提示（注入当前时间）
   d. 转换为 Eino messages 格式

4. MCP Agent 流程
   a. 获取 MCP 工具列表
   b. 第1次调用 Generate（非流式，AI 决定是否用工具）
   c. 如果不需要工具 → 回退到普通 Stream
   d. 如果需要工具 → 执行工具 → 第2次调用 Stream（流式总结）

5. 流式推送
   data: {"content": "你"}
   data: {"content": "好"}
   data: {"content": "！"}
   data: {"done": true}

6. 异步持久化
   - 发送 AI 响应到 Kafka
   - 更新会话最后消息
```

### 4.2 联网搜索对话

```
1. 前端开启"联网搜索"开关，发送 use_web_search=true

2. Service 层构建系统提示时额外注入：
   "你拥有联网搜索能力（web_search 工具）..."

3. MCP Agent 流程
   a. AI 决定调用 web_search 工具
   b. 执行 Tavily Search API → 返回搜索结果
   c. 提取结构化来源数据（sources）
   d. 第2次调用 Stream，AI 基于搜索结果流式回答

4. SSE 推送
   data: {"sources": [{"title":"...","url":"...","snippet":"..."}]}
   data: {"content": "根据最新搜索结果..."}
   data: {"content": "..."}
   data: {"done": true}

5. 前端渲染
   - 消息气泡下方显示"已搜索 N 个来源"折叠卡片
   - 点击展开可查看来源标题和链接
```

---

## 五、API 接口文档

### 5.1 创建 AI 会话

```
POST /user/aichat/createSession
Authorization: Bearer <token>
Content-Type: application/json

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
```

### 5.4 发送消息（SSE 流式）

```
POST /user/aichat/sendMessage
Authorization: Bearer <token>
Content-Type: multipart/form-data

Form Fields:
  session_id (required)  - 会话 ID
  content               - 消息文本
  model_type (required)  - 模型类型：deepseek | qwen | glm | glm-4v
  image                  - 图片文件（可选，仅 glm-4v）
  use_knowledge          - 是否使用知识库（可选）
  use_web_search         - 是否启用联网搜索（可选）

Response (SSE stream):
data: {"sources": [{"title":"...","url":"...","snippet":"..."}]}

data: {"content": "你"}

data: {"content": "好"}

data: {"content": "！"}

data: {"done": true}

// 或错误情况：
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
```

---

## 六、配置说明

### 6.1 config.toml

```toml
[kafkaConfig]
aiChatTopic = "ai_chat_message"
aiCommentTopic = "ai_comment"

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

# 智谱 GLM-4V（多模态，复用 glmApiKey/glmBaseUrl）
glm4vModel = "glm-4v"

# 智谱 Embedding（RAG 向量化）
glmEmbeddingModel = "embedding-3"

[seniverseConfig]
apiKey = "xxx"       # 心知天气 API Key

[exchangeRateConfig]
apiKey = "xxx"       # 汇率 API Key

[tavilyConfig]
apiKey = "tvly-xxx"  # Tavily 搜索 API Key（联网搜索）

[qdrantConfig]
host = "localhost"
port = 6334
```

### 6.2 环境变量（备选）

如果配置文件中 API Key 为空，会自动从环境变量读取：

```bash
export DEEPSEEK_API_KEY="sk-xxx"
export QWEN_API_KEY="sk-xxx"
export GLM_API_KEY="xxx"
export SENIVERSE_API_KEY="xxx"
export EXCHANGE_RATE_API_KEY="xxx"
export TAVILY_API_KEY="tvly-xxx"
```

---

## 七、前端实现

### 7.1 核心 Composable（`composables/useAIChat.js`）

```javascript
const sendMessage = (
    sessionId, content, modelType, imageFile,
    useKnowledge, useWebSearch,  // 功能开关
    onChunk, onSources, onComplete, onError  // 回调
) => {
    const formData = new FormData()
    formData.append('session_id', sessionId)
    formData.append('content', content)
    formData.append('model_type', modelType)
    if (useKnowledge) formData.append('use_knowledge', 'true')
    if (useWebSearch) formData.append('use_web_search', 'true')
    if (imageFile) formData.append('image', imageFile)

    fetch(url, { method: 'POST', body: formData, credentials: 'include' })
        .then(response => {
            const reader = response.body.getReader()
            // ... 手动解析 SSE
        })
    return { close: () => controller.abort() }
}
```

### 7.2 AI 对话页面（`pages/aichat/index.vue`）

- 左侧：AI 会话列表（新建 / 选择 / 删除）
- 右侧：聊天窗口
  - 头部：模型选择器 + 知识库开关 + 联网搜索开关 + 思考中指示器
  - 消息列表：用户消息（绿色气泡）+ AI 消息（白色气泡 + 搜索来源卡片）
  - 输入框：文本输入 + 图片上传（GLM-4V）

### 7.3 搜索来源组件（`components/SearchSources.vue`）

可折叠卡片，展示联网搜索的参考来源：
- 标题栏："已搜索 N 个来源"，点击展开/收起
- 每个来源：序号 + 标题 + URL，点击在新标签页打开

---

## 八、常见问题

### Q1: 为什么用 fetch + ReadableStream 而不是 EventSource？

因为 `EventSource` 只支持 GET 请求，无法上传图片文件。本项目用 `fetch` + POST + FormData 支持图片上传，同时手动解析 SSE 流。

### Q2: 为什么需要转义？

AI 返回的内容可能包含换行符和引号：
- 换行符 `\n` 会破坏 SSE 消息边界
- 引号 `"` 会破坏 JSON 格式

### Q3: 上下文窗口有限制吗？

有。当前限制读取最近 100 条消息。如果对话很长，可以：
1. 增加条数限制
2. 使用消息摘要/总结
3. 使用 Token 计数，按 Token 数量截断

### Q4: MCP 工具调用失败会影响对话吗？

不会。整个 MCP Agent 流程有完善的降级机制：
- MCP 工具获取失败 → 静默降级为普通流式对话
- 工具执行失败 → 返回错误信息给 AI，AI 继续回答
- Tavily API Key 未配置 → web_search 工具不可用，不影响其他工具

### Q5: 联网搜索的来源数据会持久化吗？

v1 不持久化。来源数据仅在当前会话的 SSE 流中推送一次，刷新页面后丢失。后续版本可考虑存入 message 表的新字段。

### Q6: 和 WebSocket 人聊的区别？

| 特性 | 人聊（WebSocket） | AI 聊（SSE） |
|------|-------------------|-------------|
| 实时性 | 双向实时 | 服务端单向推送 |
| 消息来源 | 另一个用户 | AI 模型推理 |
| 上下文 | 不需要 | 需要历史对话 |
| 响应时间 | 即时 | 几秒~几十秒 |
| 持久化 | 同步写 DB | 同步 + Kafka 降级 |
| 工具调用 | 不支持 | 支持 MCP 工具 |
| 联网搜索 | 不支持 | 支持（Tavily） |

---

## 九、文件结构

```
seekF-backend/
├── config/config.toml                        # 配置文件
├── internal/
│   ├── configs/configs.go                    # 配置结构体（含 TavilyConfig）
│   ├── dao/user_dao/
│   │   ├── session_dao.go                    # AI 会话 DAO
│   │   └── message_dao.go                    # AI 消息 DAO
│   ├── dto/user/
│   │   ├── user_req/
│   │   │   ├── create_ai_session_request.go
│   │   │   ├── get_ai_session_list_request.go
│   │   │   ├── get_ai_message_history_request.go
│   │   │   ├── send_ai_message_request.go    # 含 use_knowledge / use_web_search
│   │   │   └── delete_ai_session_request.go
│   │   └── user_resp/
│   │       ├── create_ai_session_respond.go
│   │       ├── get_ai_session_list_respond.go
│   │       └── get_ai_message_history_respond.go
│   ├── services/user_service/
│   │   ├── aichat_service.go                 # 核心业务（MCP Agent + RAG + 联网搜索）
│   │   └── knowledge_service.go              # 知识库管理
│   ├── controllers/user/
│   │   ├── aichat_controller.go              # HTTP 端点 + SSE
│   │   └── knowledge_controller.go           # 知识库端点
│   ├── pkg/ai/
│   │   ├── model_pool.go                     # 模型单例池
│   │   ├── ai_kafka.go                       # AI 消息 Kafka 消费者
│   │   ├── ai_comment_kafka.go               # AI 评论回复 Kafka
│   │   ├── mcp/
│   │   │   ├── server.go                     # MCP Server（注册 3 个工具）
│   │   │   ├── client.go                     # MCP Client（进程内连接）
│   │   │   └── tool/
│   │   │       ├── weather.go                # 天气查询工具
│   │   │       ├── exchange_rate.go          # 汇率查询工具
│   │   │       └── web_search.go             # 联网搜索工具（Tavily）
│   │   └── rag/
│   │       ├── init.go                       # RAG 单例初始化
│   │       ├── embedding.go                  # GLM Embedding 向量化
│   │       └── splitter.go                   # 文本分块器
│   ├── pkg/db/qdrant_util.go                 # Qdrant 向量数据库操作
│   ├── pkg/kafka/kafka.go                    # Kafka 初始化
│   ├── router/router.go                      # 路由注册
│   └── main.go                               # 初始化所有组件
└── docs/aichat.md                            # 本文档
```
