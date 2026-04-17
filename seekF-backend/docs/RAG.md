# SeekF 项目 RAG 实现详解

## 什么是 RAG？

RAG（Retrieval Augmented Generation，检索增强生成）是一种让 AI 模型能够访问私有知识库的技术。它的工作流程是：

1. **索引阶段**：将文档切分成小块，转换为向量，存入向量数据库
2. **查询阶段**：将用户问题转换为向量，从数据库中检索最相关的文档片段
3. **生成阶段**：将检索到的内容作为上下文，让 AI 生成答案

---

## 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         用户上传文档                             │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  1. 下载文件                                                     │
│  2. 文本分词 (Splitter)                                         │
│  3. 向量化 (Embedding)                                          │
│  4. 存储向量 (Qdrant)                                            │
│  5. 保存元数据 (MySQL)                                           │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                         用户提问                                 │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  1. 问题向量化                                                   │
│  2. 向量搜索 (Qdrant)                                            │
│  3. 获取相关文档片段                                             │
│  4. 构建提示词 + 调用 AI                                         │
│  5. 流式返回答案                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 核心组件

### 1. 文本分词器 (Splitter)

**文件**: `seekF-backend/internal/pkg/ai/rag/splitter.go`

负责将长文档拆分成小片段，便于后续向量化和检索。

```go
// 创建分词器：每块 500 字符，重叠 50 字符
splitter := NewTextSplitter(500, 50)

// 分词
chunks := splitter.SplitText(content)
```

**分词逻辑**:
- 按段落分割，保留段落结构
- 每块最大 500 字符
- 相邻块之间有 50 字符重叠（保持上下文连贯性）
- 过滤多余空行，清理换行符

### 2. 向量化 (Embedding)

**文件**: `seekF-backend/internal/pkg/ai/rag/embedding.go`

调用 GLM 模型将文本转换为 2048 维向量。

```go
emb := NewEmbedding(apiKey, model, baseURL)
vectors, err := emb.EmbedTexts(ctx, chunks)
```

**调用流程**:
```
请求 → POST {baseURL}/embeddings
      {
        "input": "文本内容",
        "model": "embedding-3"
      }

响应 ←
      {
        "data": [{
          "embedding": [0.1, 0.2, ...],  // 2048 维向量
          "index": 0
        }]
      }
```

### 3. 向量存储 (Qdrant)

**文件**: `seekF-backend/internal/pkg/db/qdrant_util.go`

向量数据库，负责存储和检索向量。

**核心操作**:

| 操作 | 说明 |
|------|------|
| `EnsureCollection` | 创建 collection（向量库） |
| `UpsertChunks` | 批量插入向量 + 文本 + 元数据 |
| `Search` | 向量相似度搜索 |
| `DeleteByDocUUID` | 按 doc_uuid 删除向量 |

**存储结构**:
```go
PointStruct{
    Id:      uint64,           // 随机生成的唯一 ID
    Vectors: []float32,       // 2048 维向量
    Payload: {
        "text": "...",        // 文档片段内容
        "doc_uuid": "K..."    // 关联的文档 ID
    }
}
```

---

## 核心流程

### 文档上传流程 (AddDocument)

**调用链**:
```
Controller → Service → RAG → Qdrant + MySQL
```

**代码位置**: `knowledge_service.go:48-97`

```go
func (s *KnowledgeServiceImpl) AddDocument(...) (*DocInfo, error) {
    // 1. 从 OSS 下载文件内容
    content, err := s.downloadFile(ctx, fileURL)

    // 2. 如果是 Markdown，移除标记
    if fileType == "md" {
        content = removeMarkdownMarkers(content)
    }

    // 3. 分词：将长文本拆分成小块
    ragInst := rag.GetRAG()
    chunks := ragInst.GetSplitter().SplitText(content)

    // 4. 确保用户的 collection 存在（向量库）
    collectionName := "knowledge_" + userId
    ragInst.EnsureCollection(ctx, collectionName)

    // 5. 向量化 + 存储到 Qdrant
    docUUID := "K" + randomString(11)
    ragInst.UpsertChunks(ctx, collectionName, chunks, docUUID)

    // 6. 保存元数据到 MySQL
    doc := &models.Knowledge{Uuid: docUUID, ...}
    knowledgeDAO.Create(doc)

    return DocInfo{ChunkCnt: len(chunks)}, nil
}
```

### 文档删除流程 (RemoveDocument)

**代码位置**: `knowledge_service.go:120-145`

