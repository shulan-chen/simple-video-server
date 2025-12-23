package api

type User struct {
	Id       int    `json:id`
	Username string `json:name`
	Password string `json:pasword`
	IsVaild  int    `json:isVaild`
	//CreatAt  int64  `json:createAt`
	//UpdateAt int64  `json:updateAt`
}

type SignedUP struct {
	Success   bool   `json:sueecess`
	SessionId string `json:session_id`
}
