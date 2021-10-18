package defs

type SignedUp struct {
	Success   bool   `json:"success"`
	SessionID string `json:"session_id"`
}
