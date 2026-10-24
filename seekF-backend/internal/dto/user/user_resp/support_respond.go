package userresp

// SupportCreatorRespond 创作者助力配置及当前查看者的权益，不包含支付凭据。
type SupportCreatorRespond struct {
	PlatformRateBPS         int    `json:"platform_rate_bps"`
	CreatorRateBPS          int    `json:"creator_rate_bps"`
	TermsVersion            string `json:"terms_version"`
	RequiresTermsAcceptance bool   `json:"requires_terms_acceptance"`
	CreatorId               string `json:"creator_id"`
	Nickname                string `json:"nickname"`
	Avatar                  string `json:"avatar"`
	Activated               bool   `json:"activated"`
	PriceYuan               int64  `json:"price_yuan"`
	HasSupport              bool   `json:"has_support"`
	ExpiresAt               string `json:"expires_at"`
}
