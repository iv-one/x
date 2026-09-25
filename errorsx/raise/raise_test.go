package raise

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	ErrA = errors.New("a")
	ErrB = errors.New("b")
	ErrC = errors.New("c")
)

func TestError(t *testing.T) {
	err := Error(ErrA)
	assert.Equal(t, "a", err.Error())

	formatted := fmt.Sprintf("%v", err)
	assert.Equal(t, "\n<raise/raise_test.go:19> a", formatted)

	err2 := Error(err)
	err3 := Error(err2)
	assert.Equal(t, "a", err3.Error())

	formatted = fmt.Sprintf("%v", err3)
	assert.Equal(t, "\n<raise/raise_test.go:26> \n<raise/raise_test.go:25> \n<raise/raise_test.go:19> a", formatted)
}

func TestErrors(t *testing.T) {
	err := Errors(ErrA, ErrB)
	assert.Equal(t, "a: b", err.Error())

	formatted := fmt.Sprintf("%v", err)
	assert.Equal(t, "\n<raise/raise_test.go:34> a b", formatted)

	errB := Error(ErrB)
	err = Errors(ErrA, errB)
	formatted = fmt.Sprintf("%v", err)
	assert.Equal(t, "\n<raise/raise_test.go:41> a \n<raise/raise_test.go:40> b", formatted)
}

func TestStd(t *testing.T) {
	err := Error(ErrA)
	require.ErrorIs(t, err, ErrA)

	err = Errors(ErrA, ErrB)
	require.ErrorIs(t, err, ErrA)
	require.ErrorIs(t, err, ErrB)
}

func TestDeepUnwrap(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		assert.NoError(t, DeepUnwrap(nil))
	})

	t.Run("unwrapped error returns itself", func(t *testing.T) {
		result := DeepUnwrap(ErrA)
		assert.Equal(t, ErrA, result)
	})

	t.Run("single wrap returns inner error", func(t *testing.T) {
		// nolint: intentionally using fmt.Errorf to test stdlib compatibility
		wrapped := fmt.Errorf("outer: %w", ErrA)
		result := DeepUnwrap(wrapped)
		assert.Equal(t, ErrA, result)
	})

	t.Run("deeply nested returns innermost", func(t *testing.T) {
		// nolint: intentionally using fmt.Errorf to test stdlib compatibility
		wrap1 := fmt.Errorf("wrap1: %w", ErrA)
		wrap2 := fmt.Errorf("wrap2: %w", wrap1)
		wrap3 := fmt.Errorf("wrap3: %w", wrap2)

		result := DeepUnwrap(wrap3)
		assert.Equal(t, ErrA, result)
	})

	t.Run("works with raise.Error", func(t *testing.T) {
		wrapped := Error(Error(Error(ErrA)))
		result := DeepUnwrap(wrapped)
		assert.Equal(t, ErrA, result)
	})

	t.Run("works with raise.Errors", func(t *testing.T) {
		wrapped := Errors(ErrA, Errors(ErrB, ErrC))
		result := DeepUnwrap(wrapped)
		assert.Equal(t, ErrC, result)
	})
}

func TestErrorsNilHandling(t *testing.T) {
	t.Run("both nil returns nil", func(t *testing.T) {
		assert.NoError(t, Errors(nil, nil))
	})

	t.Run("err nil returns wrapped cause", func(t *testing.T) {
		result := Errors(nil, ErrA)
		require.Error(t, result)
		assert.Equal(t, "a", result.Error())
		require.ErrorIs(t, result, ErrA)
	})

	t.Run("cause nil returns wrapped err", func(t *testing.T) {
		result := Errors(ErrA, nil)
		require.Error(t, result)
		assert.Equal(t, "a", result.Error())
		require.ErrorIs(t, result, ErrA)
	})

	t.Run("both non-nil returns withCause", func(t *testing.T) {
		result := Errors(ErrA, ErrB)
		require.Error(t, result)
		assert.Equal(t, "a: b", result.Error())
		require.ErrorIs(t, result, ErrA)
		require.ErrorIs(t, result, ErrB)
	})
}
