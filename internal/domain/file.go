package domain

import (
	"context"

	surrealmodels "github.com/surrealdb/surrealdb.go/pkg/models"
)

// File represents the metadata for a stored file.
// The actual file content is stored on a filesystem (e.g., local disk, S3)
// and referenced by the StoragePath.
type File struct {
	ID          *surrealmodels.RecordID       `json:"id,omitempty" surrealdb:"id,omitempty"`                 // Unique identifier for the file record.
	UserID      *surrealmodels.RecordID       `json:"user_id,omitempty" surrealdb:"user_id,omitempty"`       // The user who owns the file.
	Filename    string                        `json:"filename" surrealdb:"filename"`                         // Original name of the file at the time of upload.
	MIMEType    string                        `json:"mime_type" surrealdb:"mime_type"`                       // MIME type of the file (e.g., "image/jpeg").
	Size        int64                         `json:"size" surrealdb:"size"`                                 // Size of the file in bytes.
	StoragePath string                        `json:"storage_path" surrealdb:"storage_path"`                 // The path to the file in the configured storage backend.
	CreatedAt   *surrealmodels.CustomDateTime `json:"created_at,omitempty" surrealdb:"created_at,omitempty"` // Timestamp of when the record was created.
	UpdatedAt   *surrealmodels.CustomDateTime `json:"updated_at,omitempty" surrealdb:"updated_at,omitempty"` // Timestamp of the last update.
	DeletedAt   *surrealmodels.CustomDateTime `json:"deleted_at,omitempty" surrealdb:"deleted_at,omitempty"`
}

// FileRepository defines the interface for interacting with file metadata storage.
type FileRepository interface {
	// Create inserts a new file metadata record.
	Create(ctx context.Context, file *File) (*File, error)

	// Update modifies an existing file metadata record.
	Update(ctx context.Context, file *File) (*File, error)

	// Delete removes a file metadata record.
	DeleteByID(ctx context.Context, fileID string) error

	// GetByID retrieves file metadata by its unique ID.
	FindByID(ctx context.Context, fileID string) (*File, error)

	// FindLatestByUser retrieves the most recently created file for a given user.
	FindLatestByUser(ctx context.Context, userID *surrealmodels.RecordID) (*File, error)

	// FindByUser retrieves all file metadata records for a given user.
	FindByUser(ctx context.Context, userID *surrealmodels.RecordID) ([]*File, error)

	// FindByStoragePath retrieves file metadata by its storage path.
	FindByStoragePath(ctx context.Context, storagePath string) (*File, error)
}
