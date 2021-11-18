package entity

import "net/http"

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
	ErrorDBError = ErrResponse{
		HttpSc: 500,
		Error: Err{
			Error:     "Internal DB Error",
			ErrorCode: "003",
		},
	}
	ErrorInternalFaults = ErrResponse{
		HttpSc: 500,
		Error: Err{
			Error:     "Error Internal Faults",
			ErrorCode: "004",
		},
	}
	ErrorMethodError = ErrResponse{
		HttpSc: 405,
		Error: Err{
			Error:     "Method not allowed",
			ErrorCode: "005",
		},
	}
	ErrorNotAuthUser = ErrResponse{
		HttpSc: http.StatusUnauthorized,
		Error: Err{
			Error:     "Username not compatible,check your name and password.",
			ErrorCode: "006",
		},
	}
	ErrorTest = ErrResponse{
		HttpSc: http.StatusFound,
		Error: Err{
			Error:     "Username not compatible,check your name and password.",
			ErrorCode: "006",
		},
	}
)
