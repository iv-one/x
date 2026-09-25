package x

import (
	"errors"
	"strconv"
	"strings"

	"github.com/iv-one/x/errorsx/raise"
)

// ParseMem parses a memory string into a memory size.
func ParseMem(str string) (int64, error) {
	str = strings.ToLower(strings.TrimSpace(str))
	var multiplier int64

	switch {
	case strings.HasSuffix(str, "kb"):
		multiplier = 1024
		str = strings.TrimSuffix(str, "kb")
	case strings.HasSuffix(str, "mb"):
		multiplier = 1024 * 1024
		str = strings.TrimSuffix(str, "mb")
	case strings.HasSuffix(str, "gb"):
		multiplier = 1024 * 1024 * 1024
		str = strings.TrimSuffix(str, "gb")
	default:
		return 0, errors.New("invalid memory unit: must be kb, mb, or gb")
	}

	value, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, raise.Error(err)
	}

	return value * multiplier, nil
}
