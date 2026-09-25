package imagefetch

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/iv-one/x/errorsx/raise"
	"github.com/iv-one/x/validate"
	"golang.org/x/net/idna"
)

var (
	// ErrRefused reports a target this package will not fetch: a scheme, port,
	// host or resolved address outside what the caller allowed.
	ErrRefused = errors.New("imagefetch: target refused")

	// ErrStatus reports a response whose status was not 200.
	ErrStatus = errors.New("imagefetch: unexpected status")

	// ErrTooLarge reports a body that ran past MaxBodyBytes.
	ErrTooLarge = errors.New("imagefetch: body over the ceiling")

	// ErrRedirects reports a chain longer than MaxRedirects.
	ErrRedirects = errors.New("imagefetch: too many redirects")
)

const (
	// DialTimeout bounds the TCP connect and the TLS handshake.
	DialTimeout = 5 * time.Second

	// RequestTimeout bounds one whole fetch, redirects included.
	RequestTimeout = 10 * time.Second

	// MaxRedirects is how many hops a provider CDN may take before the fetch is
	// abandoned.
	MaxRedirects = 3

	// MaxBodyBytes is a ceiling, not the size limit, which the caller's ingest
	// pipeline enforces. This one errors rather than truncating: an operator may
	// tune the limit above it,
	// and a truncating cap would hand the pipeline a valid-but-cut image
	// instead of an overflow.
	MaxBodyBytes int64 = 32 << 20
)

// refusedPrefixes are the ranges netip.Addr has no predicate for: "this
// network" (0.1.2.3 reaches local on Linux, and IsUnspecified catches only
// 0.0.0.0), CGNAT, IETF protocol assignments, benchmarking, 6to4 relay
// anycast, reserved space, 6to4, Teredo, IPv4-compatible IPv6 (which Unmap
// does not fold, so ::7f00:1 would otherwise pass), and both NAT64 prefixes.
var refusedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/96"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("2001::/32"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
}

// Hosts is a caller's allowlist. An entry starting with "." matches any
// subdomain of it, so ".fbcdn.net" admits scontent.fbcdn.net and rejects
// evil-fbcdn.net; any other entry matches exactly.
type Hosts []string

// Match reports whether host is on the allowlist. host is expected lowercased
// and in its ASCII form.
func (h Hosts) Match(host string) bool {
	for _, entry := range h {
		if strings.HasPrefix(entry, ".") {
			// len, so ".googleusercontent.com" needs a label before the dot and
			// does not match itself or "..googleusercontent.com".
			if len(host) > len(entry) && strings.HasSuffix(host, entry) {
				return true
			}

			continue
		}

		if host == entry {
			return true
		}
	}

	return false
}

// Fetcher is the one operation this package offers, as an interface so a caller
// under test injects its own.
type Fetcher interface {
	Fetch(ctx context.Context, rawURL string, hosts Hosts) (io.ReadCloser, error)
}

// Options configures a Client.
type Options struct {
	// AllowLoopbackHTTP lets an http URL on a loopback host bypass the scheme,
	// port and allowlist rules, and admits a loopback address at dial. It
	// exists so the testing build can fetch from its own fake OIDC issuer, and
	// the caller decides: this package has no build tags of its own.
	//
	// The address half is not scoped to the relaxed URL: with it on, an https
	// allowlisted host resolving to loopback is dialed too. Every other refused
	// range stays refused either way. Do not set it outside a test binary.
	AllowLoopbackHTTP bool
}

// lookupFunc resolves a hostname. Injected so a test can drive the dial-time
// address check without a hostname that really resolves.
type lookupFunc func(ctx context.Context, host string) ([]netip.Addr, error)

// Client is the hardened fetcher. Build one with New and share it: the
// transport, and so the connection pool, is built once.
type Client struct {
	opts      Options
	lookup    lookupFunc
	transport *http.Transport
}

// New returns a Client configured by opts.
func New(opts Options) *Client { return newClient(opts, nil) }

// newClient is New with an injectable resolver, for this package's own tests.
func newClient(opts Options, lookup lookupFunc) *Client {
	if lookup == nil {
		lookup = func(ctx context.Context, host string) ([]netip.Addr, error) {
			return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		}
	}

	c := &Client{opts: opts, lookup: lookup}
	c.transport = &http.Transport{
		// Nil on purpose: an environment proxy would carry the request past the
		// dial check, which is the whole defense.
		Proxy:               nil,
		DialContext:         c.dial,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
		TLSHandshakeTimeout: DialTimeout,
		ForceAttemptHTTP2:   true,
		// A zero-valued Transport never expires an idle connection, and a CDN
		// answering on hundreds of hostnames would then hold a socket per host
		// for the process lifetime. http.DefaultTransport's bounds, which a
		// literal does not inherit.
		IdleConnTimeout:     90 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 2,
	}

	return c
}

