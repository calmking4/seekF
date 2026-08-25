package rag

import (
	"math"
	"regexp"
	"strings"
	"sync"

	"seekF-backend/internal/pkg/db"
)

// SparseVector 稀疏向量，用于关键词匹配（别名）
type SparseVector = db.SparseVector

// SparseVectorizer 稀疏向量化器，基于TF-IDF算法
type SparseVectorizer struct {
	vocabulary   map[string]uint32  // 词汇表：词 -> 索引
	idf          map[string]float32 // IDF值：词 -> IDF
	mu           sync.RWMutex
	dirty        bool             // 标记是否需要重新计算IDF
	docCount     int              // 文档数量
	termDocCount map[string]int   // 包含该词的文档数量
}

// NewSparseVectorizer 创建稀疏向量化器
func NewSparseVectorizer() *SparseVectorizer {
	return &SparseVectorizer{
		vocabulary:   make(map[string]uint32),
		idf:          make(map[string]float32),
		termDocCount: make(map[string]int),
		dirty:        true,
	}
}

// tokenize 中文分词（简单的基于字符的分词，支持2-4字的n-gram）
func tokenize(text string) []string {
	text = strings.ToLower(text)
	// 移除标点符号和特殊字符
	reg := regexp.MustCompile(`[^\p{L}\p{N}\s]`)
	text = reg.ReplaceAllString(text, " ")

	// 按空格分词
	words := strings.Fields(text)

	var tokens []string

	for _, word := range words {
		// 英文单词直接作为token
		if isASCII(word) {
			if len(word) > 1 { // 过滤单字符
				tokens = append(tokens, word)
			}
			continue
		}

		// 中文使用n-gram分词（2-4字）
		runes := []rune(word)
		length := len(runes)
		for n := 2; n <= 4 && n <= length; n++ {
			for i := 0; i <= length-n; i++ {
				token := string(runes[i : i+n])
				tokens = append(tokens, token)
			}
		}
	}

	return tokens
}

// isASCII 检查字符串是否全是ASCII字符
func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

// BuildVocabulary 从文档集合构建词汇表和IDF
func (sv *SparseVectorizer) BuildVocabulary(documents []string) {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	sv.docCount = len(documents)
	sv.termDocCount = make(map[string]int)

	// 统计每个词出现在多少文档中
	for _, doc := range documents {
		tokens := tokenize(doc)
		seen := make(map[string]struct{})
		for _, token := range tokens {
			if _, exists := seen[token]; !exists {
				seen[token] = struct{}{}
				sv.termDocCount[token]++
			}
		}
	}

	// 构建词汇表并计算IDF
	idx := uint32(0)
	for term, docFreq := range sv.termDocCount {
		if _, exists := sv.vocabulary[term]; !exists {
			sv.vocabulary[term] = idx
			idx++
		}
		// IDF = log(N / (df + 1)) + 1，平滑处理
		sv.idf[term] = float32(math.Log(float64(sv.docCount)/float64(docFreq+1)) + 1)
	}

	sv.dirty = false
}

// Vectorize 将文本转换为稀疏向量
func (sv *SparseVectorizer) Vectorize(text string) SparseVector {
	sv.mu.RLock()
	defer sv.mu.RUnlock()

	tokens := tokenize(text)
	if len(tokens) == 0 {
		return SparseVector{}
	}

	// 计算TF（词频）
	termFreq := make(map[string]int)
	for _, token := range tokens {
		termFreq[token]++
	}

	// 计算TF-IDF并构建稀疏向量
	indexMap := make(map[uint32]float32)
	for term, freq := range termFreq {
		idx, exists := sv.vocabulary[term]
		if !exists {
			continue // 词汇表中没有的词跳过
		}

		tf := float32(1 + math.Log(float64(freq))) // 对数TF
		idf, _ := sv.idf[term]
		score := tf * idf

		if existing, ok := indexMap[idx]; ok {
			indexMap[idx] = existing + score
		} else {
			indexMap[idx] = score
		}
	}

	// 转换为数组格式
	indices := make([]uint32, 0, len(indexMap))
	values := make([]float32, 0, len(indexMap))
	for idx, val := range indexMap {
		indices = append(indices, idx)
		values = append(values, val)
	}

	return SparseVector{
		Indices: indices,
		Values:  values,
	}
}

// GetVocabularySize 获取词汇表大小
func (sv *SparseVectorizer) GetVocabularySize() int {
	sv.mu.RLock()
	defer sv.mu.RUnlock()
	return len(sv.vocabulary)
}
