## 1. 先用一句话理解 MCP

MCP（Model Context Protocol）可以理解为：  
**给大模型一套“可调用工具”的标准接口**。  
大模型先判断要不要调用工具（例如天气），如果要，就让后端执行工具，再把工具结果喂回模型，让模型组织成自然语言回复。

在本项目里，MCP 工具包括天气、汇率、网页搜索和帖子搜索。

---

## 2. 本项目 MCP 的核心文件

### 2.1 MCP Server（独立进程，stdio 模式）
- 文件：`cmd/mcp-server/main.go`
- 作用：
  - 创建一个独立的 MCP Server 进程
  - 注册工具：天气、汇率、网页搜索、帖子搜索
  - 通过 stdio（stdin/stdout）与主进程进行 JSON-RPC 通信
  - 使用 `server.NewStdioServer(mcpServer)` 启动，监听 stdin/stdout

你可以把它理解成：**”工具商店后台”**，作为独立子进程运行，负责告诉外界”我有哪些工具、每个工具怎么执行”。

### 2.2 MCP Client（把 MCP 工具转换为 Eino 可用工具）
- 文件：`internal/pkg/ai/mcp/client.go`
- 作用：
  - 由启动入口调用 `InitMCPClient(ctx)`，通过 stdio transport 和 `client.NewClient` 启动 MCP Server 子进程
  - 执行 MCP 初始化握手（`Initialize`）
  - 调用 `eino-ext` 的 `mcpp.GetTools(...)`，把 MCP 工具转为 `[]tool.BaseTool`
  - 在握手、工具加载和声明校验全部成功后，一次性发布工具列表及调用映射，使用读写锁保护缓存

你可以把它理解成：**”翻译层”**，启动 MCP Server 子进程并把 MCP 工具元信息翻译成大模型可读的函数声明（tool schema）。

### 客户端初始化与关闭

主服务启动时显式调用 `InitMCPClient(ctx)`。每次握手和工具加载共用30秒超时，最多尝试3次，重试间隔2秒；失败的客户端会先关闭，再进行下一次尝试。子进程的生命周期上下文与应用保持一致，不随握手结束而取消。

`GetMCPTools(ctx)`和`FilteredMCPToolBinding(ctx, enableWebSearch)`只读取缓存，不创建子进程、不重新加载工具。初始化失败时返回错误，聊天服务按普通聊天流程继续处理。工具列表和调用映射每次返回容器副本，避免请求修改全局缓存。

`CloseMCPClient()`清空缓存并关闭客户端，由stdio transport关闭输入管道、等待子进程退出，必要时终止进程。主服务收到退出信号或入口正常返回时调用该方法。

MCP子进程显式调用`InitConfig()`、`InitLogger()`和`InitMySQL()`，帖子搜索工具使用该进程自己的数据库连接池。stdout专用于JSON-RPC，日志由stderr和日志文件输出；主进程持续读取子进程stderr，避免管道积压阻塞工具服务。

启动命令由`MCP_SERVER_COMMAND`控制，默认`go run ./cmd/mcp-server`；工作目录由`MCP_SERVER_WORKDIR`控制，默认当前目录。两个入口均应在`seekF-backend/`目录运行，以加载相对路径的配置文件。

### 2.3 具体工具实现（天气）
- 文件：`internal/pkg/ai/mcp/tool/weather.go`
- 作用：
  - `GetWeatherTool()`：声明工具名、描述、参数（`location`）
  - `HandleWeatherRequest(...)`：解析参数并执行查询
  - `queryWeather(...)`：调用心知天气 API，拼装文本结果

这部分就是“真实业务逻辑”。

### 2.4 进程内备用 Server（参考）
- 文件：`internal/pkg/ai/mcp/server.go`
- 作用：
  - 进程内 MCP Server 的备用实现，仅用于参考
  - 当前主应用使用 stdio 模式与独立的 MCP Server 进程通信

### 2.5 AI 主流程（MCP 决策 + 工具执行 + 总结回复）
- 文件：`internal/services/user_service/aichat_service.go`
- 核心函数：
  - `SendMessageStream(...)`
  - `runMCPAgentFlow(...)`
  - `streamChatModelToSSE(...)`
  - `persistAndCompleteAIMessage(...)`

---

## 3. 用户发消息后的完整链路（真实代码行为）

下面是你项目当前实现的实际流程：

1. `SendMessageStream(...)` 先做通用逻辑  
   - 校验会话、保存用户消息到 DB、更新会话最后消息  
   - 读取历史消息，构建 Eino `chatMessages`（含 system prompt、历史 user/assistant）
   - 可选拼接知识库检索结果（`UseKnowledge`）

