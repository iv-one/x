package errorsx

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/iv-one/x/errorsx/raise"
	"github.com/stretchr/testify/assert"
)

func TestToHTTPError(t *testing.T) {
	t.Run("case=status", func(t *testing.T) {
		e := &HTTPError{
			StatusField: "foo-status",
		}
		assert.Equal(t, "foo-status", ToHTTPError(e, "").Status())
	})

	t.Run("case=reason", func(t *testing.T) {
		e := &HTTPError{
			ReasonField: "foo-reason",
		}
		assert.Equal(t, "foo-reason", ToHTTPError(e, "").Reason())
	})

	t.Run("case=details", func(t *testing.T) {
		e := &HTTPError{
			DetailsField: map[string]any{"foo-debug": "bar"},
		}
		assert.Equal(t, map[string]any{"foo-debug": "bar"}, ToHTTPError(e, "").Details())
	})

	t.Run("case=rid", func(t *testing.T) {
		e := &HTTPError{
			RIDField: "foo-rid",
		}
		assert.Equal(t, "foo-rid", ToHTTPError(e, "").RequestID())
		assert.Equal(t, "fallback-rid", ToHTTPError(new(HTTPError), "fallback-rid").RequestID())
	})

	t.Run("case=id", func(t *testing.T) {
		e := &HTTPError{
			IDField: "foo-rid",
		}
		assert.Equal(t, "foo-rid", ToHTTPError(e, "").ID())
	})

	t.Run("case=code", func(t *testing.T) {
		e := &HTTPError{CodeField: 501}
		assert.Equal(t, 501, ToHTTPError(e, "").StatusCode())
		assert.Equal(t, http.StatusText(http.StatusNotImplemented), ToHTTPError(e, "").Status())

		e = &HTTPError{CodeField: 501, StatusField: "foobar"}
		assert.Equal(t, 501, ToHTTPError(e, "").StatusCode())
		assert.Equal(t, "foobar", ToHTTPError(e, "").Status())

		assert.Equal(t, 500, ToHTTPError(errors.New(""), "").StatusCode())
	})

	t.Run("case=cast", func(t *testing.T) {
		cause := raise.Error(errors.New("cause"))
		err := raise.Errors((&HTTPError{CodeField: 404}).WithReason("Details"), cause)

		httpErr := ToHTTPError(err, "")
		assert.Equal(t, 404, httpErr.StatusCode())
		assert.Equal(t, "Details", httpErr.Reason())

		formatted := fmt.Sprintf("%v", err)
		assert.Equal(t, "\n<errorsx/http_error_test.go:64>  \n<errorsx/http_error_test.go:63> cause", formatted)
	})
}
