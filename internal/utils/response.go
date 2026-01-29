// Package utils provides utility functions for the video server
package utils

import (
    "encoding/json"
    "net/http"
)

// Response represents a standard API response
type Response struct {
    Success bool        `json:"success"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

// SendSuccessResponse sends a success response
func SendSuccessResponse(w http.ResponseWriter, data interface{}, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(Response{
        Success: true,
        Message: message,
        Data:    data,
    })
}

// SendErrorResponse sends an error response
func SendErrorResponse(w http.ResponseWriter, errorMsg string, statusCode int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(Response{
        Success: false,
        Error:   errorMsg,
    })
}