package raise

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type callResult struct {
	frame Frame
}

func call() *callResult {
	return &callResult{frame: caller()}
}

func TestCaller(t *testing.T) {
	res := call()
	str := fmt.Sprintf("%t", res.frame)
	assert.Equal(t, "raise/stack_test.go:19", str)
}
