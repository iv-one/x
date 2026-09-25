package x

// ReduceOptions reduces the options.
func ReduceOptions[T any](init *T, opts ...func(*T)) *T {
	for _, opt := range opts {
		opt(init)
	}
	return init
}
