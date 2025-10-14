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
		return nil, errors.New("file to create cannot be nil")
	}
	if err := file.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed for file: %w", err)
	}

	// Instead of passing the struct directly, create a map to ensure correct
	now := time.Now().UTC()
	file.CreatedAt = &surrealmodels.CustomDateTime{Time: now} // This assumes domain.File.CreatedAt is *CustomDateTime
	file.UpdatedAt = &surrealmodels.CustomDateTime{Time: now} // This assumes domain.File.UpdatedAt is *CustomDateTime

	fileData := map[string]interface{}{
		"user_id":      file.UserID,
		"filename":     file.Filename,
		"mime_type":    file.MIMEType,
		"size":         file.Size,
		"storage_path": file.StoragePath,
		"created_at":   file.CreatedAt,
		"updated_at":   file.UpdatedAt,
	}

	createdFile, err := s.client.Create(ctx, fileTable, fileData)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	return createdFile, nil
}

// GetByID retrieves file metadata by its unique ID.
func (s *FileStore) FindByID(ctx context.Context, fileID string) (*domain.File, error) {
	return s.client.Select(ctx, fileID)
}

// FindByStoragePath retrieves file metadata by its storage path.
func (s *FileStore) FindByStoragePath(ctx context.Context, storagePath string) (*domain.File, error) {
	query := "SELECT * FROM file WHERE storage_path = $path"
	vars := map[string]interface{}{"path": storagePath}

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

	if err := file.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed for file update: %w", err)
	}

	// Use a map for updates to explicitly control which fields are modified
	file.UpdatedAt = &surrealmodels.CustomDateTime{Time: time.Now().UTC()}
	updateData := map[string]interface{}{
		"filename":   file.Filename,
		"mime_type":  file.MIMEType,
		"updated_at": file.UpdatedAt,
	}

	return s.client.Update(ctx, file.ID.String(), updateData)
}

// Delete removes a file record from the database.
func (s *FileStore) DeleteByID(ctx context.Context, fileID string) error {
	return s.client.Delete(ctx, fileID)
}

// FindLatestByUser retrieves the most recently created file for a given user from the database.
func (s *FileStore) FindLatestByUser(ctx context.Context, userID *surrealmodels.RecordID) (*domain.File, error) {
	query := "SELECT * FROM file WHERE user_id = $user ORDER BY created_at DESC LIMIT 1"
	vars := map[string]interface{}{"user": userID}

	files, err := s.client.Query(ctx, query, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to query for latest file: %w", err)
	}

	if len(files) == 0 {
		return nil, nil // No file found is not an error
	}

	return &files[0], nil
}

// FindByUser retrieves all file metadata records for a given user, ordered by creation date.
func (s *FileStore) FindByUser(ctx context.Context, userID *surrealmodels.RecordID) ([]*domain.File, error) {
	if userID == nil {
		return nil, NewDBError(ErrInvalidInput, "user ID is required")
	}

	query := "SELECT * FROM file WHERE user_id = $userID ORDER BY created_at DESC"
	files, err := s.client.Query(ctx, query, map[string]any{"userID": userID})
	if err != nil {
		return nil, fmt.Errorf("failed to query for user files: %w", err)
	}

	// Convert the slice of values to a slice of pointers as required by the interface.
	filePtrs := make([]*domain.File, 0, len(files))
	for i := range files {
		filePtrs = append(filePtrs, &files[i])
	}

	return filePtrs, nil
}
