package errorsx

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"
)

// Error is an error in the model of AIP-193.
//
// A package declares its errors as values that it makes from a base error,
// such as ErrNotFound.WithReason("USER_NOT_FOUND").WithMessage("..."). Each
// With method returns a copy, so a declared error does not change.
type Error struct {
	// Code is the canonical code. The HTTP status comes from it.
	Code Code

	// Reason is the stable name of the error for a program, in upper case
	// with underscores, such as `USER_NOT_FOUND`. A program reads the reason
	// and not the message. An error with no reason has the name of its code
	// as the reason (see [Error.ReasonOrCode]).
	Reason string

	// Message is the text for a person. An error with no message has the
	// message of its code (see [Error.Error]).
	Message string

	// Metadata is the parts of the error that change from one occurrence to
	// the next, for a program.
	Metadata map[string]string

	// Violations are the fields of the request that break a rule.
	Violations []Violation

	// RetryDelay is how long the caller waits before it sends the request
	// again. Zero is no advice.
	RetryDelay time.Duration
}

// Violation is one field of a request that breaks a rule.
type Violation struct {
	// Field is the path of the field in the request.
	Field string
	// Description says which rule the field breaks.
	Description string
}

// Error is the message, or the message of the code for an error with none.
func (e Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return codes[e.Code].message
}

// ReasonOrCode is the reason, or the name of the code in upper case for an
// error with no reason, such as `PERMISSION_DENIED`.
func (e Error) ReasonOrCode() string {
	if e.Reason != "" {
		return e.Reason
	}

	name := []byte(e.Code.String())
	for i, char := range name {
		if char >= 'a' && char <= 'z' {
			name[i] = char - 'a' + 'A'
		}
	}
	return string(name)
}

// Is reports whether e is err or an error made from it. The With methods
// copy an Error, so identity is not sufficient: err is a template, and each
// of its code, reason and message that is set must be the same in e.
// ErrNotFound matches each not-found error. An error with a reason matches
// only the errors with that reason.
func (e Error) Is(err error) bool {
	var target Error
	switch t := err.(type) {
	case Error:
		target = t
	case *Error:
		if t == nil {
			return false
		}
		target = *t
	default:
		return false
	}

	return (target.Code == 0 || target.Code == e.Code) &&
		(target.Reason == "" || target.Reason == e.Reason) &&
		(target.Message == "" || target.Message == e.Message)
}

// WithReason returns a copy with the reason.
func (e Error) WithReason(reason string) *Error {
	e.Reason = reason
	return &e
}

// WithMessage returns a copy with the message.
func (e Error) WithMessage(message string) *Error {
	e.Message = message
	return &e
}

// WithMessagef returns a copy with the formatted message.
func (e Error) WithMessagef(format string, args ...any) *Error {
	return e.WithMessage(fmt.Sprintf(format, args...))
}

// WithMetadata returns a copy with one more entry of metadata.
func (e Error) WithMetadata(key, value string) *Error {
	metadata := maps.Clone(e.Metadata)
	if metadata == nil {
		metadata = map[string]string{}
	}
	metadata[key] = value
	e.Metadata = metadata
	return &e
}

// WithViolation returns a copy with one more field violation.
func (e Error) WithViolation(field, description string) *Error {
	e.Violations = append(slices.Clone(e.Violations), Violation{Field: field, Description: description})
	return &e
}

// WithRetryDelay returns a copy with the retry delay.
func (e Error) WithRetryDelay(delay time.Duration) *Error {
	e.RetryDelay = delay
	return &e
}

// As gives the Error in the chain of err. It reports false for an error that
// has none.
func As(err error) (*Error, bool) {
	var pointer *Error
	if errors.As(err, &pointer) && pointer != nil {
		return pointer, true
	}

	var value Error
	if errors.As(err, &value) {
		return &value, true
	}

	return nil, false
}

// From gives the Error of err: the Error in its chain, or ErrInternal for an
// error that has none. The cause of such an error is for the log and not for
// the caller, so the result does not hold it.
func From(err error) *Error {
	if found, ok := As(err); ok {
		return found
	}

	internal := ErrInternal
	return &internal
}
