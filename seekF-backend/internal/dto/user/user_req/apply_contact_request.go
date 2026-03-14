package userreq

type ApplyContactRequest struct {
	ContactId string `json:"contact_id"`
	Message   string `json:"message"`
}
