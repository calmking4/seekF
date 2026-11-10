package rag

import (
	"regexp"
	"strings"
	"unicode"

	"seekF-backend/internal/pkg/constants"
)

// TextSplitter 按Unicode字符分块，优先保留段落和句子边界。
type TextSplitter struct {
	chunkSize int
	overlap   int
}

// NewTextSplitter 创建分块器，块长及重叠长度的单位均为Unicode字符。
func NewTextSplitter(chunkSize, overlap int) *TextSplitter {
	if chunkSize <= 0 {
		chunkSize = constants.KnowledgeChunkSize
	}
	if overlap < 0 {
		overlap = constants.KnowledgeChunkOverlap
	}
	// 重叠必须小于块长，异常配置退回块长的十分之一，保证持续向前推进。
	if overlap >= chunkSize {
		overlap = chunkSize / 10
	}
	return &TextSplitter{chunkSize: chunkSize, overlap: overlap}
}

// SplitText 按字符上限切分正文，相邻块保留尾部重叠且不产生空块。
func (s *TextSplitter) SplitText(text string) []string {
	paragraphs := splitIntoParagraphs(normalizeText(text))
	runes := []rune(strings.Join(paragraphs, "\n\n"))
	if len(runes) == 0 {
		return nil
	}
	var chunks []string
	for start := 0; start < len(runes); {
		end := min(start+s.chunkSize, len(runes))
		if end < len(runes) {
			// 在窗口后半段寻找语义边界，避免很短的块被重叠内容占满。
			minEnd := start + max(s.chunkSize/2, s.overlap+1)
			end = findChunkEnd(runes, start, end, minEnd)
		}
		if chunk := strings.TrimSpace(string(runes[start:end])); chunk != "" {
			chunks = append(chunks, chunk)
		}
		if end == len(runes) {
			break
		}
		start = end - s.overlap
	}
	return chunks
}

// NormalizeMarkdown 清理Markdown块标记，保留标题文字和段落空行供分块使用。
func NormalizeMarkdown(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	var result []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		title := strings.TrimLeft(line, "#")
		if title != line && (title == "" || strings.HasPrefix(title, " ") || strings.HasPrefix(title, "\t")) {
			// 标题两侧保留段落边界；只移除独立的尾部井号，保留C#等名称。
			title = strings.TrimSpace(title)
			if lastSpace := strings.LastIndexAny(title, " \t"); lastSpace >= 0 && strings.Trim(title[lastSpace+1:], "#") == "" {
				title = strings.TrimSpace(title[:lastSpace])
			}
			if len(result) > 0 && result[len(result)-1] != "" {
				result = append(result, "")
			}
			if title != "" {
				result = append(result, title, "")
			}
			continue
		}
		if strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~") || line == "---" || line == "***" || line == "___" {
			line = ""
		}
		// 合并连续空行，但保留一个段落间隔。
		if line == "" && (len(result) == 0 || result[len(result)-1] == "") {
			continue
		}
		result = append(result, line)
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}

// normalizeText 统一换行符，合并多余空行并保留段落间隔。
func normalizeText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	reg := regexp.MustCompile("\n{3,}")
	return strings.TrimSpace(reg.ReplaceAllString(text, "\n\n"))
}

// splitIntoParagraphs 按空行识别段落，将同一段落内的普通换行合并为空格。
func splitIntoParagraphs(text string) []string {
	var paragraphs []string
	var current strings.Builder
	for _, line := range strings.Split(text, "\n") {
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

// findChunkEnd 优先取窗口内靠后的段落边界，其次句末，最后按字符上限截取。
func findChunkEnd(text []rune, start, limit, minEnd int) int {
	paragraphEnd, sentenceEnd := 0, 0
	for i := start; i < limit; i++ {
		if i+1 < minEnd {
			continue
		}
		if i > start && text[i-1] == '\n' && text[i] == '\n' {
			paragraphEnd = i + 1
		}
		switch text[i] {
		case '。', '！', '？', '!', '?':
			sentenceEnd = i + 1
		case '.':
			// 英文句号后需为空白，避免将小数或域名中间当作句末。
			if i+1 == len(text) || unicode.IsSpace(text[i+1]) {
				sentenceEnd = i + 1
			}
		}
	}
	if paragraphEnd > 0 {
		return paragraphEnd
	}
	if sentenceEnd > 0 {
		return sentenceEnd
	}
	return limit
}
