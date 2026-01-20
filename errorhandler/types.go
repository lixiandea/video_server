package errorhandler

// Error 定义基础错误结构
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

// ErrorResponse 定义错误响应结构
type ErrorResponse struct {
	Error Error `json:"error"`
}

// HTTPError 实现 error 接口
type HTTPError struct {
	Code    string
	Message string
	Status  int
}
