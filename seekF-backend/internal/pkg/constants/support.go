package constants

import "time"

const (
	// 每期助力价格下限，单位元。
	SupportMinPriceYuan = 5
	// 每期助力价格上限，单位元，只允许整数元。
	SupportMaxPriceYuan = 100
	// 默认助力价格，单位分。
	SupportDefaultPriceFen int64 = 1000
	// 每期权益时长，单位小时，固定30天。
	SupportDurationHours = 720
	// 待支付订单有效期，单位分钟。
	SupportOrderTimeout = 15 * time.Minute
	// 支付成功后的退款申请期限，单位小时。
	SupportRefundWindow = 168 * time.Hour
	// 免费视频。
	SupportVideoFree int8 = 0
	// 助力专属视频。
	SupportVideoExclusive int8 = 1
	// 订单等待付款。
	SupportOrderPending int8 = 0
	// 订单已付款。
	SupportOrderPaid int8 = 1
	// 订单已关闭。
	SupportOrderClosed int8 = 2
	// 尚未申请退款。
	SupportRefundNone int8 = 0
	// 退款等待管理员审核。
	SupportRefundPending int8 = 1
	// 退款申请已驳回。
	SupportRefundRejected int8 = 2
	// 退款处理中，含结果未知待查询。
	SupportRefundProcessing int8 = 3
	// 已确认全额退款。
	SupportRefundSucceeded int8 = 4
	// 已确认退款失败，允许按原请求重试。
	SupportRefundFailed int8 = 5
)
