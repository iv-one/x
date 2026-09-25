// Package validate checks user-supplied values: passwords, emails, image URLs and hosts.
package validate

import (
	"errors"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"

	"github.com/iv-one/x"
	"github.com/iv-one/x/errorsx"
	"github.com/iv-one/x/errorsx/raise"
)

var (
	// ErrEmptyEmail is the error for empty email.
	ErrEmptyEmail = errors.New("email is empty")
	// ErrEmptyPassword is the error for empty password.
	ErrEmptyPassword = errors.New("password is empty")
	// ErrInvalidEmail is the error for invalid email.
	ErrInvalidEmail = errors.New("invalid email")
	// ErrInvalidPassword is the error for invalid password.
	ErrInvalidPassword = errors.New("invalid password")

	// ErrInvalidEmailFormat is the error for invalid email format.
	ErrInvalidEmailFormat = errorsx.ErrBadRequest.WithReason("Invalid Email Format: The provided email address does not meet the required format standards. Please check and correct the email address.")
	// ErrMissingPassword is the error for missing password.
	ErrMissingPassword = errorsx.ErrBadRequest.WithReason("Password field is empty. Please provide a password.")
	// ErrInvalidPasswordStrength is the error for invalid password strength.
	ErrInvalidPasswordStrength = errorsx.ErrBadRequest.WithReason("Invalid Password: The provided password does not meet the required strength criteria. Please use a stronger password.")
	// ErrInvalidImageSrc is the error for an invalid image source.
	ErrInvalidImageSrc = errorsx.ErrBadRequest.WithReason("Invalid Image Source: must be a URL on this instance's own origin, an http URL on a loopback host, or a same-origin path starting with '/', at most 2048 characters.")
)

// MaxImageURLLen caps image source length. Browsers and CDNs commonly
// bound URLs near 2 KB; anything longer is more likely payload than address.
const MaxImageURLLen = 2048

// PasswordStrength is the password strength.
type PasswordStrength int

const (
	// PasswordStrengthNone is the password strength none.
	PasswordStrengthNone PasswordStrength = iota
	// PasswordStrengthWeak is the password strength weak.
	PasswordStrengthWeak
	// PasswordStrengthMedium is the password strength medium.
	PasswordStrengthMedium
	// PasswordStrengthStrong is the password strength strong.
	PasswordStrengthStrong
)

// RuleResult is the rule result.
type RuleResult struct {
	Valid   bool
	Name    string
	Message string
}

// Rule is the rule function.
type Rule func(x.SensitiveStr) *RuleResult

// PasswordValidator is the password validator.
type PasswordValidator struct {
	Rules []Rule
}

// ValidationResult is the validation result.
type ValidationResult struct {
	Valid   bool
	Results []*RuleResult
}

// NewPasswordValidator creates a new password validator.
func NewPasswordValidator() *PasswordValidator {
	p := &PasswordValidator{
		Rules: []Rule{
			LengthRule,
			UppercaseRule,
			NumberRule,
			SymbolRule,
			NoRepetitionRule,
		},
	}

	return p
}

// Validate validates a password.
func (p *PasswordValidator) Validate(password x.SensitiveStr) *ValidationResult {
	results := make([]*RuleResult, 0, len(p.Rules))
	valid := true

	for _, rule := range p.Rules {
		result := rule(password)
		results = append(results, result)
		valid = valid && result.Valid
	}
	return &ValidationResult{
		Valid:   valid,
		Results: results,
	}
}

// LengthRule is the length rule.
var LengthRule = func() Rule {
	return func(password x.SensitiveStr) *RuleResult {
		valid := len(password) >= 8
		var message string
		if !valid {
			message = "Must be at least 8 characters"
		}
		return &RuleResult{
			Valid:   valid,
			Name:    "8 characters",
			Message: message,
		}
	}
}()

// UppercaseRule is the uppercase rule.
var UppercaseRule = func() Rule {
	return func(password x.SensitiveStr) *RuleResult {
		valid := regexp.MustCompile(`[A-Z]`).MatchString(password.Unwrap())
		var message string
		if !valid {
			message = "Must include at least one uppercase letter"
		}
		return &RuleResult{
			Valid:   valid,
			Name:    "capital letter",
			Message: message,
		}
	}
}()

// NumberRule is the number rule.
var NumberRule = func() Rule {
	return func(password x.SensitiveStr) *RuleResult {
		valid := regexp.MustCompile(`[0-9]`).MatchString(password.Unwrap())
		var message string
		if !valid {
			message = "Must include at least one number"
		}
		return &RuleResult{
			Valid:   valid,
			Name:    "number",
			Message: message,
		}
	}
}()

// SymbolRule is the symbol rule.
var SymbolRule = func() Rule {
	return func(password x.SensitiveStr) *RuleResult {
		valid := regexp.MustCompile(`[!@#$%^&*]`).MatchString(password.Unwrap())
		var message string
		if !valid {
			message = "Must include at least one symbol"
		}
		return &RuleResult{
			Valid:   valid,
			Name:    "special character",
			Message: message,
		}
	}
}()

