package testutils

import (
	"github.com/google/uuid"
	surrealmodels "github.com/surrealdb/surrealdb.go/pkg/models"
)

// NewTestRecordID creates a new RecordID for testing purposes.
func NewTestRecordID(table string) *surrealmodels.RecordID {
	id := surrealmodels.NewRecordID(table, uuid.NewString())
	return &id
}
