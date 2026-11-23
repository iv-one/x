package errorsx

// The base errors, one for each code that an application answers. A base
// error has no reason and no message of its own, so it matches each error
// that is made from it (see [Error.Is]).
var (
	// ErrInvalidArgument is for a request that is not correct, whatever the
	// state of the system.
	ErrInvalidArgument = Error{Code: CodeInvalidArgument}

	// ErrOutOfRange is for a value that is outside the range that the
	// system accepts, such as a body that is too large.
	ErrOutOfRange = Error{Code: CodeOutOfRange}

	// ErrFailedPrecondition is for a request that the state of the system
	// refuses. The caller must change the state before it sends the request
	// again.
	ErrFailedPrecondition = Error{Code: CodeFailedPrecondition}

	// ErrUnauthenticated is for a request with no valid credential.
	ErrUnauthenticated = Error{Code: CodeUnauthenticated}

	// ErrPermissionDenied is for a caller that has no permission for the
	// request.
	ErrPermissionDenied = Error{Code: CodePermissionDenied}

	// ErrNotFound is for a resource that does not exist.
	ErrNotFound = Error{Code: CodeNotFound}

	// ErrAlreadyExists is for a resource that the request cannot create
	// because it exists.
	ErrAlreadyExists = Error{Code: CodeAlreadyExists}

	// ErrAborted is for a request that lost to a concurrent change. The
	// caller can send it again.
	ErrAborted = Error{Code: CodeAborted}

	// ErrResourceExhausted is for a request over a limit, such as a rate
	// limit.
	ErrResourceExhausted = Error{Code: CodeResourceExhausted}

	// ErrUnimplemented is for an operation that the system does not have.
	ErrUnimplemented = Error{Code: CodeUnimplemented}

	// ErrUnavailable is for a system that cannot answer now. The caller can
	// send the request again later.
	ErrUnavailable = Error{Code: CodeUnavailable}

	// ErrInternal is for a fault of the system. The caller gets no cause.
	ErrInternal = Error{Code: CodeInternal}
)
