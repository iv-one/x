package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOffsetMovesForwardOnly(t *testing.T) {
	t.Cleanup(Reset)

	assert.WithinDuration(t, time.Now(), Now(), time.Second, "zero offset is wall time")
	assert.True(t, SetOffset(15*time.Minute))
	assert.Equal(t, 15*time.Minute, Offset())
	assert.WithinDuration(t, time.Now().Add(15*time.Minute), Now(), time.Second)

	assert.False(t, SetOffset(-time.Minute), "the clock never runs back")
	assert.False(t, SetOffset(10*time.Minute), "nor behind its current offset")
	assert.Equal(t, 15*time.Minute, Offset(), "a rejected offset leaves the clock alone")
	assert.True(t, SetOffset(20*time.Minute), "larger is fine")

	Reset()
	assert.Equal(t, time.Duration(0), Offset())
}
