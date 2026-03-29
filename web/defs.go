package web

// ApiBody API请求体结构（用于前端API透传）
type ApiBody struct {
	Url     string `json:"url"`
	Method  string `json:"method"`
	ReqBody string `json:"req_body"`
}

