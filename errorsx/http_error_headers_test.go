package errorsx

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/iv-one/x/errorsx/raise"
)

func TestWithHeader(t *testing.T) {
	t.Run("case=copies", func(t *testing.T) {
		base := HTTPError{MessageField: "base"}
		first := base.WithHeader("Retry-After", "7")

		assert.Nil(t, base.Headers(), "the receiver keeps no headers")
		assert.Equal(t, map[string]string{"Retry-After": "7"}, first.Headers())

		second := first.WithHeader("Retry-After", "9")
		assert.Equal(t, "7", first.Headers()["Retry-After"], "a variant leaves its origin alone")
		assert.Equal(t, "9", second.Headers()["Retry-After"])
	})

	t.Run("case=accumulates", func(t *testing.T) {
		err := HTTPError{}.WithHeader("Retry-After", "7").WithHeader("X-Test", "on")
		assert.Equal(t, map[string]string{"Retry-After": "7", "X-Test": "on"}, err.Headers())
	})

	t.Run("case=not_in_body", func(t *testing.T) {
		body, err := json.Marshal(ErrTooManyRequests.WithHeader("Retry-After", "7"))
		require.NoError(t, err)
		assert.NotContains(t, string(body), "Retry-After", "headers ride beside the body, not in it")
	})
}

func TestToHTTPErrorCarriesHeaders(t *testing.T) {
	t.Run("case=carried", func(t *testing.T) {
		err := raise.Error(ErrTooManyRequests.WithHeader("Retry-After", "7"))
		assert.Equal(t, "7", ToHTTPError(err, "").Headers()["Retry-After"])
		assert.Equal(t, http.StatusTooManyRequests, ToHTTPError(err, "").StatusCode())
	})

	t.Run("case=absent", func(t *testing.T) {
		assert.Nil(t, ToHTTPError(errors.New("plain"), "").Headers())
	})
}
