package constants

import "time"

const (
	// 渠道账单压缩包及解压明细大小上限，单位字节。
	SupportBillMaxBytes = 20 * 1024 * 1024
	// 助力全额退款成功通知。
	NotificationSupportRefund int8 = 8
	// 已付款订单渠道退款账单核对周期，单位小时。
	SupportRefundAuditInterval = 24 * time.Hour

	// 助力付款确认通知，可在我的助力中查看。
	NotificationSupportPaid int8 = 7
	// 持久化订单补偿扫描与失败重试间隔，单位分钟。
	SupportOrderScanInterval = time.Minute
	// 单批领取订单上限，单位笔。
	SupportOrderScanBatch = 20
	// 多实例后台任务领取租约，单位分钟。
	SupportOrderLease = 2 * time.Minute
	// 每笔订单渠道补偿的总超时，单位秒。
	SupportOrderAttemptTimeout = 45 * time.Second
	// 本人订单及权益的分页大小，单位条。
	SupportOrderPageSize = 20
	// 每期助力价格下限，单位元。
	SupportMinPriceYuan = 5
	// 每期助力价格上限，单位元，只允许整数元。
	SupportMaxPriceYuan = 100
	// 默认助力价格，单位元。
	SupportDefaultPriceYuan int64 = 10
	// 每期权益时长，单位小时，固定30天。
	SupportDurationHours = 720
	// 待支付订单有效期，单位分钟。
	SupportOrderTimeout = 15 * time.Minute
	// 支付成功后的退款申请期限，单位小时。
	SupportRefundWindow = 168 * time.Hour
	// 免费作品，适用于图文和视频。
	SupportPostFree int8 = 0
	// 助力专属作品，适用于图文和视频。
	SupportPostExclusive int8 = 1
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
