# seekF

一个集成了 AI 能力的全栈即时通讯（IM）应用，采用 Go 后端 + Nuxt 4 前端的 monorepo 结构。

## 核心功能

- **即时通讯** — 单聊、群聊、音视频通话（WebSocket + WebRTC）
- **AI 智能对话** — 支持 DeepSeek / Qwen / GLM 多模型切换，SSE 流式响应
- **AI 联网搜索** — 对话中可实时搜索互联网并展示来源
- **社区帖子 AI 问答** — AI 回答时可关联搜索社区帖子并展示
- **知识库 RAG** — 上传文档构建私有知识库，基于向量检索增强回答
- **MCP 工具系统** — 天气查询、网页搜索等外部工具集成
- **文本转语音（TTS）** — AI 回答可朗读播放
- **发现页社交动态** — 帖子发布、评论、点赞、收藏，评论区 @AI 自动回复

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go / Gin / GORM / Redis / Kafka / WebSocket / Qdrant |
| 前端 | Nuxt 4 / Vue 3 / Element Plus / Tailwind CSS |
| AI | Eino 框架 / OpenAI 兼容接口 / SSE 流式响应 / MCP 工具协议 |
| 基础设施 | 阿里云 OSS / 阿里云短信 / Kafka 异步消息 / Qdrant 向量数据库 |

## 项目结构

```
seekF/
├── seekF-backend/          # Go 后端服务
│   ├── main.go             # 组合根，手动注入依赖
│   ├── config/             # TOML 配置 + .env 敏感信息
│   ├── router/             # Gin 路由定义
│   └── internal/
│       ├── controllers/    # HTTP 处理器（user / admin）
│       ├── services/       # 业务逻辑层
│       ├── dao/            # 数据访问层（GORM）
│       ├── dto/            # 请求/响应 DTO
│       ├── models/         # GORM 数据模型
│       ├── middlewares/    # 中间件（认证、CORS 等）
│       └── pkg/            # 公共包
│           ├── ai/         # AI 子系统（模型池、MCP、RAG、TTS）
│           ├── websocket/  # WebSocket 服务器
│           ├── kafka/      # Kafka 生产/消费
│           ├── db/         # MySQL 连接
│           ├── redis/      # Redis 连接
│           └── ...
├── seekF-user/             # Nuxt 4 前端
│   ├── nuxt.config.ts      # 前端配置
│   └── app/
│       ├── pages/          # 文件路由页面
│       ├── composables/    # 组合式函数
│       └── ...
├── seekF-admin/            # 管理后台（开发中）
└── LICENSE                 # GPL-3.0
```

IM部分参考https://github.com/youngyangyang04/KamaChat
