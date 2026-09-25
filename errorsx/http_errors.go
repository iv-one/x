package errorsx

import (
	"net/http"
)

// ErrNotFound is the error for not found.
var ErrNotFound = HTTPError{
	StatusField:  http.StatusText(http.StatusNotFound),
	MessageField: "The requested resource could not be found",
	CodeField:    http.StatusNotFound,
}

// ErrUnauthorized is the error for unauthorized.
var ErrUnauthorized = HTTPError{
	StatusField:  http.StatusText(http.StatusUnauthorized),
	MessageField: "The request could not be authorized",
	CodeField:    http.StatusUnauthorized,
}

// ErrForbidden is the error for forbidden.
var ErrForbidden = HTTPError{
	StatusField:  http.StatusText(http.StatusForbidden),
	MessageField: "The requested action was forbidden",
	CodeField:    http.StatusForbidden,
}

// ErrInternalServerError is the error for internal server error.
var ErrInternalServerError = HTTPError{
	StatusField:  http.StatusText(http.StatusInternalServerError),
	MessageField: "An internal server error occurred, please contact the system administrator",
	CodeField:    http.StatusInternalServerError,
}

// ErrBadRequest is the error for bad request.
var ErrBadRequest = HTTPError{
	StatusField:  http.StatusText(http.StatusBadRequest),
	MessageField: "The request was malformed or contained invalid parameters",
	CodeField:    http.StatusBadRequest,
}

// ErrUnsupportedMediaType is the error for unsupported media type.
var ErrUnsupportedMediaType = HTTPError{
	StatusField:  http.StatusText(http.StatusUnsupportedMediaType),
	MessageField: "The request is using an unknown content type",
	CodeField:    http.StatusUnsupportedMediaType,
}

// ErrConflict is the error for conflict.
var ErrConflict = HTTPError{
	StatusField:  http.StatusText(http.StatusConflict),
	MessageField: "The resource could not be created due to a conflict",
	CodeField:    http.StatusConflict,
}

// ErrRequestEntityTooLarge is the error for a request body over the limit.
var ErrRequestEntityTooLarge = HTTPError{
	StatusField:  http.StatusText(http.StatusRequestEntityTooLarge),
	MessageField: "The request body is larger than the server accepts",
	CodeField:    http.StatusRequestEntityTooLarge,
}

// ErrTooManyRequests is the error for a rate-limited request.
var ErrTooManyRequests = HTTPError{
	StatusField:  http.StatusText(http.StatusTooManyRequests),
	MessageField: "The request was rate limited",
	CodeField:    http.StatusTooManyRequests,
}
