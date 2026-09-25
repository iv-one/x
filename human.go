package x

import (
	"fmt"
	"strconv"
	"strings"
)

// HumanBytes renders a size the way an operator reads one: powers of 1024 with
// single-letter units, because these are file sizes on a disk and in a bucket
// rather than a marketing number.
func HumanBytes(n int64) string {
	const unit = 1024

	if n < unit {
		return strconv.FormatInt(n, 10) + "B"
	}

	size := float64(n)

	for _, suffix := range []string{"K", "M", "G", "T", "P"} {
		size /= unit

		if size < unit {
			return fmt.Sprintf("%.1f%s", size, suffix)
		}
	}

	return fmt.Sprintf("%.1fE", size/unit)
}

// HumanCount renders a count with thousands separators, so six figures is
// legible at a glance.
func HumanCount(n uint64) string {
	digits := strconv.FormatUint(n, 10)
	if len(digits) <= 3 {
		return digits
	}

	lead := len(digits) % 3
	if lead == 0 {
		lead = 3
	}

	var out strings.Builder

	out.WriteString(digits[:lead])

	for i := lead; i < len(digits); i += 3 {
		out.WriteString(",")
		out.WriteString(digits[i : i+3])
	}

	return out.String()
}
