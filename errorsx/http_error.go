package errorsx

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
)

// HTTPError is the HTTP error response.
type HTTPError struct {
	// The error ID
	//
	// Useful when trying to identify various errors in application logic.
	IDField string `json:"id,omitempty"`

	// The status code
	//
	// example: 404
	CodeField int `json:"code,omitempty"`

	// The status description
	//
	// example: Not Found
	StatusField string `json:"status,omitempty"`

	// The request ID
	//
	// The request ID is often exposed internally in order to trace
	// errors across service architectures. This is often a UUID.
	//
	// example: d7ef54b1-ec15-46e6-bccb-524b82c035e6
	RIDField string `json:"request,omitempty"`

	// A human-readable reason for the error
	//
	// example: User with ID 1234 does not exist.
	ReasonField string `json:"reason,omitempty"`

	// Further error details
	DetailsField map[string]any `json:"details,omitempty"`

	// Response headers the error carries
	//
	// Written to the response before the status line, never serialized into
	// the body (Retry-After, for one).
	HeadersField map[string]string `json:"-"`

	// Error message
	//
	// The error's message.
	//
	// example: The resource could not be found
	// required: true
	MessageField string `json:"message"`
}

// HTTPErrorResponse is the HTTP error response.
type HTTPErrorResponse struct {
	Error *HTTPError `json:"error"`
}

// WithID sets the ID of the error.
func (e HTTPError) WithID(id string) *HTTPError {
	e.IDField = id
	return &e
}

// Is reports whether err is the same HTTP error or one derived from it.
// HTTPError values are copied by every With* builder, so identity is never
// enough: the target acts as a template, and every field set on it must
// match. ErrBadRequest matches all bad requests; a WithReason variant matches
// only errors carrying that reason. Details are not compared.
func (e HTTPError) Is(err error) bool {
	var target HTTPError
	switch t := err.(type) {
	case HTTPError:
		target = t
	case *HTTPError:
		if t == nil {
			return false
		}
		target = *t
	default:
		return false
	}
	return (target.IDField == "" || target.IDField == e.IDField) &&
		(target.CodeField == 0 || target.CodeField == e.CodeField) &&
		(target.StatusField == "" || target.StatusField == e.StatusField) &&
		(target.RIDField == "" || target.RIDField == e.RIDField) &&
		(target.ReasonField == "" || target.ReasonField == e.ReasonField) &&
		(target.MessageField == "" || target.MessageField == e.MessageField)
}

// Status returns the status of the error.
func (e HTTPError) Status() string {
	return e.StatusField
}

// ID returns the ID of the error.
func (e HTTPError) ID() string {
	return e.IDField
}

// Error returns the error message.
func (e HTTPError) Error() string {
	return e.MessageField
}

// Message returns the error message.
func (e HTTPError) Message() string {
	return e.MessageField
}

// RequestID returns the request ID.
func (e HTTPError) RequestID() string {
	return e.RIDField
}

// Reason returns the reason for the error.
func (e HTTPError) Reason() string {
	return e.ReasonField
}

// Details returns the details of the error.
func (e HTTPError) Details() map[string]any {
	return e.DetailsField
}

// StatusCode returns the status code of the error.
func (e HTTPError) StatusCode() int {
	return e.CodeField
}

// Headers returns the response headers of the error.
func (e HTTPError) Headers() map[string]string {
	return e.HeadersField
}

// WithHeader sets a response header on the error. The map is copied, so a
// package-level error stays free of the headers its variants carry.
func (e HTTPError) WithHeader(name, value string) *HTTPError {
	headers := maps.Clone(e.HeadersField)
	if headers == nil {
		headers = map[string]string{}
	}
	headers[name] = value
	e.HeadersField = headers
	return &e
}

// WithReason sets the reason for the error.
func (e HTTPError) WithReason(reason string) *HTTPError {
	e.ReasonField = reason
	return &e
}

// WithReasonf sets the reason for the error.
func (e HTTPError) WithReasonf(reason string, args ...any) *HTTPError {
	return e.WithReason(fmt.Sprintf(reason, args...))
}

// WithMessage sets the message for the error.
func (e HTTPError) WithMessage(message string) *HTTPError {
	e.MessageField = message
	return &e
}

// WithMessagef sets the message for the error.
func (e HTTPError) WithMessagef(message string, args ...any) *HTTPError {
	return e.WithMessage(fmt.Sprintf(message, args...))
}

