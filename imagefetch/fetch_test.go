package imagefetch

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testHosts = Hosts{".fbcdn.net", "avatars.githubusercontent.com"}

func TestCheckURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		ok   bool
	}{
		{"https on an exact host", "https://avatars.githubusercontent.com/u/1.png", true},
		{"https on port 443", "https://avatars.githubusercontent.com:443/u/1.png", true},
		{"a subdomain of a dotted entry", "https://scontent.fbcdn.net/a.jpg", true},
		{"the host in upper case", "https://AVATARS.GITHUBUSERCONTENT.COM/u/1.png", true},
		{"http refused", "http://avatars.githubusercontent.com/u/1.png", false},
		{"userinfo refused", "https://user@avatars.githubusercontent.com/u/1.png", false},
		{"an IPv4 literal refused", "https://93.184.216.34/a.png", false},
		{"an IPv6 literal refused", "https://[2606:2800::1]/a.png", false},
		{"another port refused", "https://avatars.githubusercontent.com:8443/u/1.png", false},
		{"a host off the list refused", "https://cdn.example.com/a.png", false},
		// The leading dot is what makes this a subdomain rule and not a suffix
		// match on the string.
		{"a host merely suffixed refused", "https://evil-fbcdn.net/a.jpg", false},
		{"the bare parent of a dotted entry refused", "https://fbcdn.net/a.jpg", false},
		{"no host refused", "https:///a.png", false},
	}

	c := newClient(Options{}, nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.raw)
			require.NoError(t, err)

			err = c.checkURL(u, testHosts)
			if tt.ok {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, ErrRefused)
			}
		})
	}
}

// The name checked has to be the name dialed. A spelling that only folds onto
// an allowlisted name is refused rather than rewritten, because the request and
// the dial both use the URL as given.
func TestCheckURLRefusesNamesThatOnlyFold(t *testing.T) {
	c := newClient(Options{}, nil)

	t.Run("the ascii form of an IDN is accepted", func(t *testing.T) {
		u, err := url.Parse("https://xn--e1afmkfd.example.com/a.png")
		require.NoError(t, err)

		assert.NoError(t, c.checkURL(u, Hosts{"xn--e1afmkfd.example.com"}))
	})

	// Each of these normalizes onto an allowlisted name while being a different
	// string on the wire.
	folds := []struct {
		name string
		raw  string
	}{
		{"the unicode spelling of its own punycode", "https://\u043f\u0440\u0438\u043c\u0435\u0440.example.com/a.png"},
		{"a fullwidth letter", "https://\uff41vatars.githubusercontent.com/u/1.png"},
		{"a soft hyphen", "https://evil.com\u00ad.googleusercontent.com/a.png"},
		{"an ideographic full stop", "https://evil.com\u3002googleusercontent.com/a.png"},
		{"a trailing dot", "https://avatars.githubusercontent.com./u/1.png"},
	}

	hosts := Hosts{"xn--e1afmkfd.example.com", "avatars.githubusercontent.com", ".googleusercontent.com"}
	for _, tt := range folds {
		t.Run(tt.name+" is refused", func(t *testing.T) {
			u, err := url.Parse(tt.raw)
			require.NoError(t, err)

			assert.ErrorIs(t, c.checkURL(u, hosts), ErrRefused)
		})
	}
}

// A dotted entry is a subdomain rule, so it needs a label in front of it.
func TestHostsMatchNeedsALabel(t *testing.T) {
	hosts := Hosts{".googleusercontent.com"}

	assert.True(t, hosts.Match("lh3.googleusercontent.com"))
	assert.False(t, hosts.Match(".googleusercontent.com"))
	assert.False(t, hosts.Match("googleusercontent.com"))
}

func TestRefusedAddr(t *testing.T) {
	refused := []string{
		"127.0.0.1", "::1", "::ffff:127.0.0.1", "::ffff:10.0.0.1",
		"10.0.0.1", "172.16.0.1", "192.168.1.1",
		"169.254.169.254", "fe80::1", "fc00::1",
		"100.64.1.1", "192.0.0.1", "198.18.0.1",
		"2002::1", "2001::1", "64:ff9b::1",
		"::", "0.0.0.0", "224.0.0.1", "ff02::1",
		"0.1.2.3", "192.88.99.1", "240.0.0.1", "255.255.255.255",
		"::7f00:1", "64:ff9b:1::1",
	}
	allowed := []string{"93.184.216.34", "2606:2800:220:1:248:1893:25c8:1946", "8.8.8.8"}

	c := newClient(Options{}, nil)
	for _, raw := range refused {
		t.Run(raw+" refused", func(t *testing.T) {
			addr, err := netip.ParseAddr(raw)
			require.NoError(t, err)
			assert.True(t, c.refusedAddr(addr))
		})
	}

	for _, raw := range allowed {
		t.Run(raw+" allowed", func(t *testing.T) {
			addr, err := netip.ParseAddr(raw)
			require.NoError(t, err)
			assert.False(t, c.refusedAddr(addr))
		})
	}
}

// The point of resolving inside the dialer: the name is on the allowlist and
// passes every URL rule, and it is still refused because of what it resolves
// to. A hostname check alone would have dialed this.
func TestFetchRefusesAtDialTime(t *testing.T) {
	var asked string
	lookup := func(_ context.Context, host string) ([]netip.Addr, error) {
		asked = host

		return []netip.Addr{netip.MustParseAddr("10.0.0.1")}, nil
	}

	c := newClient(Options{}, lookup)

	_, err := c.Fetch(context.Background(), "https://avatars.githubusercontent.com/u/1.png", testHosts)

	require.ErrorIs(t, err, ErrRefused)
	assert.Equal(t, "avatars.githubusercontent.com", asked, "the refusal must come after resolution")
}

