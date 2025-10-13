package storage_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"path/filepath"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/storage"
	"github.com/nfrund/goby/internal/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileHandler_Upload tests the file upload endpoint.
func TestFileHandler_Upload(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// --- Setup ---
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// 1. Database and FileStore setup
	cfg := testutils.ConfigForTests(t)
	conn := database.NewConnection(cfg)
	err := conn.Connect(ctx)
	require.NoError(t, err)
	defer conn.Close(ctx)
	conn.StartMonitoring()

	fileClient, err := database.NewClient[domain.File](conn, cfg)
	require.NoError(t, err)
	fileStore := database.NewFileStore(fileClient)

	// 2. In-memory storage setup
	memFs := afero.NewMemMapFs()
	aferoStore := storage.NewAferoStore(memFs)

	// 3. Create a test user to own the file
	userClient, err := database.NewClient[testutils.TestUser](conn, cfg)
	require.NoError(t, err)
	testUserName := "File Uploader"
	testUser := testutils.TestUser{
		User:     domain.User{Name: &testUserName, Email: fmt.Sprintf("uploader-%d@example.com", time.Now().UnixNano())},
		Password: "password",
	}
	createdUser, err := userClient.Create(ctx, "user", &testUser)
	require.NoError(t, err)
	t.Cleanup(func() { _ = userClient.Delete(ctx, createdUser.ID.String()) })

	// 4. Handler and Server setup
	// For this test, allow any size and type to test the success path.
	maxSize := int64(10 * 1024 * 1024) // 10MB
	allowedTypes := []string{"text/plain"}
	fileHandler := storage.NewFileHandler(aferoStore, fileStore, maxSize, allowedTypes)
	e := echo.New()
	// Middleware to inject the user ID into the context for the handler
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user", &createdUser.User)
			return next(c)
		}
	})
	e.POST("/upload", fileHandler.Upload)
	e.DELETE("/files/:id", fileHandler.Delete)
	e.GET("/files/:id/download", fileHandler.Download)

	// --- Create Upload Request ---
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test-upload.txt")
	// part.Header.Set("Content-Type", "text/plain") // This is incorrect, but let's fix it by using CreatePart
	require.NoError(t, err)
	fileContent := "this is the content of the uploaded file"
	_, err = io.WriteString(part, fileContent)
	require.NoError(t, err)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	rec := httptest.NewRecorder()

	// --- Execute Request ---
	e.ServeHTTP(rec, req)

	// --- Assertions ---
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "File test-upload.txt uploaded successfully.")

	// Verify file metadata was saved in the database
	// We need to find the file by something other than ID, since we don't know it.
	// A real app might have a "ListFilesForUser" method. Here, we'll query by filename.
	query := "SELECT * FROM file WHERE filename = $name AND user_id = $user"
	vars := map[string]interface{}{"name": "test-upload.txt", "user": createdUser.ID}
	files, err := fileClient.Query(ctx, query, vars)
	require.NoError(t, err)
	require.Len(t, files, 1, "expected one file record in the database")

	savedFile := files[0]
	t.Logf("Verified file metadata was saved to database with ID: %s", savedFile.ID.String())
	assert.Equal(t, "test-upload.txt", savedFile.Filename)
	assert.Equal(t, int64(len(fileContent)), savedFile.SizeBytes)
	assert.Equal(t, createdUser.ID, savedFile.UserID)
	t.Cleanup(func() { _ = fileStore.Delete(ctx, savedFile.ID.String()) })

	// Verify file content was saved in the in-memory storage
	expectedPath := filepath.Join("users", createdUser.ID.String())
	infos, err := afero.ReadDir(memFs, expectedPath)
	require.NoError(t, err)
	require.Len(t, infos, 1, "expected one file in the user's storage directory")
	storagePath := filepath.Join(expectedPath, infos[0].Name())
	t.Logf("Verified file content was saved to in-memory storage at: %s", storagePath)
	readBytes, err := afero.ReadFile(memFs, storagePath)
	require.NoError(t, err)
	assert.Equal(t, fileContent, string(readBytes))
}

// TestFileHandler_Delete tests the file deletion endpoint.
func TestFileHandler_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// --- Setup ---
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cfg := testutils.ConfigForTests(t)
	conn := database.NewConnection(cfg)
	err := conn.Connect(ctx)
	require.NoError(t, err)
	defer conn.Close(ctx)

	fileClient, err := database.NewClient[domain.File](conn, cfg)
	require.NoError(t, err)
	fileStore := database.NewFileStore(fileClient)

	memFs := afero.NewMemMapFs()
	aferoStore := storage.NewAferoStore(memFs)

	userClient, err := database.NewClient[testutils.TestUser](conn, cfg)
	require.NoError(t, err)
	testUserName := "File Deleter"
	testUser := testutils.TestUser{
		User:     domain.User{Name: &testUserName, Email: fmt.Sprintf("deleter-%d@example.com", time.Now().UnixNano())},
		Password: "password",
	}
	createdUser, err := userClient.Create(ctx, "user", &testUser)
	require.NoError(t, err)
	t.Cleanup(func() { _ = userClient.Delete(ctx, createdUser.ID.String()) })

	// --- Create a file to be deleted ---
	storagePath := filepath.Join("users", createdUser.ID.String(), "file-to-delete.txt")
	fileContent := "this file will be deleted"
	_, err = aferoStore.Save(ctx, storagePath, bytes.NewReader([]byte(fileContent)))
	require.NoError(t, err)

	fileToCreate := &domain.File{
		UserID:      createdUser.ID,
		Filename:    "file-to-delete.txt",
		StoragePath: storagePath,
		SizeBytes:   int64(len(fileContent)),
	}
	createdFile, err := fileStore.Create(ctx, fileToCreate)
	require.NoError(t, err)

	// --- Setup Handler and Server ---
	maxSize := int64(10 * 1024 * 1024) // 10MB
	allowedTypes := []string{"text/plain"}
	fileHandler := storage.NewFileHandler(aferoStore, fileStore, maxSize, allowedTypes)
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user", &createdUser.User)
			return next(c)
		}
	})
	e.DELETE("/files/:id", fileHandler.Delete)
	e.GET("/files/:id/download", fileHandler.Download)

	// --- Execute Delete Request ---
	req := httptest.NewRequest(http.MethodDelete, "/files/"+createdFile.ID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// --- Assertions ---
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Verify metadata was deleted from the database
	_, err = fileStore.GetByID(ctx, createdFile.ID.String())
	require.Error(t, err, "expected an error when getting a deleted file")
	// A more specific check for a "not found" error would be even better.
	// For now, any error suffices to show it's gone.

	// Verify file content was deleted from storage
	exists, err := afero.Exists(memFs, storagePath)
	require.NoError(t, err)
	assert.False(t, exists, "expected physical file to be deleted from storage")
}

