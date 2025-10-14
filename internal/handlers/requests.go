package handlers

import (
	"mime/multipart"

	"github.com/go-playground/validator/v10"
)

// CustomValidator wraps the go-playground/validator library to implement Echo's Validator interface.
type CustomValidator struct {
	validator *validator.Validate
}

// NewValidator creates a new CustomValidator.
func NewValidator() *CustomValidator {
	return &CustomValidator{validator: validator.New()}
}

// Validate implements the echo.Validator interface.
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// PaginationParams defines the pagination parameters for list endpoints.
type PaginationParams struct {
	Page     int `query:"page" validate:"min=1"`
	PageSize int `query:"page_size" validate:"min=1,max=100"`
}

// DefaultPagination returns the default pagination parameters.
func DefaultPagination() PaginationParams {
	return PaginationParams{
		Page:     1,
		PageSize: 20,
	}
}

// Offset returns the calculated offset for the current page and page size.
func (p PaginationParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// UploadFileRequest defines the DTO for the file upload endpoint.
type UploadFileRequest struct {
	File *multipart.FileHeader `form:"file" validate:"required"`
	// You can easily add more fields here, e.g.:
	// Description string `form:"description" validate:"max=500"`
}

// PaginatedResponse represents a paginated API response.
type PaginatedResponse[T any] struct {
	Data       []T `json:"data"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalPages int `json:"total_pages"`
}

// NewPaginatedResponse creates a new paginated response.
func NewPaginatedResponse[T any](data []T, total, page, pageSize int) *PaginatedResponse[T] {
	totalPages := (total + pageSize - 1) / pageSize // Ceiling division
	return &PaginatedResponse[T]{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}
