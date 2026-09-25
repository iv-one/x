package imagex

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A DIFF HERE ON A TOOLCHAIN OR DEPENDENCY BUMP IS EXPECTED DRIFT, NOT A BUG.
//
// These digests pin the exact bytes the pipeline produces. They depend on the
// standard library's PNG and JPEG encoders and on the Catmull-Rom scaler in
// golang.org/x/image, none of which promise stable output across releases, and
// the standard library encoders have in fact changed before.
//
// When one of these fails after an upgrade and nothing in this package changed,
// the correct response is to read the new value and update it here, not to hunt
// for a regression. Operationally the drift means content uploaded afterwards
// gets new identifiers, every existing asset keeps the identifier it has and
// goes on serving, and the store simply holds a mix.
// Nothing breaks; the only cost is that a re-upload of an old picture no longer
// deduplicates against it.
//
// The test exists to make that moment visible and deliberate. Deleting it would
// not make the drift stop happening, only stop it being noticed.
var golden = map[string]struct {
	build func(t *testing.T) []byte
	// digest of the canonical bytes
	digest string
	// the properties that must NOT drift: these are contract, not encoder
	// output, and a change to any of them is a real regression
	contentType   string
	width, height int
}{
	"opaque jpeg, scaled down": {
		build: func(t *testing.T) []byte {
			t.Helper()
			return encodeJPEG(t, gradient(1000, 1000))
		},
		digest:      "850a759c7b8c73ba4a9ce3a07ba68b8756a80061260669e50e925aedf488902c",
		contentType: ContentTypeJPEG,
		width:       512,
		height:      512,
	},
	"transparent png, untouched": {
		build: func(t *testing.T) []byte {
			t.Helper()
			return encodePNG(t, transparent(64, 48))
		},
		digest:      "bce08a0773157ec78a996d0e24dd40423f9656b49ed5ffd15fcff446790a5124",
		contentType: ContentTypePNG,
		width:       64,
		height:      48,
	},
	// Pins the third decoder and the alpha path through it: a lossy WebP with an
	// ALPH chunk decodes to NYCbCrA and comes out as a PNG.
	"webp with alpha, re-encoded as png": {
		build: func(t *testing.T) []byte {
			t.Helper()
			return readFixture(t, "alpha.webp")
		},
		digest:      "faf8b8e755ff0647e977e202ec526a270535fe36e3816e188b735bc1d6526645",
		contentType: ContentTypePNG,
		width:       64,
		height:      48,
	},
}

func TestGoldenCanonicalBytes(t *testing.T) {
	for name, tc := range golden {
		t.Run(name, func(t *testing.T) {
			got, err := canonical(t, tc.build(t))
			require.NoError(t, err)

			// Contract first: these must hold whatever the encoder does.
			assert.Equal(t, tc.contentType, got.ContentType)
			assert.Equal(t, tc.width, got.Width)
			assert.Equal(t, tc.height, got.Height)

			sum := sha256.Sum256(got.Bytes)

			assert.Equal(t, tc.digest, hex.EncodeToString(sum[:]),
				"canonical bytes changed; if the toolchain or golang.org/x/image moved, "+
					"this is expected encoder drift and the value above should be updated "+
					"(see the comment on `golden`), not investigated as a regression")
		})
	}
}
