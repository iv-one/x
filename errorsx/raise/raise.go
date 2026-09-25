package raise

import (
	"errors"
	"fmt"
	"io"
)

//nolint:errname // internal wrapper type, not exported
type withFrame struct {
	error
	frame Frame
}

// Error creates a new error with a frame.
func Error(err error) error {
	// If the error is nil, return immediately.
	if err == nil {
		return nil
	}

	return &withFrame{
		error: err,
		frame: caller(),
	}
}

func fprint(w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(w, format, a...)
}

func writeString(s fmt.State, str string) {
	_, _ = io.WriteString(s, str)
}

func (w *withFrame) Cause() error { return w.error }

// Unwrap provides compatibility for Go 1.13 error chains.
func (w *withFrame) Unwrap() error { return w.error }

func (w *withFrame) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		writeString(s, "\n<")
		w.frame.Format(s, 't')
		writeString(s, "> ")

		if s.Flag('+') {
			fprint(s, "%+v", w.Cause())
		} else {
			fprint(s, "%v", w.Cause())
		}
	case 's':
		writeString(s, w.Error())
	case 'q':
		fprint(s, "%q", w.Error())
	}
}

//nolint:errname // internal wrapper type, not exported
type withCause struct {
	err   error
	cause error
	frame Frame
}

func (w *withCause) Error() string { return fmt.Sprintf("%s: %s", w.err, w.cause) }

// Errors creates a new error with a cause and a frame.
// Returns nil if both err and cause are nil.
// Returns the non-nil error wrapped with a frame if only one is nil.
func Errors(err error, cause error) error {
	if err == nil && cause == nil {
		return nil
	}
	if err == nil {
		return Error(cause)
	}
	if cause == nil {
		return Error(err)
	}
	return &withCause{
		err:   err,
		cause: cause,
		frame: caller(),
	}
}

func (w *withCause) Cause() error { return w.cause }

// Unwrap provides compatibility for Go 1.13 error chains.
func (w *withCause) Unwrap() error { return w.cause }

func (w *withCause) Is(target error) bool {
	return errors.Is(w.err, target) || errors.Is(w.cause, target)
}

func (w *withCause) As(target any) bool {
	return errors.As(w.err, target) || errors.As(w.cause, target)
}

func (w *withCause) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		writeString(s, "\n<")
		w.frame.Format(s, 't')
		writeString(s, "> ")

		if s.Flag('+') {
			fprint(s, "%+v ", w.err)
		} else {
			fprint(s, "%v ", w.err)
		}

		if s.Flag('+') {
			fprint(s, "%+v", w.Cause())
		} else {
			fprint(s, "%v", w.Cause())
		}
	case 's':
		writeString(s, w.Error())
	case 'q':
		fprint(s, "%q", w.err)
		writeString(s, ": ")
		fprint(s, "%s", w.Cause())
	}
}

// Errorf creates a new error with a frame.
func Errorf(format string, a ...any) error {
	return &withFrame{
		error: fmt.Errorf(format, a...),
		frame: caller(),
	}
}

// DeepUnwrap fully unwraps the error chain, returning the innermost error.
// Returns nil if err is nil or has no wrapped errors.
func DeepUnwrap(err error) error {
	if err == nil {
		return nil
	}
	for {
		cause := errors.Unwrap(err)
		if cause == nil {
			return err
		}
		err = cause
	}
}
