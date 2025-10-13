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

// Delete handles the deletion of a file by its ID.
func (h *FileHandler) Delete(c echo.Context) error {
	ctx := c.Request().Context()
	logger := middleware.FromContext(ctx)

	fileID := c.Param("id")
	if fileID == "" {
		return c.String(http.StatusBadRequest, "File ID is required")
	}

	// In a real app, we would also get the current user's ID to ensure
	// they have permission to delete this file.
	userID, ok := c.Get("userID").(*surrealmodels.RecordID)
	if !ok || userID == nil {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	// 1. Get the file metadata to find its storage path and verify ownership.
	file, err := h.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		logger.Warn("Failed to get file for deletion", slog.String("fileID", fileID), slog.String("error", err.Error()))
		return c.String(http.StatusNotFound, "File not found")
	}

	// 2. Authorization check: Ensure the current user owns the file.
	if file.UserID == nil || file.UserID.String() != userID.String() {
		logger.Warn("User attempted to delete a file they don't own",
			slog.String("userID", userID.String()),
			slog.String("fileID", fileID),
			slog.String("ownerID", file.UserID.String()))
		return c.String(http.StatusForbidden, "You do not have permission to delete this file")
	}

	// 3. Delete the physical file from storage.
	if err := h.store.Delete(ctx, file.StoragePath); err != nil {
		logger.Error("Failed to delete physical file from storage", slog.String("path", file.StoragePath), slog.String("error", err.Error()))
		// We continue, to at least remove the database record.
	}

	// 4. Delete the metadata record from the database.
	if err := h.fileRepo.Delete(ctx, fileID); err != nil {
		logger.Error("Failed to delete file metadata from database", slog.String("fileID", fileID), slog.String("error", err.Error()))
		return c.String(http.StatusInternalServerError, "Failed to delete file metadata")
	}

	return c.NoContent(http.StatusNoContent)
}