// One public answer does not redeem a private one; the target is refused whole.
func TestFetchRefusesAMixedAnswer(t *testing.T) {
	lookup := func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{
			netip.MustParseAddr("93.184.216.34"),
			netip.MustParseAddr("127.0.0.1"),
		}, nil
	}

	c := newClient(Options{}, lookup)

	_, err := c.Fetch(context.Background(), "https://avatars.githubusercontent.com/u/1.png", testHosts)

	assert.ErrorIs(t, err, ErrRefused)
}

func TestFetchRefusesAnEmptyAnswer(t *testing.T) {
	lookup := func(context.Context, string) ([]netip.Addr, error) { return nil, nil }
	c := newClient(Options{}, lookup)

	_, err := c.Fetch(context.Background(), "https://avatars.githubusercontent.com/u/1.png", testHosts)

	assert.ErrorIs(t, err, ErrRefused)
}

// The escape hatch is off unless the caller asks for it, which is what keeps it
// out of a production build.
func TestLoopbackIsRefusedByDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("the server must not be reached")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := newClient(Options{}, nil)

	_, err := c.Fetch(context.Background(), server.URL+"/a.png", testHosts)

	assert.ErrorIs(t, err, ErrRefused)
}

// The ceiling errors rather than truncating: handing the pipeline a cut image
// would look like a valid small one.
func TestBodyRefusesPastTheCeiling(t *testing.T) {
	b := &body{rc: io.NopCloser(strings.NewReader("two bytes over")), cancel: func() {}, left: 12}

	_, err := io.ReadAll(b)

	assert.ErrorIs(t, err, ErrTooLarge)
}

// A body inside the ceiling reads to EOF untouched.
func TestBodyPassesUnderTheCeiling(t *testing.T) {
	const payload = "some image bytes"
	b := &body{rc: io.NopCloser(strings.NewReader(payload)), cancel: func() {}, left: MaxBodyBytes}

	got, err := io.ReadAll(b)

	require.NoError(t, err)
	assert.Equal(t, payload, string(got))
}

func TestTransportPinsTLS12(t *testing.T) {
	c := newClient(Options{}, nil)

	assert.Equal(t, uint16(tls.VersionTLS12), c.transport.TLSClientConfig.MinVersion)
	assert.Nil(t, c.transport.Proxy, "an environment proxy would bypass the dial check")
}

// The response cases, over a real loopback server the relaxation admits.
func TestFetchOverLoopback(t *testing.T) {
	const png = "\x89PNG\r\n\x1a\nfake bytes"

	mux := http.NewServeMux()
	mux.HandleFunc("/ok.png", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = io.WriteString(w, png)
	})
	mux.HandleFunc("/missing.png", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/page.html", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<html>")
	})
	mux.HandleFunc("/untyped", func(w http.ResponseWriter, _ *http.Request) {
		w.Header()["Content-Type"] = nil
		_, _ = io.WriteString(w, png)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// /hop?n=k redirects k more times, then lands on the image.
	mux.HandleFunc("/hop", func(w http.ResponseWriter, r *http.Request) {
		left, _ := strconv.Atoi(r.URL.Query().Get("n"))
		if left <= 0 {
			http.Redirect(w, r, server.URL+"/ok.png", http.StatusFound)

			return
		}

		http.Redirect(w, r, server.URL+"/hop?n="+strconv.Itoa(left-1), http.StatusFound)
	})
	mux.HandleFunc("/offsite", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://evil.example.com/x.png", http.StatusFound)
	})

	c := newClient(Options{AllowLoopbackHTTP: true}, nil)
	fetch := func(path string) (io.ReadCloser, error) {
		return c.Fetch(context.Background(), server.URL+path, testHosts)
	}

	t.Run("a 200 image streams back", func(t *testing.T) {
		body, err := fetch("/ok.png")
		require.NoError(t, err)
		defer func() { assert.NoError(t, body.Close()) }()

		got, err := io.ReadAll(body)
		require.NoError(t, err)
		assert.Equal(t, png, string(got))
	})

	t.Run("a missing image is a status error", func(t *testing.T) {
		_, err := fetch("/missing.png")
		assert.ErrorIs(t, err, ErrStatus)
	})

	t.Run("a non-image content type is refused", func(t *testing.T) {
		_, err := fetch("/page.html")
		assert.ErrorIs(t, err, ErrRefused)
	})

	t.Run("no content type is left to the pipeline", func(t *testing.T) {
		body, err := fetch("/untyped")
		require.NoError(t, err)
		defer func() { assert.NoError(t, body.Close()) }()
	})

	t.Run("three redirects still arrive", func(t *testing.T) {
		// n=2 is two more hops after the first, so three redirects in all.
		body, err := fetch("/hop?n=2")
		require.NoError(t, err)
		defer func() { assert.NoError(t, body.Close()) }()

		got, err := io.ReadAll(body)
		require.NoError(t, err)
		assert.Equal(t, png, string(got))
	})

	t.Run("a fourth redirect is refused", func(t *testing.T) {
		_, err := fetch("/hop?n=3")
		assert.ErrorIs(t, err, ErrRedirects)
	})

	// The relaxation admits loopback; it does not admit being forwarded off it.
	t.Run("a redirect off the allowlist is refused", func(t *testing.T) {
		_, err := fetch("/offsite")
		assert.ErrorIs(t, err, ErrRefused)
	})
}
