package imagex

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math/rand/v2"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// maxBytes is a cap generous enough that only the tests about the cap ever hit
// it.
const maxBytes = 5 << 20

// gradient builds an opaque image whose pixels vary in both directions, so a
// scaler has something to do and a lossy encoder has something to lose.
func gradient(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{
				R: uint8(x % 256), //nolint:gosec // deliberate wrap, this is a test pattern
				G: uint8(y % 256), //nolint:gosec // deliberate wrap, this is a test pattern
				B: 0x80,
				A: 0xFF,
			})
		}
	}

	return img
}

// transparent builds an image with genuinely non-opaque pixels.
func transparent(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))

	for y := range h {
		for x := range w {
			img.Set(x, y, color.NRGBA{R: 0xFF, G: 0x00, B: 0x00, A: uint8(x % 256)}) //nolint:gosec // test pattern
		}
	}

	return img
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))

	return buf.Bytes()
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 92}))

	return buf.Bytes()
}

func canonical(t *testing.T, b []byte) (Image, error) {
	t.Helper()

	return Canonical(bytes.NewReader(b), maxBytes)
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()

	b, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err, "fixture %s should exist", name)

	return b
}

// Each accepted input format produces canonical bytes that decode, with the
// dimensions reported on the result.
func TestCanonicalAcceptedFormats(t *testing.T) {
	tests := map[string]struct {
		input           []byte
		wantContentType string
		wantExt         string
		wantW, wantH    int
	}{
		"png with alpha": {
			input:           encodePNG(t, transparent(64, 48)),
			wantContentType: ContentTypePNG,
			wantExt:         ExtPNG,
			wantW:           64,
			wantH:           48,
		},
		"jpeg": {
			input:           encodeJPEG(t, gradient(64, 48)),
			wantContentType: ContentTypeJPEG,
			wantExt:         ExtJPEG,
			wantW:           64,
			wantH:           48,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := canonical(t, tc.input)
			require.NoError(t, err)

			assert.Equal(t, tc.wantContentType, got.ContentType)
			assert.Equal(t, tc.wantExt, got.Ext)
			assert.Equal(t, tc.wantW, got.Width)
			assert.Equal(t, tc.wantH, got.Height)

			decoded, format, err := image.Decode(bytes.NewReader(got.Bytes))
			require.NoError(t, err, "canonical bytes must decode")
			assert.Equal(t, tc.wantW, decoded.Bounds().Dx())
			assert.Equal(t, tc.wantH, decoded.Bounds().Dy())
			assert.Contains(t, []string{"png", "jpeg"}, format)
		})
	}
}

// WebP is accepted on input and never emitted: no encoder for it exists in the
// standard library, so an upload of one comes back as a PNG or a JPEG.
func TestCanonicalWebPInNeverOut(t *testing.T) {
	got, err := canonical(t, readFixture(t, "valid.webp"))
	require.NoError(t, err)

	assert.Equal(t, ContentTypeJPEG, got.ContentType, "an opaque webp stores as a jpeg")
	assert.Equal(t, 64, got.Width)
	assert.Equal(t, 48, got.Height)

	_, gotFormat, err := image.Decode(bytes.NewReader(got.Bytes))
	require.NoError(t, err)
	assert.NotEqual(t, "webp", gotFormat, "webp must never be emitted")
}

// A lossy WebP carrying an ALPH chunk keeps its transparency.
//
// This is the case that decides how the encoding is chosen. Such a file decodes
// to *image.NYCbCrA, a color model that a model-based alpha test is easy to
// omit; omitting it stores the image as a JPEG and composites the transparency
// onto whatever the encoder felt like, silently and irreversibly.
func TestCanonicalWebPWithAlphaStaysTransparent(t *testing.T) {
	raw := readFixture(t, "alpha.webp")

	decoded, _, err := image.Decode(bytes.NewReader(raw))
	require.NoError(t, err)
	require.IsType(t, &image.NYCbCrA{}, decoded,
		"the fixture must exercise the alpha-carrying webp representation")

	got, err := canonical(t, raw)
	require.NoError(t, err)

	assert.Equal(t, ContentTypePNG, got.ContentType,
		"a transparent upload must not be flattened into a jpeg")
	assert.Equal(t, ExtPNG, got.Ext)

	stored, _, err := image.Decode(bytes.NewReader(got.Bytes))
	require.NoError(t, err)

	opaque, ok := stored.(interface{ Opaque() bool })
	require.True(t, ok)
	assert.False(t, opaque.Opaque(), "the stored image must still carry transparency")
}