// NoRepetitionRule is the no repetition rule.
var NoRepetitionRule = func() Rule {
	return func(password x.SensitiveStr) *RuleResult {
		valid := !regexp.MustCompile(`(.)\\1{2}`).MatchString(password.Unwrap())
		var message string
		if !valid {
			message = "Must not include repeated characters"
		}
		return &RuleResult{
			Valid:   valid,
			Name:    "no repetition",
			Message: message,
		}
	}
}()

// Password validates a password.
func Password(password x.Sensitive, strength PasswordStrength) error {
	if password.Empty() {
		return ErrEmptyPassword
	}

	// TODO: implement password strength(s)
	if strength == PasswordStrengthNone {
		return nil
	}

	if strength == PasswordStrengthWeak {
		if len(password) < 8 {
			return ErrInvalidPassword
		}
		return nil
	}

	validator := NewPasswordValidator()
	res := validator.Validate(x.SensitiveStr(password))

	if !res.Valid {
		return ErrInvalidPassword
	}

	return nil
}

// Email validates an email.
func Email(email string) error {
	if email == "" {
		return ErrEmptyEmail
	}
	if err := validEmail(email); err != nil {
		return raise.Errors(ErrInvalidEmail, err)
	}
	return nil
}

func validEmail(email string) error {
	_, err := mail.ParseAddress(email)
	return raise.Error(err)
}

// ImageSrc validates a configured image source: empty, a same-origin relative path — one starting with a single
// "/" ("//host" is protocol-relative and stays external) — an http URL on a
// loopback host, or an absolute URL on one of origins. The loopback case is
// what lets a local install render its own images, whose host is plain http
// when the install runs locally.
//
// origins are this instance's own image origins, as the caller's config spells
// them; with none, only the first three branches pass. The rule is the origin,
// not the scheme: an https install's own images arrive as absolute https URLs,
// and a foreign host's do too.
func ImageSrc(raw string, origins ...string) error {
	// Empty clears the source. A sanitizer may run over every config load, so
	// refusing it would warn about each source never set.
	if raw == "" {
		return nil
	}

	// One cap for every branch below; each of them parses or scans raw.
	if len(raw) > MaxImageURLLen {
		return raise.Error(ErrInvalidImageSrc)
	}

	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		return nil
	}

	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raise.Error(ErrInvalidImageSrc)
	}

	if loopbackHTTP(u) || onOwnOrigin(u, origins) {
		return nil
	}

	return raise.Error(ErrInvalidImageSrc)
}

// onOwnOrigin reports whether u is an http or https URL whose scheme and host
// (port included) match one of origins exactly. Ports are compared as
// configured, not normalized: both sides come from the same strings.
func onOwnOrigin(u *url.URL, origins []string) bool {
	if u.User != nil {
		return false
	}
	if !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		return false
	}

	for _, origin := range origins {
		if origin == "" {
			continue
		}

		own, err := url.Parse(origin)
		if err != nil || own.Host == "" {
			continue
		}

		if strings.EqualFold(u.Scheme, own.Scheme) && strings.EqualFold(u.Host, own.Host) {
			return true
		}
	}

	return false
}

// loopbackHTTP reports whether u is an http URL addressed to a loopback host.
func loopbackHTTP(u *url.URL) bool {
	return strings.EqualFold(u.Scheme, "http") && LoopbackHost(u.Hostname())
}

// LoopbackHost reports whether host is a genuine loopback target: "localhost",
// a "*.localhost" subdomain (RFC 6761 §6.3), or a loopback IP (127.0.0.0/8,
// ::1). It takes a bare hostname, so a spoof that only reads as local —
// "localhost.evil.com", "evil-localhost.com" — does not match.
func LoopbackHost(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}

	return false
}

// specialUseSuffixes are names that never resolve to a host anyone can register
// on the public internet: the special-use names of RFC 6761 §6, mDNS
// (RFC 6762 §3), the home network zone of RFC 8375, and ICANN's reserved
// `.internal`. A suffix matches the whole name or any name under it.
var specialUseSuffixes = []string{
	"localhost",
	"local",
	"internal",
	"test",
	"invalid",
	"example",
	"home.arpa",
}

// PublicHost reports whether host is a name that can be registered and resolved
// on the public internet. It rejects IP literals, loopback targets, and
// special-use suffixes. It takes a hostname rather than a URL — a bracketed
// IPv6 literal and a trailing root dot are tolerated — so it rejects
// "127.0.0.1" but not the ordinary name "ip-127-0-0-1.example.com".
//
// It answers from the name alone: no lookup is performed, because the address a
// name resolves to at registration time is not the one it will resolve to when
// the host is served.
func PublicHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	// A fully qualified name carries the root label as a trailing dot; without
	// this the suffix comparison would run against an empty last label.
	host = strings.TrimSuffix(host, ".")
	if host == "" {
		return false
	}

	// An IP literal is never a registrable name, which covers the private and
	// link-local ranges without enumerating them.
	if net.ParseIP(strings.Trim(host, "[]")) != nil {
		return false
	}

	if LoopbackHost(host) {
		return false
	}

	for _, suffix := range specialUseSuffixes {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return false
		}
	}

	return true
}
