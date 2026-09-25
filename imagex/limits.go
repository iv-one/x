package imagex

// Two families of limit, which must not be conflated. Accept bounds decide what may be decoded (memory
// safety, deliberately generous). Canonical bounds decide what is stored, and so
// what every viewer downloads, since there are no variants.
const (
	// MaxAcceptWidth and MaxAcceptHeight bound each edge this package will
	// decode, generous enough for a phone photograph (4032 wide).
	MaxAcceptWidth  = 4096
	MaxAcceptHeight = 4096

	// MaxAcceptPixels bounds the decoded bitmap: 4096x4096 is 64 MiB, and
	// compressed size bounds nothing. Currently implied by the edge bounds, so it
	// never binds; kept because it states the real concern, and pinned by
	// TestAcceptBoundsAgree so the coincidence cannot rot unnoticed.
	MaxAcceptPixels = 16777216

	// MaxCanonicalShortEdge governs, because avatars and logos are displayed
	// height-bound. MaxCanonicalLongEdge is a sanity cap: governing on it
	// instead would crush a wide wordmark.
	MaxCanonicalShortEdge = 512
	MaxCanonicalLongEdge  = 2048

	// JPEGQuality is part of the stored bytes' identity, like the encoder and
	// scaler: changing it changes every future identifier.
	JPEGQuality = 85
)

// The two stored encodings. WebP is accepted on input and never emitted: the
// standard library has no encoder, so a WebP upload yields a PNG or JPEG
// instead.
const (
	ContentTypePNG  = "image/png"  // transparency
	ContentTypeJPEG = "image/jpeg" // fully opaque

	ExtPNG  = "png"
	ExtJPEG = "jpg"
)
