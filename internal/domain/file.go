package domain

import (
	"context"

	surrealmodels "github.com/surrealdb/surrealdb.go/pkg/models"
)

// File represents the metadata for a stored file.
// The actual file content is stored on a filesystem (e.g., local disk, S3)
// and referenced by the StoragePath.
type File struct {
	ID          *surrealmodels.RecordID       `json:"id,omitempty" surrealdb:"id,omitempty"`
	UserID      *surrealmodels.RecordID       `json:"user_id,omitempty" surrealdb:"user_id,omitempty"` // The user who uploaded the file
	Filename    string                        `json:"filename" surrealdb:"filename"`                   // Original name of the file
	MimeType    string                        `json:"mime_type" surrealdb:"mime_type"`                 // MIME type of the file
	SizeBytes   int64                         `json:"size_bytes" surrealdb:"size_bytes"`               // Size of the file in bytes
	StoragePath string                        `json:"storage_path" surrealdb:"storage_path"`           // The path to the file in the storage backend
	CreatedAt   *surrealmodels.CustomDateTime `json:"created_at,omitempty" surrealdb:"created_at,omitempty"`
	UpdatedAt   *surrealmodels.CustomDateTime `json:"updated_at,omitempty" surrealdb:"updated_at,omitempty"`
}

// FileRepository defines the interface for interacting with file metadata storage.
type FileRepository interface {
	// Create inserts a new file metadata record.
	Create(ctx context.Context, file *File) (*File, error)

	// GetByID retrieves file metadata by its unique ID.
	GetByID(ctx context.Context, id string) (*File, error)

	// GetByStoragePath retrieves file metadata by its storage path.
	GetByStoragePath(ctx context.Context, path string) (*File, error)

	// Update modifies an existing file metadata record.
	Update(ctx context.Context, file *File) (*File, error)

	// Delete removes a file metadata record.
	Delete(ctx context.Context, id string) error

	// FindLatestByUser retrieves the most recently created file for a given user.
	FindLatestByUser(ctx context.Context, userID *surrealmodels.RecordID) (*File, error)

	// FindByUser retrieves all file metadata records for a given user.
	FindByUser(ctx context.Context, userID *surrealmodels.RecordID) ([]*File, error)
}
