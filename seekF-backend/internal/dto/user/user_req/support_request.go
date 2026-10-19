package userreq

// SupportOrderRequest 助力下单、查询及分页请求，金额完全由服务端确定。
type SupportOrderRequest struct {
	CreatorId  string `json:"creator_id"`
	RequestKey string `json:"request_key"`
	OrderNo    string `json:"order_no"`
	Page       int    `json:"page"`
}

// SupportCreatorRequest 查询创作者助力设置，未指定时查询本人。
type SupportCreatorRequest struct {
	CreatorId string `json:"creator_id" binding:"max=20"`
}

// SaveSupportActivateRequest 开通助力或改价，价格仅接受5至100整数元。
type SaveSupportActivateRequest struct {
	PriceYuan int `json:"price_yuan" binding:"required,min=5,max=100"`
}
