package v2

import (
	"errors"
	"fmt"
)

// Common database errors that can be checked using errors.Is()
var (
	// ErrNotFound is returned when a record is not found in the database.
	ErrNotFound = errors.New("record not found")

	// ErrInvalidID is returned when an invalid ID format is provided.
	ErrInvalidID = errors.New("invalid ID format")

	// ErrInvalidInput is returned when invalid input is provided to a method.
	ErrInvalidInput = errors.New("invalid input data")

	// ErrAlreadyExists is returned when trying to create a record that already exists.
	ErrAlreadyExists = errors.New("record already exists")

	// ErrQueryFailed is returned when a query execution fails.
	ErrQueryFailed = errors.New("query execution failed")

	// ErrMultipleResults is returned when a query that expects a single result returns multiple.
	ErrMultipleResults = errors.New("multiple results found when one was expected")
)

// DBError represents a database error with additional context.
type DBError struct {
	// The underlying error that was returned by the database driver.
	err error

	// Additional context about where the error occurred.
	context string

	// The query that was being executed when the error occurred.
	query string

	// Optional parameters that were used with the query.
	params map[string]any
}

// NewDBError creates a new DBError with the given error and context.
// The context should describe what operation was being performed when the error occurred.
func NewDBError(err error, context string) *DBError {
	return &DBError{
		err:     err,
		context: context,
	}
}

// WithQuery adds query information to the error.
func (e *DBError) WithQuery(query string) *DBError {
	e.query = query
	return e
}

// WithParams adds query parameters to the error.
func (e *DBError) WithParams(params map[string]any) *DBError {
	e.params = params
	return e
}

// Error returns the error message.
func (e *DBError) Error() string {
	msg := e.context
	if e.query != "" {
		msg = fmt.Sprintf("%s\nQuery: %s", msg, e.query)
	}
	if len(e.params) > 0 {
		msg = fmt.Sprintf("%s\nParams: %+v", msg, e.params)
	}
	if e.err != nil {
		msg = fmt.Sprintf("%s: %v", msg, e.err)
	}
	return msg
}

// Unwrap returns the underlying error.
func (e *DBError) Unwrap() error {
	return e.err
}

// Is checks if the target error is of type DBError or matches one of the common database errors.
func (e *DBError) Is(target error) bool {
	if target == nil {
		return e == nil
	}

	switch target {
	case ErrNotFound, ErrInvalidID, ErrInvalidInput, ErrAlreadyExists, ErrQueryFailed, ErrMultipleResults:
		return errors.Is(e.err, target)
	}

	return false
}

// WrapError wraps an error with additional context.
// If the error is already a DBError, it adds the context to the existing error.
// Otherwise, it creates a new DBError with the given context.
func WrapError(err error, context string) *DBError {
	if err == nil {
		return nil
	}

	if dbErr, ok := err.(*DBError); ok {
		// If the error already has context, preserve it
		if dbErr.context != "" {
			context = fmt.Sprintf("%s: %s", context, dbErr.context)
		}
		dbErr.context = context
		return dbErr
	}

	return NewDBError(err, context)
}
