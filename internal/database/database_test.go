package database

import (
	"context"
	"testing"

	"github.com/nfrund/goby/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Load configuration using the application's standard mechanism.
	baseCfg := config.New()

	tests := []struct {
		name    string
		prepare func() *config.Config
		wantErr bool
	}{
		{
			name: "success - valid configuration",
			prepare: func() *config.Config {
				return baseCfg
			},
			wantErr: false,
		},
		{
			name: "error - invalid URL",
			prepare: func() *config.Config {
				invalidCfg := *baseCfg
				invalidCfg.DBUrl = "invalid://url"
				return &invalidCfg
			},
			wantErr: true,
		},
		{
			name: "error - invalid credentials",
			prepare: func() *config.Config {
				invalidCfg := *baseCfg
				invalidCfg.DBPass = "wrongpassword"
				return &invalidCfg
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			testCfg := tt.prepare()

			db, err := NewDB(ctx, testCfg)

			if tt.wantErr {
				assert.Error(t, err, "expected an error")
				assert.Nil(t, db, "db should be nil on error")
			} else {
				require.NoError(t, err, "unexpected error creating DB connection")
				assert.NotNil(t, db, "db should not be nil")

				// Verify connection by using the Info method
				_, err = db.Info(ctx)
				assert.NoError(t, err, "should be able to get database info")

				// Cleanup
				_ = db.Close(ctx) // Ignoring error for cleanup
			}
		})
	}
}

func TestNewDB_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := config.New()

	db, err := NewDB(ctx, cfg)
	assert.Error(t, err, "should return error with cancelled context")
	assert.Nil(t, db, "db should be nil on error")
}
