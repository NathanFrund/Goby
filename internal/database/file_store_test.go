package database

import (
	"context"
	"errors"
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
	assert.True(t, errors.Is(err, ErrNotFound), "Expected not found error")

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

func TestFileStore_FindByUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	store, fileClient, userClient, cleanup := setupFileStoreTest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Create a test user
	name := "File Test User"
	testUser := TestUser{
		User: domain.User{
			Name:  &name,
			Email: fmt.Sprintf("findbyuser-%d@example.com", time.Now().UnixNano()),
		},
		Password: "password",
	}
	createdUser, err := userClient.Create(ctx, "user", &testUser)
	require.NoError(t, err, "failed to create test user")
	t.Cleanup(func() { _ = userClient.Delete(ctx, createdUser.ID.String()) })

	// 2. Create test files with different timestamps
	now := time.Now()
	testFiles := []*domain.File{
		{
			UserID:   createdUser.ID,
			Filename: "file1.txt",
			MIMEType: "text/plain",
			Size:     100,
		},
		{
			UserID:   createdUser.ID,
			Filename: "file2.txt",
			MIMEType: "text/plain",
			Size:     200,
		},
		{
			UserID:   createdUser.ID,
			Filename: "file3.txt",
			MIMEType: "text/plain",
			Size:     300,
		},
	}

	// Create files with staggered timestamps to ensure consistent ordering
	for i, file := range testFiles {
		file.StoragePath = fmt.Sprintf("user/files/test-%d-%d.txt", i, time.Now().UnixNano())
		created, err := store.Create(ctx, file)
		require.NoError(t, err, "failed to create test file %d", i)
		t.Cleanup(func() { _ = fileClient.Delete(ctx, created.ID.String()) })

		// Update the created at time to ensure they're in a known order
		_, err = fileClient.Update(ctx, created.ID.String(), map[string]interface{}{
			"created_at": now.Add(time.Duration(i) * time.Hour),
		})
		require.NoError(t, err, "failed to update file timestamp")
	}

	t.Run("returns all files for user", func(t *testing.T) {
		files, total, err := store.FindByUser(ctx, createdUser.ID, 10, 0)
		require.NoError(t, err)
		require.Len(t, files, 3)
		require.Equal(t, int64(3), total)

		// Should be ordered by created_at DESC (newest first)
		assert.Equal(t, "file3.txt", files[0].Filename)
		assert.Equal(t, "file2.txt", files[1].Filename)
		assert.Equal(t, "file1.txt", files[2].Filename)
	})

	t.Run("respects pagination", func(t *testing.T) {
		// First page
		files, total, err := store.FindByUser(ctx, createdUser.ID, 2, 0)
		require.NoError(t, err)
		require.Len(t, files, 2)
		require.Equal(t, int64(3), total)
		assert.Equal(t, "file3.txt", files[0].Filename)
		assert.Equal(t, "file2.txt", files[1].Filename)

		// Second page
		files, total, err = store.FindByUser(ctx, createdUser.ID, 2, 2)
		require.NoError(t, err)
		require.Len(t, files, 1)
		require.Equal(t, int64(3), total)
		assert.Equal(t, "file1.txt", files[0].Filename)
	})

	t.Run("returns empty slice for user with no files", func(t *testing.T) {
		// Create another user with no files
		otherUserName := "Other User"
		otherUser := TestUser{
			User: domain.User{
				Name:  &otherUserName,
				Email: fmt.Sprintf("other-%d@example.com", time.Now().UnixNano()),
			},
			Password: "password",
		}
		createdOtherUser, err := userClient.Create(ctx, "user", &otherUser)
		require.NoError(t, err)
		t.Cleanup(func() { _ = userClient.Delete(ctx, createdOtherUser.ID.String()) })

		files, total, err := store.FindByUser(ctx, createdOtherUser.ID, 10, 0)
		require.NoError(t, err)
		assert.Empty(t, files)
		assert.Equal(t, int64(0), total)
	})

	t.Run("returns error for nil user ID", func(t *testing.T) {
		_, _, err := store.FindByUser(ctx, nil, 10, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user ID is required", "Expected error about missing user ID")
	})

	t.Run("returns all files when limit is 0", func(t *testing.T) {
		files, total, err := store.FindByUser(ctx, createdUser.ID, 0, 0)
		require.NoError(t, err)
		// Should return all 3 files regardless of limit/offset when limit is 0
		assert.Len(t, files, 3)
		assert.Equal(t, int64(3), total)
	})

	t.Run("returns all files when limit is negative", func(t *testing.T) {
		files, total, err := store.FindByUser(ctx, createdUser.ID, -1, 0)
		require.NoError(t, err)
		// Should return all 3 files regardless of limit/offset when limit is negative
		assert.Len(t, files, 3)
		assert.Equal(t, int64(3), total)
	})

	t.Run("ignores offset when limit is 0", func(t *testing.T) {
		files, total, err := store.FindByUser(ctx, createdUser.ID, 0, 10) // Large offset with limit=0
		require.NoError(t, err)
		// Should still return all files even with offset when limit is 0
		assert.Len(t, files, 3)
		assert.Equal(t, int64(3), total)
	})

	t.Run("handles offset beyond total count", func(t *testing.T) {
		files, total, err := store.FindByUser(ctx, createdUser.ID, 10, 100) // Offset beyond total
		require.NoError(t, err)
		// Should return empty slice when offset is beyond total count
		assert.Empty(t, files)
		assert.Equal(t, int64(3), total)
	})
}
