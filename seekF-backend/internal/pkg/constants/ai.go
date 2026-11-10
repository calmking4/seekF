package constants

import "time"

// 知识库正文分块参数。
const (
	// 每块最多500个Unicode字符。
	KnowledgeChunkSize = 500
	// 相邻块保留最多50个尾部字符作为上下文重叠。
	KnowledgeChunkOverlap = 50
)

// 知识库BM25索引参数，调整预处理或参数后需重新生成已有稀疏向量。
const (
	// GLM语义向量的命名字段。
	KnowledgeDenseVector = "dense"
	// Qdrant服务端内置的BM25模型。
	KnowledgeBM25Model = "qdrant/bm25"
	// 复用既有稀疏向量字段，兼容Qdrant 1.16.3不支持追加命名向量的限制。
	KnowledgeBM25Vector = "sparse"
	// 文档与问题共用的ES中文分析器。
	KnowledgeBM25Analyzer = "ik_max_word"
	// 单次ES分词最多等待5秒。
	KnowledgeBM25AnalyzeTimeout = 5 * time.Second
	// 词频饱和参数。
	KnowledgeBM25K = 1.2
	// 文档长度归一化参数。
	KnowledgeBM25B = 0.75
	// IK分词后的固定平均词项数估计，并非集合实时统计。
	KnowledgeBM25AvgLen = 256.0
	// 每批迁移的最大分块数量。
	KnowledgeBM25BatchSize = 100
	// 知识库最多注入模型的分块数量。
	KnowledgeContextLimit = 5
)

// AI回答生成与回放时限。
const (
	// 单次后台生成最多运行15分钟。
	AIStreamTimeout = 15 * time.Minute
	// 回答事件在最近写入后保留30分钟。
	AIStreamTTL = 30 * time.Minute
	// 后台任务每5秒写入一次心跳。
	AIStreamHeartbeat = 5 * time.Second
	// 超过30秒未更新且未落库的任务视为中断。
	AIStreamStaleTimeout = 30 * time.Second
	// 单次Redis读写最多等待3秒。
	AIStreamRedisTimeout = 3 * time.Second
	// 每轮最多读取100条事件。
	AIStreamReadCount = 100
	// 回答事件流的Redis键前缀。
	AIStreamKeyPrefix = "ai:stream:"
)

// AI助手身份与展示信息。
const (
	// AI评论使用的助手用户编号。
	AIAssistantUserId = "AI_ASSISTANT"
	// AI助手的展示名称。
	AIAssistantName = "AI助手"
	// AI助手的默认头像地址。
	AIAssistantAvatar = "https://seekf.oss-cn-shenzhen.aliyuncs.com/common/ai_avatar/AI%E5%8A%A9%E6%89%8B%E5%A4%B4%E5%83%8F.png"
)

// AI评论回复协程池。
const (
	// AI评论工作协程数量及批次最大任务数。
	AICommentWorkers = 4
	// AI评论收集批次的等待时间。
	AICommentBatchWait = 100 * time.Millisecond
)

// MCP客户端启动限制。
const (
	// MCP单次握手及工具加载最多等待30秒。
	MCPInitTimeout = 30 * time.Second
	// MCP启动最多尝试3次。
	MCPInitAttempts = 3
	// MCP启动失败后等待2秒再重试。
	MCPInitRetryDelay = 2 * time.Second
)
