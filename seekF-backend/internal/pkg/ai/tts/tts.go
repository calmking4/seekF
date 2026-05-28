package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"
)

// ttsRequest TTS请求，启用 stream 时 API 以 SSE + base64 返回 PCM
type ttsRequest struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	Voice          string `json:"voice"`
	ResponseFormat string `json:"response_format"`
	EncodeFormat   string `json:"encode_format,omitempty"`
	Stream         bool   `json:"stream"`
}

// StreamResult 流式TTS结果
type StreamResult struct {
	Body   io.ReadCloser // PCM 音频数据流
	Format string        // 音频格式 (pcm)
}

// Synthesize 调用智谱 TTS API 的流式接口，返回 PCM 音频数据流
func Synthesize(ctx context.Context, content string, voice string) (*StreamResult, error) {
	cfg := configs.GetConfig().AIModelConfig

	model := cfg.TTSModel
	if model == "" {
		model = "glm-tts"
	}
	if voice == "" {
		voice = cfg.TTSVoice
	}
	if voice == "" {
		voice = "chuichui"
	}

	reqBody := ttsRequest{
		Model:          model,
		Input:          content,
		Voice:          voice,
		ResponseFormat: "pcm",
		EncodeFormat:   "base64",
		Stream:         true,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("构建TTS请求失败: %w", err)
	}

	ttsURL := cfg.GlmBaseUrl + "audio/speech"
	req, err := http.NewRequestWithContext(ctx, "POST", ttsURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建TTS请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.GlmApiKey)

	// 使用带超时的 HTTP 客户端，避免无限等待
	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		zlog.Error("TTS API调用失败: " + err.Error())
		return nil, fmt.Errorf("调用TTS服务失败")
	}
	// 注意：不 defer resp.Body.Close()，调用方负责关闭

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		zlog.Error(fmt.Sprintf("TTS API返回 %d: %s", resp.StatusCode, string(errBody)))
		return nil, fmt.Errorf("TTS服务返回错误: %d", resp.StatusCode)
	}

	zlog.Info("TTS流式响应开始，解析SSE并输出裸PCM")

	return &StreamResult{
		Body:   wrapSSEPCMStream(resp.Body),
		Format: "pcm",
	}, nil
}
