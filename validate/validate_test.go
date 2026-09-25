package validate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPasswordValidator(t *testing.T) {
	p := NewPasswordValidator()
	res := map[string]*RuleResult{}
	for _, r := range p.Validate("abc").Results {
		res[r.Name] = r
	}

	t.Run("Length Rule", func(t *testing.T) {
		assert.False(t, res["8 characters"].Valid)
	})

	t.Run("Uppercase Rule", func(t *testing.T) {
		assert.False(t, res["capital letter"].Valid)
	})

	t.Run("Number Rule", func(t *testing.T) {
		assert.False(t, res["number"].Valid)
	})

	t.Run("Symbol Rule", func(t *testing.T) {
		assert.False(t, res["special character"].Valid)
	})

	t.Run("No Repetition Rule", func(t *testing.T) {
		assert.True(t, res["no repetition"].Valid)
	})

	t.Run("All Rules", func(t *testing.T) {
		res := p.Validate("Abc@123xyz")
		assert.True(t, res.Valid)
	})

	t.Run("All Rules negative", func(t *testing.T) {
		res := p.Validate("Abc@123")
		assert.False(t, res.Valid)
	})
}

func TestImageSrc(t *testing.T) {
	// The instance's own origins, as a caller's config spells them.
	auth := []string{"https://auth.example.com"}
	both := []string{"https://auth.example.com", "https://assets.example.com:8443"}

	tests := []struct {
		name    string
		input   string
		origins []string
		valid   bool
	}{
		{"foreign https rejected", "https://cdn.example.com/logo.svg", auth, false},
		{"Same-origin relative path", "/logo.svg", nil, true},
		{"Nested relative path", "/static/auth/default-logo.svg", nil, true},
		{"Empty allowed", "", nil, true},
		{"http on localhost", "http://localhost:8888/static/auth/logo.svg", nil, true},
		{"http on a .localhost subdomain", "http://portal.localhost:8888/logo.svg", nil, true},
		{"http on 127.0.0.1", "http://127.0.0.1:8888/logo.svg", nil, true},
		{"http on ::1", "http://[::1]:8888/logo.svg", nil, true},
		{"https on localhost", "https://localhost:8888/logo.svg", []string{"https://localhost:8888"}, true},
		{"own auth origin over https", "https://auth.example.com/static/auth/favicon.ico", auth, true},
		{"own asset origin carrying a port", "https://assets.example.com:8443/assets/x.png", both, true},
		{"own origin over plain http", "http://assets.internal/assets/x.png", []string{"http://assets.internal"}, true},
		{"same host on another port rejected", "https://auth.example.com:9443/x.png", auth, false},
		{"same host on another scheme rejected", "http://auth.example.com/x.png", auth, false},
		{"host merely suffixed by the origin rejected", "https://evil-auth.example.com/x.png", auth, false},
		{"userinfo rejected", "https://user@auth.example.com/x.png", auth, false},
		{"unparsable origin ignored", "https://auth.example.com/x.png", []string{"://nope"}, false},
		{"absolute https with no origins rejected", "https://auth.example.com/x.png", nil, false},
		{"Protocol-relative rejected", "//evil.example.com/logo.svg", auth, false},
		{"http rejected", "http://cdn.example.com/logo.svg", auth, false},
		{"localhost-suffixed host rejected", "http://localhost.evil.com/logo.svg", nil, false},
		{"localhost-prefixed host rejected", "http://evil-localhost.com/logo.svg", nil, false},
		{"non-loopback IP rejected", "http://10.0.0.1/logo.svg", nil, false},
		{"over-length loopback URL rejected", "http://localhost/" + strings.Repeat("a", MaxImageURLLen), nil, false},
		{"over-length own-origin URL rejected", "https://auth.example.com/" + strings.Repeat("a", MaxImageURLLen), auth, false},
		{"data URI rejected", "data:image/svg+xml,<svg/>", auth, false},
		{"javascript rejected", "javascript:alert(1)", auth, false},
		{"bare filename rejected", "logo.svg", auth, false},
		{"over-length path rejected", "/" + strings.Repeat("a", MaxImageURLLen), nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ImageSrc(tt.input, tt.origins...)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, ErrInvalidImageSrc)
			}
		})
	}
}

func TestLoopbackHost(t *testing.T) {
	tests := []struct {
		host     string
		loopback bool
	}{
		{"localhost", true},
		{"portal.localhost", true},
		{"tenant.portal.localhost", true},
		{"127.0.0.1", true},
		{"127.1.2.3", true},
		{"::1", true},
		{"localhost.evil.com", false},
		{"evil-localhost.com", false},
		{"notlocalhost", false},
		{"10.0.0.1", false},
		{"example.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			assert.Equal(t, tt.loopback, LoopbackHost(tt.host))
		})
	}
}

func TestPublicHost(t *testing.T) {
	tests := []struct {
		host   string
		public bool
	}{
		{"example.com", true},
		{"auth.example.com", true},
		{"sign-in.acme.co.uk", true},
		{"EXAMPLE.COM", true},
		{"ip-127-0-0-1.example.com", true},
		{"localhost.evil.com", true},
		{"example.com.", true},
		// Unregistrable in practice, but outside the deny list: kept as rows so a
		// change of mind is a one-line edit.
		{"intranet", true},
		{"box.localdomain", true},
		{"abc.onion", true},
		{"1.0.0.127.in-addr.arpa", true},

		{"localhost", false},
		{"portal.localhost", false},
		{"127.0.0.1", false},
		{"127.1.2.3", false},
		{"::1", false},
		{"[::1]", false},
		{"10.0.0.1", false},
		{"192.168.1.10", false},
		{"169.254.169.254", false},
		{"172.16.0.1", false},
		{"8.8.8.8", false},
		{"fd00::1", false},
		{"fe80::1", false},
		{"[2001:db8::1]", false},
		{"printer.local", false},
		{"metadata.internal", false},
		{"router.home.arpa", false},
		{"home.arpa", false},
		{"local", false},
		{"foo.test", false},
		{"foo.invalid", false},
		{"foo.example", false},
		{"printer.local.", false},
		{"localhost.", false},
		{"", false},
		{"   ", false},
		{".", false},
	}

	names := map[string]string{"": "empty", "   ": "whitespace"}
	for _, tt := range tests {
		name := tt.host
		if n, ok := names[name]; ok {
			name = n
		}

		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.public, PublicHost(tt.host))
		})
	}
}
