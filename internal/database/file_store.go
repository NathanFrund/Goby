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
		return nil, ErrNotFound
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

// FindByUser retrieves file metadata records for a given user, ordered by creation date (newest first).
//
// Parameters:
//   - ctx: The context for the database operation
//   - userID: The ID of the user whose files to retrieve
//   - limit: Maximum number of files to return (0 or negative for no limit)
//   - offset: Number of records to skip before starting to return files (for pagination)
//
// Returns:
//   - []*domain.File: A slice of file pointers matching the criteria
//   - int64: Total number of files available (not just the returned slice)
//   - error: Any error that occurred during the operation
//
// Example usage:
//
//	// Get all files without pagination
//	files, total, err := fileStore.FindByUser(ctx, userID, 0, 0)
//
//	// Get paginated results (first page, 10 items)
//	files, total, err := fileStore.FindByUser(ctx, userID, 10, 0)
//
//	// Get next page
//	files, total, err = fileStore.FindByUser(ctx, userID, 10, 10)
//
// Notes:
// - Files are always ordered by created_at in descending order (newest first)
// - If no files are found, returns an empty slice with total=0 and nil error
// - When limit <= 0, all matching files are returned (no pagination)
// - The total count reflects all files for the user, not just the returned slice
func (s *FileStore) FindByUser(ctx context.Context, userID *surrealmodels.RecordID, limit, offset int) ([]*domain.File, int64, error) {
	if userID == nil {
		return nil, 0, NewDBError(ErrInvalidInput, "user ID is required")
	}

	// Define a local struct to unmarshal the query result, which includes the total count.
	// The `total` field from the subquery will be an array with one object: `[{ "count": 5 }]`.
	type fileWithTotal struct {
		domain.File
		Total []struct {
			Count int64 `json:"count"`
		} `json:"total"`
	}

	// Use a single query to fetch both the paginated data and the total count.
	// The outer SELECT handles pagination, and the inner subquery provides the total count.
	query := `
		SELECT
			*,
			(SELECT count() FROM file WHERE user_id = $userID GROUP ALL) AS total
		FROM file WHERE user_id = $userID
		ORDER BY created_at DESC
	`
	vars := map[string]any{"userID": userID}

	// Add pagination only if limit > 0
	if limit > 0 {
		query += " LIMIT $limit START $offset"
		vars["limit"] = limit
		vars["offset"] = offset
	}

	// Since this query has a custom return shape (with the 'total' field), we create a
	// temporary client typed to our local struct. This is a clean way to handle
	// one-off query shapes without modifying the store's primary client.
	// We can reuse the underlying connection from the main client.
	tempClient, err := NewClient[fileWithTotal](s.client.(*client[domain.File]).conn)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create temporary client for user files query: %w", err)
	}

	// Execute the query using the temporary client.
	results, err := tempClient.Query(ctx, query, vars)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query for user files with count: %w", err)
	}

	// Extract total count - it's always on the first record if results exist
	var total int64
	if len(results) > 0 {
		// The total count is returned on the first record of the subquery result.
		total = results[0].Total[0].Count
	} else {
		// When offset is beyond total count, we still need to get the total.
		// Run a separate count query to get the total count.
		countQuery := "SELECT count() FROM file WHERE user_id = $userID GROUP ALL"
		countVars := map[string]any{"userID": userID}
		type countResult struct {
			Count int64 `json:"count"`
		}
		countClient, err := NewClient[countResult](s.client.(*client[domain.File]).conn)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create count client: %w", err)
		}
		countResults, err := countClient.Query(ctx, countQuery, countVars)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to query for total count: %w", err)
		}
		if len(countResults) > 0 {
			total = countResults[0].Count
		}
		return []*domain.File{}, total, nil
	}

	filePtrs := make([]*domain.File, 0, len(results))
	for i := range results {
		filePtrs = append(filePtrs, &results[i].File)
	}

	return filePtrs, total, nil
}
