package errorsx

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPErrorIs(t *testing.T) {
	missingBody := ErrBadRequest.WithReason("Missing request body.")
	badJSON := ErrBadRequest.WithReason("Failed to decode request body.")

	require.ErrorIs(t, missingBody, ErrBadRequest, "a reason variant is still its base error")
	require.ErrorIs(t, missingBody.WithID("x"), missingBody, "a refined error still matches its origin")
	require.ErrorIs(t, errors.Join(errors.New("wrapped"), missingBody), ErrBadRequest, "matches through wrapping")
	require.NotErrorIs(t, missingBody, badJSON, "different reasons are different errors")
	require.NotErrorIs(t, ErrBadRequest, missingBody, "the base does not carry the reason")
	require.NotErrorIs(t, ErrNotFound, ErrBadRequest)
	require.NotErrorIs(t, ErrNotFound, errors.New("not found"))
	assert.False(t, ErrNotFound.Is((*HTTPError)(nil)))
}
