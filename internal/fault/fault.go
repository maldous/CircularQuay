// Package fault provides safe error categories without upstream response bodies.
package fault

import "errors"

// Kind is a stable, non-secret failure classification.
type Kind string

const (
	Invalid     Kind = "invalid_policy"
	Trust       Kind = "trust_failure"
	Denied      Kind = "access_denied"
	Unavailable Kind = "unavailable"
	Missing     Kind = "missing_value"
	Expired     Kind = "expired_value"
	Unsupported Kind = "unsupported_capability"
	Apply       Kind = "activation_failed"
)

// Error contains a safe operation name, never a raw SDK error or value.
type Error struct {
	Kind      Kind
	Operation string
}

func (e *Error) Error() string { return string(e.Kind) + ": " + e.Operation }

// New constructs a sanitized failure.
func New(kind Kind, operation string) error { return &Error{Kind: kind, Operation: operation} }

// Is tests a failure without parsing human-readable diagnostics.
func Is(err error, kind Kind) bool { var e *Error; return errors.As(err, &e) && e.Kind == kind }

// Category returns only the safe error class.
func Category(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return Apply
}
