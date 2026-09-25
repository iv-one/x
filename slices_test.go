package x

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	assert.Equal(t, []string{"1", "2"}, Map([]int{1, 2}, strconv.Itoa))
	assert.Equal(t, []string{}, Map(nil, strconv.Itoa))
}

func TestFlatMap(t *testing.T) {
	got := FlatMap([]int{1, 2, 0}, func(n int) []int {
		res := make([]int, n)
		for i := range res {
			res[i] = n
		}
		return res
	})
	assert.Equal(t, []int{1, 2, 2}, got)
}

func TestUniq(t *testing.T) {
	assert.Equal(t, []string{"b", "a", "c"}, Uniq([]string{"b", "a", "b", "c", "a"}))
	assert.Equal(t, []string{}, Uniq[string](nil))
}
