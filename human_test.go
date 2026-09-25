package x

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHumanBytes(t *testing.T) {
	sizes := map[int64]string{
		0:             "0B",
		1:             "1B",
		1023:          "1023B",
		1024:          "1.0K",
		1536:          "1.5K",
		1 << 20:       "1.0M",
		1 << 30:       "1.0G",
		1 << 40:       "1.0T",
		1 << 50:       "1.0P",
		1 << 60:       "1.0E",
		5 * (1 << 20): "5.0M",
	}

	for size, want := range sizes {
		assert.Equal(t, want, HumanBytes(size), "%d", size)
	}
}

func TestHumanCount(t *testing.T) {
	counts := map[uint64]string{
		0:          "0",
		7:          "7",
		999:        "999",
		1000:       "1,000",
		12345:      "12,345",
		999999:     "999,999",
		1000000:    "1,000,000",
		1234567890: "1,234,567,890",
	}

	for count, want := range counts {
		assert.Equal(t, want, HumanCount(count), "%d", count)
	}
}
