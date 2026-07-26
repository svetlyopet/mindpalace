package clictx

import "fmt"

// UsageError marks cobra usage failures for exit code 2.
type UsageError struct{ err error }

func (e UsageError) Error() string { return e.err.Error() }
func (e UsageError) Unwrap() error { return e.err }
func (e UsageError) IsUsage() bool { return true }

func Usagef(format string, args ...any) error {
	return UsageError{err: fmt.Errorf(format, args...)}
}
