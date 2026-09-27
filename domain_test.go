package x

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidHostname(t *testing.T) {
	tests := []struct {
		name  string
		input string
		err   error
	}{
		{"Valid hostname", "example.com", nil},
		{"Empty hostname", "", ErrMinLen},
		{"Long hostname", "a." + strings.Repeat("a", 252), ErrHostnameMaxLen},
		{"Invalid label in hostname", "test.-example.com", ErrInvalidDNSLabel},
		{"Label too long", strings.Repeat("a", 64), ErrMaxLen},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsValidHostname(tt.input)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestIsValidDNSLabel(t *testing.T) {
	tests := []struct {
		name  string
		input string
		err   error
	}{
		{"Valid label", "example", nil},
		{"Label starts with hyphen", "-example", ErrInvalidDNSLabel},
		{"Label ends with hyphen", "example-", ErrInvalidDNSLabel},
		{"Empty label", "", ErrMinLen},
		{"Label too long", strings.Repeat("a", 64), ErrMaxLen},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsValidDNSLabel(tt.input)
			assert.Equal(t, tt.err, err)
		})
	}
}

func TestIsValidSubdomain(t *testing.T) {
	// Reuse the DNSLabel tests since it's effectively the same function
	tests := []struct {
		name  string
		input string
		err   error
	}{
		{"Valid subdomain", "sub", nil},
		{"Subdomain starts with hyphen", "-sub", ErrInvalidDNSLabel},
		{"Subdomain ends with hyphen", "sub-", ErrInvalidDNSLabel},
		{"Empty subdomain", "", ErrMinLen},
		{"Subdomain too long", strings.Repeat("a", 64), ErrMaxLen},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsValidSubdomain(tt.input)
			assert.Equal(t, tt.err, err)
		})
	}
}

func TestURLWithSubdomain(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		subdomain string
		want      string
		wantErr   bool
	}{
		{"Valid URL with subdomain", "http://example.com", "sub", "http://sub.example.com", false},
		{"Valid URL for localhost", "http://localhost:8080", "sub", "http://sub.localhost:8080", false},
		{"Invalid URL", ":", "sub", "", true},
		{"Invalid subdomain", "http://example.com", "-sub", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := URLWithSubdomain(tt.uri, tt.subdomain)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, u.String())
			}
		})
	}
}

func TestCanonicalOrigin(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"Bare origin unchanged", "https://app.example.com", "https://app.example.com", false},
		{"Trailing slash stripped", "https://app.example.com/", "https://app.example.com", false},
		{"Path stripped", "https://app.example.com/dashboard", "https://app.example.com", false},
		{"Query and fragment stripped", "https://app.example.com/x?a=1#top", "https://app.example.com", false},
		{"Host lowercased", "https://App.Example.COM", "https://app.example.com", false},
		{"Scheme lowercased", "HTTPS://app.example.com", "https://app.example.com", false},
		{"Default https port dropped", "https://app.example.com:443", "https://app.example.com", false},
		{"Default http port dropped", "http://app.example.com:80", "http://app.example.com", false},
		{"Custom port kept", "https://app.example.com:8443", "https://app.example.com:8443", false},
		{"Localhost port kept", "http://localhost:3000", "http://localhost:3000", false},
		{"Userinfo stripped", "https://user:pass@app.example.com", "https://app.example.com", false},
		{"IPv6 default port dropped keeps brackets", "https://[::1]:443", "https://[::1]", false},
		{"Surrounding whitespace trimmed", " https://app.example.com ", "https://app.example.com", false},
		{"No scheme", "app.example.com", "", true},
		{"No host", "https://", "", true},
		{"Empty", "", "", true},
		{"Unparseable", "https://exa mple.com", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CanonicalOrigin(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateSubdomain(t *testing.T) {
	require.Error(t, ValidateSubdomain(""), "Empty domain")
	require.Error(t, ValidateSubdomain("example.com"), "Should be a subdomain")
	require.Error(t, ValidateSubdomain("aaaa"), "Too short")
	require.Error(t, ValidateSubdomain(strings.Repeat("a", 64)), "Too long")

	assert.NoError(t, ValidateSubdomain("test-domain"), "Valid subdomain")
}

func TestParseDomain(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "Invalid domain",
			input:   "://invalid",
			wantErr: true,
		},
		{
			name:  "Domain only",
			input: "example.com",
			want:  "https://example.com",
		},
		{
			name:  "Domain with port",
			input: "example.com:8080",
			want:  "https://example.com:8080",
		},
		{
			name:  "Domain with protocol",
			input: "http://example.com",
			want:  "https://example.com", // Enforce https
		},
		{
			name:  "Domain with protocol and port",
			input: "https://example.com:8080",
			want:  "https://example.com:8080",
		},
		{
			name:  "Trailing slash is dropped",
			input: "example.com/",
			want:  "https://example.com",
		},
		{
			name:    "Credentials",
			input:   "user@example.com",
			wantErr: true,
		},
		{
			name:    "Credentials in front of an IP literal",
			input:   "x@10.20.30.40",
			wantErr: true,
		},
		{
			name:    "Path",
			input:   "example.com/path",
			wantErr: true,
		},
		{
			name:    "Query",
			input:   "example.com?x=1",
			wantErr: true,
		},
		{
			name:    "Fragment",
			input:   "example.com#x",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := ParseDomain(tt.input)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, d)
				assert.Equal(t, tt.want, d.String())
			}
		})
	}
}

func TestHostname(t *testing.T) {
	assert.Equal(t, "example.com", Hostname("https://example.com"), "Valid URL")
	assert.Equal(t, "sub.example.com", Hostname("https://sub.example.com"), "Valid URL with Subdomain")
	assert.Equal(t, "example.com", Hostname("https://example.com:8080"), "Valid URL with Domain and Port")
	assert.Equal(t, "sub.example.com", Hostname("https://sub.example.com:9090"), "Valid URL with Subdomain and Port")

	assert.Empty(t, Hostname(""), "Empty URL")
}

func TestSubdomain(t *testing.T) {
	assert.Equal(t, "sub", Subdomain("https://sub.example.com"), "Valid URL with Subdomain")
	assert.Equal(t, "multi", Subdomain("https://multi.sub.example.com"), "Valid URL with Multiple Subdomains")
	assert.Equal(t, "sub", Subdomain("https://sub.example.com:9090"), "Valid URL with Subdomain and Port")

	assert.Equal(t, "example", Subdomain("https://example.com"), "Valid URL without Subdomain")
	assert.Empty(t, Subdomain(""), "Empty URL")
}

func TestIsSameHostname(t *testing.T) {
	assert.True(t, IsSameHostname("https://example.com:8443", "http://example.com"))
	assert.False(t, IsSameHostname("https://example.com", "https://example.org"))
}

func TestURLToPath(t *testing.T) {
	assert.Equal(t, "https://example.com", URLToPath("https://example.com"))
	assert.Equal(t, "https://example.com/a/b", URLToPath("https://example.com", "a", "/b"))
}
