package plane

import (
	"errors"
	"fmt"
)

// Sentinels for errors.Is.
var (
	ErrUnauthorized = errors.New("plane: unauthorized")
	ErrNotFound     = errors.New("plane: not found")
	ErrRateLimited  = errors.New("plane: rate limited")
)

// APIError is an API response error.
type APIError struct {
	Status  int
	Message string
}

// Error implements error.
func (e *APIError) Error() string {
	return fmt.Sprintf("plane: HTTP %d: %s", e.Status, e.Message)
}

// Is maps statuses to sentinels for errors.Is.
func (e *APIError) Is(target error) bool {
	switch target {
	case ErrUnauthorized:
		return e.Status == 401 || e.Status == 403
	case ErrNotFound:
		return e.Status == 404
	case ErrRateLimited:
		return e.Status == 429
	}
	return false
}
