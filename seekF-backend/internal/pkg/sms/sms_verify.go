package sms

import (
	"encoding/json"
	"fmt"
	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dypnsapi "github.com/alibabacloud-go/dypnsapi-20170525/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
)

// SendCode 根据配置选择号码认证或原短信服务发送验证码。
func SendCode(telephone, code string) error {
	cfg := configs.GetConfig().AuthCodeConfig
	switch strings.TrimSpace(cfg.Provider) {
	case "pnvs":
		return SendSMSVerifyCode(telephone, code)
	case "sms", "":
		return SendSMSCode(telephone, code)
	default:
		return fmt.Errorf("不支持的短信发送方式：%s", cfg.Provider)
	}
}

// SendSMSVerifyCode 通过号码认证发送自定义验证码，由现有 Redis 流程负责校验。
func SendSMSVerifyCode(telephone, code string) error {
	cfg := configs.GetConfig().AuthCodeConfig
	if strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.AccessKeySecret) == "" {
		return fmt.Errorf("号码认证访问密钥未配置")
	}
	if strings.TrimSpace(cfg.VerifySignName) == "" || strings.TrimSpace(cfg.VerifyTemplateCode) == "" {
		return fmt.Errorf("号码认证签名或模板未配置")
	}
	client, err := dypnsapi.NewClient(&openapi.Config{
		AccessKeyId:     tea.String(cfg.AccessKeyID),
		AccessKeySecret: tea.String(cfg.AccessKeySecret),
		Endpoint:        tea.String("dypnsapi.aliyuncs.com"),
		ConnectTimeout:  tea.Int(10000),
		ReadTimeout:     tea.Int(10000),
	})
	if err != nil {
		return fmt.Errorf("创建号码认证客户端失败：%w", err)
	}
	return sendSMSVerifyCode(client, cfg, telephone, code)
}

// sendSMSVerifyCode 提交号码认证请求并检查业务结果，避免将失败或空响应视为成功。
func sendSMSVerifyCode(client *dypnsapi.Client, cfg configs.AuthCodeConfig, telephone, code string) error {
	// 显式传入验证码及五分钟有效期，与 Redis 的验证码内容和有效期一致。
	params, err := json.Marshal(map[string]string{"code": code, "min": "5"})
	if err != nil {
		return fmt.Errorf("构建号码认证模板参数失败：%w", err)
	}
	resp, err := client.SendSmsVerifyCodeWithOptions(&dypnsapi.SendSmsVerifyCodeRequest{
		PhoneNumber:      tea.String(telephone),
		CountryCode:      tea.String("86"),
		SignName:         tea.String(cfg.VerifySignName),
		TemplateCode:     tea.String(cfg.VerifyTemplateCode),
		TemplateParam:    tea.String(string(params)),
		ReturnVerifyCode: tea.Bool(false),
	}, &util.RuntimeOptions{})
	if err != nil {
		return fmt.Errorf("发送号码认证短信失败：%w", err)
	}
	if resp == nil || resp.Body == nil {
		return fmt.Errorf("发送号码认证短信失败：响应为空")
	}
	if tea.StringValue(resp.Body.Code) != "OK" || !tea.BoolValue(resp.Body.Success) {
		return fmt.Errorf("发送号码认证短信失败：%s，%s", tea.StringValue(resp.Body.Code), tea.StringValue(resp.Body.Message))
	}
	requestID := ""
	if resp.Body.Model != nil {
		requestID = tea.StringValue(resp.Body.Model.RequestId)
	}
	zlog.Info("号码认证短信提交成功，请求编号：" + requestID)
	return nil
}
