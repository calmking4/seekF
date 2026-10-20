# seekF

**基于 Go、Nuxt 4 feed流内容社区。**

seekF 围绕实时交流、智能问答和内容创作构建完整业务链路：支持用户私聊、多模型 AI 对话、私有知识库、图文与视频社区，以及创作者付费权益、收益分配、自动提现和管理控制台。

项目采用 monorepo 结构，包含 Go 后端、Nuxt 4 用户端和 React 管理控制台。

## 核心功能

| 模块 | 能力 |
| --- | --- |
| 账号与认证 | 密码登录、手机与邮箱验证码、GitHub / Gitee OAuth 登录、个人资料维护、JWT 或 Redis 会话认证 |
| 用户关系 | 关注与粉丝列表弹窗、用户主页、拉黑与解除拉黑、个人黑名单管理；任一方拉黑后禁止私信 |
| 即时通讯 | 从用户主页发起私聊，无需添加好友；文字与图片消息、聊天记录搜索与联想、WebSocket 心跳与断线重连、持久化未读数与导航总角标、消息入场与平滑滚动、会话删除、点击头像在新标签页查看个人主页 |
| AI 对话 | DeepSeek / Qwen / GLM 多模型切换、多会话与历史记录、SSE 流式回答、联网搜索与来源展示、社区帖子检索、ASR 语音输入与 TTS 回答朗读 |
| MCP 工具 | 天气查询、汇率查询、网页搜索和帖子搜索；通过 stdio 协议调用独立工具进程 |
| 私有知识库 | 文档添加、查看与删除、文本分块、Embedding 向量化、Dense / Sparse 混合检索、RAG 增强回答、按用户隔离集合 |
| 内容创作 | 图文与视频发布、草稿、AI 图片标题与标签建议、视频封面、分片上传与断点续传、作品编辑与可见性管理、发布与编辑送审、驳回原因及修改重提 |
| 社区互动 | 发现页信息流、关注动态与作者更新提醒、帖子搜索、个人主页、评论与回复、评论区 @AI、帖子与评论点赞、收藏夹管理、内容反馈与站内通知 |
| 创作者助力 | 创作者开通与定价、支付宝单次购买 30 天权益与手动续期、专属图文与视频访问控制、按有效权益显示“已助力”或“助力专属”、收益与助力记录、退款申请与管理员审核 |
| 收益与提现 | 按订单比例自动分配收益、可提现余额与收益流水、本人支付宝绑定、免审核自动提现、提现记录与渠道查单、失败解冻、退款冻结与后续收益抵扣 |
| 管理控制台 | 管理员鉴权、业务概览与七天趋势、平台收款与抽成收益汇总、在线人数与 Go 运行状态、依赖探测、帖子封面列表与状态筛选、图文 / 视频 / 标签预览、审核通过与驳回、下架与恢复发布、助力订单与退款管理、提现查询与账户核对、Grafana / Kibana 入口 |

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 后端框架与工程 | Go 1.25.5 / Gin / GORM / Google Wire |
| 用户端 | Nuxt 4 / Vue 3 Composition API / Element Plus / Tailwind CSS |
| 管理端 | React 19 / TypeScript / Vite / React Router / Tailwind CSS |
| 数据存储 | MySQL（业务数据与事务）/ Redis（会话、验证码与缓存）/ 阿里云 OSS（文件与媒体） |
| 搜索与向量检索 | Elasticsearch 8 / Qdrant / GLM Embedding / TF-IDF 稀疏向量 / RRF 结果融合 |
| 实时通信 | Gorilla WebSocket / SSE（EventSource） |
| 消息与后台任务 | kafka-go / Kafka / Transactional Outbox / 消费幂等与失败重试 / 定时补偿 |
| AI 与工具编排 | CloudWeGo Eino / OpenAI 兼容接口 / DeepSeek、Qwen、GLM / MCP（mcp-go）/ RAG / ASR / TTS |
| 身份认证 | JWT / Redis 会话 / bcrypt 密码哈希 / OAuth 2.0 / HttpOnly 管理端 Cookie |
| 支付与外部接口 | 支付宝电脑网站支付、退款与商家转账 / RSA2 签名与验签 / 阿里云短信 / SMTP 邮件 / Seniverse / Tavily |
| 日志与可观测性 | Zap JSON 日志 / 日志轮转与慢 SQL 日志 / Prometheus / Grafana / Kibana 入口 / pprof |

## 架构与工程亮点