2. 选择模型实例  
   - 通过 `aipkg.GetModelPool().GetModel(req.ModelType)` 获取 `ToolCallingChatModel`

3. 进入 MCP Agent 分支（默认尝试）  
   - 调用 `runMCPAgentFlow(...)`

4. `runMCPAgentFlow(...)` 第 1 次请求（非流式）  
   - 从 `mcppkg.GetMCPTools(ctx)` 获取工具定义
   - `chatModel.WithTools(toolInfos)` 绑定工具
   - 调 `Generate(ctx, chatMessages)` 让模型做“是否调用工具”决策

5. 分支判断  
   - **无 ToolCalls**：返回 `handled=false`，外层回退普通流式（保证非工具问题也流式）  
   - **有 ToolCalls**：后端执行每个工具（`InvokableRun`），并拼 `ToolMessage`

6. 第 2 次请求（流式总结）  
   - 把“原对话 + assistant 工具调用指令 + tool 执行结果”组装成 `msgs2`
   - 调 `streamChatModelToSSE(...)`（无工具绑定）进行流式总结回复

7. 持久化与收尾  
   - `persistAndCompleteAIMessage(...)`：兜底空回复、发送 Kafka 异步持久化、更新会话 last_message、回调 `onComplete`

---

## 4. 一共调用几次大模型？

### 场景 A：模型判断不需要工具
- 第 1 次：`Generate`（带 tools 做决策）  
- 然后回退普通流式：`Stream`  
- **总计 2 次**

### 场景 B：模型判断需要工具（例如天气）
- 第 1 次：`Generate`（带 tools 做决策，返回 ToolCalls）  
- 工具执行：本地/MCP 调用（不算 LLM 请求）  
- 第 2 次：`Stream`（基于工具结果总结）  
- **总计 2 次**

所以你现在的实现是：**默认按两段式 Agent 走**，区别只在于第二段的输入上下文是否包含工具结果。

---

## 5. 为什么非天气问题也能流式？

之前常见坑是：  
“无工具时直接把第 1 次 `Generate` 的整段文本返回”，这会看起来不是流式。

当前代码已经修正为：
- 无工具时 `runMCPAgentFlow` 返回 `handled=false`
- 外层继续走 `streamChatModelToSSE`，所以仍是流式输出

---

## 6. 配置项与依赖

### 6.1 模型配置
- 文件：`config/config.toml` 下 `[aiModelConfig]`
- 支持：`deepseek / qwen / glm / glm-4v`
- 读取逻辑：`internal/pkg/ai/model_pool.go`

### 6.2 天气 API 配置
- 文件：`config/config.toml` 下 `[seniverseConfig] apiKey`
- 结构定义：`internal/configs/configs.go` 的 `SeniverseConfig`
- 也支持环境变量覆盖：`SENIVERSE_API_KEY`

### 6.3 关键依赖（go.mod）
- `github.com/mark3labs/mcp-go`（MCP server/client 协议实现）
- `github.com/cloudwego/eino`（模型与消息抽象）
- `github.com/cloudwego/eino-ext/components/tool/mcp`（MCP 工具桥接到 Eino）

---

## 7. 工具调用的组织方式

- 工具接入清晰：Server（注册）+ Client（桥接）+ Service（编排）
- 对前端体验友好：非工具场景也保持流式
- 有降级能力：MCP 工具不可用时可回退普通聊天

---

## 8. 如何在本项目新增一个 MCP 工具（最小步骤）

1. 在 `internal/pkg/ai/mcp/tool/` 新建工具文件（例如 `calendar.go`）  
2. 参考 `WeatherTool` 实现：
   - 工具声明函数（名称、描述、参数）
   - 处理函数（参数解析 + 业务调用 + 文本结果）
3. 在 `cmd/mcp-server/main.go` 的 `main()` 里 `AddTool(...)` 注册（独立 MCP Server 进程）  
4. 不需要改 `runMCPAgentFlow` 主流程（它会动态读取 MCP 工具列表）  
5. 确保工具描述写清”何时使用”，这样模型更容易正确触发工具

---

## 9. 给 AI 应用初学者的总结

你可以把这个项目 MCP 流程记成三步：

1. **先问模型要不要查工具**（第 1 次 LLM）  
2. **后端真去查工具**（MCP，不是 LLM）  
3. **再让模型用查到的数据回答人话**（第 2 次 LLM，流式输出）

这就是一个标准的“LLM + Tool Calling”工程化落地方式。  
你现在这套代码已经具备可扩展基础，后面只需要不断加工具和提示词策略，就能从“天气助手”升级为“多工具 AI 助手”。
