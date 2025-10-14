package handlers

import (
	"fmt"
	"time"

	"github.com/nfrund/goby/internal/domain"
)

// ErrorResponse is the standard format for API error responses.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// FileResponse is the DTO for a single file.
// We can use this to control which fields are exposed in the API.
// It also includes a generated URL for client convenience.
type FileResponse struct {
	ID          string    `json:"id"`
	Filename    string    `json:"filename"`
	MIMEType    string    `json:"mime_type"`
	Size        int64     `json:"size"`
	DownloadURL string    `json:"download_url"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewFileResponse creates a new FileResponse DTO from a domain.File model.
func NewFileResponse(file *domain.File) *FileResponse {
	return &FileResponse{
		ID:          file.ID.String(),
		Filename:    file.Filename,
		MIMEType:    file.MIMEType,
		Size:        file.Size,
		DownloadURL: fmt.Sprintf("/app/files/%s/download", file.ID.String()),
		CreatedAt:   file.CreatedAt.Time,
	}
}
