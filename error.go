package command

import "fmt"

// Error is the package's sentinel-error type. Every error Tee can emit is a
// const of this type, so each path is matchable with errors.Is rather than by
// string comparison.
type Error string

func (e Error) Error() string { return string(e) }

const (
	// ErrOpenFile is returned when a named tee destination cannot be opened.
	ErrOpenFile Error = "tee: cannot open file"
	// ErrWrite is returned when writing a line to a destination fails.
	ErrWrite Error = "tee: write failed"
)

// With wraps a cause with %w so errors.Is still matches both this sentinel and
// the cause, optionally appending space-separated context args.
func (e Error) With(cause error, args ...any) error {
	out := error(e)
	if cause != nil {
		out = fmt.Errorf("%w: %w", e, cause)
	}
	if len(args) > 0 {
		out = fmt.Errorf("%w: %v", out, args)
	}
	return out
}

var _ error = Error("")
