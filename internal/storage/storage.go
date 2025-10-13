package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// FileStorage defines the interface for an abstract file storage system.
// This allows us to swap out the underlying storage (e.g., local disk, S3)
// without changing the application logic.
type FileStorage interface {
	// Save writes the content from an io.Reader to the specified path.
	// It returns the number of bytes written and any error encountered.
	Save(ctx context.Context, path string, content io.Reader) (int64, error)

	// Open retrieves a file from the specified path.
	// It returns an afero.File, which implements io.Reader, io.Closer, and io.Seeker.
	Open(ctx context.Context, path string) (afero.File, error)

	// Delete removes a file from the specified path.
	Delete(ctx context.Context, path string) error

	// Fs returns the underlying afero.Fs instance for more advanced operations.
	Fs() afero.Fs
}

// aferoStore is an implementation of FileStorage that uses afero.
type aferoStore struct {
	fs afero.Fs
}

// NewAferoStore creates a new FileStorage instance backed by the given afero.Fs.
func NewAferoStore(fs afero.Fs) FileStorage {
	return &aferoStore{fs: fs}
}

// Save writes the file content to the afero filesystem.
func (s *aferoStore) Save(ctx context.Context, path string, content io.Reader) (int64, error) {
	// Ensure the directory exists.
	dir := filepath.Dir(path)
	if err := s.fs.MkdirAll(dir, 0755); err != nil {
		return 0, err
	}

	f, err := s.fs.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	return io.Copy(f, content)
}

// Open retrieves a file from the afero filesystem.
func (s *aferoStore) Open(ctx context.Context, path string) (afero.File, error) {
	return s.fs.Open(path)
}

// Delete removes a file from the afero filesystem.
func (s *aferoStore) Delete(ctx context.Context, path string) error {
	return s.fs.Remove(path)
}

// Fs returns the raw afero filesystem.
func (s *aferoStore) Fs() afero.Fs {
	return s.fs
}
