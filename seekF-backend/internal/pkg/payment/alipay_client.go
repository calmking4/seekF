package payment

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"seekF-backend/internal/configs"
)

// AlipayTrade 支付宝查单或通知中的业务结果。
type AlipayTrade struct {
	Code     string `json:"code"`
	SubCode  string `json:"sub_code"`
	OrderNo  string `json:"out_trade_no"`
	TradeNo  string `json:"trade_no"`
	Status   string `json:"trade_status"`
	Amount   string `json:"total_amount"`
	SellerID string `json:"seller_id"`
	PaidTime string `json:"send_pay_date"`
}

// alipayCanonical 对参数按名称排序并拼接未转义的签名原文。
func alipayCanonical(values url.Values) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		if k != "sign" && values.Get(k) != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+values.Get(k))
	}
	return strings.Join(parts, "&")
}

// alipayVerify 校验支付宝RSA2签名。
func alipayVerify(cfg configs.AlipayConfig, data []byte, signature string) error {
	der, err := alipayKeyDER(cfg.AlipayPublicKey)
	if err != nil {
		return fmt.Errorf("支付宝公钥格式错误")
	}
	key, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return fmt.Errorf("支付宝公钥格式错误")
	}
	pub, ok := key.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("支付宝公钥类型错误")
	}
	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("支付宝签名格式错误")
	}
	digest := sha256.Sum256(data)
	if rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sig) != nil {
		return fmt.Errorf("支付宝签名校验失败")
	}
	return nil
}

