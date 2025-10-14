package domain

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	surrealmodels "github.com/surrealdb/surrealdb.go/pkg/models"
)

// validatorInstance is a package-level validator instance.
// Using a single instance is more efficient as it caches struct information.
var validatorInstance = validator.New()

// init registers custom validation functions with the validator instance.
func init() {
	// Register the safepath validator to prevent directory traversal attacks.
	_ = validatorInstance.RegisterValidation("safepath", validateSafePath)
}

// validateSafePath ensures the path doesn't contain any directory traversal attempts.
func validateSafePath(fl validator.FieldLevel) bool {
	path := fl.Field().String()

	// Check for common path traversal patterns.
	if strings.Contains(path, "..") ||
		strings.Contains(path, "~") ||
		strings.HasPrefix(path, "/") ||
		strings.Contains(path, "\\") {
		return false
	}

	// Clean the path and check if it still matches the original.
	// This catches more subtle issues like "uploads/./../file".
	return path == filepath.Clean(path)
}

// File represents the metadata for a stored file.
// The actual file content is stored on a filesystem (e.g., local disk, S3)
// and referenced by the StoragePath.
type File struct {
	ID          *surrealmodels.RecordID       `json:"id,omitempty" surrealdb:"id,omitempty"`                               // Unique identifier for the file record.
	UserID      *surrealmodels.RecordID       `json:"user_id,omitempty" surrealdb:"user_id,omitempty" validate:"required"` // The user who owns the file. Must be a valid user record ID.
	Filename    string                        `json:"filename" surrealdb:"filename" validate:"required,min=1,max=255"`     // Original name of the file. Length: 1-255 chars.
	MIMEType    string                        `json:"mime_type" surrealdb:"mime_type" validate:"required"`                 // MIME type of the file. Handler-level validation uses a configurable list.
	Size        int64                         `json:"size" surrealdb:"size" validate:"gte=0"`                              // Size of the file in bytes. Must be non-negative.
	StoragePath string                        `json:"storage_path" surrealdb:"storage_path" validate:"required,safepath"`  // The path to the file in the configured storage backend. Must be a safe, relative path.
	CreatedAt   *surrealmodels.CustomDateTime `json:"created_at,omitempty" surrealdb:"created_at,omitempty"`               // Timestamp of when the record was created.
	UpdatedAt   *surrealmodels.CustomDateTime `json:"updated_at,omitempty" surrealdb:"updated_at,omitempty"`               // Timestamp of the last update.
	DeletedAt   *surrealmodels.CustomDateTime `json:"deleted_at,omitempty" surrealdb:"deleted_at,omitempty"`
}

// Validate runs validation checks on the File struct using the defined tags.
// This ensures that the domain model is always in a valid state.
func (f *File) Validate() error {
	return validatorInstance.Struct(f)
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
	// FindByUser retrieves paginated file metadata records for a given user.
	// Returns the list of files and the total count of files for the user.
	FindByUser(ctx context.Context, userID *surrealmodels.RecordID, limit, offset int) ([]*File, int64, error)

	// FindByStoragePath retrieves file metadata by its storage path.
	FindByStoragePath(ctx context.Context, storagePath string) (*File, error)
}

// Pagination constants
const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)
