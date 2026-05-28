package tts

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ttsStreamEvent 智谱 TTS 流式 SSE 单条 data 载荷
type ttsStreamEvent struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// ssePCMReader 将智谱 TTS 的 SSE（base64 PCM）转为裸 PCM 字节流
type ssePCMReader struct {
	scanner *bufio.Scanner
	pcmBuf  []byte
	err     error
	closed  bool
}

func newSSEPCMReader(r io.Reader) *ssePCMReader {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return &ssePCMReader{scanner: sc}
}

func (r *ssePCMReader) Read(p []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	if len(r.pcmBuf) > 0 {
		n := copy(p, r.pcmBuf)
		r.pcmBuf = r.pcmBuf[n:]
		return n, nil
	}
	if r.closed {
		return 0, io.EOF
	}

	for len(r.pcmBuf) == 0 {
		if !r.scanner.Scan() {
			if err := r.scanner.Err(); err != nil {
				r.err = fmt.Errorf("读取TTS流失败: %w", err)
				return 0, r.err
			}
			r.closed = true
			return 0, io.EOF
		}

		line := strings.TrimSpace(r.scanner.Text())
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			r.closed = payload == "[DONE]"
			if r.closed {
				return 0, io.EOF
			}
			continue
		}

		if err := r.decodeEvent(payload); err != nil {
			r.err = err
			return 0, err
		}
		if len(r.pcmBuf) > 0 {
			break
		}
	}

	n := copy(p, r.pcmBuf)
	r.pcmBuf = r.pcmBuf[n:]
	return n, nil
}

func (r *ssePCMReader) decodeEvent(payload string) error {
	var evt ttsStreamEvent
	if err := json.Unmarshal([]byte(payload), &evt); err != nil {
		return fmt.Errorf("解析TTS流数据失败: %w", err)
	}
	if evt.Error != nil {
		return fmt.Errorf("TTS服务错误: %s", evt.Error.Message)
	}
	for _, choice := range evt.Choices {
		content := choice.Delta.Content
		if content == "" {
			continue
		}
		pcm, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			return fmt.Errorf("解码TTS音频数据失败: %w", err)
		}
		r.pcmBuf = append(r.pcmBuf, pcm...)
	}
	return nil
}

// ssePCMReadCloser 包装底层 HTTP 响应体，Close 时一并关闭
type ssePCMReadCloser struct {
	*ssePCMReader
	underlying io.ReadCloser
}

func (c *ssePCMReadCloser) Close() error {
	return c.underlying.Close()
}

func wrapSSEPCMStream(body io.ReadCloser) io.ReadCloser {
	return &ssePCMReadCloser{
		ssePCMReader: newSSEPCMReader(body),
		underlying:   body,
	}
}
