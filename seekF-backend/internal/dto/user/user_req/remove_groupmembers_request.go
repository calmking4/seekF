package userreq

type RemoveGroupMembersRequest struct {
	GroupId  string   `json:"group_id"`
	UuidList []string `json:"uuid_list"`
}
