package userreq

type AVData struct {
	MessageId string        `json:"message_id"` // PROXY (需要存储) 或其他值
	Type      string        `json:"type"`       // 信令类型: start_call, accept_call, reject_call, offer, answer, ice_candidate, end_call
	Sdp       *SdpData      `json:"sdp,omitempty"`
	Candidate *IceCandidate `json:"candidate,omitempty"`
}

type SdpData struct {
	Type string `json:"type"` // offer 或 answer
	Sdp  string `json:"sdp"`
}

type IceCandidate struct {
	Candidate     string `json:"candidate"`
	SdpMLineIndex int    `json:"sdpMLineIndex"`
	SdpMid        string `json:"sdpMid"`
}
