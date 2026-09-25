package imagex

import "errors"

// The four ways an upload is refused. Separate values rather than one coded
// error because each maps to a different HTTP status.
var (
	// ErrTooLarge is over the configured byte cap (413).
	ErrTooLarge = errors.New("imagex: upload exceeds the maximum size")

	// ErrUnsupportedFormat is not an accepted input format, SVG and ICO
	// included (415).
	ErrUnsupportedFormat = errors.New("imagex: unsupported image format")

	// ErrDimensions is over an accept bound, raised from the header before any
	// pixel is decoded since the bound is a memory limit; §3.8 answers 400.
	ErrDimensions = errors.New("imagex: image dimensions exceed the accepted bounds")

	// ErrMalformed is a recognized format that does not decode, decoder panic
	// included; §3.8 answers 400.
	ErrMalformed = errors.New("imagex: malformed image")
)
