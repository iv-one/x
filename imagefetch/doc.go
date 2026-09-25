// Package imagefetch fetches an image from a URL the server did not compose,
// such as the avatar an identity provider returns, so the bytes can be stored
// rather than the URL handed to a browser.
//
// It is deliberately narrow. The target must be https, on a host the caller
// allowlists, resolving to a public address, and the body comes back as a
// stream for the asset pipeline to judge. The address rules follow OWASP's
// Server Side Request Forgery Prevention Cheat Sheet; the reason they are
// applied at dial time rather than to the hostname is that a name checked
// before resolution can resolve to something else by the time it is dialed.
//
// Hostnames are matched as spelled, never normalized. The request and the dial
// both use the URL as given, so a host whose IDN form differs from its own
// spelling (a fullwidth letter, a soft hyphen, an ideographic full stop) would
// be checked under one name and dialed under another; every provider CDN is
// already ASCII, so such a spelling is refused outright.
//
// It is not a general HTTP client and must not grow into one.
package imagefetch
