package db

// PGNImportError wraps an error that occurred during PGN parsing
// and includes the PGN text that caused the error.
type PGNImportError struct {
	OriginalError error
	PGNText       string
}

// Error returns the message of the original error.
func (e *PGNImportError) Error() string {
	if e.OriginalError == nil {
		return "unknown PGN import error"
	}
	return e.OriginalError.Error()
}

// Unwrap provides compatibility for errors.Is and errors.As,
// returning the original wrapped error.
func (e *PGNImportError) Unwrap() error {
	return e.OriginalError
}