// An opaque PNG stores as a JPEG.
//
// The other half of the same decision. Go's PNG decoder returns an *image.RGBA
// for color-type 2, which has no alpha channel in the file at all, so a
// model-based test would send every opaque photograph down the lossless path and
// store it many times larger than necessary.
func TestCanonicalOpaquePNGStoresAsJPEG(t *testing.T) {
	// Noise rather than a gradient: a smooth synthetic pattern compresses so
	// well losslessly that it would hide the very cost this test is about. Noise
	// is what a photograph looks like to an encoder.
	raw := encodePNG(t, noise(1024, 1024))

	decoded, _, err := image.Decode(bytes.NewReader(raw))
	require.NoError(t, err)
	require.IsType(t, &image.RGBA{}, decoded,
		"the fixture must exercise the alpha-modeled but opaque representation")

	got, err := canonical(t, raw)
	require.NoError(t, err)

	require.Equal(t, ContentTypeJPEG, got.ContentType,
		"an opaque png must not be stored losslessly")

	// The size the model-based reading would have cost, on the same pixels.
	stored, _, err := image.Decode(bytes.NewReader(got.Bytes))
	require.NoError(t, err)

	assert.Less(t, len(got.Bytes), len(encodePNG(t, stored)),
		"storing an opaque photograph as png costs multiples of the jpeg")
}

// noise builds an opaque image whose pixels a lossless encoder cannot predict,
// from a fixed seed so the fixture never varies between runs.
func noise(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	rng := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // a test fixture, not a secret

	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{
				R: uint8(rng.UintN(256)), //nolint:gosec // bounded by UintN
				G: uint8(rng.UintN(256)), //nolint:gosec // bounded by UintN
				B: uint8(rng.UintN(256)), //nolint:gosec // bounded by UintN
				A: 0xFF,
			})
		}
	}

	return img
}

// An image within the accept bounds but over the canonical bounds is scaled, not
// rejected, with the aspect ratio preserved and no crop applied.
func TestCanonicalScaling(t *testing.T) {
	tests := map[string]struct {
		w, h         int
		wantW, wantH int
		governing    string
	}{
		"photograph, short edge governs": {w: 4032, h: 3024, wantW: 683, wantH: 512, governing: "short"},
		"wordmark, long edge governs":    {w: 2400, h: 300, wantW: 2048, wantH: 256, governing: "long"},
		"square mark":                    {w: 1000, h: 1000, wantW: 512, wantH: 512, governing: "short"},
		"already inside the bounds":      {w: 400, h: 200, wantW: 400, wantH: 200, governing: "none"},
		"tall wordmark, rotated":         {w: 300, h: 2400, wantW: 256, wantH: 2048, governing: "long"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := canonical(t, encodeJPEG(t, gradient(tc.w, tc.h)))
			require.NoError(t, err)

			assert.Equal(t, tc.wantW, got.Width)
			assert.Equal(t, tc.wantH, got.Height)

			// Aspect preserved to within a pixel, which is all rounding allows.
			in := float64(tc.w) / float64(tc.h)
			out := float64(got.Width) / float64(got.Height)
			assert.InDelta(t, in, out, in/float64(min(got.Width, got.Height)),
				"aspect ratio must survive scaling")

			assert.LessOrEqual(t, min(got.Width, got.Height), MaxCanonicalShortEdge)
			assert.LessOrEqual(t, max(got.Width, got.Height), MaxCanonicalLongEdge)
		})
	}
}

// An image already inside the canonical bounds is never scaled up.
func TestCanonicalNeverScalesUp(t *testing.T) {
	got, err := canonical(t, encodePNG(t, transparent(8, 4)))
	require.NoError(t, err)

	assert.Equal(t, 8, got.Width)
	assert.Equal(t, 4, got.Height)
}

