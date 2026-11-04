# seekF

**基于 Go 与 Nuxt 4 的 feed流 内容社区，集成实时私聊与创作者助力。**

seekF 围绕实时交流、智能问答和内容创作构建完整业务链路：支持用户私聊、多模型 AI 对话、私有知识库、图文与视频社区，以及创作者付费权益、收益分配、自动提现和管理控制台。

项目采用 monorepo 结构，包含 Go 后端、Nuxt 4 用户端和 React 管理控制台。

## 核心功能

| 模块 | 能力 |
| --- | --- |
| 账号与认证 | 密码登录、手机与邮箱验证码、GitHub / Gitee OAuth 登录、个人资料维护、JWT 或 Redis 会话认证 |
| 用户关系 | 关注与粉丝列表弹窗、用户主页、拉黑与解除拉黑、个人黑名单管理；任一方拉黑后禁止私信 |
| 即时通讯 | 从用户主页发起私聊，无需添加好友；文字与图片消息、聊天记录搜索与联想、WebSocket 心跳与断线重连、持久化未读数与导航总角标、会话置顶与删除、消息右键菜单、两分钟内撤回本人消息、仅对本人删除消息、点击头像在新标签页查看个人主页 |
| AI 对话 | DeepSeek / Qwen / GLM 多模型切换、多会话与历史记录、SSE 流式回答、联网搜索与来源展示、社区帖子检索、ASR 语音输入与 TTS 回答朗读 |
| MCP 工具 | 天气查询、汇率查询、网页搜索和帖子搜索；通过 stdio 协议调用独立工具进程 |
| 私有知识库 | 文档添加、查看与删除、文本分块、Embedding 向量化、Dense / Sparse 混合检索、RAG 增强回答、按用户隔离集合 |
| 内容创作 | 图文与视频发布、草稿、AI 图片标题与标签建议、视频封面、分片上传与断点续传、作品编辑与可见性管理、发布与编辑送审、驳回原因及修改重提 |
| 社区互动 | 发现页信息流、关注动态与作者更新提醒、帖子搜索、个人主页、评论与回复、评论区 @AI、帖子与评论点赞、收藏夹管理、内容反馈与站内通知 |
| 创作者数据中心 | 点击量、净增粉丝、点赞、收藏、助力数、评论与分享七项指标；点击量与收益趋势、近 7 / 30 / 90 天筛选、每日明细、刷新与 CSV 导出 |
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

### 分层设计与 AI 编排

- **分层与依赖注入**：后端采用 Controller → Service → DAO 分层，由 Wire 生成依赖组装代码。Service 统一管理事务，业务写入、计数和站内通知使用同一事务连接；并发修改结合行锁与目标状态校验，保持重复请求下的数据一致性。
- **多模型与 MCP 工具**：通过 Eino 接入 OpenAI 兼容模型，复用 DeepSeek / Qwen / GLM 模型实例。对话先判断是否调用工具、执行 MCP 工具，再通过 SSE 流式输出回答；工具运行于独立 stdio 子进程，提供联网搜索、社区帖子检索与外部 API 能力。
- **用户隔离的 RAG**：文档按 500 字符分块、50 字符重叠，结合 GLM 稠密向量与 TF-IDF 稀疏向量，通过 Qdrant 的 RRF 融合检索结果；每个用户使用独立集合，隔离私有知识库。

### 实时通信与内容分发

- **消息传输与权限复核**：WebSocket 承载私聊，配合心跳与自动重连维持连接，Kafka 异步处理私聊及 AI 消息。私聊在入队、消费和落库阶段复核发送权限，任一方拉黑后禁止继续发送。
- **未读与消息状态同步**：以入站消息主键记录已读边界，事务内锁定会话并单调推进进度，避免重复或乱序确认误清新消息。撤回与个人删除统一影响历史、搜索、未读及会话摘要，前端过滤迟到响应，避免旧内容重新出现。
- **搜索与信息流**：帖子搜索优先使用 Elasticsearch，异常时降级到 MySQL；私聊搜索限定本人会话并过滤个人删除记录。关注流采用混合推拉：普通作者通过 Outbox 与 Kafka 将帖子索引写入粉丝 Redis ZSet，大 V 作品读取时从 MySQL 召回；缓存缺失时从帖子及关注关系重建，故障或历史翻页超出缓存范围时回退 MySQL 全拉，统一复核当前权限并按时间与主键游标分页。
- **审核与媒体访问**：帖子审核结合行锁与版本校验，防止旧页面审核已修改内容，公开帖子通过审核后再创建关注分发事件。OSS 支持分片续传，媒体读取支持 Range 请求，并在后端校验作品可见性与助力权益。

### 异步可靠性与并发控制

- **Transactional Outbox**：帖子变更与事件同事务落库，后台投递 Kafka；消费者读取帖子最新状态同步 Elasticsearch，结合持久化重试、幂等处理和外部版本控制，应对重复投递与乱序写入。仅公开、已发布且未删除的帖子可搜索，查询结果再经 MySQL 复核状态。
- **有界协程池**：封装泛型任务队列，以固定 worker 复用协程。浏览量更新使用 4 个 worker、256 个排队容量、5 秒任务超时，队列满时等待空位并支持请求取消；AI 评论使用 4 个 worker，每批最多拉取 4 条消息并行处理，等待本批全部完成后同步提交 Kafka 位点，提交成功才拉取下一批。处理失败采用退避重试，稳定评论 ID 与事务内去重防止重放产生重复回复和计数。
- **限流与资源保护**：验证码发送通过 Redis Lua 原子执行冷却及账号、IP 窗口计数，登录与验证码校验结合滑动窗口和凭据失败统计。AI 请求使用用户令牌桶，流式请求与后台评论共用单实例的用户及总并发上限；超限返回 `429` 和建议重试时间，Redis 异常返回 `503`。