// alipayParameters 构造RSA2签名请求，不进行网络调用。
func alipayParameters(cfg configs.AlipayConfig, method string, biz any) (url.Values, error) {
	if err := ValidateAlipayConfig(cfg); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(biz)
	if err != nil {
		return nil, fmt.Errorf("编码支付宝业务参数失败: %w", err)
	}
	values := url.Values{"app_id": {cfg.AppID}, "method": {method}, "format": {"JSON"}, "charset": {"utf-8"}, "sign_type": {"RSA2"}, "timestamp": {time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04:05")}, "version": {"1.0"}, "biz_content": {string(raw)}}
	return values, nil
}

// alipaySign 为完整参数集合签名，包含支付回跳与通知地址。
func alipaySign(cfg configs.AlipayConfig, values url.Values) error {
	der, err := alipayKeyDER(cfg.PrivateKey)
	if err != nil {
		return fmt.Errorf("支付宝私钥格式错误")
	}
	key, err := x509.ParsePKCS8PrivateKey(der)
	if err != nil {
		key, err = x509.ParsePKCS1PrivateKey(der)
	}
	priv, ok := key.(*rsa.PrivateKey)
	if err != nil || !ok {
		return fmt.Errorf("支付宝私钥格式错误")
	}
	digest := sha256.Sum256([]byte(alipayCanonical(values)))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, digest[:])
	if err != nil {
		return fmt.Errorf("支付宝请求签名失败: %w", err)
	}
	values.Set("sign", base64.StdEncoding.EncodeToString(sig))
	return nil
}

// AlipayPageURL 生成电脑网站收银台地址，超时时间沿用订单的固定截止时间。
func AlipayPageURL(cfg configs.AlipayConfig, orderNo string, fen int64, expires time.Time) (string, error) {
	return alipayPageURL(cfg, orderNo, fen, expires, false)
}

// AlipayQRURL 使用电脑网站支付的嵌入式二维码，与网页付款共用同一笔订单。
func AlipayQRURL(cfg configs.AlipayConfig, orderNo string, fen int64, expires time.Time) (string, error) {
	return alipayPageURL(cfg, orderNo, fen, expires, true)
}

// alipayPageURL 按支付方式生成签名地址，二维码不设置回跳以保留当前作品。
func alipayPageURL(cfg configs.AlipayConfig, orderNo string, fen int64, expires time.Time, embedded bool) (string, error) {
	biz := map[string]any{"out_trade_no": orderNo, "total_amount": FormatAlipayAmount(fen), "subject": "创作者30天助力", "product_code": "FAST_INSTANT_TRADE_PAY", "time_expire": expires.In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04:05"), "seller_id": cfg.SellerID}
	if embedded {
		biz["qr_pay_mode"] = "4"
		biz["qrcode_width"] = 168
	}
	values, err := alipayParameters(cfg, "alipay.trade.page.pay", biz)
	if err != nil {
		return "", err
	}
	back, err := url.Parse(cfg.ReturnURL)
	if err != nil {
		return "", fmt.Errorf("支付回跳地址错误")
	}
	q := back.Query()
	q.Set("order_no", orderNo)
	back.RawQuery = q.Encode()
	if !embedded {
		values.Set("return_url", back.String())
	}
	values.Set("notify_url", cfg.NotifyURL)
	if err := alipaySign(cfg, values); err != nil {
		return "", err
	}
	return AlipayGatewayURL(cfg.Environment) + "?" + values.Encode(), nil
}

// AlipayRequest 执行查单或关单，先验证响应原始JSON签名再读取业务结果。
func AlipayRequest(ctx context.Context, cfg configs.AlipayConfig, method, orderNo string) (*AlipayTrade, error) {
	values, err := alipayParameters(cfg, method, map[string]string{"out_trade_no": orderNo})
	if err != nil {
		return nil, err
	}
	if err = alipaySign(cfg, values); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, AlipayGatewayURL(cfg.Environment), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("构造支付宝请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := (&http.Client{Timeout: 12 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求支付宝失败: %w", err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("读取支付宝响应失败: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("支付宝响应异常，状态码%d", response.StatusCode)
	}
	var envelope map[string]json.RawMessage
	if json.Unmarshal(raw, &envelope) != nil {
		return nil, fmt.Errorf("支付宝响应格式错误")
	}
	node := envelope[strings.ReplaceAll(method, ".", "_")+"_response"]
	var signature string
	if json.Unmarshal(envelope["sign"], &signature) != nil || len(node) == 0 {
		return nil, fmt.Errorf("支付宝响应缺少签名或结果")
	}
	if err := alipayVerify(cfg, node, signature); err != nil {
		return nil, err
	}
	var trade AlipayTrade
	if json.Unmarshal(node, &trade) != nil {
		return nil, fmt.Errorf("支付宝交易结果格式错误")
	}
	// 查单响应不保证包含seller_id；已验签的结果归属于本次应用私钥签名请求对应的商户。
	if method == "alipay.trade.query" && trade.Code == "10000" && trade.SellerID == "" {
		trade.SellerID = cfg.SellerID
	}
	return &trade, nil
}

// VerifyAlipayNotify 验证通知签名及应用、收款方，拒绝重复参数歧义。
func VerifyAlipayNotify(cfg configs.AlipayConfig, values url.Values) (*AlipayTrade, error) {
	if err := ValidateAlipayConfig(cfg); err != nil {
		return nil, err
	}
	for _, v := range values {
		if len(v) != 1 {
			return nil, fmt.Errorf("支付宝通知包含重复参数")
		}
	}
	if values.Get("sign_type") != "RSA2" {
		return nil, fmt.Errorf("支付宝通知签名类型错误")
	}
	copied := url.Values{}
	for k, v := range values {
		if k != "sign_type" {
			copied[k] = v
		}
	}
	if err := alipayVerify(cfg, []byte(alipayCanonical(copied)), values.Get("sign")); err != nil {
		return nil, err
	}
	if values.Get("app_id") != cfg.AppID || values.Get("seller_id") != cfg.SellerID {
		return nil, fmt.Errorf("支付宝通知应用或收款方不匹配")
	}
	return &AlipayTrade{OrderNo: values.Get("out_trade_no"), TradeNo: values.Get("trade_no"), Status: values.Get("trade_status"), Amount: values.Get("total_amount"), SellerID: values.Get("seller_id"), PaidTime: values.Get("gmt_payment")}, nil
}

// FormatAlipayAmount 将整数分转换为两位小数元，避免浮点误差。
func FormatAlipayAmount(fen int64) string { return fmt.Sprintf("%d.%02d", fen/100, fen%100) }

// ParseAlipayAmount 严格解析渠道元金额为整数分。
func ParseAlipayAmount(value string) (int64, error) {
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, fmt.Errorf("支付宝金额格式错误")
	}
	for _, c := range strings.ReplaceAll(value, ".", "") {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("支付宝金额格式错误")
		}
	}
	fraction := "00"
	if len(parts) == 2 {
		if len(parts[1]) < 1 || len(parts[1]) > 2 {
			return 0, fmt.Errorf("支付宝金额格式错误")
		}
		fraction = (parts[1] + "0")[:2]
	}
	yuan, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || yuan > 1000000 {
		return 0, fmt.Errorf("支付宝金额超出范围")
	}
	fen, _ := strconv.ParseInt(fraction, 10, 64)
	return yuan*100 + fen, nil
}