// Fetch GETs rawURL and returns its body as a stream. The caller closes it,
// which also releases the fetch's deadline.
//
// It returns ErrRefused for a target the rules reject, at the URL or at the
// resolved address; ErrRedirects for a chain past MaxRedirects; ErrStatus for a
// response that was not 200; and ErrTooLarge, from a Read, for a body past
// MaxBodyBytes.
func (c *Client) Fetch(ctx context.Context, rawURL string, hosts Hosts) (io.ReadCloser, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, raise.Error(ErrRefused)
	}

	if err := c.checkURL(u, hosts); err != nil {
		return nil, raise.Error(err)
	}

	// Its own deadline, never the request context's: a caller's context may
	// outlive what this fetch is allowed to cost.
	ctx, cancel := context.WithTimeout(ctx, RequestTimeout)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		cancel()

		return nil, raise.Error(err)
	}
	req.Header.Set("Accept", "image/*")

	client := &http.Client{
		Transport: c.transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// via holds the requests already made, so this admits exactly
			// MaxRedirects hops and refuses the one after.
			if len(via) > MaxRedirects {
				return ErrRedirects
			}

			return c.checkURL(req.URL, hosts)
		},
	}

	// Every early exit below has to release the deadline, and two of them a
	// body; one closure so no new exit can forget either.
	bail := func(res *http.Response, err error) (io.ReadCloser, error) {
		if res != nil {
			_ = res.Body.Close()
		}
		cancel()

		return nil, raise.Error(err)
	}

	res, err := client.Do(req) //nolint:bodyclose // closed by bail on every early exit, else owned by the returned body
	if err != nil {
		return bail(nil, err)
	}

	if res.StatusCode != http.StatusOK {
		return bail(res, ErrStatus)
	}

	// A cheap fail-fast only. The ingest pipeline sniffs the bytes and is the
	// authority on what this actually is.
	if ct := res.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "image/") {
		return bail(res, ErrRefused)
	}

	return &body{rc: res.Body, cancel: cancel, left: MaxBodyBytes}, nil
}

// body ties the deadline to the stream, so closing the stream releases it, and
// fails the read rather than truncating once the byte budget is spent.
type body struct {
	rc     io.ReadCloser
	cancel context.CancelFunc
	left   int64
}

func (b *body) Read(p []byte) (int, error) {
	n, err := b.rc.Read(p)
	b.left -= int64(n)

	if b.left < 0 {
		return n, ErrTooLarge
	}

	return n, err
}

func (b *body) Close() error {
	err := b.rc.Close()
	b.cancel()

	return err
}

// checkURL applies the target rules, to the first request and to every redirect
// hop. Refusing a redirect is what stops an allowed host from forwarding the
// fetch somewhere it could not have been sent directly.
func (c *Client) checkURL(u *url.URL, hosts Hosts) error {
	if c.relaxed(u) {
		return nil
	}

	if !strings.EqualFold(u.Scheme, "https") || u.User != nil {
		return ErrRefused
	}

	if port := u.Port(); port != "" && port != "443" {
		return ErrRefused
	}

	host := u.Hostname()
	if host == "" {
		return ErrRefused
	}

	// An IP literal skips DNS, and with it the reason the caller's allowlist is
	// a meaningful check at all.
	if _, err := netip.ParseAddr(host); err == nil {
		return ErrRefused
	}

	// A host whose ASCII form differs from its own spelling is refused, not
	// rewritten: the request and the dial use the URL as given, so rewriting
	// would check one name and dial another (the package comment has the rest).
	ascii, err := idna.Lookup.ToASCII(host)
	if err != nil || !strings.EqualFold(ascii, host) {
		return ErrRefused
	}

	if !hosts.Match(strings.ToLower(host)) {
		return ErrRefused
	}

	return nil
}

// relaxed reports whether opts admits this URL as the testing build's loopback
// escape hatch.
func (c *Client) relaxed(u *url.URL) bool {
	return c.opts.AllowLoopbackHTTP &&
		strings.EqualFold(u.Scheme, "http") &&
		validate.LoopbackHost(u.Hostname())
}

// dial resolves the host, refuses the whole target if any answer is an address
// this package will not talk to, then dials the address it checked. Dialing the
// literal is what closes DNS rebinding: a name checked before resolution can
// resolve to something else by the time it is dialed. TLS still verifies
// against the URL's hostname, because DialContext replaces only the TCP step.
func (c *Client) dial(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, raise.Error(err)
	}

	addrs, err := c.resolve(ctx, host)
	if err != nil {
		return nil, raise.Error(err)
	}

	// All or nothing: a host answering one public and one private address is
	// refused outright rather than raced down to the acceptable one.
	if len(addrs) == 0 {
		return nil, raise.Error(ErrRefused)
	}
	for _, addr := range addrs {
		if c.refusedAddr(addr) {
			return nil, raise.Error(ErrRefused)
		}
	}

	dialer := &net.Dialer{Timeout: DialTimeout}

	return dialer.DialContext(ctx, network, net.JoinHostPort(addrs[0].String(), port))
}

// resolve answers the addresses host stands for. A literal, which reaches
// here only under the loopback relaxation, stands for itself.
func (c *Client) resolve(ctx context.Context, host string) ([]netip.Addr, error) {
	if addr, err := netip.ParseAddr(host); err == nil {
		return []netip.Addr{addr}, nil
	}

	return c.lookup(ctx, host)
}

// refusedAddr reports whether addr is one this package will not connect to.
func (c *Client) refusedAddr(addr netip.Addr) bool {
	// Unmap first: an IPv4-mapped ::ffff:127.0.0.1 satisfies none of the v4
	// predicates while still naming loopback.
	addr = addr.Unmap()

	if !addr.IsValid() {
		return true
	}

	if addr.IsLoopback() {
		return !c.opts.AllowLoopbackHTTP
	}

	if addr.IsPrivate() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsInterfaceLocalMulticast() ||
		addr.IsMulticast() ||
		addr.IsUnspecified() {
		return true
	}

	for _, prefix := range refusedPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}

	return false
}
