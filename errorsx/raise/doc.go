// Package raise wraps errors with the caller's location.
//
// Use [Error] in place of fmt.Errorf("...: %w", err) so every returned error
// carries a stack of the frames it passed through:
//
//	return raise.Error(err)
package raise
