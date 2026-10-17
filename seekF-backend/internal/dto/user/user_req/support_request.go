package userreq

// SupportCreatorRequest 查询创作者助力设置，未指定时查询本人。
type SupportCreatorRequest struct {
	CreatorId string `json:"creator_id" binding:"max=20"`
}

// SaveSupportActivateRequest 开通助力或改价，价格仅接受5至100整数元。
type SaveSupportActivateRequest struct {
	PriceYuan int `json:"price_yuan" binding:"required,min=5,max=100"`
}