- **分层架构与事务一致性**：Controller → Service → DAO 分层，Wire 生成依赖注入代码；业务写入、计数和站内通知由 Service 在同一事务中编排，点赞等并发操作结合行锁与目标状态接口实现幂等。
- **多协议实时交互**：WebSocket 负责用户私聊，配合心跳与自动重连维持连接；SSE 输出 AI 回答，Kafka 异步处理私聊及 AI 消息。私聊消息入队、消费和落库时复核发送权限，任一方拉黑后禁止继续发送，历史消息保留。
- **持久化未读与快照确认**：在本人会话中保存已读入站消息主键边界，服务端聚合会话未读与总数；确认时核验会话归属及消息范围，在事务内锁定会话并单调推进进度，避免重复或乱序确认清除新消息。前端共享统计，结合实时消息、窗口可见性与轮询恢复未读状态，隔离跨账号及过期响应。
- **多模型与工具调用**：通过 Eino 统一接入 OpenAI 兼容模型，使用模型池组织 DeepSeek / Qwen / GLM；先判断并执行 MCP 工具，再流式输出回答，将联网搜索、社区内容和外部 API 接入对话。
- **RAG 混合检索**：文档按 500 字符分块、50 字符重叠，结合 GLM Dense Embedding 与 TF-IDF Sparse 向量，通过 Qdrant 的 RRF 融合召回结果，并按用户隔离集合。
- **搜索与信息流**：消息和帖子搜索优先使用 Elasticsearch，异常时降级到 MySQL；发现页采用游标分页，关注动态结合异步分发与已读进度维护，处理重复和乱序确认。
- **内容审核与状态一致性**：复用已发布、已下架、审核中、未通过四种状态，管理员通过独立鉴权入口预览图片、视频和封面；审核结合帖子行锁与版本校验，避免旧页面审核已修改内容，处理原因与状态一同保存。新内容送审后暂不进入公开信息流，审核通过再创建公开帖子的关注分发事件。
- **Outbox 与消费可靠性**：帖子发布、编辑、审核、下架、删除和可见性变更与 Outbox 事件同事务落库，后台投递 Kafka。消费者读取帖子最新状态更新 Elasticsearch，仅未删除、公开且已发布的内容可搜索，其余写入不含正文的不可见标记；搜索结果再通过 MySQL 复核公开状态。结合事件幂等、持久化重试和 Elasticsearch 外部版本控制，处理重复事件与旧写覆盖。
- **媒体上传与访问控制**：OSS 分片上传支持进度恢复与续传；媒体读取支持 Range 请求，并在后端校验作品可见性与助力权益，后台清理未被作品引用的过期资源。
- **支付与权益一致性**：支付通知通过 RSA2 验签，订单与权益在加锁事务中幂等更新；使用 DECIMAL 存储金额，后台执行查单、关单、退款补偿和退款账单核对，以租约支持任务重启恢复。
- **收益分配与自动提现**：按订单保存抽成比例与创作者份额，收益账户和流水同事务更新；提现通过账户行锁与请求幂等键防止透支和重复申请，保存固定转账单号与收款快照。渠道确认成功后结算，明确失败后解冻，结果未知时保留冻结并查原单；退款与在途提现联动，后续收益优先抵扣欠款。收款信息加密保存、页面脱敏展示，管理端按支付环境汇总抽成及退款冲减后的净收益。
- **可观测与管理闭环**：Zap 输出结构化日志并记录慢 SQL，Prometheus 暴露 HTTP、在线人数及发现页后台任务指标；管理端提供业务趋势、运行状态、依赖探测和外部观测工具入口，pprof 支持性能分析。

## 项目结构

```text
seekF/
├── seekF-backend/              # Go 后端
│   ├── cmd/
│   │   ├── seekf-server/       # 主服务入口与 Wire 依赖注入
│   │   └── mcp-server/         # 独立 MCP 工具进程（stdio）
│   ├── config/config.toml      # 主配置，敏感信息由后端根目录 .env 覆盖
│   ├── scripts/migrations/     # 增量数据库迁移，如私聊未读进度
│   └── internal/
│       ├── router/             # Gin 路由
│       ├── controllers/        # 用户端与管理端 HTTP 处理器
│       ├── services/           # 业务逻辑与事务编排
│       ├── dao/                # 数据访问
│       ├── dto/                # 请求与响应结构
│       ├── models/             # 数据表模型
│       ├── middlewares/        # 认证、CORS、指标等中间件
│       └── pkg/                # AI、支付、通信、存储与可观测性等
├── seekF-user/                 # Nuxt 4 用户端
│   ├── nuxt.config.ts          # API 与 WebSocket 等配置
│   └── app/
│       ├── pages/              # 文件路由页面
│       ├── components/         # 页面与业务组件
│       └── composables/        # API、聊天、未读状态与知识库等组合式函数
├── seekF-admin/                # React 管理控制台
├── deploy/observability/       # Grafana 看板配置
└── LICENSE                     # GPL-3.0
```

## 本地开发

### 环境与配置

- Go 版本以 [go.mod](seekF-backend/go.mod) 为准，当前为 **1.25.5**；前端可使用 **Node.js 22.12+（22.x）** 与 npm，具体要求以安装的依赖为准。
- 准备 MySQL、Redis、Kafka、Elasticsearch 和 Qdrant，配置可访问的服务地址，并准备与当前模型及迁移记录一致的数据库结构。以下命令不包含基础设施部署和数据库初始化。
- 后端配置见 [config.toml](seekF-backend/config/config.toml)，参考 [后端环境变量模板](seekF-backend/.env.example)，在 `seekF-backend/` 下复制 `.env.example` 为 `.env` 后按需填写模型、OSS、短信、邮件、OAuth、工具 API 与支付宝凭据。已有 `.env` 时仅补充缺少的配置，不要覆盖或提交真实凭据；覆盖项见[配置加载代码](seekF-backend/internal/configs/configs.go)。
- 用户端 API 与 WebSocket 地址见 [nuxt.config.ts](seekF-user/nuxt.config.ts)，默认分别为 `http://localhost:8080/` 和 `ws://localhost:8080/`。

