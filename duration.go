package x

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/iv-one/x/errorsx/raise"
)

// ParseDuration parses a duration string into a time.Duration.
func ParseDuration(durationStr string) (time.Duration, error) {
	const (
		day  = time.Hour * 24
		week = day * 7
	)

	if strings.Contains(durationStr, "mo") {
		months, err := strconv.Atoi(strings.TrimSuffix(durationStr, "mo"))
		if err != nil {
			return 0, raise.Error(err)
		}
		return time.Duration(months) * 30 * day, nil
	}

	if strings.Contains(durationStr, "w") {
		weeks, err := strconv.Atoi(strings.TrimSuffix(durationStr, "w"))
		if err != nil {
			return 0, raise.Error(err)
		}
		return time.Duration(weeks) * week, nil
	}

	if strings.Contains(durationStr, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(durationStr, "d"))
		if err != nil {
			return 0, raise.Error(err)
		}
		return time.Duration(days) * day, nil
	}

	return time.ParseDuration(durationStr)
}

// FormatDuration formats a duration into a string.
func FormatDuration(d time.Duration) string {
	hours := int(d.Hours())
	days := hours / 24

	// Check for weeks
	if days == 7 {
		return "1 week"
	}

	if days > 7 {
		weeks := days / 7
		return fmt.Sprintf("%d weeks", weeks)
	}

	if days == 1 {
		return "1 day"
	}

	if days > 1 {
		return fmt.Sprintf("%d days", days)
	}

	if hours == 1 {
		return "1 hour"
	}

	if hours > 1 {
		return fmt.Sprintf("%d hours", hours)
	}

	minutes := int(d.Minutes())
	if minutes == 1 {
		return "1 minute"
	}

	return fmt.Sprintf("%d minutes", minutes)
}
