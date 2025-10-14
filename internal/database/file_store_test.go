package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupFileStoreTest is a test helper that creates a connection to the test database
// and returns a fully initialized FileStore along with a cleanup function.
func setupFileStoreTest(t *testing.T) (*FileStore, Client[domain.File], Client[TestUser], func()) {
	cfg := testutils.ConfigForTests(t)
	conn := NewConnection(cfg)
	err := conn.Connect(context.Background())
	require.NoError(t, err, "Failed to connect to test database with new connection manager")
	conn.StartMonitoring()

	// Client for file operations.
	fileClient, err := NewClient[domain.File](conn, cfg)
	require.NoError(t, err)

	// Client for creating test users.
	userClient, err := NewClient[TestUser](conn, cfg)
	require.NoError(t, err)

	store := NewFileStore(fileClient)

	cleanup := func() {
		conn.Close(context.Background())
	}
	return store, fileClient, userClient, cleanup
}

func TestFileStore_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	store, fileClient, userClient, cleanup := setupFileStoreTest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Create a prerequisite: a test user to own the file.
	testUserName := "File Test User"
	testUser := TestUser{
		User:     domain.User{Name: &testUserName, Email: fmt.Sprintf("file-user-%d@example.com", time.Now().UnixNano())},
		Password: "password",
	}
	createdUser, err := userClient.Create(ctx, "user", &testUser)
	require.NoError(t, err, "failed to create test user")
	t.Cleanup(func() { _ = userClient.Delete(ctx, createdUser.ID.String()) })

	// 2. Create file metadata using the real user's ID.
	storagePath := fmt.Sprintf("user/files/test-%d.txt", time.Now().UnixNano())
	fileToCreate := domain.File{
		UserID:      createdUser.ID,
		Filename:    "test.txt",
		MIMEType:    "text/plain",
		Size:        12345,
		StoragePath: storagePath,
	}

	createdFile, err := store.Create(ctx, &fileToCreate)
	require.NoError(t, err, "failed to create file")
	require.NotNil(t, createdFile)
	require.NotNil(t, createdFile.ID, "Created file should have an ID")
	require.NotEmpty(t, createdFile.ID.String(), "Created file ID should not be empty")

	// Ensure cleanup with proper ID handling
	idStr := createdFile.ID.String()
	t.Cleanup(func() { _ = fileClient.Delete(ctx, idStr) })

	// 3. Test GetByID
	fetchedByID, err := store.FindByID(ctx, idStr)
	require.NoError(t, err)
	require.NotNil(t, fetchedByID)
	assert.Equal(t, createdFile.ID, fetchedByID.ID)
	assert.Equal(t, fileToCreate.Filename, fetchedByID.Filename)
	assert.Equal(t, fileToCreate.StoragePath, fetchedByID.StoragePath)
	require.NotNil(t, fetchedByID.CreatedAt, "CreatedAt should not be nil")
	require.False(t, fetchedByID.CreatedAt.IsZero(), "CreatedAt should not be a zero time")
	assert.WithinDuration(t, time.Now(), fetchedByID.CreatedAt.Time, 5*time.Second, "CreatedAt should be recent")

	// 4. Test FindByStoragePath
	fetchedByPath, err := store.FindByStoragePath(ctx, storagePath)
	require.NoError(t, err)
	require.NotNil(t, fetchedByPath)
	assert.Equal(t, createdFile.ID, fetchedByPath.ID)

	// 5. Test FindByStoragePath with non-existent path
	_, err = store.FindByStoragePath(ctx, "non/existent/path.txt")
	require.Error(t, err)
	assert.Equal(t, "file not found", err.Error())

	// 5. Test Update
	updatedFilename := "updated_test.txt"
	fetchedByID.Filename = updatedFilename
	updatedFile, err := store.Update(ctx, fetchedByID)
	require.NoError(t, err)
	require.NotNil(t, updatedFile)
	assert.Equal(t, updatedFilename, updatedFile.Filename)
	require.NotNil(t, updatedFile.UpdatedAt, "UpdatedAt should not be nil")
	require.NotNil(t, updatedFile.CreatedAt, "CreatedAt should not be nil")
	assert.False(t, updatedFile.UpdatedAt.IsZero(), "UpdatedAt should not be a zero time")
	assert.True(t, updatedFile.UpdatedAt.Time.After(updatedFile.CreatedAt.Time), "UpdatedAt should be after CreatedAt")

	// 7. Test Delete
	err = store.DeleteByID(ctx, idStr)
	require.NoError(t, err)

	deletedFile, err := store.FindByID(ctx, idStr)
	require.Error(t, err)
	assert.Nil(t, deletedFile)
}
