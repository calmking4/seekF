package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"
)

type ttsRequest struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	Voice          string `json:"voice"`
	ResponseFormat string `json:"response_format"`
}

// Synthesize 调用智谱 TTS API，返回 WAV 格式音频数据
func Synthesize(ctx context.Context, content string, voice string) ([]byte, error) {
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
		ResponseFormat: "wav",
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		zlog.Error("TTS API调用失败: " + err.Error())
		return nil, fmt.Errorf("调用TTS服务失败")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		zlog.Error(fmt.Sprintf("TTS API returned %d: %s", resp.StatusCode, string(errBody)))
		return nil, fmt.Errorf("TTS服务返回错误: %d", resp.StatusCode)
	}

	wavData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取TTS响应失败: %w", err)
	}

	zlog.Info(fmt.Sprintf("TTS response: contentType=%s, size=%d", resp.Header.Get("Content-Type"), len(wavData)))

	return wavData, nil
}
