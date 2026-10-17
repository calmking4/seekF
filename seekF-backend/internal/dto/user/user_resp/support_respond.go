package userresp

// SupportCreatorRespond 创作者助力配置及当前查看者的权益，不包含支付凭据。
type SupportCreatorRespond struct {
	CreatorId  string `json:"creator_id"`
	Activated  bool   `json:"activated"`
	PriceYuan  int64  `json:"price_yuan"`
	HasSupport bool   `json:"has_support"`
	ExpiresAt  string `json:"expires_at"`
}
