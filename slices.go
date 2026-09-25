package x

// Map returns a new slice with fn applied to every element of items.
func Map[T, R any](items []T, fn func(T) R) []R {
	res := make([]R, len(items))
	for i, item := range items {
		res[i] = fn(item)
	}
	return res
}

// FlatMap returns a new slice with the slices returned by fn concatenated
// in order.
func FlatMap[T, R any](items []T, fn func(T) []R) []R {
	res := make([]R, 0, len(items))
	for _, item := range items {
		res = append(res, fn(item)...)
	}
	return res
}

// Uniq returns the distinct elements of items, keeping first occurrences in
// their original order.
func Uniq[T comparable](items []T) []T {
	seen := make(map[T]struct{}, len(items))
	res := make([]T, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		res = append(res, item)
	}
	return res
}
