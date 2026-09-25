package imagex

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"testing"

	"github.com/stretchr/testify/require"
)

// Fixtures Go's encoders cannot produce, built here rather than committed.
//
// Two binaries are committed under testdata/ because nothing in Go can write
// them: a real WebP and a truncated copy. Everything else, EXIF included, is
// assembled here, where the diff shows what the bytes are instead of asking a
// reviewer to hexdump a file.

// pngHeaderOnly returns a PNG signature and a valid IHDR chunk declaring w x h,
// and then nothing at all.
//
// This is the fixture the dimension check exists for: the header is truthful
// enough to parse and the body is absent, so a decoder that trusted the header
// would allocate w x h x 4 bytes for pixels that were never sent. Reaching
// ErrDimensions here proves the bound is read from the header; reaching
// ErrMalformed instead would prove it is not.
func pngHeaderOnly(w, h uint32) []byte {
	var buf bytes.Buffer

	buf.Write(magicPNG)

	ihdr := make([]byte, 0, 13)
	ihdr = binary.BigEndian.AppendUint32(ihdr, w)
	ihdr = binary.BigEndian.AppendUint32(ihdr, h)
	ihdr = append(ihdr,
		8, // bit depth
		2, // color type: truecolor
		0, // compression: deflate
		0, // filter: adaptive
		0, // interlace: none
	)

	writePNGChunk(&buf, "IHDR", ihdr)

	return buf.Bytes()
}

// writePNGChunk writes one length-prefixed, CRC-suffixed PNG chunk.
func writePNGChunk(buf *bytes.Buffer, kind string, data []byte) {
	_ = binary.Write(buf, binary.BigEndian, uint32(len(data)))

	body := append([]byte(kind), data...)
	buf.Write(body)

	_ = binary.Write(buf, binary.BigEndian, crc32.ChecksumIEEE(body))
}

// exifMarkerValue is a recognizable string planted inside the EXIF block, so a
// test can assert on the payload surviving or not rather than only on the
// "Exif\x00\x00" header.
const exifMarkerValue = "EXIF-MARKER-MUST-NOT-SURVIVE"

// spliceEXIF returns jpg with an APP1 EXIF segment inserted after the
// start-of-image marker, which is where a camera writes one.
//
// image/jpeg.Encode never writes EXIF, and the decoder skips APP segments it
// does not understand, so this is the only way to get a JPEG that carries
// metadata into the pipeline.
func spliceEXIF(t *testing.T, jpg []byte) []byte {
	t.Helper()

	require.True(t, bytes.HasPrefix(jpg, []byte{0xFF, 0xD8}), "input must start with SOI")

	payload := append([]byte("Exif\x00\x00"), exifMarkerValue...)

	// An APP1 segment is: marker, then a two-byte length that counts itself.
	segment := []byte{0xFF, 0xE1}
	segment = binary.BigEndian.AppendUint16(segment, uint16(len(payload)+2))
	segment = append(segment, payload...)

	out := make([]byte, 0, len(jpg)+len(segment))
	out = append(out, jpg[:2]...)
	out = append(out, segment...)
	out = append(out, jpg[2:]...)

	return out
}
