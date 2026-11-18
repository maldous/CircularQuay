//go:build !linux

package quota

// Native reports unsupported where the kernel cannot provide the Linux
// descriptor-bound proof used by fetch. Fetch therefore fails closed.
func Native() Provider { return unsupportedProvider{} }

type unsupportedProvider struct{}

func (unsupportedProvider) Status(string) (Status, error) { return Status{}, ErrUnsupported }

// ValidateFetchIdentity is a no-op outside Linux; Native still fails closed.
func ValidateFetchIdentity() error { return nil }

// ValidateUnprivilegedExecution is a no-op outside Linux; Native still fails
// closed before remote fetch can begin.
func ValidateUnprivilegedExecution(string) error { return nil }
