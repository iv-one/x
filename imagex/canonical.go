package imagex

import (
	"bytes"
	"image"
	"image/jpeg" // also registers the JPEG decoder for image.Decode
	"image/png"  // also registers the PNG decoder for image.Decode
	"io"
	"math"

	"golang.org/x/image/draw"

	// The only decoder that has to be imported for its side effect alone: WebP
	// is accepted on input and never emitted, so nothing else here names it.
	_ "golang.org/x/image/webp"

	"github.com/iv-one/x/errorsx/raise"
)

// Image is an accepted upload, scaled and re-encoded.
//
// Bytes are this service's encoder output, never the upload. Only pixels
// survive: EXIF, color profiles and trailing data are gone by construction.
type Image struct {
	// Bytes is the canonical encoding, and the input to the asset identifier.
	Bytes []byte
	// ContentType is the stored type, never the uploaded one.
	ContentType string
	// Ext is the file extension matching ContentType, carried in the asset
	// identifier so retrieval needs no metadata lookup.
	Ext string
	// Width and Height are the dimensions of the canonical image, after scaling.
	Width, Height int
}

// pngEncoder is pinned: compression level is part of the stored bytes' identity,
// so changing it changes every future identifier.
var pngEncoder = png.Encoder{CompressionLevel: png.DefaultCompression}

// scaler is pinned for the same reason as pngEncoder.
var scaler = draw.CatmullRom

// Canonical reads an upload and returns the bytes to store, or one of
// ErrTooLarge, ErrUnsupportedFormat, ErrDimensions, ErrMalformed.
//
// Step order is contract: the header is read before
// decoding because a decoder sizes its pixel buffer from it, and the resulting
// OOM is fatal rather than recoverable.
//
// A read error from r and an encoder failure are deliberately none of the four:
// neither judges the upload, so §3.8 callers should treat them as internal.
//
// Reads at most maxBytes+1 from r; the extra byte separates "at the cap" from
// "over it". ReadCapped and CanonicalBytes are its two halves, for a caller
// that bounds the read before it can afford the decode.
func Canonical(r io.Reader, maxBytes int64) (Image, error) {
	buf, err := ReadCapped(r, maxBytes)
	if err != nil {
		return Image{}, err
	}

	return CanonicalBytes(buf)
}

// CanonicalBytes is Canonical over bytes already read.
func CanonicalBytes(buf []byte) (Image, error) {
	if !accepted(buf) {
		return Image{}, raise.Error(ErrUnsupportedFormat)
	}

	if err := checkDimensions(buf); err != nil {
		return Image{}, err
	}

	img, err := decode(buf)
	if err != nil {
		return Image{}, err
	}

	// Decided before scaling: the destination type scaling picks would otherwise
	// decide the stored format, making every scaled photograph a PNG.
	alpha := hasAlpha(img)

	return encode(scale(img, alpha), alpha)
}

// ReadCapped reads r into memory, refusing an upload over the cap. Bounded
// before anything is read, so a gigabyte claim costs maxBytes+1.
func ReadCapped(r io.Reader, maxBytes int64) ([]byte, error) {
	buf, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		// Not one of the four: the upload never arrived, so nothing was judged.
		// ErrMalformed here would answer a dropped connection with 400.
		return nil, raise.Error(err)
	}

	if int64(len(buf)) > maxBytes {
		return nil, raise.Error(ErrTooLarge)
	}

	return buf, nil
}

// checkDimensions reads the image header and refuses anything over an accept
// bound, before any pixel is decoded.
func checkDimensions(buf []byte) error {
	cfg, err := decodeConfig(buf)
	if err != nil {
		return err
	}

	if cfg.Width <= 0 || cfg.Height <= 0 {
		return raise.Error(ErrMalformed)
	}

	return withinAcceptBounds(cfg.Width, cfg.Height)
}

// withinAcceptBounds applies the three accept bounds. The pixel
// bound cannot bind at the current edge bounds; see MaxAcceptPixels.
func withinAcceptBounds(w, h int) error {
	if w > MaxAcceptWidth || h > MaxAcceptHeight {
		return raise.Error(ErrDimensions)
	}

	// int64 so a 32-bit build cannot overflow the product.
	if int64(w)*int64(h) > MaxAcceptPixels {
		return raise.Error(ErrDimensions)
	}

	return nil
}

