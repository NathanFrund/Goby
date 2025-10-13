package filestore

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/storage"
	surrealmodels "github.com/surrealdb/surrealdb.go/pkg/models"
)

// Service defines the interface for file management operations.
type Service interface {
	// UploadFile handles saving file content and creating its metadata record.
	UploadFile(ctx context.Context, userID, originalFilename, mimeType string, content io.Reader) (*domain.File, error)
	// GetFile retrieves the file metadata and a reader for the file content.
	GetFile(ctx context.Context, fileID string) (*domain.File, io.ReadCloser, error)
}

// serviceImpl implements the Service interface.
type serviceImpl struct {
	repo    domain.FileRepository
	storage storage.FileStorage
}

// NewService creates a new file service.
func NewService(repo domain.FileRepository, storage storage.FileStorage) Service {
	return &serviceImpl{
		repo:    repo,
		storage: storage,
	}
}

// UploadFile orchestrates saving a file and its metadata.
func (s *serviceImpl) UploadFile(ctx context.Context, userID, originalFilename, mimeType string, content io.Reader) (*domain.File, error) {
	// 1. Generate a unique path for the file to prevent collisions.
	uniqueFilename := fmt.Sprintf("%s%s", uuid.NewString(), filepath.Ext(originalFilename))
	storagePath := filepath.Join("uploads", userID, uniqueFilename) // Example path structure

	// 2. Save the file content using the storage service.
	bytesWritten, err := s.storage.Save(ctx, storagePath, content)
	if err != nil {
		return nil, fmt.Errorf("failed to save file content: %w", err)
	}

	// Split the UserID string "table:id" into its parts for the NewRecordID constructor.
	parts := strings.SplitN(userID, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid user ID format: expected 'table:id', got '%s'", userID)
	}
	userIDRecord := surrealmodels.NewRecordID(parts[0], parts[1])

	// 3. Create the metadata record in the database.
	fileMetadata := &domain.File{
		UserID:      &userIDRecord,
		Filename:    originalFilename,
		MimeType:    mimeType,
		SizeBytes:   bytesWritten,
		StoragePath: storagePath,
	}

	return s.repo.Create(ctx, fileMetadata)
}

// GetFile is a placeholder for file retrieval logic.
func (s *serviceImpl) GetFile(ctx context.Context, fileID string) (*domain.File, io.ReadCloser, error) {
	return nil, nil, fmt.Errorf("not implemented")
}
