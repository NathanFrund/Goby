package storage

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAferoStore_Unit(t *testing.T) {
	// 1. Setup: Create an in-memory filesystem for the test.
	// This is the core benefit of using afero for testing. No disk I/O is performed.
	memFs := afero.NewMemMapFs()
	store := NewAferoStore(memFs)
	ctx := context.Background()

	// Test data
	filePath := "test/dir/my-file.txt"
	fileContent := "hello world, this is a test"

	// 2. Test Save
	t.Run("Save", func(t *testing.T) {
		contentReader := bytes.NewReader([]byte(fileContent))
		bytesWritten, err := store.Save(ctx, filePath, contentReader)

		require.NoError(t, err)
		assert.Equal(t, int64(len(fileContent)), bytesWritten)

		// Verify the file was actually written using afero's helpers
		exists, err := afero.Exists(memFs, filePath)
		require.NoError(t, err)
		assert.True(t, exists, "file should exist after saving")

		// Verify content
		readBytes, err := afero.ReadFile(memFs, filePath)
		require.NoError(t, err)
		assert.Equal(t, fileContent, string(readBytes))
	})

	// 3. Test Open
	t.Run("Open", func(t *testing.T) {
		file, err := store.Open(ctx, filePath)
		require.NoError(t, err)
		defer file.Close()

		readBytes, err := io.ReadAll(file)
		require.NoError(t, err)
		assert.Equal(t, fileContent, string(readBytes))
	})

	// 4. Test Delete
	t.Run("Delete", func(t *testing.T) {
		err := store.Delete(ctx, filePath)
		require.NoError(t, err)

		// Verify the file was actually deleted
		exists, err := afero.Exists(memFs, filePath)
		require.NoError(t, err)
		assert.False(t, exists, "file should not exist after deleting")
	})

	// 5. Test edge cases
	t.Run("Open non-existent file", func(t *testing.T) {
		_, err := store.Open(ctx, "path/to/nothing.txt")
		assert.Error(t, err, "opening a non-existent file should return an error")
	})
}
