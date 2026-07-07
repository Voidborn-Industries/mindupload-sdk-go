package mindupload

import (
	"errors"
	"fmt"
)

// Sentinel errors for classification with errors.Is.
var (
	// ErrAuthentication indicates a missing, malformed, or rejected partner key (HTTP 401).
	ErrAuthentication = errors.New("mindupload: authentication failed")
	// ErrRateLimit indicates a rate limit or credit cap was hit (HTTP 429).
	ErrRateLimit = errors.New("mindupload: rate limited")
	// ErrConnection indicates the API could not be reached.
	ErrConnection = errors.New("mindupload: connection failed")
)

// Error is returned by every operation on failure.
//
// Classify it with errors.Is, or inspect the details with errors.As:
//
//	if errors.Is(err, mindupload.ErrAuthentication) { ... }
//
//	var apiErr *mindupload.Error
//	if errors.As(err, &apiErr) {
//		log.Printf("%s failed: HTTP %d: %s", apiErr.Operation, apiErr.Status, apiErr.Message)
//	}
type Error struct {
	Operation  string         // the operation that failed, e.g. "rag"
	Status     int            // HTTP status; 0 for a logical or connection failure
	Message    string         // server-provided message
	RetryAfter float64        // seconds to wait, for rate-limit errors (0 if unknown)
	Response   map[string]any // full decoded response, when available
	err        error          // wrapped sentinel
}

func (e *Error) Error() string {
	if e.Status > 0 {
		return fmt.Sprintf("mindupload: %s: %s (HTTP %d)", e.Operation, e.Message, e.Status)
	}
	return fmt.Sprintf("mindupload: %s: %s", e.Operation, e.Message)
}

// Unwrap exposes the sentinel (ErrAuthentication, ErrRateLimit, ErrConnection)
// so errors.Is works.
func (e *Error) Unwrap() error { return e.err }