// canonicalSize is the scaling rule on its own, without an encoder in the way.
func TestCanonicalSize(t *testing.T) {
	tests := map[string]struct {
		w, h         int
		wantW, wantH int
	}{
		"photograph":                 {w: 4032, h: 3024, wantW: 683, wantH: 512},
		"wordmark":                   {w: 2400, h: 300, wantW: 2048, wantH: 256},
		"square":                     {w: 1000, h: 1000, wantW: 512, wantH: 512},
		"inside bounds":              {w: 400, h: 200, wantW: 400, wantH: 200},
		"exactly on the short bound": {w: 512, h: 512, wantW: 512, wantH: 512},
		"one pixel tall":             {w: 4000, h: 1, wantW: 2048, wantH: 1},
		"single pixel":               {w: 1, h: 1, wantW: 1, wantH: 1},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			w, h := canonicalSize(tc.w, tc.h)
			assert.Equal(t, tc.wantW, w)
			assert.Equal(t, tc.wantH, h)
			assert.GreaterOrEqual(t, w, 1, "an edge must never round to zero")
			assert.GreaterOrEqual(t, h, 1, "an edge must never round to zero")
		})
	}
}

// countingReader records how much was actually pulled from the source, so the
// cap can be shown to bound the read rather than only the result.
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)

	return n, err
}

// A file over the byte cap is rejected, and no more than the cap plus one byte
// is ever read: the cap bounds the stream, not the parsed result, so an upload
// claiming to be a gigabyte costs the cap in memory and no more.
func TestCanonicalByteCap(t *testing.T) {
	const cap64 = 1024

	big := encodePNG(t, gradient(512, 512))
	require.Greater(t, len(big), cap64, "the fixture must exceed the cap")

	counter := &countingReader{r: bytes.NewReader(big)}

	_, err := Canonical(counter, cap64)
	require.ErrorIs(t, err, ErrTooLarge)

	assert.LessOrEqual(t, counter.n, int64(cap64)+1,
		"no more than the cap plus one byte may be read")

	// Exactly at the cap is accepted: the extra byte is what distinguishes
	// "at the limit" from "over it".
	small := encodePNG(t, transparent(4, 4))

	_, err = Canonical(bytes.NewReader(small), int64(len(small)))
	require.NoError(t, err, "an upload exactly at the cap is within it")

	_, err = Canonical(bytes.NewReader(small), int64(len(small))-1)
	require.ErrorIs(t, err, ErrTooLarge, "one byte over the cap is over it")
}

// Dimensions are refused from the header, before any pixel is decoded. The
// fixture is a real PNG header declaring huge dimensions over a body that stops
// immediately: if the check ran after decoding, this would be a malformed-image
// error or an out-of-memory kill instead.
func TestCanonicalDimensionsRejectedBeforeDecode(t *testing.T) {
	tests := map[string]struct{ w, h uint32 }{
		"width over the accept bound":  {w: MaxAcceptWidth + 1, h: 16},
		"height over the accept bound": {w: 16, h: MaxAcceptHeight + 1},
		"both edges over the bound":    {w: MaxAcceptWidth + 1, h: MaxAcceptHeight + 1},
		"absurd square":                {w: 60000, h: 60000},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := canonical(t, pngHeaderOnly(tc.w, tc.h))
			require.ErrorIs(t, err, ErrDimensions)
		})
	}

	// Headers exactly on the bounds are accepted, so the boundary is where it is
	// claimed to be rather than one short of it. The body is absent, so these
	// fail as malformed — which is itself the proof that the dimension check ran
	// from the header, before any pixel was asked for.
	onBounds := map[string]struct{ w, h uint32 }{
		"widest accepted":  {w: MaxAcceptWidth, h: 16},
		"tallest accepted": {w: 16, h: MaxAcceptHeight},
		"largest accepted": {w: MaxAcceptWidth, h: MaxAcceptHeight},
	}

	for name, tc := range onBounds {
		t.Run(name, func(t *testing.T) {
			_, err := canonical(t, pngHeaderOnly(tc.w, tc.h))
			require.ErrorIs(t, err, ErrMalformed)
			require.NotErrorIs(t, err, ErrDimensions,
				"a header exactly on the bounds is inside them")
		})
	}
}

