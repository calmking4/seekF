package user

import userservice "seekF-backend/internal/services/user_service"

type ContactController struct {
	contactService userservice.ContactService
}

func NewContactController(contactService userservice.ContactService) *ContactController {
	return &ContactController{
		contactService: contactService,
	}
}
