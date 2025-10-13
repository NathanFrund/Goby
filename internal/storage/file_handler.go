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

// getUserFromContext is a helper to retrieve the authenticated user from the context.
func getUserFromContext(c echo.Context) (*domain.User, error) {
	user, ok := c.Get("user").(*domain.User)
	if !ok || user == nil || user.ID == nil {
		return nil, echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}
	return user, nil
}

// Upload handles file uploads from a multipart form.
func (h *FileHandler) Upload(c echo.Context) error {
	ctx := c.Request().Context()
	logger := middleware.FromContext(ctx)

	// Retrieve the authenticated user from the context, set by the Auth middleware.
	user, err := getUserFromContext(c)
	if err != nil {
		return err
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
	// Sanitize the filename to prevent path traversal attacks.
	sanitizedFilename := filepath.Base(fileHeader.Filename)
	storagePath := filepath.Join("users", user.ID.String(), fmt.Sprintf("%d-%s", time.Now().UnixNano(), sanitizedFilename))

	bytesWritten, err := h.store.Save(ctx, storagePath, src)
	if err != nil {
		logger.Error("Failed to save file to storage", slog.String("error", err.Error()))
		return c.String(http.StatusInternalServerError, "Failed to save file")
	}

	// Save metadata to the database.
	fileMetadata := &domain.File{
		UserID:      user.ID,
		Filename:    sanitizedFilename,
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

	// Retrieve the authenticated user from the context.
	user, err := getUserFromContext(c)
	if err != nil {
		return err
	}

	// 1. Get the file metadata to find its storage path and verify ownership.
	file, err := h.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		logger.Warn("Failed to get file for deletion", slog.String("fileID", fileID), slog.String("error", err.Error()))
		return c.String(http.StatusNotFound, "File not found")
	}

	// 2. Authorization check: Ensure the current user owns the file.
	if file.UserID == nil || file.UserID.String() != user.ID.String() {
		logger.Warn("User attempted to delete a file they don't own",
			slog.String("userID", user.ID.String()),
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

// Download handles serving a file's content.
func (h *FileHandler) Download(c echo.Context) error {
	ctx := c.Request().Context()
	logger := middleware.FromContext(ctx)

	fileID := c.Param("id")
	if fileID == "" {
		return c.String(http.StatusBadRequest, "File ID is required")
	}

	// Retrieve the authenticated user from the context.
	user, err := getUserFromContext(c)
	if err != nil {
		return err
	}

	// 1. Get the file metadata to verify ownership and get content type.
	file, err := h.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		logger.Warn("Failed to get file for download", slog.String("fileID", fileID), slog.String("error", err.Error()))
		return c.String(http.StatusNotFound, "File not found")
	}

	// 2. Authorization check: Ensure the current user owns the file.
	if file.UserID == nil || file.UserID.String() != user.ID.String() {
		logger.Warn("User attempted to download a file they don't own",
			slog.String("userID", user.ID.String()),
			slog.String("fileID", fileID),
			slog.String("ownerID", file.UserID.String()))
		return c.String(http.StatusForbidden, "You do not have permission to download this file")
	}

	// 3. Get the file content from storage.
	content, err := h.store.Get(ctx, file.StoragePath)
	if err != nil {
		logger.Error("Failed to get physical file from storage", slog.String("path", file.StoragePath), slog.String("error", err.Error()))
		return c.String(http.StatusInternalServerError, "Could not retrieve file")
	}
	defer content.Close()

	return c.Stream(http.StatusOK, file.MimeType, content)
}