```go
func (s *KnowledgeServiceImpl) RemoveDocument(...) error {
    // 1. 查询文档，确认权限
    doc, _ := knowledgeDAO.FindByUuid(uuid)

    // 2. 删除 Qdrant 中的向量（按 doc_uuid 过滤）
    ragInst.DeleteChunks(ctx, collectionName, uuid)

    // 3. 删除 MySQL 中的元数据记录
    knowledgeDAO.Delete(uuid)
}
```

---

## AI 对话中的 RAG 应用

### 流程图

```
用户问题
    │
    ▼
┌───────────────────┐
│  向量化问题        │
│  emb.EmbedTexts   │
└────────┬──────────┘
         │
         ▼
┌───────────────────┐
│  Qdrant 语义搜索   │  ←─ 从用户的知识库中检索
│  rag.Search       │      相关文档片段
└────────┬──────────┘
         │
         ▼
┌───────────────────┐
│  构建 Prompt      │
│  [上下文] + 问题   │
└────────┬──────────┘
         │
         ▼
┌───────────────────┐
│  调用 AI 模型     │
│  生成回答          │
└────────┬──────────┘
         │
         ▼
     SSE 流式返回
```

### RAG 初始化

**文件**: `seekF-backend/internal/pkg/ai/rag/init.go`

```go
type RAG struct {
    embedding *Embedding  // 向量化组件
    splitter  *TextSplitter  // 分词器
}

func GetRAG() *RAG {
    ragOnce.Do(func() {
        // 使用 GLM 的 embedding-3 模型
        emb := NewEmbedding(apiKey, "embedding-3", baseURL)

        // 每块 500 字符，重叠 50
        spl := NewTextSplitter(500, 50)

        ragInstance = &RAG{embedding: emb, splitter: spl}
    })
    return ragInstance
}
```

### 搜索方法

```go
func (r *RAG) Search(ctx context.Context, collectionName, query string, topK int) ([]string, error) {
    // 1. 将用户问题转换为向量
    vectors, _ := r.embedding.EmbedTexts(ctx, []string{query})

    // 2. 在 Qdrant 中搜索最相似的 topK 个向量
    return db.GetQdrant().Search(ctx, collectionName, vectors[0], topK)
}
```

---

## 数据存储

### MySQL (元数据)

表: `knowledge`

| 字段 | 类型 | 说明 |
|------|------|------|
| uuid | varchar | 文档唯一ID (如 K2026041780625137333) |
| user_id | varchar | 所属用户ID |
| file_name | varchar | 文件名 |
| file_url | varchar | OSS 地址 |
| file_type | varchar | 文件类型 (md/txt等) |
| chunk_count | int | 切片数量 |
| created_at | datetime | 创建时间 |

### Qdrant (向量)

Collection: `knowledge_{userId}`

| 字段 | 说明 |
|------|------|
| id | 随机 uint64 |
| vector | 2048 维向量 |
| payload.text | 文档片段 |
| payload.doc_uuid | 关联的文档ID |

---

## 关键技术点

### 1. Collection 命名
```go
collectionName = "knowledge_" + userId
```
每个用户有独立的向量库，实现数据隔离。

### 2. 文档 ID 生成
```go
docUUID = "K" + randomString(11)
// 例: K2026041780625137333
```
- 前缀 "K" 标识为知识库文档
- 后 11 位随机字符串

### 3. 文本预处理
- Markdown 文件：移除 `#`、`````、`---` 等标记
- 统一换行符：`\r\n` → `\n`
- 合并连续空行

### 4. 向量维度
- 使用 **embedding-3** 模型，输出 **2048 维**向量
- Qdrant collection 创建时指定 `vectorSize: 2048`

### 5. 流式响应
AI 对话使用 **SSE (Server-Sent Events)** 实现流式输出：
```go
ctx.Header("Content-Type", "text/event-stream")
fmt.Fprintf(ctx.Writer, "data: {\"content\": \"...\"}\n\n")
ctx.Writer.Flush()
```

---

## 常见问题排查

### Q: 向量插入失败
**错误**: `Unable to parse UUID: K2026041780625137333_0`

**原因**: Qdrant 默认要求 UUID 格式

**解决**: 使用 `qdrant.NewIDNum()` 传入数值型 ID

---

## 总结

SeekF 的 RAG 实现简洁高效：

1. **文档上传** → 下载 → 分词 → 向量化 → 存入 Qdrant
2. **用户提问** → 向量化 → 检索 → 构建 Prompt → AI 生成
3. 每个用户有独立的 collection，实现数据隔离
4. 使用 GLM embedding-3 生成 2048 维向量
5. Markdown 文件自动清理标记符号