// decodeConfig reads the header inside the recover boundary. This call itself
// panicked on malformed WebP before x/image v0.43.0 (CVE-2026-46601); go.mod
// pins the floor.
func decodeConfig(buf []byte) (cfg image.Config, err error) {
	defer recoverAsMalformed(&err)

	cfg, _, err = image.DecodeConfig(bytes.NewReader(buf))
	if err != nil {
		return image.Config{}, raise.Errors(ErrMalformed, err)
	}

	return cfg, nil
}

// decode decodes the pixels inside the recover boundary. Defense in depth
// against x/image's recurring panic class (GO-2026-4815); an unrecovered panic
// here is a request-crash primitive. checkDimensions covers the OOM case.
func decode(buf []byte) (img image.Image, err error) {
	defer recoverAsMalformed(&err)

	img, _, err = image.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, raise.Errors(ErrMalformed, err)
	}

	return img, nil
}

// recoverAsMalformed turns a decoder panic into ErrMalformed. The recovered
// value is dropped, not wrapped: panic text carries offsets derived from the
// crafted input, and §3.8 forbids echoing caller-supplied strings.
func recoverAsMalformed(err *error) {
	if recover() != nil {
		*err = raise.Error(ErrMalformed)
	}
}

// scale reduces img to the canonical bounds, or returns it unchanged.
// Factor is min(1, shortBound/short, longBound/long): never scales up, preserves
// aspect, never crops.
//
// Destination type follows alpha, so transparent pixels reach the PNG encoder
// unpremultiplied.
func scale(img image.Image, alpha bool) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	width, height := canonicalSize(w, h)
	if width == w && height == h {
		return img
	}

	rect := image.Rect(0, 0, width, height)

	var dst draw.Image
	if alpha {
		dst = image.NewNRGBA(rect)
	} else {
		dst = image.NewRGBA(rect)
	}

	scaler.Scale(dst, rect, img, b, draw.Src, nil)

	return dst
}

// canonicalSize returns the dimensions w x h scales down to.
func canonicalSize(w, h int) (width, height int) {
	short, long := min(w, h), max(w, h)

	factor := min(
		1.0,
		float64(MaxCanonicalShortEdge)/float64(short),
		float64(MaxCanonicalLongEdge)/float64(long),
	)

	if factor >= 1 {
		return w, h
	}

	return clampEdge(w, factor), clampEdge(h, factor)
}

// clampEdge scales one edge, floored at 1 so a 2400x1 image keeps its height.
func clampEdge(edge int, factor float64) int {
	scaled := int(math.Round(float64(edge) * factor))

	return min(max(scaled, 1), MaxCanonicalLongEdge)
}

// encode writes img in the canonical encoding: PNG when the upload carried an
// alpha channel, JPEG otherwise.
func encode(img image.Image, alpha bool) (Image, error) {
	b := img.Bounds()
	out := Image{Width: b.Dx(), Height: b.Dy()}

	var buf bytes.Buffer

	if alpha {
		if err := pngEncoder.Encode(&buf, img); err != nil {
			return Image{}, raise.Error(err)
		}

		out.ContentType, out.Ext = ContentTypePNG, ExtPNG
	} else {
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: JPEGQuality}); err != nil {
			return Image{}, raise.Error(err)
		}

		out.ContentType, out.Ext = ContentTypeJPEG, ExtJPEG
	}

	out.Bytes = buf.Bytes()

	return out, nil
}

// hasAlpha reports whether any pixel is not fully opaque, which picks the
// canonical encoding.
//
// Asks the pixels, not the color model: Go returns *image.RGBA for
// color-type-2 PNG (no alpha in the file, ~10x larger stored losslessly), and
// alpha-carrying WebP decodes to *image.NYCbCrA (easy to miss, transparency
// destroyed). Both mislead; Opaque() does not.
//
// A type that cannot answer is assumed to carry alpha: wrong that way costs
// size, the other way costs pixels.
func hasAlpha(img image.Image) bool {
	if o, ok := img.(interface{ Opaque() bool }); ok {
		return !o.Opaque()
	}

	return true
}
