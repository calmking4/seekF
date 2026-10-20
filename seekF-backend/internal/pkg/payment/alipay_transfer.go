package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/constants"
	"strings"
	"time"
)

// AlipayTransferResult 经过RSA2验签的转账提交或查询结果。
type AlipayTransferResult struct {
	Code        string `json:"code"`
	SubCode     string `json:"sub_code"`
	OutBizNo    string `json:"out_biz_no"`
	OrderID     string `json:"order_id"`
	Status      string `json:"status"`
	Amount      string `json:"amount"`
	TransAmount string `json:"trans_amount"`
}

// TransferPayee 保存收款账号、姓名及本次转账的固定商户信息。
type TransferPayee struct {
	AccountVersion int64  `json:"account_version"`
	Account        string `json:"account"`
	Name           string `json:"name"`
	AppID          string `json:"app_id"`
	SellerID       string `json:"seller_id"`
}

// ValidateTransferConfig 复用现有支付宝RSA2配置校验，不在启动时增加额外初始化。
func ValidateTransferConfig(cfg configs.AlipayConfig) error { return ValidateAlipayConfig(cfg) }

// AlipayTransfer 复用支付签名方式提交或查询转账，接口失败不当作资金成功。
func AlipayTransfer(ctx context.Context, cfg configs.AlipayConfig, no, amount string, payee TransferPayee, query bool) (*AlipayTransferResult, error) {
	method := "alipay.fund.trans.uni.transfer"
	biz := map[string]any{"out_biz_no": no, "product_code": constants.SupportTransferProduct, "biz_scene": constants.SupportTransferScene}
	if query {
		method = "alipay.fund.trans.common.query"
	} else {
		biz["trans_amount"] = amount
		biz["order_title"] = "创作者收益提现"
		biz["payee_info"] = map[string]string{"identity": payee.Account, "identity_type": "ALIPAY_LOGON_ID", "name": payee.Name}
	}
	values, err := alipayParameters(cfg, method, biz)
	if err != nil {
		return nil, err
	}
	if err := alipaySign(cfg, values); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, AlipayGatewayURL(cfg.Environment), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, fmt.Errorf("构造转账请求失败")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	response, err := (&http.Client{Timeout: 12 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("转账渠道网络异常，请按原单号查询")
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("转账响应异常，请按原单号查询")
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("转账渠道响应异常，状态码%d，请按原单号查询", response.StatusCode)
	}
	var envelope map[string]json.RawMessage
	if json.Unmarshal(raw, &envelope) != nil {
		return nil, fmt.Errorf("转账响应格式错误")
	}
	node := envelope[strings.ReplaceAll(method, ".", "_")+"_response"]
	var signature string
	if len(node) == 0 || json.Unmarshal(envelope["sign"], &signature) != nil {
		return nil, fmt.Errorf("转账响应缺少验签信息")
	}
	if err := alipayVerify(cfg, node, signature); err != nil {
		return nil, err
	}
	var result AlipayTransferResult
	if json.Unmarshal(node, &result) != nil {
		return nil, fmt.Errorf("转账结果格式错误")
	}
	return &result, nil
}