助力支付与提现复用支付宝环境及 RSA2 凭据。启用提现时，还需在后端 `.env` 配置 `SUPPORT_ACCOUNT_ENCRYPTION_KEY`，取值为 **32 字节随机密钥的 Base64 编码**，用于加密收款账号、姓名和提现快照；部署及迁移时保持该密钥稳定。平台抽成由 `config.toml` 的 `[supportSettlementConfig].platformRateBPS` 配置，`1000` 表示 10%，仅影响新订单。

### 创作者与管理端入口

| 页面 | 路径 | 用途 |
| --- | --- | --- |
| 私聊消息 | `/chat` | 查看会话、未读数、文字与图片消息，搜索记录及删除会话 |
| 用户主页 | `/user/:id` | 查看作品与公开收藏夹、关注与粉丝列表，发起私聊或拉黑用户 |
| 我的主页 | `/my` | 维护个人资料，查看自己的帖子、收藏与点赞 |
| 黑名单管理 | `/my/blacklist` | 分页查看已拉黑用户并解除拉黑 |
| 作品发布与编辑 | `/discover/create` | 创建图文或视频，保存草稿并提交审核 |
| 作品管理 | `/discover/manage` | 按发布与审核状态查看作品、阅读处理原因、编辑与修改可见范围 |
| 助力设置 | `/discover/support` | 开通助力、设置价格 |
| 助力记录 | `/discover/support-records` | 查看助力订单、收款与退款情况 |
| 收益与提现 | `/discover/support-wallet` | 查看余额与流水、绑定支付宝、申请提现 |
| 管理端系统概览 | `/` | 查看平台总收款、累计抽成、退款退回抽成与平台净收益 |
| 管理端助力订单与退款 | `/support` | 查询订单、审核购买者退款 |
| 管理端收益与提现核对 | `/settlement` | 查询提现、核对渠道结果和账户资金 |
| 管理端帖子审核 | `/posts` | 查看封面、按标题 / 作者 / 编号及状态筛选，预览内容与媒体并审核、驳回、下架或恢复 |

管理端平台收益按当前支付环境的新规则已付款订单统计，使用每单实际份额计算；退款成功后扣除对应抽成，净收益未扣渠道手续费。沙箱标注为虚拟资金，概览统计最多缓存 30 秒。

### 私聊未读数升级

已有数据库升级当前版本前，在后端根目录执行[未读进度迁移](seekF-backend/scripts/migrations/chat-unread/main.go)：

```sh
cd seekF-backend
go run ./scripts/migrations/chat-unread
```

迁移为已有 `session` 表增加 `last_read_message_id`，首次执行将迁移时的历史消息初始化为已读；重复执行不会重置已经保存的正常进度，并核查字段默认值及中文表、字段注释。该命令仅负责此项增量升级，不创建完整业务数据库。迁移完成后重新启动后端，使未读统计、已读接口及消息主键响应生效。

### 启动应用

下面各组命令均从仓库根目录开始，在独立终端中执行。

**后端**

```sh
cd seekF-backend
go mod download
go run ./cmd/seekf-server
```

默认监听 `http://localhost:8080`。必须在 `seekF-backend/` 下执行，因为配置按 `./config/config.toml` 加载。MCP 工具初始化时会自动启动 stdio 子进程，默认命令为 `go run ./cmd/mcp-server`，无需另开终端；可通过 `MCP_SERVER_COMMAND` 指定其他启动命令。

**用户端**

```sh
cd seekF-user
npm install
npm run dev
```

默认访问 `http://localhost:3000`。

**管理端**

```sh
cd seekF-admin
npm install
npm run dev
```

访问 `http://127.0.0.1:3002`，开发代理默认连接本地 8080 后端，可通过管理端的 `VITE_BACKEND_TARGET` 修改目标地址。登录使用已有正常账号，数据库 `user_info.is_admin` 必须为 `1`；没有预置管理员密码。

### 编译与构建

在各应用目录执行：

| 目录 | 命令 | 用途 |
| --- | --- | --- |
| `seekF-backend/` | `go build ./...` | 后端编译检查 |
| `seekF-user/` | `npm run build` | 用户端生产构建 |
| `seekF-admin/` | `npm run build` | 管理端 TypeScript 检查与生产构建 |

## 许可证与致谢

项目采用 [GNU GPL-3.0](LICENSE) 许可证。

即时通讯部分参考了 [KamaChat](https://github.com/youngyangyang04/KamaChat)，感谢原项目的分享。
