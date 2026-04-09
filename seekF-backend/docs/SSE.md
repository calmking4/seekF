# SSE 详解 - 新手入门指南

> 面向第一次接触 Server-Sent Events 的开发者，用你熟悉的 Web 概念来类比解释。

---

## 一、什么是 SSE？

### 1.1 类比解释

想象你在餐厅点餐：

| 传统 HTTP 请求 | SSE |
|---------------|-----|
| 你点完菜，厨师全部做好后一次性端上来 | 厨师做好一道菜就立刻端给你 |
| 你需要等待完整响应 | 边做边吃，体验更好 |

**SSE = 服务端主动推送数据给客户端的技术**

### 1.2 与 WebSocket 的对比

| 特性 | WebSocket | SSE |
|------|-----------|-----|
| 通信方向 | 双向（客户端 ↔ 服务端） | 单向（服务端 → 客户端） |
| 协议 | 独立的 `ws://` 协议 | 复用 HTTP 协议 |
| 复杂度 | 高（需要握手、心跳、重连） | 低（浏览器原生支持） |
| 适用场景 | 实时聊天、游戏 | AI 流式响应、股票行情、通知 |

**本项目用 SSE 的原因**：AI 生成内容是单向的（服务端 → 客户端），不需要客户端发送数据给服务端，SSE 更简单。

---

## 二、SSE 的格式

SSE 消息格式非常简单，**每个消息以空行结束**：

```
data: {"content": "第一块内容"}

data: {"content": "第二块内容"}

data: {"content": "第三块内容"}

data: {"done": true}
```

### 格式说明

- `data: ` 开头表示这是数据
- 空行 `\n\n` 表示一条消息结束
- `done: true` 表示服务端发送完毕

### 一个完整的例子

假设 AI 分 3 次返回内容：`"你好"`, `"，我是"`, `"AI助手"`，最后发送完成信号：

```
data: {"content": "你好"}

data: {"content": "，我是"}

data: {"content": "AI助手"}

data: {"done": true}
```

---

## 三、本项目中的 SSE 实现

### 3.1 Controller 层（SSE 格式输出）

**文件**：`seekF-backend/internal/controllers/user/aichat_controller.go`

```go
func (c *AIChatController) SendMessage(ctx *gin.Context) {
    // 1. 设置 SSE 响应头（关键！）
    ctx.Header("Content-Type", "text/event-stream")
    ctx.Header("Cache-Control", "no-cache")
    ctx.Header("Connection", "keep-alive")
    ctx.Status(http.StatusOK)
    ctx.Writer.Flush()

    // 2. 定义 onChunk 回调：每收到一块数据就推送给客户端
    onChunk := func(chunk string) error {
        // 转义特殊字符，避免破坏 JSON 格式
        escaped := strings.ReplaceAll(chunk, "\n", "\\n")
        escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
        
        // 输出 SSE 格式
        _, err := fmt.Fprintf(ctx.Writer, "data: {\"content\": \"%s\"}\n\n", escaped)
        if err != nil {
            return err
        }
        ctx.Writer.Flush()  // 关键：立即推送给客户端
        return nil
    }

    // 3. 定义 onComplete 回调：流式推送完成后执行
    onComplete := func(fullContent string) error {
        _, err := fmt.Fprintf(ctx.Writer, "data: {\"done\": true}\n\n")
        if err != nil {
            return err
        }
        ctx.Writer.Flush()
        return nil
    }

    // 4. 调用 Service，传入回调函数
    err := c.aiChatService.SendMessageStream(ctx.Request.Context(), userId, req, onChunk, onComplete)
}
```

**为什么要设置这些响应头？**

| 响应头 | 作用 |
|--------|------|
| `Content-Type: text/event-stream` | 告诉浏览器这是 SSE 流 |
| `Cache-Control: no-cache` | 禁用缓存，因为数据是流式的 |
| `Connection: keep-alive` | 保持连接不断开 |

### 3.2 Service 层（调用模型 + 执行回调）

**文件**：`seekF-backend/internal/services/ai_service/aichat_service.go`

