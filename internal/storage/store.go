package storage

import (
	"context"
	"io"
)

// Store defines the interface for a file storage backend.
type Store interface {
	Save(ctx context.Context, path string, reader io.Reader) (int64, error)
	Delete(ctx context.Context, path string) error
}
