package payment

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"strings"

	"seekF-backend/internal/configs"
)

// AlipayGatewayURL 按明确的环境返回固定网关，防止误用网关或将密钥发送到其他地址。
func AlipayGatewayURL(environment string) string {
	switch environment {
	case "sandbox":
		return "https://openapi-sandbox.dl.alipaydev.com/gateway.do"
	case "production":
		return "https://openapi.alipay.com/gateway.do"
	default:
		return ""
	}
}

// ValidateAlipayConfig 校验必要参数、RSA2密钥和回调地址，供实际支付请求使用。
func ValidateAlipayConfig(cfg configs.AlipayConfig) error {
	if AlipayGatewayURL(cfg.Environment) == "" {
		return fmt.Errorf("支付宝环境必须为sandbox或production")
	}
	if cfg.AppID == "" || cfg.SellerID == "" || cfg.PrivateKey == "" || cfg.AlipayPublicKey == "" {
		return fmt.Errorf("支付宝配置不完整，请设置应用ID、收款方ID、应用私钥和支付宝公钥")
	}
	privateDER, err := alipayKeyDER(cfg.PrivateKey)
	if err != nil {
		return fmt.Errorf("支付宝应用私钥格式错误")
	}
	privateKey, err := x509.ParsePKCS8PrivateKey(privateDER)
	if err != nil {
		privateKey, err = x509.ParsePKCS1PrivateKey(privateDER)
	}
	key, ok := privateKey.(*rsa.PrivateKey)
	if err != nil || !ok || key.N.BitLen() < 2048 || key.Validate() != nil {
		return fmt.Errorf("支付宝应用私钥必须为有效的2048位或以上RSA密钥")
	}
	publicDER, err := alipayKeyDER(cfg.AlipayPublicKey)
	if err != nil {
		return fmt.Errorf("支付宝公钥格式错误")
	}
	publicKey, err := x509.ParsePKIXPublicKey(publicDER)
	pub, ok := publicKey.(*rsa.PublicKey)
	if err != nil || !ok || pub.N.BitLen() < 2048 {
		return fmt.Errorf("支付宝公钥必须为2048位或以上RSA公钥")
	}
	if err := validateAlipayURL(cfg.ReturnURL, cfg.Environment, false); err != nil {
		return fmt.Errorf("支付宝回跳地址无效: %w", err)
	}
	if err := validateAlipayURL(cfg.NotifyURL, cfg.Environment, true); err != nil {
		return fmt.Errorf("支付宝通知地址无效: %w", err)
	}
	return nil
}

// alipayKeyDER 解析单行Base64或PEM密钥，兼容环境变量中的换行转义。
func alipayKeyDER(value string) ([]byte, error) {
	value = strings.ReplaceAll(value, `\n`, "\n")
	if block, rest := pem.Decode([]byte(value)); block != nil {
		if strings.TrimSpace(string(rest)) != "" {
			return nil, fmt.Errorf("密钥包含多余内容")
		}
		return block.Bytes, nil
	}
	return base64.StdEncoding.DecodeString(strings.Join(strings.Fields(value), ""))
}

// validateAlipayURL 限制回调协议，通知地址必须使用可公开访问的HTTPS地址。
func validateAlipayURL(value, environment string, notify bool) error {
	u, err := url.Parse(value)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
		return fmt.Errorf("请填写完整且不含凭据或片段的地址")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && environment == "sandbox" && !notify) {
		return fmt.Errorf("请使用HTTPS地址，沙箱页面回跳允许HTTP")
	}
	if notify {
		host := strings.ToLower(u.Hostname())
		ip := net.ParseIP(host)
		if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || (ip != nil && (!ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast())) {
			return fmt.Errorf("请使用公网通知地址，不能填写本机或内网地址")
		}
	}
	return nil
}
