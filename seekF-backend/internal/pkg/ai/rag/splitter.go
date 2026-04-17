package rag

import (
	"regexp"
	"strings"
)

const (
	DefaultChunkSize = 500
	DefaultOverlap   = 50
)

// TextSplitter 文本分块器,将长文本分割成适合向量化的较小块
type TextSplitter struct {
	chunkSize int
	overlap   int
}

// NewTextSplitter 创建文本分块器实例
func NewTextSplitter(chunkSize, overlap int) *TextSplitter {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	if overlap < 0 {
		overlap = DefaultOverlap
	}
	return &TextSplitter{
		chunkSize: chunkSize,
		overlap:   overlap,
	}
}

// SplitText 将文本分割成多个块
func (s *TextSplitter) SplitText(text string) []string {
	if text == "" {
		return nil
	}

	text = normalizeText(text)
	if len(text) <= s.chunkSize {
		return []string{text}
	}

	var chunks []string
	paragraphs := splitIntoParagraphs(text)

	var currentChunk strings.Builder
	currentLen := 0

	for _, para := range paragraphs {
		paraLen := len(para)

		if currentLen+paraLen+1 <= s.chunkSize {
			if currentLen > 0 {
				currentChunk.WriteString("\n")
				currentLen++
			}
			currentChunk.WriteString(para)
			currentLen += paraLen
		} else {
			if currentLen > 0 {
				chunks = append(chunks, currentChunk.String())
			}

			if currentLen > s.overlap && s.overlap > 0 {
				prevChunk := currentChunk.String()
				prevStart := len(prevChunk) - s.overlap
				if prevStart > 0 {
					overlapText := prevChunk[prevStart:]
					currentChunk.Reset()
					currentChunk.WriteString(overlapText)
					currentLen = len(overlapText)
				} else {
					currentChunk.Reset()
					currentLen = 0
				}
			} else {
				currentChunk.Reset()
				currentLen = 0
			}

			if currentLen > 0 {
				currentChunk.WriteString("\n")
				currentLen++
			}
			currentChunk.WriteString(para)
			currentLen += paraLen
		}
	}

	if currentLen > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	if len(chunks) == 0 {
		return []string{text}
	}

	return chunks
}

// normalizeText 规范化文本(统一换行符,去除多余空行)
func normalizeText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	reg := regexp.MustCompile(`\n{3,}`)
	text = reg.ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

// splitIntoParagraphs 按段落分割文本
func splitIntoParagraphs(text string) []string {
	var paragraphs []string

	lines := strings.Split(text, "\n")
	var current strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current.Len() > 0 {
				paragraphs = append(paragraphs, current.String())
				current.Reset()
			}
		} else {
			if current.Len() > 0 {
				current.WriteString(" ")
			}
			current.WriteString(line)
		}
	}

	if current.Len() > 0 {
		paragraphs = append(paragraphs, current.String())
	}

	return paragraphs
}
