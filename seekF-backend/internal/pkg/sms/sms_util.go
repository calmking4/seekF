package sms

import (
	"fmt"
	"seekF-backend/internal/configs"
	zlog "seekF-backend/internal/pkg/zlog"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v5/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	credential "github.com/aliyun/credentials-go/credentials"
)

// CreateClient 创建阿里云短信客户端
func CreateClient() (*dysmsapi20170525.Client, error) {
	// 获取配置
	cfg := configs.GetConfig()

	credConfig := &credential.Config{
		Type:            tea.String("access_key"),
		AccessKeyId:     tea.String(cfg.AuthCodeConfig.AccessKeyID),
		AccessKeySecret: tea.String(cfg.AuthCodeConfig.AccessKeySecret),
	}

	cred, err := credential.NewCredential(credConfig)
	if err != nil {
		zlog.Info("创建凭据失败: " + err.Error())
		return nil, err
	}

	config := &openapi.Config{
		Credential: cred,
		Endpoint:   tea.String("dysmsapi.aliyuncs.com"),
	}

	client, err := dysmsapi20170525.NewClient(config)
	if err != nil {
		zlog.Info("创建短信客户端失败: " + err.Error())
		return nil, err
	}

	return client, nil
}

// SendSMS 发送短信
// telephone: 手机号
// code: 验证码
func SendSMSCode(telephone, code string) error {
	// 创建客户端
	client, err := CreateClient()
	if err != nil {
		return err
	}

	// 获取配置
	cfg := configs.GetConfig()

	// 构建请求
	req := &dysmsapi20170525.SendSmsRequest{
		PhoneNumbers:  tea.String(telephone),
		SignName:      tea.String(cfg.AuthCodeConfig.SignName),
		TemplateCode:  tea.String(cfg.AuthCodeConfig.TemplateCode),
		TemplateParam: tea.String(fmt.Sprintf(`{"code":"%s"}`, code)),
	}

	// 发送短信
	resp, err := client.SendSmsWithOptions(req, &util.RuntimeOptions{})
	if err != nil {
		zlog.Info("发送短信失败: " + err.Error())
		return err
	}

	// 检查发送结果
	if resp.Body.Code != nil && *resp.Body.Code != "OK" {
		zlog.Info("发送短信失败: " + *resp.Body.Code + ", " + *resp.Body.Message)
		return fmt.Errorf("发送短信失败: %s, %s", *resp.Body.Code, *resp.Body.Message)
	}

	zlog.Info("发送短信成功，RequestId: " + *resp.Body.RequestId)
	return nil
}