```go
// SendMessageStream 的签名
func (s *AIChatServiceImpl) SendMessageStream(
    ctx context.Context,
    userId string,
    req userreq.SendAIMessageRequest,
    onChunk func(chunk string) error,        // 回调：每收到一块数据
    onComplete func(fullContent string) error // 回调：流式结束
) error {
    // 1. 调用 AI 模型，获取流式响应
    stream, err := model.Stream(ctx, chatMessages)
    if err != nil {
        return err
    }
    defer stream.Close()

    // 2. 逐块读取模型返回的内容
    var fullContent strings.Builder
    for {
        chunk, err := stream.Recv()
        if errors.Is(err, io.EOF) {
            break  // 流结束
        }
        if err != nil {
            break
        }

        if chunk != nil && len(chunk.Content) > 0 {
            fullContent.WriteString(chunk.Content)
            // 调用 onChunk 回调，推送这一块数据
            if err := onChunk(chunk.Content); err != nil {
                break
            }
        }
    }

    // 3. 流式结束后，调用 onComplete 回调
    if onComplete != nil {
        onComplete(fullContent.String())
    }

    return nil
}
```

---

## 四、为什么用回调函数？

### 4.1 问题

流式响应需要两个时机：
1. **每收到一块数据** → 立即推送给客户端（用户体验好，看到"打字机效果"）
2. **全部数据发送完毕** → 发送结束标志

这两个时机都在 Service 层内部发生，但输出格式（SSE 格式）在 Controller 层定义。

### 4.2 解决方案：回调函数

```
┌─────────────────────────────────────────────────────────┐
│  Controller 层                                          │
│  - 定义 onChunk：输出 SSE 格式                           │
│  - 定义 onComplete：发送完成信号                         │
│         ↓ 传入回调                                       │
│  Service 层                                             │
│  - 调用模型获取流                                         │
│  - 每收到一块数据 → 调用 onChunk                         │
│  - 流结束 → 调用 onComplete                             │
└─────────────────────────────────────────────────────────┘
```

**好处**：
1. **解耦**：Service 只管"何时发送"，Controller 管"怎么发送"
2. **复用**：同一个 Service 可以用于不同输出场景（SSE、WebSocket、文件等）
3. **灵活性**：换个输出格式只需要换回调函数，不需要改 Service

---

## 五、前端如何接收 SSE

### 5.1 使用 EventSource（推荐）

```javascript
const evtSource = new EventSource(
    '/user/aichat/sendMessage?session_id=xxx&content=你好&model_type=deepseek'
);

evtSource.onmessage = (event) => {
    const data = JSON.parse(event.data);
    
    if (data.content) {
        // 追加到页面上（打字机效果）
        aiText += data.content;
        document.getElementById('ai-response').innerText = aiText;
    }
    
    if (data.done) {
        // 流式响应结束
        evtSource.close();
    }
};

evtSource.onerror = (error) => {
    console.error('SSE 连接错误', error);
    evtSource.close();
};
```

### 5.2 使用 fetch + ReadableStream（POST 请求）

浏览器原生 `EventSource` 只支持 GET 请求，如果需要 POST：

```javascript
fetch('/user/aichat/sendMessage', {
    method: 'POST',
    body: JSON.stringify({
        session_id: 'xxx',
        content: '你好',
        model_type: 'deepseek'
    })
}).then(response => {
    const reader = response.body.getReader();
    const decoder = new TextDecoder();

    function read() {
        reader.read().then(({ done, value }) => {
            if (done) return;
            
            const text = decoder.decode(value);
            // 解析 SSE 格式...
            read();
        });
    }
    read();
});
```

---

## 六、完整的请求-响应流程

```
前端                                         后端
  │                                             │
  │  GET /user/aichat/sendMessage?session_id=xxx&content=你好
  │  ──────────────────────────────────────────►
  │                                             │
  │  设置 SSE 响应头                             │
  │  Content-Type: text/event-stream           │
  │  ◄──────────────────────────────────────────
  │                                             │
  │                                    保存用户消息到 DB
  │                                    调用 AI 模型
  │                                             │
  │  data: {"content": "你"}                    │
  │  ◄──────────────────────────────────────────
  │                                             │
  │  data: {"content": "好，"}                  │
  │  ◄──────────────────────────────────────────
  │                                             │
  │  data: {"content": "我是 AI 助手"}           │
  │  ◄──────────────────────────────────────────
  │                                             │
  │  data: {"done": true}                       │
  │  ◄──────────────────────────────────────────
  │                                             │
  │  evtSource.close()                          │
  │  (连接关闭)                                  │
```

---

## 七、关键代码位置汇总

| 文件 | 作用 |
|------|------|
| `controllers/user/aichat_controller.go` | 设置 SSE 响应头、定义回调函数 |
| `services/ai_service/aichat_service.go` | 调用模型流式推理、执行回调 |
| `pkg/ai/model_pool.go` | 管理 AI 模型单例 |
| `pkg/kafka/kafka.go` | Kafka 异步持久化 AI 响应 |