// The accept bounds on their own.
//
// Note the pixel bound cannot bind through Canonical: 4096x4096 is exactly
// MaxAcceptPixels, so no image with both edges inside the edge bounds can exceed
// it. It is exercised here directly because it is the bound that expresses the
// real concern, decoded bytes in memory, and it is what would hold if either
// edge bound were ever raised.
func TestWithinAcceptBounds(t *testing.T) {
	tests := map[string]struct {
		w, h    int
		wantErr bool
	}{
		"ordinary photograph":        {w: 4032, h: 3024},
		"exactly on the edge bounds": {w: MaxAcceptWidth, h: MaxAcceptHeight},
		"one pixel":                  {w: 1, h: 1},
		"one over on width":          {w: MaxAcceptWidth + 1, h: 1, wantErr: true},
		"one over on height":         {w: 1, h: MaxAcceptHeight + 1, wantErr: true},
		// Unreachable through Canonical today, and the reason the bound stays.
		"pixel count over the bound, edges inside": {
			w: MaxAcceptWidth, h: MaxAcceptPixels/MaxAcceptWidth + 1, wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := withinAcceptBounds(tc.w, tc.h)

			if tc.wantErr {
				require.ErrorIs(t, err, ErrDimensions)
				return
			}

			require.NoError(t, err)
		})
	}
}

// Unsupported formats are refused on their content, with the status that says
// so, rather than being handed to a decoder.
func TestCanonicalUnsupportedFormats(t *testing.T) {
	tests := map[string][]byte{
		"svg":               []byte(`<svg xmlns="http://www.w3.org/2000/svg"><rect/></svg>`),
		"svg with prologue": []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"/>`),
		"ico":               {0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x10, 0x10, 0x00, 0x00},
		"gif":               []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00"),
		"bmp":               []byte("BM\x1e\x00\x00\x00\x00\x00\x00\x00"),
		"tiff":              {0x49, 0x49, 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00},
		"pdf":               []byte("%PDF-1.7\n"),
		"html":              []byte("<!DOCTYPE html><html><body>hi</body></html>"),
		"empty":             {},
		"riff but not webp": []byte("RIFF\x24\x00\x00\x00WAVEfmt "),
		"zip":               {0x50, 0x4B, 0x03, 0x04},
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := canonical(t, input)
			require.ErrorIs(t, err, ErrUnsupportedFormat)
		})
	}
}

// Malformed bytes inside an accepted format are rejected rather than crashing
// the process. A panic here would be a request-crash primitive against the
// ingest endpoint, so the assertion is on not panicking as much as on the error.
func TestCanonicalMalformedNeverPanics(t *testing.T) {
	valid := encodePNG(t, gradient(64, 48))
	validJPEG := encodeJPEG(t, gradient(64, 48))
	webp := readFixture(t, "valid.webp")

	tests := map[string][]byte{
		"truncated png":       valid[:len(valid)/2],
		"truncated jpeg":      validJPEG[:len(validJPEG)/2],
		"truncated webp":      readFixture(t, "truncated.webp"),
		"png header only":     valid[:8],
		"corrupted png body":  corrupt(valid),
		"corrupted jpeg body": corrupt(validJPEG),
		"corrupted webp body": corrupt(webp),
		"png magic, no body":  append([]byte(nil), magicPNG...),
		"jpeg magic, no body": append([]byte(nil), magicJPEG...),
		"webp magic, no body": []byte("RIFF\x00\x00\x00\x00WEBP"),
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() {
				_, err := canonical(t, input)
				require.Error(t, err, "malformed input must be refused")
				require.NotErrorIs(t, err, ErrUnsupportedFormat,
					"the format was recognized, so this is malformed, not unsupported")
			})
		})
	}
}

// corrupt flips the second half of b, leaving the header intact so the bytes
// still sniff as their format.
func corrupt(b []byte) []byte {
	out := append([]byte(nil), b...)
	for i := len(out) / 2; i < len(out); i++ {
		out[i] ^= 0xFF
	}

	return out
}

// A file is judged on its contents, never on a name or a declared type. There is
// no parameter through which a caller could claim otherwise, which is the point:
// the JPEG below would be a PNG if anything but the bytes were consulted.
func TestCanonicalJudgesContentNotClaims(t *testing.T) {
	got, err := canonical(t, encodeJPEG(t, gradient(32, 32)))
	require.NoError(t, err)
	assert.Equal(t, ContentTypeJPEG, got.ContentType)

	got, err = canonical(t, encodePNG(t, transparent(32, 32)))
	require.NoError(t, err)
	assert.Equal(t, ContentTypePNG, got.ContentType)
}

