package errorhandler

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Error 返回错误消息
func (e HTTPError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// SendErrorResponse 发送错误响应
func SendErrorResponse(w http.ResponseWriter, err HTTPError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)

	response := ErrorResponse{
		Error: Error{
			Code:    err.Code,
			Message: err.Message,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// 如果JSON编码失败，返回简单文本响应
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// SendCustomErrorResponse 发送自定义错误响应
func SendCustomErrorResponse(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := ErrorResponse{
		Error: Error{
			Code:    code,
			Message: message,
			Status:  status,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// WrapError 包装错误信息
func WrapError(err error, code string, status int) HTTPError {
	return HTTPError{
		Code:    code,
		Message: err.Error(),
		Status:  status,
	}
}

// ValidationError 创建验证错误
func ValidationError(field, message string) HTTPError {
	return HTTPError{
		Code:    "VALIDATION_ERROR",
		Message: fmt.Sprintf("Field '%s' %s", field, message),
		Status:  http.StatusBadRequest,
	}
}
