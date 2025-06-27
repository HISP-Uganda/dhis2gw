package models

import (
	"time"
)

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

type PaginatedResponse[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total" example:"100"`
	Page       int   `json:"page" example:"1"`
	TotalPages int   `json:"total_pages" example:"10"`
	PageSize   int   `json:"page_size" example:"10"`
}

type ImportResponse[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total" example:"100"`
}