// TestFileHandler_Download tests the file download endpoint.
func TestFileHandler_Download(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// --- Setup ---
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cfg := testutils.ConfigForTests(t)
	conn := database.NewConnection(cfg)
	err := conn.Connect(ctx)
	require.NoError(t, err)
	defer conn.Close(ctx)

	fileClient, err := database.NewClient[domain.File](conn, cfg)
	require.NoError(t, err)
	fileStore := database.NewFileStore(fileClient)

	memFs := afero.NewMemMapFs()
	aferoStore := storage.NewAferoStore(memFs)

	userClient, err := database.NewClient[testutils.TestUser](conn, cfg)
	require.NoError(t, err)
	testUserName := "File Downloader"
	testUser := testutils.TestUser{
		User:     domain.User{Name: &testUserName, Email: fmt.Sprintf("downloader-%d@example.com", time.Now().UnixNano())},
		Password: "password",
	}
	createdUser, err := userClient.Create(ctx, "user", &testUser)
	require.NoError(t, err)
	t.Cleanup(func() { _ = userClient.Delete(ctx, createdUser.ID.String()) })

	// --- Create a file to be downloaded ---
	storagePath := filepath.Join("users", createdUser.ID.String(), "file-to-download.txt")
	fileContent := "this is the content of the downloaded file"
	_, err = aferoStore.Save(ctx, storagePath, bytes.NewReader([]byte(fileContent)))
	require.NoError(t, err)

	fileToCreate := &domain.File{
		UserID:      createdUser.ID,
		Filename:    "file-to-download.txt",
		MimeType:    "text/plain",
		StoragePath: storagePath,
		SizeBytes:   int64(len(fileContent)),
	}
	createdFile, err := fileStore.Create(ctx, fileToCreate)
	require.NoError(t, err)
	t.Cleanup(func() { _ = fileStore.Delete(ctx, createdFile.ID.String()) })

	// --- Setup Handler and Server ---
	maxSize := int64(10 * 1024 * 1024) // 10MB
	allowedTypes := []string{"text/plain"}
	fileHandler := storage.NewFileHandler(aferoStore, fileStore, maxSize, allowedTypes)
	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user", &createdUser.User)
			return next(c)
		}
	})
	e.GET("/files/:id/download", fileHandler.Download)

	// --- Execute Download Request ---
	req := httptest.NewRequest(http.MethodGet, "/files/"+createdFile.ID.String()+"/download", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// --- Assertions ---
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/plain", rec.Header().Get("Content-Type"))
	bodyBytes, err := ioutil.ReadAll(rec.Body)
	require.NoError(t, err)
	assert.Equal(t, fileContent, string(bodyBytes))
}

// TestFileHandler_Upload_Validation tests the security validations on upload.
func TestFileHandler_Upload_Validation(t *testing.T) {
	// --- Setup ---
	// No database needed for these validation tests
	fileStore := database.NewFileStore(nil)
	memFs := afero.NewMemMapFs()
	aferoStore := storage.NewAferoStore(memFs)

	// Create a dummy user, does not need to be in the DB for this test.
	user := &domain.User{ID: testutils.NewTestRecordID("user")}

	// Configure handler with strict limits
	maxSize := int64(1024) // 1 KB
	allowedTypes := []string{"image/png", "image/jpeg"}
	fileHandler := storage.NewFileHandler(aferoStore, fileStore, maxSize, allowedTypes)

	e := echo.New()
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user", user)
			return next(c)
		}
	})
	e.POST("/upload", fileHandler.Upload)

	t.Run("rejects unsupported MIME type", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", `form-data; name="file"; filename="script.sh"`)
		h.Set("Content-Type", "application/x-shellscript") // Disallowed type
		part, err := writer.CreatePart(h)
		require.NoError(t, err)
		_, err = io.WriteString(part, "echo 'pwned'")
		require.NoError(t, err)
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
		assert.Contains(t, rec.Body.String(), "File type 'application/x-shellscript' is not allowed")
	})

	t.Run("rejects file that is too large", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", `form-data; name="file"; filename="large-file.png"`)
		h.Set("Content-Type", "image/png") // Allowed type
		part, err := writer.CreatePart(h)
		require.NoError(t, err)
		// Create content larger than the 1KB limit
		largeContent := make([]byte, 2048)
		_, err = part.Write(largeContent)
		require.NoError(t, err)
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/upload", body)
		req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
		assert.Contains(t, rec.Body.String(), "File size of 2048 bytes exceeds the limit")
	})
}
