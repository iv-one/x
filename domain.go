package x

import (
	"errors"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/iv-one/x/errorsx/raise"
)

// TODO: think to move it to validate package
var (
	// ErrInvalidDomain is the error for invalid domain.
	ErrInvalidDomain = errors.New("domain is invalid")
	// ErrInvalidSubdomain is the error for invalid sub-domain.
	ErrInvalidSubdomain = errors.New("sub-domain is invalid")
	// ErrInvalidDNSLabel is the error for invalid DNS label.
	ErrInvalidDNSLabel = errors.New("DNS label is invalid")
	// ErrMaxLen is the error for max length.
	ErrMaxLen = errors.New("max length is 63")
	// ErrMinLen is the error for min length.
	ErrMinLen = errors.New("min length is 1")
	// ErrHostnameMaxLen is the error for hostname max length.
	ErrHostnameMaxLen = errors.New("hostname max length is 253")
	// ErrDomainNotBare is the error for a domain carrying more than a host and port.
	ErrDomainNotBare = errors.New("domain must be a host name with an optional port: no credentials, path, query, or fragment")
	// ErrInvalidURI is the error for invalid URI.
	ErrInvalidURI = errors.New("the provided URI is invalid or malformed. Please check the URI format and try again")
	// ErrSubdomainMinLen is the error for sub-domain min length.
	ErrSubdomainMinLen = errors.New("min length is 5")
)

// IsValidHostname validates if the given string is a valid DNS host name.
func IsValidHostname(host string) error {
	if len(host) > 253 {
		return ErrHostnameMaxLen
	}

	labels := strings.SplitSeq(host, ".")
	for label := range labels {
		if err := IsValidDNSLabel(label); err != nil {
			return raise.Error(err)
		}
	}

	return nil
}

// IsValidDNSLabel validates if the given string is a valid DNS label.
func IsValidDNSLabel(label string) error {
	if len(label) > 63 {
		return ErrMaxLen
	}

	if len(label) < 1 {
		return ErrMinLen
	}

	pattern := `^[A-Za-z0-9](?:[A-Za-z0-9\-]{0,61}[A-Za-z0-9])$`
	matched, _ := regexp.MatchString(pattern, label)
	if !matched {
		return ErrInvalidDNSLabel
	}

	return nil
}

// IsValidSubdomain validates if the given string is a valid sub-domain.
func IsValidSubdomain(domain string) error {
	return IsValidDNSLabel(domain)
}

// URLWithSubdomain returns a URL with the given sub-domain.
func URLWithSubdomain(uri string, subdomain string) (*url.URL, error) {
	if err := IsValidSubdomain(subdomain); err != nil {
		return nil, raise.Error(err)
	}

	u, err := url.Parse(uri)
	if err != nil {
		return nil, raise.Error(ErrInvalidURI)
	}

	u.Host = subdomain + "." + u.Host
	return u, nil
}

// ParseDomain parses a domain string into a URL.
// Custom domain could be:
// - domain name (e.g. example.com)
// - domain name with port (e.g. example.com:8080)
// - domain name with protocol (and port) (e.g. https://example.com or https://example.com:8080)
//
// Anything beyond scheme, host, and port is an error: credentials, a path
// (a lone trailing slash is tolerated), a query, or a fragment. Callers read
// the host back out of the result, and each of those would otherwise ride
// along inside it.
func ParseDomain(domainStr string) (*url.URL, error) {
	domain := strings.ToLower(domainStr)
	if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
		domain = "https://" + domain
	}
	uri, err := url.Parse(domain)
	if err != nil {
		return nil, ErrInvalidDomain
	}

	if uri.User != nil || (uri.Path != "" && uri.Path != "/") || uri.RawQuery != "" || uri.Fragment != "" {
		return nil, ErrDomainNotBare
	}

	uri.Scheme = "https"
	uri.Path = ""
	hostname := uri.Hostname()

	if err := IsValidHostname(hostname); err != nil {
		return nil, raise.Error(err)
	}

	return uri, nil
}

// ValidateSubdomain validates if the given string is a valid sub-domain.
func ValidateSubdomain(domain string) error {
	if domain == "" {
		return ErrInvalidSubdomain
	}

	if len(domain) > 63 {
		return ErrMaxLen
	}

	if len(domain) < 5 {
		return ErrSubdomainMinLen
	}

	if err := IsValidSubdomain(domain); err != nil {
		if errors.Is(err, ErrInvalidDNSLabel) {
			return ErrInvalidSubdomain
		}
		return raise.Error(err)
	}

	return nil
}

// CanonicalOrigin returns the canonical web origin of rawURL: lowercase
// scheme and host, default port dropped, and no path, query, fragment, or
// userinfo — the exact form a browser sends in the Origin header. It errors
// when rawURL does not parse or lacks a scheme or host.
func CanonicalOrigin(rawURL string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", raise.Error(ErrInvalidURI)
	}

	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Host)
	if scheme == "" || host == "" {
		return "", raise.Error(ErrInvalidURI)
	}

	if h, p, err := net.SplitHostPort(host); err == nil {
		if (scheme == "http" && p == "80") || (scheme == "https" && p == "443") {
			// Re-bracket IPv6 hosts, which SplitHostPort unwraps.
			if strings.Contains(h, ":") {
				h = "[" + h + "]"
			}
			host = h
		}
	}

	return scheme + "://" + host, nil
}

/* ------------------
Domain functions
------------------ */

// Hostname extracts the hostname from a URL string.
// It does not include the port number. The function does not validate the URL
// format or Hostname format. rawURL is expected to be a valid URL or Host.
func Hostname(rawURL string) string {
	res := strings.TrimPrefix(rawURL, "http://")
	res = strings.TrimPrefix(res, "https://")

	if strings.Contains(res, ":") {
		h, _, _ := net.SplitHostPort(res)
		return h
	}

	return res
}

// Subdomain extracts the subdomain from a URL string.
func Subdomain(rawURL string) string {
	hostname := Hostname(rawURL)
	parts := strings.Split(hostname, ".")

	if len(parts) == 0 {
		return ""
	}

	return parts[0]
}

// IsSameHostname checks if the given hostnames are the same.
func IsSameHostname(a, b string) bool {
	return Hostname(a) == Hostname(b)
}

// URLToPath returns a URL with the given path.
func URLToPath(endpoint string, path ...string) string {
	if len(path) == 0 {
		return endpoint
	}

	sb := strings.Builder{}
	sb.WriteString(endpoint)
	for _, p := range path {
		if !strings.HasPrefix(p, "/") {
			sb.WriteString("/")
		}
		sb.WriteString(p)
	}
	return sb.String()
}