// WithDetail sets the detail for the error.
func (e HTTPError) WithDetail(key string, detail any) *HTTPError {
	if e.DetailsField == nil {
		e.DetailsField = map[string]any{}
	}
	e.DetailsField[key] = detail
	return &e
}

// WithDetailf sets the detail for the error.
func (e HTTPError) WithDetailf(key string, message string, args ...any) *HTTPError {
	if e.DetailsField == nil {
		e.DetailsField = map[string]any{}
	}
	e.DetailsField[key] = fmt.Sprintf(message, args...)
	return &e
}

// Format formats the error.
func (e HTTPError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		_, _ = io.WriteString(s, e.MessageField)
		return
	case 'x':
		_, _ = fmt.Fprintf(s, "id=%s\n", e.IDField)
		_, _ = fmt.Fprintf(s, "rid=%s\n", e.RIDField)
		_, _ = fmt.Fprintf(s, "error=%s\n", e.MessageField)
		_, _ = fmt.Fprintf(s, "reason=%s\n", e.ReasonField)
		_, _ = fmt.Fprintf(s, "details=%+v\n", e.DetailsField)
		fallthrough
	case 's':
		_, _ = io.WriteString(s, e.MessageField)
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.MessageField)
	}
}

// ToHTTPError converts the error to an HTTP error.
func ToHTTPError(err error, requestID string) *HTTPError {
	de := &HTTPError{
		RIDField:     requestID,
		CodeField:    http.StatusInternalServerError,
		DetailsField: map[string]any{},
	}

	if c := ReasonCarrier(nil); errors.As(err, &c) {
		de.ReasonField = c.Reason()
	}
	if c := RequestIDCarrier(nil); errors.As(err, &c) && c.RequestID() != "" {
		de.RIDField = c.RequestID()
	}
	if c := DetailsCarrier(nil); errors.As(err, &c) && c.Details() != nil {
		de.DetailsField = c.Details()
	}
	if c := HeadersCarrier(nil); errors.As(err, &c) && c.Headers() != nil {
		de.HeadersField = c.Headers()
	}
	if c := StatusCarrier(nil); errors.As(err, &c) && c.Status() != "" {
		de.StatusField = c.Status()
	}
	if c := StatusCodeCarrier(nil); errors.As(err, &c) && c.StatusCode() != 0 {
		de.CodeField = c.StatusCode()
	}
	if c := IDCarrier(nil); errors.As(err, &c) {
		de.IDField = c.ID()
	}
	if c := MessageCarrier(nil); errors.As(err, &c) {
		de.MessageField = c.Message()
	}

	if de.StatusField == "" {
		de.StatusField = http.StatusText(de.StatusCode())
	}

	return de
}

// StatusCodeCarrier can be implemented by an error to support setting status codes in the error itself.
type StatusCodeCarrier interface {
	// StatusCode returns the status code of this error.
	StatusCode() int
}

// RequestIDCarrier can be implemented by an error to support error contexts.
type RequestIDCarrier interface {
	// RequestID returns the ID of the request that caused the error, if applicable.
	RequestID() string
}

// ReasonCarrier can be implemented by an error to support error contexts.
type ReasonCarrier interface {
	// Reason returns the reason for the error, if applicable.
	Reason() string
}

// DebugCarrier can be implemented by an error to support error contexts.
type DebugCarrier interface {
	// Debug returns debugging information for the error, if applicable.
	Debug() string
}

// StatusCarrier can be implemented by an error to support error contexts.
type StatusCarrier interface {
	// ID returns the error id, if applicable.
	Status() string
}

// DetailsCarrier can be implemented by an error to support error contexts.
type DetailsCarrier interface {
	// Details returns details on the error, if applicable.
	Details() map[string]any
}

// HeadersCarrier can be implemented by an error to support response headers.
type HeadersCarrier interface {
	// Headers returns the response headers the error carries, if applicable.
	Headers() map[string]string
}

// IDCarrier can be implemented by an error to support error contexts.
type IDCarrier interface {
	// ID returns application error ID on the error, if applicable.
	ID() string
}

// MessageCarrier can be implemented by an error to support error contexts.
type MessageCarrier interface {
	// ID returns application error ID on the error, if applicable.
	Message() string
}

// IsHTTPError checks if the error is an HTTP error.
func IsHTTPError(err error) bool {
	var e *HTTPError
	return errors.As(err, &e)
}
