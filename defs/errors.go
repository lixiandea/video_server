package defs

type Err struct {
	Error     string `json:"error"`
	ErrorCode string `json:"error_code"`
}

type ErrResponse struct {
	HttpSc int
	Error  Err
}

var (
	ErrorRequestBodyParseFailed = ErrResponse{
		HttpSc: 400,
		Error: Err{
			Error:     "Request Body Parse Failed",
			ErrorCode: "001",
		},
	}
	ErrorUserAuthenticationFailed = ErrResponse{
		HttpSc: 401,
		Error: Err{
			Error:     "User auth failed",
			ErrorCode: "002",
		},
	}
)
