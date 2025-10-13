package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nfrund/goby/internal/domain"
	surrealmodels "github.com/surrealdb/surrealdb.go/pkg/models"
)

const fileTable = "file"

// var _ ensures that FileStore implements the domain.FileRepository interface at compile time.
var _ domain.FileRepository = (*FileStore)(nil)

// FileStore implements operations for managing file metadata in the database.
// It provides a type-safe interface over the underlying database client.
type FileStore struct {
	client Client[domain.File]
}

// NewFileStore creates a new FileStore with the given database client.
func NewFileStore(client Client[domain.File]) *FileStore {
	return &FileStore{client: client}
}

// Create inserts a new file metadata record into the database.
func (s *FileStore) Create(ctx context.Context, file *domain.File) (*domain.File, error) {
	if file == nil {
		return nil, errors.New("file cannot be nil")
	}
	if file.Filename == "" {
		return nil, errors.New("file name is required")
	}
	if file.StoragePath == "" {
		return nil, errors.New("file storage path is required")
	}

	// Set timestamps directly on the struct. The SurrealDB driver will handle
	// marshalling `time.Time` correctly when the struct is passed to `Create`.
	now := time.Now().UTC()
	file.CreatedAt = &surrealmodels.CustomDateTime{Time: now}
	file.UpdatedAt = &surrealmodels.CustomDateTime{Time: now}

	createdFile, err := s.client.Create(ctx, fileTable, file)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	return createdFile, nil
}

// GetByID retrieves file metadata by its unique ID.
func (s *FileStore) GetByID(ctx context.Context, id string) (*domain.File, error) {
	return s.client.Select(ctx, id)
}

// GetByStoragePath retrieves file metadata by its storage path.
func (s *FileStore) GetByStoragePath(ctx context.Context, path string) (*domain.File, error) {
	query := "SELECT * FROM file WHERE storage_path = $path"
	vars := map[string]interface{}{"path": path}

	file, err := s.client.QueryOne(ctx, query, vars)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, errors.New("file not found")
	}

	return file, nil
}

// Update updates an existing file record.
func (s *FileStore) Update(ctx context.Context, file *domain.File) (*domain.File, error) {
	if file == nil || file.ID == nil || file.ID.String() == "" {
		return nil, errors.New("file and file ID are required for update")
	}

	// For updates, it's still best practice to use a map to explicitly control
	// which fields are being modified. The database schema will automatically
	// handle updating the `updated_at` field.
	updateData := map[string]interface{}{
		"filename":   file.Filename,
		"mime_type":  file.MimeType,
		"updated_at": &surrealmodels.CustomDateTime{Time: time.Now().UTC()},
	}

	return s.client.Update(ctx, file.ID.String(), &updateData)
}

// Delete removes a file record from the database.
func (s *FileStore) Delete(ctx context.Context, id string) error {
	return s.client.Delete(ctx, id)
}