### 资金一致性与运行观测

- **支付、权益与补偿**：支付通知通过 RSA2 验签，订单与权益在加锁事务中幂等更新，金额使用 DECIMAL 存储。后台执行查单、关单、退款补偿和账单核对，通过任务租约支持重启恢复。
- **收益结算与自动提现**：订单保存抽成比例与创作者份额，账户和流水同事务更新；提现通过账户行锁、请求幂等键及固定转账单号防止透支与重复转账。渠道确认成功后结算，明确失败后解冻，结果未知时保留冻结并持续查单；退款与在途提现联动，后续收益优先抵扣欠款，收款信息加密保存并脱敏展示。
- **统计口径与可观测性**：点击按北京时间、用户与帖子每日去重并排除作者自身，关注事件与关系变更同事务保存，分享通过请求键去重；收益按流水发生日汇总入账与退款冲销。Zap 记录结构化日志与慢 SQL，Prometheus 暴露 HTTP、在线人数和后台任务指标，结合 Grafana、管理端依赖探测及 pprof 支持运行排查。

## 项目结构

```text
seekF/
├── seekF-backend/              # Go 后端
│   ├── cmd/
│   │   ├── seekf-server/       # 主服务入口与 Wire 依赖注入
│   │   └── mcp-server/         # 独立 MCP 工具进程（stdio）
│   ├── config/config.toml      # 主配置，敏感信息由后端根目录 .env 覆盖
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
├── plan/                       # 功能计划、实施记录与验收说明
└── LICENSE                     # GPL-3.0
```

## 本地开发

### 环境与配置

- Go 版本以 [go.mod](seekF-backend/go.mod) 为准，当前为 **1.25.5**；前端 Node.js 版本需满足各自锁定依赖的 `engines` 要求，使用 npm 安装依赖。
- 准备 MySQL、Redis、Kafka、Elasticsearch 和 Qdrant，配置可访问的服务地址，并准备与当前模型及迁移记录一致的数据库结构。以下命令不包含基础设施部署和数据库初始化。
- 后端配置见 [config.toml](seekF-backend/config/config.toml)，参考 [后端环境变量模板](seekF-backend/.env.example)，在 `seekF-backend/` 下复制 `.env.example` 为 `.env` 后按需填写模型、OSS、短信、邮件、OAuth、工具 API 与支付宝凭据。已有 `.env` 时仅补充缺少的配置，不要覆盖或提交真实凭据；覆盖项见[配置加载代码](seekF-backend/internal/configs/configs.go)。
- 用户端 API 与 WebSocket 地址见 [nuxt.config.ts](seekF-user/nuxt.config.ts)，默认分别为 `http://localhost:8080/` 和 `ws://localhost:8080/`。

助力支付与提现复用支付宝环境及 RSA2 凭据。启用提现时，还需在后端 `.env` 配置 `SUPPORT_ACCOUNT_ENCRYPTION_KEY`，取值为 **32 字节随机密钥的 Base64 编码**，用于加密收款账号、姓名和提现快照；部署及迁移时保持该密钥稳定。平台抽成由 `config.toml` 的 `[supportSettlementConfig].platformRateBPS` 配置，`1000` 表示 10%，仅影响新订单。

### 创作者与管理端入口

| 页面 | 路径 | 用途 |
| --- | --- | --- |
| 私聊消息 | `/chat` | 查看会话与未读数、置顶或删除会话，搜索记录，右键撤回或仅对本人删除消息 |
| 用户主页 | `/user/:id` | 查看作品与公开收藏夹、关注与粉丝列表，发起私聊或拉黑用户 |
| 我的主页 | `/my` | 维护个人资料，查看自己的帖子、收藏与点赞 |
| 黑名单管理 | `/my/blacklist` | 分页查看已拉黑用户并解除拉黑 |
| 作品发布与编辑 | `/discover/create` | 创建图文或视频，保存草稿并提交审核 |
| 作品管理 | `/discover/manage` | 按发布与审核状态查看作品、阅读处理原因、编辑与修改可见范围 |
| 创作者数据中心 | `/discover/analytics` | 查看七项核心指标、点击与收益趋势，筛选时间、展开每日明细并导出 CSV |
| 助力设置 | `/discover/support` | 开通助力、设置价格 |
| 助力记录 | `/discover/support-records` | 查看助力订单、收款与退款情况 |
| 收益与提现 | `/discover/support-wallet` | 查看余额与流水、绑定支付宝、申请提现 |
| 管理端系统概览 | `/` | 查看平台总收款、累计抽成、退款退回抽成与平台净收益 |
| 管理端助力订单与退款 | `/support` | 查询订单、审核购买者退款 |
| 管理端收益与提现核对 | `/settlement` | 查询提现、核对渠道结果和账户资金 |
| 管理端帖子审核 | `/posts` | 查看封面、按标题 / 作者 / 编号及状态筛选，预览内容与媒体并审核、驳回、下架或恢复 |



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
