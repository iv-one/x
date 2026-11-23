package errorsx

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/iv-one/x/errorsx/raise"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodeHasANameAndAnHTTPStatus(t *testing.T) {
	assert.Equal(t, "not_found", CodeNotFound.String())
	assert.Equal(t, http.StatusNotFound, CodeNotFound.HTTPStatus())

	// The two codes of a conflict have one status, and the code tells them apart.
	assert.Equal(t, http.StatusConflict, CodeAlreadyExists.HTTPStatus())
	assert.Equal(t, http.StatusConflict, CodeAborted.HTTPStatus())
	assert.Equal(t, http.StatusBadRequest, CodeFailedPrecondition.HTTPStatus())

	// Each code has a name of its own and a status.
	names := map[string]bool{}
	for code := CodeCanceled; code <= CodeUnauthenticated; code++ {
		assert.False(t, names[code.String()], code.String())
		names[code.String()] = true
		assert.GreaterOrEqual(t, code.HTTPStatus(), 400)
	}

	assert.Equal(t, "unknown", Code(99).String())
	assert.Equal(t, http.StatusInternalServerError, Code(99).HTTPStatus())
}

func TestWithReturnsACopy(t *testing.T) {
	base := ErrNotFound.WithReason("USER_NOT_FOUND")
	made := base.
		WithMessagef("User %s does not exist.", "ada").
		WithMetadata("user", "ada").
		WithViolation("user_id", "must name a user").
		WithRetryDelay(3 * time.Second)

	assert.Equal(t, CodeNotFound, made.Code)
	assert.Equal(t, "USER_NOT_FOUND", made.Reason)
	assert.Equal(t, "User ada does not exist.", made.Error())
	assert.Equal(t, map[string]string{"user": "ada"}, made.Metadata)
	assert.Equal(t, []Violation{{Field: "user_id", Description: "must name a user"}}, made.Violations)
	assert.Equal(t, 3*time.Second, made.RetryDelay)

	// The error that made is from does not change, and a second copy shares
	// no map and no slice with the first.
	assert.Nil(t, base.Metadata)
	assert.Nil(t, base.Violations)
	other := made.WithMetadata("account", "x").WithViolation("b", "c")
	assert.Len(t, made.Metadata, 1)
	assert.Len(t, made.Violations, 1)
	assert.Len(t, other.Metadata, 2)
	assert.Len(t, other.Violations, 2)

	assert.Nil(t, ErrNotFound.Metadata, "a base error has no metadata")
}

func TestReasonOrCode(t *testing.T) {
	assert.Equal(t, "USER_NOT_FOUND", ErrNotFound.WithReason("USER_NOT_FOUND").ReasonOrCode())
	assert.Equal(t, "PERMISSION_DENIED", ErrPermissionDenied.ReasonOrCode())
}

func TestIs(t *testing.T) {
	missingBody := ErrInvalidArgument.WithMessage("Missing request body.")
	badJSON := ErrInvalidArgument.WithMessage("Failed to decode request body.")

	require.ErrorIs(t, missingBody, ErrInvalidArgument, "an error with a message is still its base error")
	require.ErrorIs(t, missingBody.WithReason("X"), missingBody, "a refined error still matches its origin")
	require.ErrorIs(t, errors.Join(errors.New("wrapped"), missingBody), ErrInvalidArgument, "matches through wrapping")
	require.ErrorIs(t, raise.Error(missingBody), ErrInvalidArgument, "matches through raise")
	require.NotErrorIs(t, missingBody, badJSON, "different messages are different errors")
	require.NotErrorIs(t, ErrNotFound.WithReason("A"), ErrNotFound.WithReason("B"))
	require.NotErrorIs(t, ErrNotFound, ErrInvalidArgument)
	require.NotErrorIs(t, ErrAborted, ErrAlreadyExists, "the two codes of status 409 are different errors")
	require.NotErrorIs(t, ErrNotFound, errors.New("not found"))
	assert.False(t, ErrNotFound.Is((*Error)(nil)))
}

func TestFromAndAs(t *testing.T) {
	declared := ErrPermissionDenied.WithReason("USER_BLOCKED")

	found, ok := As(raise.Error(declared))
	require.True(t, ok)
	assert.Equal(t, "USER_BLOCKED", found.Reason)

	// A value in the chain is found as a pointer is.
	found, ok = As(errors.Join(errors.New("wrapped"), ErrNotFound))
	require.True(t, ok)
	assert.Equal(t, CodeNotFound, found.Code)

	_, ok = As(errors.New("plain"))
	assert.False(t, ok)
	_, ok = As(nil)
	assert.False(t, ok)

	assert.Equal(t, "USER_BLOCKED", From(raise.Error(declared)).Reason)

	// An error with no Error in its chain is internal, and its text does not
	// get to the caller.
	internal := From(errors.New("the password of the database is hunter2"))
	assert.Equal(t, CodeInternal, internal.Code)
	assert.NotContains(t, internal.Error(), "hunter2")
	assert.Equal(t, ErrInternal.Error(), internal.Error())
	assert.NotEmpty(t, internal.Error(), "an error with no message has the message of its code")
}
