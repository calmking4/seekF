#  seekF 是一个集成了 AI 能力的全栈即时通讯（IM）应用，采用 Go 后端 + Nuxt 4 前端的 monorepo 结构。

  核心功能：
  - 单聊、群聊、音视频通话（WebSocket + WebRTC）
  - AI 智能对话，支持 DeepSeek / Qwen / GLM 多模型切换
  - AI 联网搜索、知识库 RAG、文本转语音（TTS）
  - MCP 工具系统（天气、汇率、网页搜索）
  - 发现页社交动态（帖子、评论、收藏）

  技术栈：
  - 后端：Go / Gin / GORM / Redis / Kafka / WebSocket / Qdrant
  - 前端：Nuxt 4 / Vue 3 / Element Plus / Tailwind CSS
  - AI：Eino 框架 / OpenAI 兼容接口 / SSE 流式响应
  - 基础设施：阿里云 OSS / 阿里云短信 / Kafka 异步消息

