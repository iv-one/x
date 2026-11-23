package errorsx

import "net/http"

// Code is a canonical error code. The values are the values of
// google.rpc.Code, so a Code converts to the code of gRPC or of Connect with
// no table.
type Code uint32

// The canonical codes. OK is not an error and has no constant.
const (
	CodeCanceled           Code = 1
	CodeUnknown            Code = 2
	CodeInvalidArgument    Code = 3
	CodeDeadlineExceeded   Code = 4
	CodeNotFound           Code = 5
	CodeAlreadyExists      Code = 6
	CodePermissionDenied   Code = 7
	CodeResourceExhausted  Code = 8
	CodeFailedPrecondition Code = 9
	CodeAborted            Code = 10
	CodeOutOfRange         Code = 11
	CodeUnimplemented      Code = 12
	CodeInternal           Code = 13
	CodeUnavailable        Code = 14
	CodeDataLoss           Code = 15
	CodeUnauthenticated    Code = 16
)

// codes gives each code its name, its HTTP status and the message of an
// error that has none. The statuses are the ones of the Connect protocol.
var codes = map[Code]struct {
	name    string
	status  int
	message string
}{
	CodeCanceled:           {"canceled", 499, "The request was canceled"},
	CodeUnknown:            {"unknown", http.StatusInternalServerError, "An unknown error occurred"},
	CodeInvalidArgument:    {"invalid_argument", http.StatusBadRequest, "The request was malformed or contained invalid parameters"},
	CodeDeadlineExceeded:   {"deadline_exceeded", http.StatusGatewayTimeout, "The request took too long"},
	CodeNotFound:           {"not_found", http.StatusNotFound, "The requested resource could not be found"},
	CodeAlreadyExists:      {"already_exists", http.StatusConflict, "The resource already exists"},
	CodePermissionDenied:   {"permission_denied", http.StatusForbidden, "The requested action was forbidden"},
	CodeResourceExhausted:  {"resource_exhausted", http.StatusTooManyRequests, "The request was rate limited"},
	CodeFailedPrecondition: {"failed_precondition", http.StatusBadRequest, "The system is not in the state that the request needs"},
	CodeAborted:            {"aborted", http.StatusConflict, "The request was aborted by a concurrent change"},
	CodeOutOfRange:         {"out_of_range", http.StatusBadRequest, "A value of the request is outside the accepted range"},
	CodeUnimplemented:      {"unimplemented", http.StatusNotImplemented, "The operation is not implemented"},
	CodeInternal:           {"internal", http.StatusInternalServerError, "An internal server error occurred, please contact the system administrator"},
	CodeUnavailable:        {"unavailable", http.StatusServiceUnavailable, "The service is not available, try again later"},
	CodeDataLoss:           {"data_loss", http.StatusInternalServerError, "Data was lost or damaged"},
	CodeUnauthenticated:    {"unauthenticated", http.StatusUnauthorized, "The request could not be authorized"},
}

// String is the name of the code in snake_case, such as `not_found`. A value
// that is not a code is `unknown`.
func (c Code) String() string {
	if code, ok := codes[c]; ok {
		return code.name
	}
	return codes[CodeUnknown].name
}

// HTTPStatus is the HTTP status of the code. A value that is not a code is
// status 500.
func (c Code) HTTPStatus() int {
	if code, ok := codes[c]; ok {
		return code.status
	}
	return http.StatusInternalServerError
}