// Metadata in the upload is absent from the canonical bytes. Nothing strips it:
// the encoder is handed a decoded image, so an EXIF block has no route through.
func TestCanonicalDropsEXIF(t *testing.T) {
	withEXIF := spliceEXIF(t, encodeJPEG(t, gradient(64, 48)))

	require.Contains(t, string(withEXIF), "Exif\x00\x00", "the fixture must carry EXIF")
	require.Contains(t, string(withEXIF), exifMarkerValue, "the fixture must carry the marker")

	got, err := canonical(t, withEXIF)
	require.NoError(t, err)

	assert.NotContains(t, string(got.Bytes), "Exif\x00\x00", "no EXIF block may survive")
	assert.NotContains(t, string(got.Bytes), exifMarkerValue, "no EXIF payload may survive")
}

// Encoding the same upload twice in one process yields identical bytes. Without
// this, content addressing does not deduplicate and the same picture uploaded
// twice would occupy two identifiers.
func TestCanonicalDeterministic(t *testing.T) {
	inputs := map[string][]byte{
		"jpeg":           encodeJPEG(t, gradient(800, 600)),
		"png with alpha": encodePNG(t, transparent(64, 48)),
		"webp":           readFixture(t, "valid.webp"),
	}

	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			first, err := canonical(t, input)
			require.NoError(t, err)

			second, err := canonical(t, input)
			require.NoError(t, err)

			assert.Equal(t, first.Bytes, second.Bytes, "canonical bytes must be reproducible")
			assert.Equal(t, first.ContentType, second.ContentType)
		})
	}
}

// hasAlpha asks the pixels, not the color model. The rows that matter are the
// ones where those two answers disagree: an *image.RGBA whose pixels are all
// opaque, which Go's PNG decoder produces for a file with no alpha channel at
// all, and an *image.NYCbCrA, whose model a color-model test tends to miss.
func TestHasAlpha(t *testing.T) {
	transparentRGBA := image.NewRGBA(image.Rect(0, 0, 1, 1))
	transparentRGBA.Set(0, 0, color.RGBA{R: 0x10, A: 0x10})

	tests := map[string]struct {
		img  image.Image
		want bool
	}{
		"opaque rgba, alpha-carrying model": {img: opaqueRGBA(2, 2), want: false},
		"rgba with a transparent pixel":     {img: transparentRGBA, want: true},
		"nrgba with a transparent pixel":    {img: transparent(4, 4), want: true},
		"gray":                              {img: image.NewGray(image.Rect(0, 0, 1, 1)), want: false},
		"ycbcr":                             {img: image.NewYCbCr(image.Rect(0, 0, 1, 1), image.YCbCrSubsampleRatio420), want: false},
		"cmyk":                              {img: image.NewCMYK(image.Rect(0, 0, 1, 1)), want: false},
		"nycbcra, the alpha-carrying webp":  {img: image.NewNYCbCrA(image.Rect(0, 0, 1, 1), image.YCbCrSubsampleRatio420), want: true},
		"paletted, fully opaque": {
			img:  image.NewPaletted(image.Rect(0, 0, 1, 1), color.Palette{color.RGBA{A: 0xFF}}),
			want: false,
		},
		"paletted with a transparent entry": {
			img:  image.NewPaletted(image.Rect(0, 0, 1, 1), color.Palette{color.NRGBA{}}),
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, hasAlpha(tc.img))
		})
	}
}

// opaqueRGBA builds an *image.RGBA with every pixel fully opaque: the shape Go's
// PNG decoder returns for a color-type-2 file, which carries no alpha channel.
func opaqueRGBA(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{R: 0x20, G: 0x40, B: 0x60, A: 0xFF})
		}
	}

	return img
}

// The accept bounds are consistent with one another. When this fails, an edge
// bound has moved and MaxAcceptPixels has become the binding check: the comment
// on it, and TestWithinAcceptBounds' note about unreachability, both need
// revisiting.
func TestAcceptBoundsAgree(t *testing.T) {
	assert.Equal(t, MaxAcceptPixels, MaxAcceptWidth*MaxAcceptHeight,
		"the pixel bound is currently implied by the edge bounds; see MaxAcceptPixels")
}
