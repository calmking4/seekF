package payment

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/shopspring/decimal"
	"golang.org/x/text/encoding/simplifiedchinese"

	"seekF-backend/internal/configs"
	"seekF-backend/internal/pkg/constants"
)

// AlipayBillRefund 下载验签地址对应的渠道账单并核对当日退款。
func AlipayBillRefund(ctx context.Context, cfg configs.AlipayConfig, day time.Time, orderNo, tradeNo string) (decimal.Decimal, error) {
	result, err := alipayRequest(ctx, cfg, "alipay.data.dataservice.bill.downloadurl.query", map[string]string{"bill_type": "trade", "bill_date": day.Format("2006-01-02")})
	if err != nil {
		return decimal.Zero, err
	}
	if result.Code != "10000" {
		return decimal.Zero, fmt.Errorf("支付宝交易账单尚不可用，业务代码%s", result.SubCode)
	}
	// 只有已验签的明确空账单才能作为零退款，不把账单缺失或生成延迟当作零金额。
	if result.BillFileCode == "EMPTY_DATA_WITH_BILL_FILE" {
		return decimal.Zero, nil
	}
	u, err := url.Parse(result.BillURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || !alipayBillHost(u.Hostname()) {
		return decimal.Zero, fmt.Errorf("支付宝账单下载地址无效")
	}
	// 官方历史响应可能返回HTTP下载地址，统一升级为HTTPS后读取资金明细。
	u.Scheme = "https"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return decimal.Zero, fmt.Errorf("构造支付宝账单请求失败")
	}
	client := &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return fmt.Errorf("支付宝账单下载不允许跳转")
	}}
	response, err := client.Do(req)
	if err != nil {
		return decimal.Zero, fmt.Errorf("下载支付宝账单失败")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return decimal.Zero, fmt.Errorf("下载支付宝账单响应异常，状态码%d", response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, constants.SupportBillMaxBytes+1))
	if err != nil || len(raw) > constants.SupportBillMaxBytes {
		return decimal.Zero, fmt.Errorf("支付宝账单读取失败或超过20MiB限制")
	}
	return parseAlipayBillRefund(raw, orderNo, tradeNo)
}

// alipayBillHost 限定支付宝官方账单下载域名。
func alipayBillHost(host string) bool {
	return strings.HasSuffix(host, ".alipay.com") || strings.HasSuffix(host, ".alipaydev.com") || strings.HasSuffix(host, ".alipayobjects.com")
}

// parseAlipayBillRefund 解析ZIP内GBK或UTF8账单并核对订单退款。
func parseAlipayBillRefund(raw []byte, orderNo, tradeNo string) (decimal.Decimal, error) {
	archive, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return decimal.Zero, fmt.Errorf("支付宝账单压缩格式错误")
	}
	total := decimal.Zero
	found := false
	var expanded int64
	for _, file := range archive.File {
		if !strings.HasSuffix(strings.ToLower(file.Name), ".csv") || strings.Contains(file.Name, "汇总") {
			continue
		}
		stream, err := file.Open()
		if err != nil {
			return decimal.Zero, fmt.Errorf("读取支付宝账单明细失败")
		}
		data, readErr := io.ReadAll(io.LimitReader(stream, constants.SupportBillMaxBytes+1))
		stream.Close()
		expanded += int64(len(data))
		if readErr != nil || expanded > constants.SupportBillMaxBytes {
			return decimal.Zero, fmt.Errorf("支付宝账单解压内容超过20MiB限制")
		}
		if !utf8.Valid(data) {
			data, err = simplifiedchinese.GBK.NewDecoder().Bytes(data)
			if err != nil {
				return decimal.Zero, fmt.Errorf("支付宝账单字符编码错误")
			}
		}
		reader := csv.NewReader(bytes.NewReader(data))
		reader.FieldsPerRecord = -1
		reader.TrimLeadingSpace = true
		reader.LazyQuotes = true
		orderIdx, tradeIdx, kindIdx, amountIdx := -1, -1, -1, -1
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return decimal.Zero, fmt.Errorf("支付宝账单CSV格式错误")
			}
			if orderIdx < 0 {
				for i, value := range record {
					value = strings.TrimSpace(strings.TrimPrefix(value, "\ufeff"))
					switch value {
					case "商户订单号":
						orderIdx = i
					case "支付宝交易号":
						tradeIdx = i
					case "业务类型":
						kindIdx = i
					case "订单金额（元）", "订单金额(元)":
						amountIdx = i
					}
				}
				if orderIdx >= 0 {
					if tradeIdx < 0 || kindIdx < 0 || amountIdx < 0 {
						return decimal.Zero, fmt.Errorf("支付宝账单缺少退款核对字段")
					}
					found = true
				}
				continue
			}
			if len(record) <= orderIdx || strings.TrimSpace(record[orderIdx]) != orderNo {
				continue
			}
			if len(record) <= tradeIdx || len(record) <= kindIdx || len(record) <= amountIdx || strings.TrimSpace(record[tradeIdx]) != tradeNo {
				return decimal.Zero, fmt.Errorf("支付宝账单订单交易号或字段不匹配")
			}
			kind := strings.TrimSpace(record[kindIdx])
			if kind == "交易" {
				continue
			}
			if kind != "退款" {
				return decimal.Zero, fmt.Errorf("支付宝账单出现未知业务类型")
			}
			amount, err := ParseAlipayAmount(strings.TrimPrefix(strings.TrimSpace(record[amountIdx]), "-"))
			if err != nil {
				return decimal.Zero, err
			}
			total = total.Add(amount)
		}
	}
	if !found {
		return decimal.Zero, fmt.Errorf("支付宝账单缺少业务明细，无法确认退款金额")
	}
	return total, nil
}
