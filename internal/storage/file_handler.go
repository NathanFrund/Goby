package storage

import (
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/middleware"
)

// FileHandler handles HTTP requests related to files.
type FileHandler struct {
	fileStore        Store
	fileRepo         domain.FileRepository
	maxFileSize      int64
	allowedMimeTypes map[string]bool
}

// NewFileHandler creates a new FileHandler.
func NewFileHandler(fileStore Store, fileRepo domain.FileRepository, maxFileSize int64, allowedMimeTypes []string) *FileHandler {
	mimeTypesMap := make(map[string]bool)
	for _, mimeType := range allowedMimeTypes {
		mimeTypesMap[strings.TrimSpace(mimeType)] = true
	}

	return &FileHandler{
		fileStore:        fileStore,
		fileRepo:         fileRepo,
		maxFileSize:      maxFileSize,
		allowedMimeTypes: mimeTypesMap,
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

// UploadFile handles file uploads from a multipart form.
func (h *FileHandler) UploadFile(c echo.Context) error {
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

	// Security: Validate file size.
	if h.maxFileSize > 0 && fileHeader.Size > h.maxFileSize {
		return c.String(http.StatusRequestEntityTooLarge, fmt.Sprintf("File size of %d bytes exceeds the limit of %d bytes", fileHeader.Size, h.maxFileSize))
	}

	// Security: Validate MIME type.
	mimeType := fileHeader.Header.Get("Content-Type")
	if len(h.allowedMimeTypes) > 0 && !h.allowedMimeTypes[mimeType] {
		return c.String(http.StatusUnsupportedMediaType, fmt.Sprintf("File type '%s' is not allowed", mimeType))
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

	bytesWritten, err := h.fileStore.Save(ctx, storagePath, src)
	if err != nil {
		logger.Error("Failed to save file to storage", slog.String("error", err.Error()))
		return c.String(http.StatusInternalServerError, "Failed to save file")
	}

	// Save metadata to the database.
	fileMetadata := &domain.File{
		UserID:      user.ID,
		Filename:    sanitizedFilename,
		MIMEType:    mimeType,
		Size:        bytesWritten,
		StoragePath: storagePath,
	}

	if _, err := h.fileRepo.Create(ctx, fileMetadata); err != nil {
		logger.Error("Failed to save file metadata", slog.String("error", err.Error()))
		// Attempt to clean up the stored file if metadata saving fails.
		_ = h.fileStore.Delete(ctx, storagePath)
		return c.String(http.StatusInternalServerError, "Failed to save file metadata")
	}

	return c.String(http.StatusOK, fmt.Sprintf("File %s uploaded successfully.", fileHeader.Filename))
}

// DeleteFile handles the deletion of a file by its ID.
func (h *FileHandler) DeleteFile(c echo.Context) error {
	ctx := c.Request().Context()
	logger := middleware.FromContext(ctx)

	fileIDParam := c.Param("id")
	if fileIDParam == "" {
		return c.String(http.StatusBadRequest, "File ID is required")
	}

	// Retrieve the authenticated user from the context.
	user, err := getUserFromContext(c)
	if err != nil {
		return err
	}

	// 1. Get the file metadata to find its storage path and verify ownership.
	file, err := h.fileRepo.FindByID(ctx, fileIDParam)
	if err != nil {
		logger.Warn("Failed to get file for deletion", slog.String("fileID", fileIDParam), slog.String("error", err.Error()))
		return c.String(http.StatusNotFound, "File not found")
	}

	// 2. Authorization check: Ensure the current user owns the file.
	if file.UserID == nil || file.UserID.String() != user.ID.String() {
		logger.Warn("User attempted to delete a file they don't own",
			slog.String("userID", user.ID.String()),
			slog.String("fileID", fileIDParam),
			slog.String("ownerID", file.UserID.String()))
		return c.String(http.StatusForbidden, "You do not have permission to delete this file")
	}

	// 3. Delete the physical file from storage.
	if err := h.fileStore.Delete(ctx, file.StoragePath); err != nil {
		logger.Error("Failed to delete physical file from storage", slog.String("path", file.StoragePath), slog.String("error", err.Error()))
		// We continue, to at least remove the database record.
	}

	// 4. Delete the metadata record from the database.
	if err := h.fileRepo.DeleteByID(ctx, file.ID.String()); err != nil {
		logger.Error("Failed to delete file metadata from database",
			slog.String("fileID", file.ID.String()),
			slog.String("error", err.Error()))
		return c.String(http.StatusInternalServerError, "Failed to delete file metadata")
	}

	return c.NoContent(http.StatusNoContent)
}

// DownloadFile handles serving a file's content.
func (h *FileHandler) DownloadFile(c echo.Context) error {
	ctx := c.Request().Context()
	logger := middleware.FromContext(ctx)

	fileIDParam := c.Param("id")
	if fileIDParam == "" {
		return c.String(http.StatusBadRequest, "File ID is required")
	}

	// Retrieve the authenticated user from the context.
	user, err := getUserFromContext(c)
	if err != nil {
		return err
	}

	// 1. Get the file metadata to verify ownership and get content type.
	file, err := h.fileRepo.FindByID(ctx, fileIDParam)
	if err != nil {
		logger.Warn("Failed to get file for download", slog.String("fileID", fileIDParam), slog.String("error", err.Error()))
		return c.String(http.StatusNotFound, "File not found")
	}

	// 2. Authorization check: Ensure the current user owns the file.
	if file.UserID == nil || file.UserID.String() != user.ID.String() {
		logger.Warn("User attempted to download a file they don't own",
			slog.String("userID", user.ID.String()),
			slog.String("fileID", fileIDParam),
			slog.String("ownerID", file.UserID.String()))
		return c.String(http.StatusForbidden, "You do not have permission to download this file")
	}

	// 3. Get the file content from storage.
	content, err := h.fileStore.Get(ctx, file.StoragePath)
	if err != nil {
		logger.Error("Failed to get physical file from storage", slog.String("path", file.StoragePath), slog.String("error", err.Error()))
		return c.String(http.StatusInternalServerError, "Could not retrieve file")
	}
	defer content.Close()

	return c.Stream(http.StatusOK, file.MIMEType, content)
}

// ListFiles returns a list of all files owned by the authenticated user.
func (h *FileHandler) ListFiles(c echo.Context) error {
	ctx := c.Request().Context()
	logger := middleware.FromContext(ctx)

	user, err := getUserFromContext(c)
	if err != nil {
		return err
	}

	files, err := h.fileRepo.FindByUser(ctx, user.ID)
	if err != nil {
		logger.Error("failed to find files for user", "user_id", user.ID.String(), "error", err)
		return c.String(http.StatusInternalServerError, "Could not retrieve files")
	}

	return c.JSON(http.StatusOK, files)
}
