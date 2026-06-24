package email

import (
	"fmt"
	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/zlog"

	"gopkg.in/gomail.v2"
)

// SendVerifyCodeEmail 发送验证码邮件
// toEmail: 收件人邮箱
// code: 6位验证码
func SendVerifyCodeEmail(toEmail, code string) error {
	cfg := configs.GetConfig()

	// 构建邮件内容
	subject := "【seekF】验证码"
	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
			<h2 style="color: #333;">seekF 验证码</h2>
			<p style="color: #666; font-size: 16px;">您好，您正在登录 seekF，验证码为：</p>
			<div style="background-color: #f5f5f5; padding: 15px; text-align: center; margin: 20px 0;">
				<span style="font-size: 32px; font-weight: bold; color: #60a5fa; letter-spacing: 5px;">%s</span>
			</div>
			<p style="color: #999; font-size: 14px;">验证码 5 分钟内有效，请勿泄露给他人。</p>
			<p style="color: #999; font-size: 14px;">如非本人操作，请忽略此邮件。</p>
			<hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
			<p style="color: #ccc; font-size: 12px;">此邮件由系统自动发送，请勿回复。</p>
		</div>
	`, code)

	// 创建邮件消息
	m := gomail.NewMessage()
	m.SetHeader("From", m.FormatAddress(cfg.EmailConfig.FromAddress, cfg.EmailConfig.FromName))
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	// 创建 SMTP 拨号器
	d := gomail.NewDialer(
		cfg.EmailConfig.SmtpHost,
		cfg.EmailConfig.SmtpPort,
		cfg.EmailConfig.Username,
		cfg.EmailConfig.Password,
	)

	// 发送邮件
	if err := d.DialAndSend(m); err != nil {
		zlog.Error("发送验证码邮件失败: " + err.Error())
		return fmt.Errorf("发送验证码邮件失败：%v", err)
	}

	zlog.Info("验证码邮件发送成功: " + toEmail)
	return nil
}
