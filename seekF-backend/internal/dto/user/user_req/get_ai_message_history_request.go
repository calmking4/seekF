package userreq

type GetAIMessageHistoryRequest struct {
	SessionId string `json:"session_id" binding:"required"`
	PageSize  int    `json:"page_size"`
	Cursor    string `json:"cursor"`    // 游标：最后一条消息的时间戳，为空则获取最新消息
	Direction string `json:"direction"` // prev=向前翻页(更旧), next=向后翻页(更新)
}
