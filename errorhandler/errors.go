package errorhandler

import "net/http"

// 预定义错误
var (
	ErrBadRequest = HTTPError{
		Code:    "BAD_REQUEST",
		Message: "Request format is invalid",
		Status:  http.StatusBadRequest,
	}

	ErrUnauthorized = HTTPError{
		Code:    "UNAUTHORIZED",
		Message: "Authentication required",
		Status:  http.StatusUnauthorized,
	}

	ErrForbidden = HTTPError{
		Code:    "FORBIDDEN",
		Message: "Access denied",
		Status:  http.StatusForbidden,
	}

	ErrNotFound = HTTPError{
		Code:    "NOT_FOUND",
		Message: "Resource not found",
		Status:  http.StatusNotFound,
	}

	ErrInternalServer = HTTPError{
		Code:    "INTERNAL_SERVER_ERROR",
		Message: "Internal server error occurred",
		Status:  http.StatusInternalServerError,
	}

	ErrDatabase = HTTPError{
		Code:    "DATABASE_ERROR",
		Message: "Database operation failed",
		Status:  http.StatusInternalServerError,
	}

	ErrValidation = HTTPError{
		Code:    "VALIDATION_ERROR",
		Message: "Validation failed",
		Status:  http.StatusBadRequest,
	}
)
