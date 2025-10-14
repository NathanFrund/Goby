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

// UploadFileRequest defines the DTO for the file upload endpoint.
type UploadFileRequest struct {
	File *multipart.FileHeader `form:"file" validate:"required"`
	// You can easily add more fields here, e.g.:
	// Description string `form:"description" validate:"max=500"`
}
