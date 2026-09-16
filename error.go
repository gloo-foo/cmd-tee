package command

import errs "github.com/gomatic/go-error"

const (
	// ErrOpenFile is returned when a named tee destination cannot be opened.
	ErrOpenFile errs.Const = "tee: cannot open file"
	// ErrWrite is returned when writing a line to a destination fails.
	ErrWrite errs.Const = "tee: write failed"
)
