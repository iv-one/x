package imagex

import "bytes"

// Magic bytes of the accepted input formats. The set is defined by what is
// listed here, which is how SVG, ICO, GIF, TIFF and BMP
// are refused without enumerating them.
var (
	magicPNG  = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	magicJPEG = []byte{0xFF, 0xD8, 0xFF}
	magicRIFF = []byte("RIFF")
	magicWEBP = []byte("WEBP")
)

// webpTagOffset is where the "WEBP" tag sits in a RIFF container, after the
// four-byte "RIFF" marker and the four-byte chunk length.
const webpTagOffset = 8

// accepted reports whether b's content is a format this package will decode.
//
// Decided here rather than by image.DecodeConfig, which reports image.ErrFormat
// alike for an unsupported format and a corrupted supported one, which
// callers answer with 415 and 400 respectively. Deciding first makes that structural.
//
// Given nothing but the bytes, so a filename, extension or caller-supplied
// Content-Type cannot influence it (§3.5).
func accepted(b []byte) bool {
	switch {
	case bytes.HasPrefix(b, magicPNG), bytes.HasPrefix(b, magicJPEG):
		return true
	case bytes.HasPrefix(b, magicRIFF):
		return len(b) >= webpTagOffset+len(magicWEBP) &&
			bytes.Equal(b[webpTagOffset:webpTagOffset+len(magicWEBP)], magicWEBP)
	}

	return false
}
