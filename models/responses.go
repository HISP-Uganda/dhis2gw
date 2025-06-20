package models

import "time"

type ErrorResponse struct {
	Error  string      `json:"error" example:"Invalid JSON"`
	Detail interface{} `json:"detail,omitempty"`
}

type UserTokenResponse struct {
	Message string    `json:"message" example:"Token created successfully"`
	Token   string    `json:"token" example:"abc123xyzTOKEN"`
	Expires time.Time `json:"expires" example:"2026-06-20T10:00:00Z"`
}

type SuccessResponse struct {
	Message string `json:"message" example:"User updated successfully"`
}
