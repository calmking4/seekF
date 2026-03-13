package userservice

import userdao "seekF-backend/internal/dao/user_dao"

type ContactService interface {
}

type ContactServiceImpl struct {
	contactDAO userdao.ContactDAO
	sessionDAO userdao.SessionDAO
}

func NewContactService(contactDAO userdao.ContactDAO, sessionDAO userdao.SessionDAO) ContactService {
	return &ContactServiceImpl{
		contactDAO: contactDAO,
		sessionDAO: sessionDAO,
	}
}
