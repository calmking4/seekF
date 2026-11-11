# seekF

[![Project Demo](https://img.shields.io/badge/Demo-Yuque-25b864)](https://www.yuque.com/lengjing-7k1h6/iadugr/coq4r9zpocyuwgyr)
[![Stars](https://img.shields.io/github/stars/calmking4/seekF?style=social)](https://github.com/calmking4/seekF/stargazers)
[![Forks](https://img.shields.io/github/forks/calmking4/seekF?style=social)](https://github.com/calmking4/seekF/forks)

[中文](README.md) | **English**

**A feed-based content community built with Go and Nuxt 4, with real-time private messaging and paid creator support.**

seekF combines private messaging, multi-model AI conversations, private knowledge bases, image and video posts, paid creator benefits, revenue sharing, automatic withdrawals, and an admin console.

The monorepo contains a Go backend, a Nuxt 4 user frontend, and a React admin console.

## Features

| Module | Capabilities |
| --- | --- |
| Accounts and authentication | Password login, phone and email verification codes, GitHub / Gitee OAuth, profile management, and JWT or Redis session authentication |
| User relationships | Follow and follower lists, public profiles, blocking and unblocking, and a personal blocklist; blocking in either direction prevents private messages |
| Private messaging | Start conversations from profiles without friend requests; text and images, history search and suggestions, WebSocket heartbeats and reconnection, persistent unread counts, pinned and deleted conversations, message context menus, recall within two minutes, personal message deletion, and profile links from avatars |
| AI conversations | DeepSeek / Qwen / GLM, multiple conversations and history, SSE streaming, web search with sources, community post search, ASR voice input, and TTS playback |
| MCP tools | Weather, exchange rates, web search, and post search through an independent stdio tool process |
| Private knowledge bases | Add, view, and delete documents; chunking, embeddings, dense / sparse hybrid retrieval, RAG answers, and collections isolated by user |
| Content creation | Image and video posts, drafts, AI title and tag suggestions from images, video covers, resumable multipart uploads, editing and visibility controls, moderation submissions, and resubmission after rejection |
| Community | Discovery feeds, followed-author updates and notifications, post search, profiles, comments and replies, @AI comments, likes, favorites, feedback, and in-app notifications |
| Creator analytics | Views, net follower growth, likes, favorites, support purchases, comments, and shares; view and revenue trends, 7 / 30 / 90-day filters, daily details, refresh, and CSV export |
| Paid creator support | Creator enrollment and pricing, Alipay purchases of 30-day benefits with manual renewal, exclusive image and video access, entitlement labels, support and revenue records, and refund requests with admin review |
| Revenue and withdrawals | Order-based revenue sharing, balances and ledgers, personal Alipay binding, automatic withdrawals, transfer status queries, unfreezing after failure, and refund-related freezes and offsets against future revenue |
| Admin console | Admin authentication, business overview and seven-day trends, collections and commissions, online users and Go runtime status, dependency checks, post previews and moderation, takedown and restoration, support orders and refunds, withdrawal reconciliation, and Grafana / Kibana links |

## Technology Stack

| Layer | Technologies |
| --- | --- |
| Backend | Go 1.25.5 / Gin / GORM / Google Wire |
| User frontend | Nuxt 4 / Vue 3 Composition API / Element Plus / Tailwind CSS |
| Admin frontend | React 19 / TypeScript / Vite / React Router / Tailwind CSS |
| Storage | MySQL for business data and transactions / Redis for sessions, verification codes, caching, and AI response event streams / Alibaba Cloud OSS for files and media |
| Search and retrieval | Elasticsearch / IK Chinese tokenization / Qdrant / GLM Embedding / BM25 sparse vectors / reciprocal rank fusion (RRF) |
| Real-time communication | Gorilla WebSocket / SSE with fetch streaming and reconnection |
| Background jobs | kafka-go / Kafka / Transactional Outbox / idempotent consumers / retries and scheduled recovery |
| AI and tools | CloudWeGo Eino / OpenAI-compatible APIs / DeepSeek, Qwen, GLM / MCP via mcp-go / RAG / ASR / TTS |
| Authentication | JWT / Redis sessions / bcrypt / OAuth 2.0 / HttpOnly admin cookies |
| Payments and integrations | Alipay website payments, refunds, and merchant transfers / RSA2 signing and verification / Alibaba Cloud SMS / SMTP / Seniverse / Tavily |
| Observability | Zap JSON logs / log rotation and slow SQL logs / Prometheus / Grafana / Kibana links / pprof |

## Architecture Highlights

### Layers and AI Orchestration

- **Layers and dependency injection:** Controller → Service → DAO, assembled through Wire. Services own transactions; business writes, counters, and in-app notifications share the transaction connection. Row locks and state validation protect concurrent updates and repeated requests.
- **Multiple models and MCP:** Eino connects to reusable OpenAI-compatible model instances. Conversations determine whether tools are needed, execute MCP tools, and then stream the answer. Background jobs write events to Redis Streams; browsers subscribe through SSE and resume by cursor. Services persist complete answers and conversation summaries in MySQL. Tools run in an independent stdio process.
- **User-isolated RAG:** Documents use 500-character chunks with a 50-character overlap. GLM produces dense vectors; Elasticsearch IK tokenizes Chinese text for Qdrant BM25 sparse vectors. RRF combines retrieval results. Each user has a separate collection, with semantic retrieval as a fallback if tokenization fails.

### Messaging and Content Distribution

- **Message delivery and permissions:** WebSocket carries private messages with heartbeats and reconnection. Services check user status, blocking relationships, and conversation ownership inside MySQL transactions, deduplicate by client request ID, and save messages. After commit, they acknowledge delivery and push to online users. Private chat does not depend on Kafka or Elasticsearch; Kafka handles AI comments, post search synchronization, and following-feed distribution.
- **Unread counts and message state:** Read progress is tracked by incoming message IDs and advances monotonically under conversation locks. Recall and personal deletion consistently affect history, search, unread counts, and conversation summaries. The frontend filters stale responses.
- **Search and feeds:** Post search uses Elasticsearch with a MySQL fallback. Private message search stays within the user's conversations and excludes personally deleted messages. Following feeds combine fan-out on write for regular authors through Outbox, Kafka, and Redis ZSets with reads from MySQL for high-follower authors. Missing caches are rebuilt; failures or older pagination fall back to MySQL. Results are checked against current permissions and paginated by time and ID.
- **Moderation and media:** Row locks and version checks prevent moderation of stale content. Public posts enter following-feed distribution after approval. OSS uploads support multipart resume; media reads support Range requests and backend visibility and entitlement checks.

### Reliability and Concurrency

- **Transactional Outbox:** Post changes and events commit together before background delivery to Kafka. Consumers synchronize current post state to Elasticsearch with persistent retries, idempotency, and external version control. Only public, published, undeleted posts are searchable, and results are checked against MySQL.
- **Bounded worker pools:** Generic task queues reuse a fixed number of workers. View updates use four workers, a queue capacity of 256, and five-second task timeouts, with cancellation while waiting for capacity. AI comments process batches of up to four messages with four workers, then synchronously commit Kafka offsets before reading the next batch. Backoff, stable comment IDs, and transactional deduplication prevent duplicate replies and counters.
- **Rate limits:** Redis Lua atomically applies verification-code cooldowns and account / IP limits. Login and code checks track sliding windows and credential failures. AI requests use per-user token buckets and instance-wide concurrency limits shared with background comments. Limits return `429` with retry guidance; Redis failures return `503`.

### Payments and Observability

- **Payments and recovery:** RSA2-verified notifications update orders and entitlements idempotently under transactional locks. Amounts use DECIMAL storage. Background jobs query orders, close expired orders, recover refunds, and reconcile bills, with task leases supporting restart recovery.
- **Revenue and withdrawals:** Orders retain commission rates and creator shares. Balances and ledger entries update in one transaction. Account locks, idempotency keys, and stable transfer IDs prevent overdrafts and duplicate transfers. Confirmed transfers settle balances; definitive failures unfreeze funds. Unknown results stay frozen while queries continue. Refunds coordinate with pending transfers, and future earnings offset outstanding amounts. Recipient information is encrypted and masked.
- **Analytics and operations:** Views are deduplicated daily by user and post in Beijing time, excluding authors' own views. Follow events and relationship changes commit together; shares use request-key deduplication. Revenue is aggregated by ledger date. Structured logs, slow SQL logs, Prometheus metrics, Grafana, dependency checks, and pprof support troubleshooting.

## Project Structure

```text
seekF/
├── seekF-backend/              # Go backend
│   ├── cmd/
│   │   ├── seekf-server/       # Main entry point and Wire dependency injection
│   │   └── mcp-server/         # Independent MCP tool process (stdio)
│   ├── config/config.toml      # Main configuration; secrets overridden by .env
│   └── internal/
│       ├── router/             # Gin routes
│       ├── controllers/        # User and admin HTTP handlers
│       ├── services/           # Business logic and transactions
│       ├── dao/                # Data access
│       ├── dto/                # Request and response types
│       ├── models/             # Database models
│       ├── middlewares/        # Authentication, CORS, metrics, etc.
│       └── pkg/                # AI, payments, messaging, storage, observability
├── seekF-user/                 # Nuxt 4 user frontend
│   ├── nuxt.config.ts          # API and WebSocket configuration
│   └── app/
│       ├── pages/              # File-based routes
│       ├── components/         # UI and business components
│       └── composables/        # API, chat, unread state, knowledge bases, etc.
├── seekF-admin/                # React admin console
├── deploy/observability/       # Grafana dashboards
├── plan/                       # Local plans and implementation notes
└── LICENSE                     # GPL-3.0
```

## Local Development

### Requirements and Configuration

- Go **1.25.5**, as specified in [go.mod](seekF-backend/go.mod); Node.js **22** and npm for the frontends.
- Prepare MySQL, Redis, Kafka, Elasticsearch, Qdrant, and the database schema. Elasticsearch requires a matching IK plugin; Qdrant must support BM25.
- Configure backend connections in [config.toml](seekF-backend/config/config.toml). Copy the [backend environment template](seekF-backend/.env.example) to `.env` in the same directory and fill in credentials. Do not commit secrets.
- For user access through Nginx, copy the [user environment template](seekF-user/.env.example) to `.env` in the same directory. If it already exists, add only missing settings.
- Configure Alipay credentials as needed for payments and withdrawals. See the [payment guide (Chinese)](seekF-backend/docs/支付.md) and [RAG guide (Chinese)](seekF-backend/docs/RAG.md).

### Creator and Admin Pages

| Page | Path | Purpose |
| --- | --- | --- |
| Private messages | `/chat` | Conversations, unread counts, pinning, deletion, search, recall, and personal message deletion |
| Public profile | `/user/:id` | Posts, public favorites, followers, following, messaging, and blocking |
| My profile | `/my` | Profile editing, own posts, favorites, and likes |
| Blocklist | `/my/blacklist` | Browse and unblock users |
| Create or edit a post | `/discover/create` | Drafts, image and video posts, and moderation submissions |
| Post management | `/discover/manage` | Publication and review status, rejection reasons, editing, and visibility |
| Creator analytics | `/discover/analytics` | Seven core metrics, trends, time filters, daily details, and CSV export |
| Support settings | `/discover/support` | Enable paid support and set pricing |
| Support records | `/discover/support-records` | Orders, collections, and refunds |
| Revenue and withdrawals | `/discover/support-wallet` | Balances, ledgers, Alipay binding, and withdrawals |
| Admin overview | `/admin/` | Collections, commissions, refunds, and net platform revenue |
| Admin support orders | `/admin/support` | Order queries and refund review |
| Admin settlement | `/admin/settlement` | Withdrawals, transfer status, and account reconciliation |
| Admin moderation | `/admin/posts` | Post filtering, previews, approval, rejection, takedown, and restoration |

### Start the Applications

Run each group from the repository root in a separate terminal.

**Backend**

```sh
cd seekF-backend
go mod download
go run ./cmd/seekf-server
```

Listens on port 8080. Run from the backend directory; MCP starts automatically.

**User frontend**

```sh
cd seekF-user
npm install
npm run dev
```

Listens on port 3000. After setting up Nginx below, visit `http://localhost`.

**Admin frontend**

```sh
cd seekF-admin
npm install
npm run dev
```

Listens on port 3002. With Nginx, visit `http://localhost/admin/`. Use an existing account with `user_info.is_admin` set to `1` in the database.

### Build

Run commands from each application directory:

| Directory | Command | Purpose |
| --- | --- | --- |
| `seekF-backend/` | `go build ./...` | Backend compilation check |
| `seekF-user/` | `npm run build` | User production build |
| `seekF-admin/` | `npm run build` | TypeScript check and admin production build |

Run frontend build outputs from their application directories:

| Application | Output | Command |
| --- | --- | --- |
| User frontend | Complete `.output/` directory | `node .output/server/index.mjs`, port 3000 by default |
| Admin frontend | `dist/` | `npm run preview`, local preview on port 3002 |

Build outputs do not automatically reflect source changes; rebuild after editing. Admin preview is for local verification and does not replace a production static file server.

## Docker

These configurations target local Docker Desktop with Linux containers. Start each application separately and prepare dependencies such as MySQL and Redis yourself.

### Preparation

- Start Docker and the required dependencies.
- Prepare the backend `.env` and check [config.docker.toml](seekF-backend/config/config.docker.toml). The current Kafka connection uses host port 9093.
- Stop local development services using ports `8080`, `3000`, or `3002`.

### Build and Start

**Run each group from the repository root. Start the backend before the frontends.**

**Backend**

```sh
cd seekF-backend
docker compose up -d --build backend
```

Listens on port 8080. The image includes the backend and MCP binaries.

**User frontend**

```sh
cd seekF-user
docker compose up -d --build user
```

Listens on port 3000 and uses `/api/` for API and WebSocket connections. With Nginx, visit `http://localhost`.

**Admin frontend**

```sh
cd seekF-admin
docker compose up -d --build admin
```

Listens on port 3002. With Nginx, visit `http://localhost/admin/`. The container currently uses Vite preview for local verification.

### Unified Nginx Entry Point

Nginx uses [default.conf](deploy/nginx/default.conf). Publish port 80 to the host:

| Address | Destination |
| --- | --- |
| `http://localhost/` | User frontend, port 3000 |
| `http://localhost/admin/` | Admin frontend, port 3002 |
| `http://localhost/api/...` | Backend, port 8080, with `/api` removed |

For an existing container named `nginx`, connect it to a custom network once:

```sh
docker network create seekf-nginx-network
docker network connect seekf-nginx-network nginx
```

Back up the existing configuration and copy the repository configuration to `conf.d/default.conf` in the mounted Nginx configuration directory. Then check and reload:

```sh
docker exec nginx nginx -t           # Check configuration
docker exec nginx nginx -s reload    # Run only after a successful check
```

Requires Nginx 1.27.3 or later. When recreating the container, include `--network seekf-nginx-network`.

Run the applications either locally or in Docker, not both on the same ports. For local development, configure the user `.env` from the [template](seekF-user/.env.example) and access the site through Nginx.

OAuth provider callback URLs must match the backend configuration. If a payment tunnel points to Nginx port 80, add `/api` to the payment notification path.

## License and Acknowledgments

Licensed under [GNU GPL-3.0](LICENSE).

The private messaging implementation draws on [KamaChat](https://github.com/youngyangyang04/KamaChat). Thanks to the original project for sharing its work.
