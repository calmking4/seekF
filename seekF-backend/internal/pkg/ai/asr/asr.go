package asr

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"seekF-backend/internal/configs"
)

// MaxAudioBytes 为30秒PCM录音及44字节WAV文件头的最大大小。
const MaxAudioBytes = 44 + 16000*2*30

// ValidateRecording 校验浏览器生成的单声道、16kHz、16位PCM WAV，最多30秒。
func ValidateRecording(audio []byte) error {
	if len(audio) <= 44 || len(audio) > MaxAudioBytes {
		return fmt.Errorf("录音不能为空且不能超过30秒")
	}
	// 核对标准PCM文件头和数据长度，避免仅凭文件扩展名接受无效录音。
	if string(audio[:4]) != "RIFF" || string(audio[8:12]) != "WAVE" ||
		string(audio[12:16]) != "fmt " || binary.LittleEndian.Uint32(audio[16:20]) != 16 ||
		binary.LittleEndian.Uint16(audio[20:22]) != 1 || binary.LittleEndian.Uint16(audio[22:24]) != 1 ||
		binary.LittleEndian.Uint32(audio[24:28]) != 16000 || binary.LittleEndian.Uint32(audio[28:32]) != 32000 ||
		binary.LittleEndian.Uint16(audio[32:34]) != 2 || binary.LittleEndian.Uint16(audio[34:36]) != 16 ||
		string(audio[36:40]) != "data" || int(binary.LittleEndian.Uint32(audio[40:44])) != len(audio)-44 ||
		int(binary.LittleEndian.Uint32(audio[4:8])) != len(audio)-8 || (len(audio)-44)%2 != 0 {
		return fmt.Errorf("录音格式无效，请重新录音")
	}
	return nil
}

// StreamResult 保存供应商的语音识别事件流，由调用方关闭。
type StreamResult struct {
	Body io.ReadCloser
}

// Transcribe 校验录音并调用配置的ASR模型返回识别事件流。
func Transcribe(ctx context.Context, audio []byte) (*StreamResult, error) {
	if err := ValidateRecording(audio); err != nil {
		return nil, err
	}
	cfg := configs.GetConfig().AIModelConfig
	model, baseURL, key := resolveConfig(cfg)
	if key == "" || baseURL == "" {
		return nil, fmt.Errorf("语音识别服务未配置，请配置GLM接口地址和密钥")
	}
	return transcribe(ctx, &http.Client{Timeout: 60 * time.Second}, strings.TrimRight(baseURL, "/")+"/audio/transcriptions", key, model, audio)
}

// resolveConfig 获取ASR模型配置，复用GLM接口地址和密钥。
func resolveConfig(cfg configs.AIModelConfig) (model, baseURL, key string) {
	model = strings.TrimSpace(cfg.ASRModel)
	if model == "" {
		model = "glm-asr-2512"
	}
	baseURL = strings.TrimSpace(cfg.GlmBaseUrl)
	key = strings.TrimSpace(cfg.GlmApiKey)
	return
}

// transcribe 上传WAV录音并开启SSE识别，不缓冲完整响应。
func transcribe(ctx context.Context, client *http.Client, endpoint, key, model string, audio []byte) (*StreamResult, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("model", model); err != nil {
		return nil, fmt.Errorf("构建语音识别模型参数失败: %w", err)
	}
	if err := writer.WriteField("stream", "true"); err != nil {
		return nil, fmt.Errorf("构建语音识别参数失败: %w", err)
	}
	part, err := writer.CreateFormFile("file", "recording.wav")
	if err != nil {
		return nil, fmt.Errorf("构建录音文件失败: %w", err)
	}
	if _, err = part.Write(audio); err != nil {
		return nil, fmt.Errorf("写入录音文件失败: %w", err)
	}
	if err = writer.Close(); err != nil {
		return nil, fmt.Errorf("完成录音请求失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return nil, fmt.Errorf("创建语音识别请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "text/event-stream")
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用语音识别服务失败: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		// 不向客户端暴露供应商响应中的内部信息。
		return nil, fmt.Errorf("语音识别服务返回错误（状态码%d），请稍后重试", res.StatusCode)
	}
	if !strings.HasPrefix(strings.ToLower(res.Header.Get("Content-Type")), "text/event-stream") {
		res.Body.Close()
		return nil, fmt.Errorf("语音识别服务未返回事件流，请检查模型接口")
	}
	return &StreamResult{Body: res.Body}, nil
}
