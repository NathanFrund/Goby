package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// aferoStore is an in-memory implementation of the Store interface for testing.
type aferoStore struct {
	fs afero.Fs
}

// NewAferoStore creates a new AferoStore.
func NewAferoStore(fs afero.Fs) Store {
	return &aferoStore{fs: fs}
}

// Save writes the content of the reader to the given path in the in-memory filesystem.
func (s *aferoStore) Save(ctx context.Context, path string, reader io.Reader) (int64, error) {
	if err := s.fs.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return 0, err
	}
	f, err := s.fs.Create(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(f, reader)
}

// Delete removes a file from the in-memory filesystem.
func (s *aferoStore) Delete(ctx context.Context, path string) error {
	return s.fs.Remove(path)
}

// Get opens a file from the in-memory filesystem for reading.
func (s *aferoStore) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	return s.fs.OpenFile(path, os.O_RDONLY, 0)
}
