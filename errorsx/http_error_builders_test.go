package errorsx

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuilders(t *testing.T) {
	base := HTTPError{IDField: "bad", MessageField: "base"}

	err := base.WithReasonf("field %s", "name").
		WithMessagef("got %d", 3).
		WithDetail("field", "name").
		WithDetailf("hint", "use %q", "x")

	assert.Equal(t, "field name", err.Reason())
	assert.Equal(t, "got 3", err.Error())
	assert.Equal(t, "got 3", err.Message())
	assert.Equal(t, map[string]any{"field": "name", "hint": `use "x"`}, err.Details())

	assert.Equal(t, "base", base.Message(), "builders leave the receiver alone")
	assert.Empty(t, base.Reason())
	assert.Nil(t, base.Details())
	assert.Equal(t, "base", base.WithMessage("base").Error())
}

func TestFormat(t *testing.T) {
	err := HTTPError{IDField: "bad", RIDField: "r1", MessageField: "boom", ReasonField: "why"}

	assert.Equal(t, "boom", fmt.Sprintf("%v", err))
	assert.Equal(t, "boom", fmt.Sprintf("%s", err))
	assert.Equal(t, `"boom"`, fmt.Sprintf("%q", err))
	assert.Equal(t, "id=bad\nrid=r1\nerror=boom\nreason=why\ndetails=map[]\nboom", fmt.Sprintf("%x", err))
}

func TestIsHTTPError(t *testing.T) {
	assert.True(t, IsHTTPError(errors.Join(errors.New("context"), ErrBadRequest.WithReason("x"))))
	assert.False(t, IsHTTPError(errors.New("plain")))
}
