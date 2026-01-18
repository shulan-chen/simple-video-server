package web

type ApiBody struct {
	Url     string `json:"url"`
	Method  string `json:"method"`
	ReqBody string `json:"req_body"`
}

type Err struct {
	Error     string `json:"error"`
	ErrorCode string `json:"error_code"`
}

type ErrResponse struct {
	HttpSC int
	Error  Err
}

var (
	ErrorRequestNotRecognized = ErrResponse{
		HttpSC: 400,
		Error: Err{
			Error:     "Request Not Recognized",
			ErrorCode: "001",
		},
	}
	ErrorRequestBodyParseFailed = ErrResponse{
		HttpSC: 400,
		Error: Err{
			Error:     "Request body is not correct",
			ErrorCode: "002",
		},
	}
	ErrorInternalFaults = ErrResponse{
		HttpSC: 500,
		Error: Err{
			Error:     "Internal service error",
			ErrorCode: "003",
		},
	}
)
