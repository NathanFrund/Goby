package storage

import (
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/middleware"
	surrealmodels "github.com/surrealdb/surrealdb.go/pkg/models"
)

// FileHandler handles HTTP requests related to files.
type FileHandler struct {
	store    Store
	fileRepo domain.FileRepository
}

// NewFileHandler creates a new FileHandler.
func NewFileHandler(s Store, fr domain.FileRepository) *FileHandler {
	return &FileHandler{
		store:    s,
		fileRepo: fr,
	}
}

// Upload handles file uploads from a multipart form.
func (h *FileHandler) Upload(c echo.Context) error {
	ctx := c.Request().Context()
	logger := middleware.FromContext(ctx)

	// For this example, we'll assume the user is authenticated and their ID is available.
	// In a real app, this would come from the session or a JWT.
	// Using a placeholder until auth is integrated here.
	userID, ok := c.Get("userID").(*surrealmodels.RecordID)
	if !ok || userID == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid file upload request")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to open uploaded file")
	}
	defer src.Close()

	// Create a unique storage path.
	storagePath := filepath.Join("users", userID.String(), fmt.Sprintf("%d-%s", time.Now().UnixNano(), fileHeader.Filename))

	bytesWritten, err := h.store.Save(ctx, storagePath, src)
	if err != nil {
		logger.Error("Failed to save file to storage", slog.String("error", err.Error()))
		return c.String(http.StatusInternalServerError, "Failed to save file")
	}

	// Save metadata to the database.
	fileMetadata := &domain.File{
		UserID:      userID,
		Filename:    fileHeader.Filename,
		MimeType:    fileHeader.Header.Get("Content-Type"),
		SizeBytes:   bytesWritten,
		StoragePath: storagePath,
	}

	if _, err := h.fileRepo.Create(ctx, fileMetadata); err != nil {
		logger.Error("Failed to save file metadata", slog.String("error", err.Error()))
		// Attempt to clean up the stored file if metadata saving fails.
		_ = h.store.Delete(ctx, storagePath)
		return c.String(http.StatusInternalServerError, "Failed to save file metadata")
	}

	return c.String(http.StatusOK, fmt.Sprintf("File %s uploaded successfully.", fileHeader.Filename))
}
